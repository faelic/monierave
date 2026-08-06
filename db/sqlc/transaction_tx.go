package db

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	BankingTransactionTypeDeposit          = "deposit"
	BankingTransactionTypeWithdrawal       = "withdrawal"
	BankingTransactionTypeInternalTransfer = "internal_transfer"
	BankingTransactionTypeReversal         = "reversal"

	BankingTransactionStatusPending  = "pending"
	BankingTransactionStatusPosted   = "posted"
	BankingTransactionStatusReversed = "reversed"

	IdempotencyOperationInternalTransfer = "internal_transfer"
	idempotencyResponseStatusCreated     = int16(201)

	USDPerTransferLimit = int64(1_000_000)
	USDDailyLimit       = int64(2_500_000)
	EURPerTransferLimit = int64(1_000_000)
	EURDailyLimit       = int64(2_500_000)
)

var (
	ErrInsufficientBalance        = errors.New("insufficient balance")
	ErrIdempotencyConflict        = errors.New("idempotency key was already used with a different request")
	ErrInvalidIdempotencyKey      = errors.New("invalid idempotency key")
	ErrPerTransferLimitExceeded   = errors.New("per-transfer limit exceeded")
	ErrDailyTransferLimitExceeded = errors.New("daily transfer limit exceeded")
)

type TransferTxParams struct {
	FromAccountPublicID pgtype.UUID `json:"from_account_id"`
	ToAccountPublicID   pgtype.UUID `json:"to_account_id"`
	Amount              int64       `json:"amount"`
	Currency            string      `json:"currency"`
	Username            string      `json:"username"`
	Narration           string      `json:"narration"`
	CorrelationID       pgtype.UUID `json:"-"`
}

type TransferTxResult struct {
	Transaction BankingTransaction `json:"transaction"`
	Postings    []LedgerPosting    `json:"postings"`
	FromAccount Account            `json:"from_account"`
	ToAccount   Account            `json:"to_account"`
	Replayed    bool               `json:"-"`
}

type IdempotentTransferTxParams struct {
	TransferTxParams
	IdempotencyKey string
	RequestHash    []byte
}

type DepositTxParams struct {
	AccountPublicID pgtype.UUID
	Amount          int64
	Narration       string
	Actor           string
	CorrelationID   pgtype.UUID
}

type WithdrawTxParams struct {
	AccountPublicID pgtype.UUID
	Amount          int64
	Narration       string
	Actor           string
	CorrelationID   pgtype.UUID
}

type MoneyMovementTxResult struct {
	Transaction BankingTransaction
	Postings    []LedgerPosting
	Account     Account
	AuditLog    AuditLog
}

type pendingTransactionParams struct {
	TransactionType string
	Currency        string
	Amount          int64
	Narration       string
	InitiatedBy     string
	ReversalOf      pgtype.UUID
}

type postingParams struct {
	LedgerAccountID int64
	Amount          int64
}

// TransferTx posts a balanced customer-to-customer transfer while both account
// rows are locked in a stable order.
func (store *SQLStore) TransferTx(
	ctx context.Context,
	arg TransferTxParams,
) (TransferTxResult, error) {
	var result TransferTxResult
	arg.CorrelationID = auditCorrelation(arg.CorrelationID)

	err := store.execTx(ctx, func(q *Queries) error {
		var err error
		result, err = transferWithQueries(ctx, q, arg)
		return err
	})
	if err != nil {
		return TransferTxResult{}, store.auditTransferFailure(ctx, arg, err)
	}
	return result, nil
}

// IdempotentTransferTx commits the key reservation, ledger transaction, cached
// balances, and replay snapshot atomically.
func (store *SQLStore) IdempotentTransferTx(
	ctx context.Context,
	arg IdempotentTransferTxParams,
) (TransferTxResult, error) {
	if arg.IdempotencyKey == "" || len(arg.IdempotencyKey) > 128 ||
		len(arg.RequestHash) != 32 {
		return TransferTxResult{}, ErrInvalidIdempotencyKey
	}

	var result TransferTxResult
	arg.CorrelationID = auditCorrelation(arg.CorrelationID)
	err := store.execTx(ctx, func(q *Queries) error {
		keyParams := DeleteExpiredIdempotencyKeyParams{
			Username:       arg.Username,
			Operation:      IdempotencyOperationInternalTransfer,
			IdempotencyKey: arg.IdempotencyKey,
		}
		if err := q.DeleteExpiredIdempotencyKey(ctx, keyParams); err != nil {
			return err
		}

		_, err := q.CreateIdempotencyKey(ctx, CreateIdempotencyKeyParams{
			Username:       arg.Username,
			Operation:      IdempotencyOperationInternalTransfer,
			IdempotencyKey: arg.IdempotencyKey,
			RequestHash:    arg.RequestHash,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			existing, getErr := q.GetIdempotencyKeyForUpdate(
				ctx,
				GetIdempotencyKeyForUpdateParams(keyParams),
			)
			if getErr != nil {
				return getErr
			}
			if !bytes.Equal(existing.RequestHash, arg.RequestHash) {
				return ErrIdempotencyConflict
			}
			if !existing.TransactionID.Valid || len(existing.ResultSnapshot) == 0 {
				return fmt.Errorf("incomplete idempotency record")
			}
			if unmarshalErr := json.Unmarshal(existing.ResultSnapshot, &result); unmarshalErr != nil {
				return fmt.Errorf("decode idempotency result: %w", unmarshalErr)
			}
			result.Replayed = true
			return nil
		}
		if err != nil {
			return err
		}

		result, err = transferWithQueries(ctx, q, arg.TransferTxParams)
		if err != nil {
			return err
		}
		snapshot, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("encode idempotency result: %w", err)
		}
		_, err = q.CompleteIdempotencyKey(ctx, CompleteIdempotencyKeyParams{
			Username:       arg.Username,
			Operation:      IdempotencyOperationInternalTransfer,
			IdempotencyKey: arg.IdempotencyKey,
			TransactionID:  result.Transaction.ID,
			ResponseStatus: pgtype.Int2{
				Int16: idempotencyResponseStatusCreated,
				Valid: true,
			},
			ResultSnapshot: snapshot,
		})
		return err
	})
	if err != nil {
		return TransferTxResult{}, store.auditTransferFailure(
			ctx,
			arg.TransferTxParams,
			err,
		)
	}
	return result, nil
}

func transferWithQueries(
	ctx context.Context,
	q *Queries,
	arg TransferTxParams,
) (TransferTxResult, error) {
	var result TransferTxResult
	fromAccount, toAccount, err := loadAndLockTransferAccounts(
		ctx,
		q,
		arg.FromAccountPublicID,
		arg.ToAccountPublicID,
	)
	if err != nil {
		return result, err
	}
	if fromAccount.Owner != arg.Username {
		return result, ErrAccountNotOwned
	}
	if fromAccount.Currency != arg.Currency || toAccount.Currency != arg.Currency {
		return result, ErrCurrencyMismatch
	}
	if fromAccount.Status == FinancialAccountStatusFrozen {
		return result, ErrAccountFrozen
	}
	if fromAccount.Status == FinancialAccountStatusClosed ||
		toAccount.Status == FinancialAccountStatusClosed {
		return result, ErrAccountClosed
	}
	perTransferLimit, dailyLimit, err := transferLimits(arg.Currency)
	if err != nil {
		return result, err
	}
	if arg.Amount > perTransferLimit {
		return result, ErrPerTransferLimitExceeded
	}

	fromLedger, err := customerLedgerAccount(ctx, q, fromAccount.ID)
	if err != nil {
		return result, err
	}
	dailyTotal, err := q.GetDailyOutgoingTransferTotal(ctx, fromLedger.ID)
	if err != nil {
		return result, err
	}
	if dailyTotal > dailyLimit-arg.Amount {
		return result, ErrDailyTransferLimitExceeded
	}
	toLedger, err := customerLedgerAccount(ctx, q, toAccount.ID)
	if err != nil {
		return result, err
	}

	transaction, postings, err := createBalancedTransaction(
		ctx,
		q,
		pendingTransactionParams{
			TransactionType: BankingTransactionTypeInternalTransfer,
			Currency:        arg.Currency,
			Amount:          arg.Amount,
			Narration:       arg.Narration,
			InitiatedBy:     arg.Username,
		},
		[]postingParams{
			{LedgerAccountID: fromLedger.ID, Amount: -arg.Amount},
			{LedgerAccountID: toLedger.ID, Amount: arg.Amount},
		},
	)
	if err != nil {
		return result, err
	}
	if _, err := createFinancialAudit(ctx, q, financialAuditParams{
		EntityType:    "banking_transaction",
		EntityID:      transaction.ID,
		CorrelationID: arg.CorrelationID,
		EventType:     "transfer_created",
		Actor:         arg.Username,
		ToState:       BankingTransactionStatusPending,
		Metadata:      transactionAuditMetadata(transaction),
	}); err != nil {
		return result, err
	}

	result.FromAccount, err = subtractAccountBalance(ctx, q, fromAccount.ID, arg.Amount)
	if err != nil {
		return TransferTxResult{}, err
	}
	result.ToAccount, err = q.AddAccountBalanceInternal(ctx, AddAccountBalanceInternalParams{
		ID: toAccount.ID, Amount: arg.Amount,
	})
	if err != nil {
		return TransferTxResult{}, err
	}

	result.Transaction, err = q.MarkBankingTransactionPosted(ctx, transaction.ID)
	result.Postings = postings
	if err != nil {
		return TransferTxResult{}, err
	}
	if _, err := createFinancialAudit(ctx, q, financialAuditParams{
		EntityType:    "banking_transaction",
		EntityID:      result.Transaction.ID,
		CorrelationID: arg.CorrelationID,
		EventType:     "transfer_posted",
		Actor:         arg.Username,
		FromState:     BankingTransactionStatusPending,
		ToState:       BankingTransactionStatusPosted,
		Metadata:      transactionAuditMetadata(result.Transaction),
	}); err != nil {
		return TransferTxResult{}, err
	}
	occurredAt := result.Transaction.PostedAt.Time
	notifications := []financialNotificationParams{
		{
			Username:      result.FromAccount.Owner,
			EventType:     DomainEventTransactionPosted,
			EntityType:    "banking_transaction",
			EntityID:      result.Transaction.ID,
			CorrelationID: arg.CorrelationID,
			Reference:     result.Transaction.Reference,
			Amount:        result.Transaction.Amount,
			Currency:      result.Transaction.Currency,
			Direction:     "outgoing",
			OccurredAt:    occurredAt,
		},
		{
			Username:      result.ToAccount.Owner,
			EventType:     DomainEventTransactionPosted,
			EntityType:    "banking_transaction",
			EntityID:      result.Transaction.ID,
			CorrelationID: arg.CorrelationID,
			Reference:     result.Transaction.Reference,
			Amount:        result.Transaction.Amount,
			Currency:      result.Transaction.Currency,
			Direction:     "incoming",
			OccurredAt:    occurredAt,
		},
	}
	for _, notification := range notifications {
		if _, _, err := createFinancialNotification(ctx, q, notification); err != nil {
			return TransferTxResult{}, err
		}
	}
	return result, nil
}

func (store *SQLStore) auditTransferFailure(
	ctx context.Context,
	arg TransferTxParams,
	transferErr error,
) error {
	eventType := "transfer_failed"
	switch {
	case errors.Is(transferErr, ErrInsufficientBalance):
		eventType = "transfer_rejected_insufficient_funds"
	case errors.Is(transferErr, ErrPerTransferLimitExceeded),
		errors.Is(transferErr, ErrDailyTransferLimitExceeded):
		eventType = "transfer_rejected_limit"
	}
	auditErr := store.execTx(ctx, func(q *Queries) error {
		if _, err := createFinancialAudit(ctx, q, financialAuditParams{
			EntityType:    "transfer_attempt",
			EntityID:      arg.CorrelationID,
			CorrelationID: arg.CorrelationID,
			EventType:     eventType,
			Actor:         arg.Username,
			ToState:       "rejected",
			Message:       transferErr.Error(),
			Metadata: map[string]any{
				"amount":          arg.Amount,
				"currency":        arg.Currency,
				"from_account_id": uuidStringOrEmpty(arg.FromAccountPublicID),
				"to_account_id":   uuidStringOrEmpty(arg.ToAccountPublicID),
			},
		}); err != nil {
			return err
		}
		_, _, err := createFinancialNotification(ctx, q, financialNotificationParams{
			Username:      arg.Username,
			EventType:     DomainEventTransactionFailed,
			EntityType:    "transfer_attempt",
			EntityID:      arg.CorrelationID,
			CorrelationID: arg.CorrelationID,
			Reference: "ATTEMPT-" + strings.ToUpper(strings.ReplaceAll(
				uuidStringOrEmpty(arg.CorrelationID),
				"-",
				"",
			)),
			Amount:     arg.Amount,
			Currency:   arg.Currency,
			Direction:  "outgoing",
			OccurredAt: time.Now().UTC(),
			Reason:     transferNotificationFailureReason(transferErr),
		})
		return err
	})
	if auditErr != nil {
		return errors.Join(transferErr, fmt.Errorf("record transfer failure audit: %w", auditErr))
	}
	return transferErr
}

func transferNotificationFailureReason(err error) string {
	switch {
	case errors.Is(err, ErrInsufficientBalance):
		return "Insufficient funds"
	case errors.Is(err, ErrPerTransferLimitExceeded),
		errors.Is(err, ErrDailyTransferLimitExceeded):
		return "Transfer limit exceeded"
	case errors.Is(err, ErrAccountFrozen):
		return "The sending account is frozen"
	case errors.Is(err, ErrAccountClosed):
		return "An account involved in the transfer is closed"
	default:
		return "The transfer could not be completed"
	}
}

func transferLimits(currency string) (int64, int64, error) {
	switch currency {
	case "USD":
		return USDPerTransferLimit, USDDailyLimit, nil
	case "EUR":
		return EURPerTransferLimit, EURDailyLimit, nil
	default:
		return 0, 0, ErrCurrencyMismatch
	}
}

func (store *SQLStore) DepositTx(
	ctx context.Context,
	arg DepositTxParams,
) (MoneyMovementTxResult, error) {
	return store.settlementMovementTx(
		ctx,
		arg.AccountPublicID,
		arg.Amount,
		arg.Narration,
		BankingTransactionTypeDeposit,
		arg.Actor,
		arg.CorrelationID,
	)
}

func (store *SQLStore) WithdrawTx(
	ctx context.Context,
	arg WithdrawTxParams,
) (MoneyMovementTxResult, error) {
	return store.settlementMovementTx(
		ctx,
		arg.AccountPublicID,
		arg.Amount,
		arg.Narration,
		BankingTransactionTypeWithdrawal,
		arg.Actor,
		arg.CorrelationID,
	)
}

func (store *SQLStore) settlementMovementTx(
	ctx context.Context,
	accountPublicID pgtype.UUID,
	amount int64,
	narration string,
	transactionType string,
	actor string,
	correlationID pgtype.UUID,
) (MoneyMovementTxResult, error) {
	var result MoneyMovementTxResult
	if amount <= 0 {
		return result, fmt.Errorf("amount must be positive")
	}
	if actor == "" {
		actor = "operator_cli"
	}
	correlationID = auditCorrelation(correlationID)

	err := store.execTx(ctx, func(q *Queries) error {
		account, err := q.GetAccountByPublicID(ctx, accountPublicID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAccountNotFound
		}
		if err != nil {
			return err
		}
		account, err = q.GetAccountForUpdate(ctx, account.ID)
		if err != nil {
			return err
		}
		if account.Status == FinancialAccountStatusClosed {
			return ErrAccountClosed
		}
		if transactionType == BankingTransactionTypeWithdrawal &&
			account.Status == FinancialAccountStatusFrozen {
			return ErrAccountFrozen
		}

		customerLedger, err := customerLedgerAccount(ctx, q, account.ID)
		if err != nil {
			return err
		}
		settlementLedger, err := q.GetSettlementLedgerAccount(
			ctx,
			pgtype.Text{String: account.Currency, Valid: true},
		)
		if err != nil {
			return err
		}

		postings := []postingParams{
			{LedgerAccountID: settlementLedger.ID, Amount: -amount},
			{LedgerAccountID: customerLedger.ID, Amount: amount},
		}
		if transactionType == BankingTransactionTypeWithdrawal {
			postings[0].Amount = amount
			postings[1].Amount = -amount
		}

		transaction, createdPostings, err := createBalancedTransaction(
			ctx,
			q,
			pendingTransactionParams{
				TransactionType: transactionType,
				Currency:        account.Currency,
				Amount:          amount,
				Narration:       narration,
				InitiatedBy:     actor,
			},
			postings,
		)
		if err != nil {
			return err
		}
		if _, err := createFinancialAudit(ctx, q, financialAuditParams{
			EntityType:    "banking_transaction",
			EntityID:      transaction.ID,
			CorrelationID: correlationID,
			EventType:     "operator_transaction_created",
			Actor:         actor,
			ToState:       BankingTransactionStatusPending,
			Metadata:      transactionAuditMetadata(transaction),
		}); err != nil {
			return err
		}

		if transactionType == BankingTransactionTypeWithdrawal {
			result.Account, err = subtractAccountBalance(ctx, q, account.ID, amount)
		} else {
			result.Account, err = q.AddAccountBalanceInternal(ctx, AddAccountBalanceInternalParams{
				ID: account.ID, Amount: amount,
			})
		}
		if err != nil {
			return err
		}

		result.Transaction, err = q.MarkBankingTransactionPosted(ctx, transaction.ID)
		result.Postings = createdPostings
		if err != nil {
			return err
		}
		result.AuditLog, err = createFinancialAudit(ctx, q, financialAuditParams{
			EntityType:    "banking_transaction",
			EntityID:      result.Transaction.ID,
			CorrelationID: correlationID,
			EventType:     "operator_transaction_posted",
			Actor:         actor,
			FromState:     BankingTransactionStatusPending,
			ToState:       BankingTransactionStatusPosted,
			Metadata:      transactionAuditMetadata(result.Transaction),
		})
		if err != nil {
			return err
		}
		direction := "incoming"
		if transactionType == BankingTransactionTypeWithdrawal {
			direction = "outgoing"
		}
		_, _, err = createFinancialNotification(ctx, q, financialNotificationParams{
			Username:      result.Account.Owner,
			EventType:     DomainEventTransactionPosted,
			EntityType:    "banking_transaction",
			EntityID:      result.Transaction.ID,
			CorrelationID: correlationID,
			Reference:     result.Transaction.Reference,
			Amount:        result.Transaction.Amount,
			Currency:      result.Transaction.Currency,
			Direction:     direction,
			OccurredAt:    result.Transaction.PostedAt.Time,
		})
		return err
	})
	if err != nil {
		return MoneyMovementTxResult{}, err
	}
	return result, nil
}

func createBalancedTransaction(
	ctx context.Context,
	q *Queries,
	arg pendingTransactionParams,
	postings []postingParams,
) (BankingTransaction, []LedgerPosting, error) {
	if arg.Amount <= 0 {
		return BankingTransaction{}, nil, fmt.Errorf("amount must be positive")
	}
	if len(postings) < 2 {
		return BankingTransaction{}, nil, fmt.Errorf("at least two postings are required")
	}

	var total int64
	for _, posting := range postings {
		if posting.Amount == 0 {
			return BankingTransaction{}, nil, fmt.Errorf("posting amount cannot be zero")
		}
		total += posting.Amount
	}
	if total != 0 {
		return BankingTransaction{}, nil, fmt.Errorf("postings must balance to zero")
	}

	id, reference := newTransactionIdentity()
	transaction, err := q.CreateBankingTransaction(ctx, CreateBankingTransactionParams{
		ID:              id,
		Reference:       reference,
		TransactionType: arg.TransactionType,
		Currency:        arg.Currency,
		Amount:          arg.Amount,
		Narration:       strings.TrimSpace(arg.Narration),
		InitiatedBy:     arg.InitiatedBy,
		ReversalOf:      arg.ReversalOf,
	})
	if err != nil {
		return BankingTransaction{}, nil, err
	}

	created := make([]LedgerPosting, 0, len(postings))
	for _, posting := range postings {
		value, err := q.CreateLedgerPosting(ctx, CreateLedgerPostingParams{
			TransactionID:   transaction.ID,
			LedgerAccountID: posting.LedgerAccountID,
			Amount:          posting.Amount,
		})
		if err != nil {
			return BankingTransaction{}, nil, err
		}
		created = append(created, value)
	}
	return transaction, created, nil
}

func customerLedgerAccount(
	ctx context.Context,
	q *Queries,
	accountID int64,
) (LedgerAccount, error) {
	return q.GetCustomerLedgerAccount(
		ctx,
		pgtype.Int8{Int64: accountID, Valid: true},
	)
}

func subtractAccountBalance(
	ctx context.Context,
	q *Queries,
	accountID int64,
	amount int64,
) (Account, error) {
	account, err := q.AddAccountBalanceInternal(ctx, AddAccountBalanceInternalParams{
		ID: accountID, Amount: -amount,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrInsufficientBalance
	}
	return account, err
}

func loadAndLockTransferAccounts(
	ctx context.Context,
	q *Queries,
	fromPublicID pgtype.UUID,
	toPublicID pgtype.UUID,
) (Account, Account, error) {
	fromAccount, err := q.GetAccountByPublicID(ctx, fromPublicID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, Account{}, ErrAccountNotFound
	}
	if err != nil {
		return Account{}, Account{}, err
	}
	toAccount, err := q.GetAccountByPublicID(ctx, toPublicID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, Account{}, ErrAccountNotFound
	}
	if err != nil {
		return Account{}, Account{}, err
	}
	if fromAccount.ID == toAccount.ID {
		return Account{}, Account{}, ErrSameAccount
	}
	return lockTransferAccounts(ctx, q, fromAccount.ID, toAccount.ID)
}

func lockTransferAccounts(
	ctx context.Context,
	q *Queries,
	fromAccountID int64,
	toAccountID int64,
) (Account, Account, error) {
	firstID, secondID := fromAccountID, toAccountID
	if firstID > secondID {
		firstID, secondID = secondID, firstID
	}

	first, err := q.GetAccountForUpdate(ctx, firstID)
	if err != nil {
		return Account{}, Account{}, err
	}
	second, err := q.GetAccountForUpdate(ctx, secondID)
	if err != nil {
		return Account{}, Account{}, err
	}
	if first.ID == fromAccountID {
		return first, second, nil
	}
	return second, first, nil
}

func newTransactionIdentity() (pgtype.UUID, string) {
	id := uuid.New()
	return pgtype.UUID{Bytes: id, Valid: true},
		"TXN-" + strings.ToUpper(strings.ReplaceAll(id.String(), "-", ""))
}
