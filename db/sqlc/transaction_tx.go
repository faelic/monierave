package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	BankingTransactionTypeDeposit          = "deposit"
	BankingTransactionTypeWithdrawal       = "withdrawal"
	BankingTransactionTypeInternalTransfer = "internal_transfer"

	BankingTransactionStatusPending = "pending"
	BankingTransactionStatusPosted  = "posted"
)

var ErrInsufficientBalance = errors.New("insufficient balance")

type TransferTxParams struct {
	FromAccountPublicID pgtype.UUID `json:"from_account_id"`
	ToAccountPublicID   pgtype.UUID `json:"to_account_id"`
	Amount              int64       `json:"amount"`
	Currency            string      `json:"currency"`
	Username            string      `json:"username"`
	Narration           string      `json:"narration"`
}

type TransferTxResult struct {
	Transaction BankingTransaction `json:"transaction"`
	Postings    []LedgerPosting    `json:"postings"`
	FromAccount Account            `json:"from_account"`
	ToAccount   Account            `json:"to_account"`
}

type DepositTxParams struct {
	AccountPublicID pgtype.UUID
	Amount          int64
	Narration       string
}

type WithdrawTxParams struct {
	AccountPublicID pgtype.UUID
	Amount          int64
	Narration       string
}

type MoneyMovementTxResult struct {
	Transaction BankingTransaction
	Postings    []LedgerPosting
	Account     Account
}

type pendingTransactionParams struct {
	TransactionType string
	Currency        string
	Amount          int64
	Narration       string
	InitiatedBy     string
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

	err := store.execTx(ctx, func(q *Queries) error {
		fromAccount, toAccount, err := loadAndLockTransferAccounts(
			ctx,
			q,
			arg.FromAccountPublicID,
			arg.ToAccountPublicID,
		)
		if err != nil {
			return err
		}
		if fromAccount.Owner != arg.Username {
			return ErrAccountNotOwned
		}
		if fromAccount.Currency != arg.Currency || toAccount.Currency != arg.Currency {
			return ErrCurrencyMismatch
		}
		if fromAccount.Status == FinancialAccountStatusFrozen {
			return ErrAccountFrozen
		}
		if fromAccount.Status == FinancialAccountStatusClosed ||
			toAccount.Status == FinancialAccountStatusClosed {
			return ErrAccountClosed
		}

		fromLedger, err := customerLedgerAccount(ctx, q, fromAccount.ID)
		if err != nil {
			return err
		}
		toLedger, err := customerLedgerAccount(ctx, q, toAccount.ID)
		if err != nil {
			return err
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
			return err
		}

		result.FromAccount, err = subtractAccountBalance(ctx, q, fromAccount.ID, arg.Amount)
		if err != nil {
			return err
		}
		result.ToAccount, err = q.AddAccountBalanceInternal(ctx, AddAccountBalanceInternalParams{
			ID: toAccount.ID, Amount: arg.Amount,
		})
		if err != nil {
			return err
		}

		result.Transaction, err = q.MarkBankingTransactionPosted(ctx, transaction.ID)
		result.Postings = postings
		return err
	})
	if err != nil {
		return TransferTxResult{}, err
	}
	return result, nil
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
	)
}

func (store *SQLStore) settlementMovementTx(
	ctx context.Context,
	accountPublicID pgtype.UUID,
	amount int64,
	narration string,
	transactionType string,
) (MoneyMovementTxResult, error) {
	var result MoneyMovementTxResult
	if amount <= 0 {
		return result, fmt.Errorf("amount must be positive")
	}

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
				InitiatedBy:     "operator_cli",
			},
			postings,
		)
		if err != nil {
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
