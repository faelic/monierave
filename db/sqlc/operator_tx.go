package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrInvalidOperatorAction      = errors.New("valid operator actor, reason, and correlation ID are required")
	ErrAccountAlreadyFrozen       = errors.New("account is already frozen")
	ErrAccountAlreadyActive       = errors.New("account is already active")
	ErrTransactionNotFound        = errors.New("transaction not found")
	ErrTransactionNotPosted       = errors.New("only posted transactions can be reversed")
	ErrTransactionAlreadyReversed = errors.New("transaction has already been reversed")
	ErrTransactionNotReversible   = errors.New("reversal transactions cannot be reversed")
	ErrReversalInsufficientFunds  = errors.New("insufficient funds to reverse transaction")
)

type AccountStatusTxParams struct {
	AccountPublicID pgtype.UUID
	TargetStatus    string
	Actor           string
	Reason          string
	CorrelationID   pgtype.UUID
}

type AccountStatusTxResult struct {
	Account  Account
	AuditLog AuditLog
}

type ReverseTransactionTxParams struct {
	TransactionID pgtype.UUID
	Actor         string
	Reason        string
	CorrelationID pgtype.UUID
}

type ReverseTransactionTxResult struct {
	Original BankingTransaction
	Reversal BankingTransaction
	Postings []LedgerPosting
	Accounts []Account
	AuditLog AuditLog
}

func (store *SQLStore) SetAccountStatusTx(
	ctx context.Context,
	arg AccountStatusTxParams,
) (AccountStatusTxResult, error) {
	if err := validateOperatorAction(arg.Actor, arg.Reason, arg.CorrelationID); err != nil {
		return AccountStatusTxResult{}, err
	}
	if arg.TargetStatus != FinancialAccountStatusFrozen &&
		arg.TargetStatus != FinancialAccountStatusActive {
		return AccountStatusTxResult{}, fmt.Errorf("invalid account target status")
	}

	var result AccountStatusTxResult
	err := store.execTx(ctx, func(q *Queries) error {
		account, err := q.GetAccountByPublicIDForUpdate(ctx, arg.AccountPublicID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAccountNotFound
		}
		if err != nil {
			return err
		}
		if account.Status == FinancialAccountStatusClosed {
			return ErrAccountClosed
		}
		if account.Status == arg.TargetStatus {
			if arg.TargetStatus == FinancialAccountStatusFrozen {
				return ErrAccountAlreadyFrozen
			}
			return ErrAccountAlreadyActive
		}

		updated, err := q.SetAccountStatus(ctx, SetAccountStatusParams{
			ID: account.ID, Status: arg.TargetStatus,
		})
		if err != nil {
			return err
		}
		eventType := "account_unfrozen"
		if arg.TargetStatus == FinancialAccountStatusFrozen {
			eventType = "account_frozen"
		}
		audit, err := createOperatorAudit(ctx, q, operatorAuditParams{
			EntityType:    "account",
			EntityID:      account.PublicID,
			CorrelationID: arg.CorrelationID,
			EventType:     eventType,
			Actor:         arg.Actor,
			FromState:     account.Status,
			ToState:       updated.Status,
			Reason:        arg.Reason,
			Metadata: map[string]any{
				"account_id": publicUUID(account.PublicID),
				"currency":   account.Currency,
			},
		})
		if err != nil {
			return err
		}
		domainEvent := DomainEventAccountUnfrozen
		if updated.Status == FinancialAccountStatusFrozen {
			domainEvent = DomainEventAccountFrozen
		}
		if _, _, err := createFinancialNotification(
			ctx,
			q,
			financialNotificationParams{
				Username:      updated.Owner,
				EventType:     domainEvent,
				EntityType:    "account",
				EntityID:      updated.PublicID,
				CorrelationID: arg.CorrelationID,
				OccurredAt:    updated.UpdatedAt.Time,
				AccountStatus: updated.Status,
				Reason:        arg.Reason,
			},
		); err != nil {
			return err
		}
		result.Account = updated
		result.AuditLog = audit
		return nil
	})
	if err != nil {
		return AccountStatusTxResult{}, store.auditOperatorFailure(
			ctx,
			operatorAuditParams{
				EntityType:    "account",
				EntityID:      arg.AccountPublicID,
				CorrelationID: arg.CorrelationID,
				EventType:     "account_status_change_failed",
				Actor:         arg.Actor,
				ToState:       "rejected",
				Reason:        arg.Reason,
				Metadata: map[string]any{
					"requested_status": arg.TargetStatus,
				},
			},
			err,
		)
	}
	return result, nil
}

func (store *SQLStore) ReverseTransactionTx(
	ctx context.Context,
	arg ReverseTransactionTxParams,
) (ReverseTransactionTxResult, error) {
	if err := validateOperatorAction(arg.Actor, arg.Reason, arg.CorrelationID); err != nil {
		return ReverseTransactionTxResult{}, err
	}

	var result ReverseTransactionTxResult
	err := store.execTx(ctx, func(q *Queries) error {
		original, err := q.GetBankingTransactionForUpdate(ctx, arg.TransactionID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTransactionNotFound
		}
		if err != nil {
			return err
		}
		if original.TransactionType == BankingTransactionTypeReversal {
			return ErrTransactionNotReversible
		}
		if original.Status == BankingTransactionStatusReversed {
			return ErrTransactionAlreadyReversed
		}
		if original.Status != BankingTransactionStatusPosted {
			return ErrTransactionNotPosted
		}

		sourcePostings, err := q.ListReversalSourcePostings(ctx, original.ID)
		if err != nil {
			return err
		}
		if len(sourcePostings) < 2 {
			return fmt.Errorf("transaction has insufficient ledger postings")
		}

		accountIDs := make([]int64, 0, len(sourcePostings))
		oppositePostings := make([]postingParams, 0, len(sourcePostings))
		for _, posting := range sourcePostings {
			oppositePostings = append(oppositePostings, postingParams{
				LedgerAccountID: posting.LedgerAccountID,
				Amount:          -posting.Amount,
			})
			if posting.Kind == "customer" {
				if !posting.CustomerAccountID.Valid {
					return fmt.Errorf("customer ledger account is missing its account")
				}
				accountIDs = append(accountIDs, posting.CustomerAccountID.Int64)
			}
		}

		lockedAccounts, orderedIDs, err := lockAccountsForOperator(
			ctx,
			q,
			accountIDs,
		)
		if err != nil {
			return err
		}
		for _, account := range lockedAccounts {
			if account.Status == FinancialAccountStatusClosed {
				return ErrAccountClosed
			}
		}

		reversal, postings, err := createBalancedTransaction(
			ctx,
			q,
			pendingTransactionParams{
				TransactionType: BankingTransactionTypeReversal,
				Currency:        original.Currency,
				Amount:          original.Amount,
				Narration:       "Reversal: " + strings.TrimSpace(arg.Reason),
				InitiatedBy:     strings.TrimSpace(arg.Actor),
				ReversalOf:      original.ID,
			},
			oppositePostings,
		)
		if err != nil {
			return err
		}

		updatedAccounts := make(map[int64]Account, len(lockedAccounts))
		reversalDirections := make(map[int64]string, len(lockedAccounts))
		for _, posting := range sourcePostings {
			if posting.Kind != "customer" {
				continue
			}
			accountID := posting.CustomerAccountID.Int64
			delta := -posting.Amount
			reversalDirections[accountID] = "incoming"
			if delta < 0 {
				reversalDirections[accountID] = "outgoing"
			}
			var updated Account
			if delta < 0 {
				updated, err = subtractAccountBalance(ctx, q, accountID, -delta)
				if errors.Is(err, ErrInsufficientBalance) {
					return ErrReversalInsufficientFunds
				}
			} else {
				updated, err = q.AddAccountBalanceInternal(
					ctx,
					AddAccountBalanceInternalParams{
						ID: accountID, Amount: delta,
					},
				)
			}
			if err != nil {
				return err
			}
			updatedAccounts[accountID] = updated
		}

		reversal, err = q.MarkBankingTransactionPosted(ctx, reversal.ID)
		if err != nil {
			return err
		}
		original, err = q.MarkBankingTransactionReversed(ctx, original.ID)
		if err != nil {
			return err
		}
		audit, err := createOperatorAudit(ctx, q, operatorAuditParams{
			EntityType:    "banking_transaction",
			EntityID:      original.ID,
			CorrelationID: arg.CorrelationID,
			EventType:     "transaction_reversed",
			Actor:         arg.Actor,
			FromState:     BankingTransactionStatusPosted,
			ToState:       BankingTransactionStatusReversed,
			Reason:        arg.Reason,
			Metadata: map[string]any{
				"original_reference":      original.Reference,
				"reversal_transaction_id": publicUUID(reversal.ID),
				"reversal_reference":      reversal.Reference,
			},
		})
		if err != nil {
			return err
		}
		for _, accountID := range orderedIDs {
			account := updatedAccounts[accountID]
			if _, _, err := createFinancialNotification(
				ctx,
				q,
				financialNotificationParams{
					Username:      account.Owner,
					EventType:     DomainEventTransactionReversed,
					EntityType:    "banking_transaction",
					EntityID:      original.ID,
					CorrelationID: arg.CorrelationID,
					Reference:     original.Reference,
					Amount:        original.Amount,
					Currency:      original.Currency,
					Direction:     reversalDirections[accountID],
					OccurredAt:    reversal.PostedAt.Time,
					Reason:        arg.Reason,
				},
			); err != nil {
				return err
			}
		}

		result.Original = original
		result.Reversal = reversal
		result.Postings = postings
		result.AuditLog = audit
		result.Accounts = make([]Account, 0, len(updatedAccounts))
		for _, accountID := range orderedIDs {
			result.Accounts = append(result.Accounts, updatedAccounts[accountID])
		}
		return nil
	})
	if err != nil {
		return ReverseTransactionTxResult{}, store.auditOperatorFailure(
			ctx,
			operatorAuditParams{
				EntityType:    "banking_transaction",
				EntityID:      arg.TransactionID,
				CorrelationID: arg.CorrelationID,
				EventType:     "transaction_reversal_failed",
				Actor:         arg.Actor,
				ToState:       "rejected",
				Reason:        arg.Reason,
			},
			err,
		)
	}
	return result, nil
}

func (store *SQLStore) auditOperatorFailure(
	ctx context.Context,
	arg operatorAuditParams,
	operatorErr error,
) error {
	if arg.Metadata == nil {
		arg.Metadata = map[string]any{}
	}
	arg.Metadata["error"] = operatorErr.Error()
	if _, auditErr := createOperatorAudit(ctx, store.Queries, arg); auditErr != nil {
		return errors.Join(
			operatorErr,
			fmt.Errorf("record operator failure audit: %w", auditErr),
		)
	}
	return operatorErr
}

type operatorAuditParams struct {
	EntityType    string
	EntityID      pgtype.UUID
	CorrelationID pgtype.UUID
	EventType     string
	Actor         string
	FromState     string
	ToState       string
	Reason        string
	Metadata      map[string]any
}

func createOperatorAudit(
	ctx context.Context,
	q *Queries,
	arg operatorAuditParams,
) (AuditLog, error) {
	metadata, err := json.Marshal(arg.Metadata)
	if err != nil {
		return AuditLog{}, fmt.Errorf("marshal operator audit metadata: %w", err)
	}
	return q.CreateAuditLog(ctx, CreateAuditLogParams{
		EntityType:    arg.EntityType,
		EntityID:      arg.EntityID,
		CorrelationID: arg.CorrelationID,
		EventType:     arg.EventType,
		Actor:         strings.TrimSpace(arg.Actor),
		FromState:     pgtype.Text{String: arg.FromState, Valid: arg.FromState != ""},
		ToState:       pgtype.Text{String: arg.ToState, Valid: arg.ToState != ""},
		Message: pgtype.Text{
			String: strings.TrimSpace(arg.Reason), Valid: true,
		},
		Metadata: metadata,
	})
}

func validateOperatorAction(
	actor string,
	reason string,
	correlationID pgtype.UUID,
) error {
	if strings.TrimSpace(actor) == "" ||
		strings.TrimSpace(reason) == "" ||
		len([]rune(strings.TrimSpace(actor))) > 128 ||
		len([]rune(strings.TrimSpace(reason))) > 500 ||
		!correlationID.Valid {
		return ErrInvalidOperatorAction
	}
	return nil
}

func lockAccountsForOperator(
	ctx context.Context,
	q *Queries,
	accountIDs []int64,
) (map[int64]Account, []int64, error) {
	unique := make(map[int64]struct{}, len(accountIDs))
	for _, accountID := range accountIDs {
		unique[accountID] = struct{}{}
	}
	ordered := make([]int64, 0, len(unique))
	for accountID := range unique {
		ordered = append(ordered, accountID)
	}
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i] < ordered[j]
	})

	accounts := make(map[int64]Account, len(ordered))
	for _, accountID := range ordered {
		account, err := q.GetAccountForUpdate(ctx, accountID)
		if err != nil {
			return nil, nil, err
		}
		accounts[accountID] = account
	}
	return accounts, ordered, nil
}

func publicUUID(value pgtype.UUID) string {
	return uuid.UUID(value.Bytes).String()
}
