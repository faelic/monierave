package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	ReconciliationIssueMissingLedgerAccount    = "missing_customer_ledger_account"
	ReconciliationIssueBalanceDrift            = "cached_balance_mismatch"
	ReconciliationIssueUnbalancedTransaction   = "unbalanced_transaction"
	ReconciliationIssueInsufficientPostings    = "insufficient_postings"
	ReconciliationIssueDuplicateTransactionRev = "duplicate_reversal"
)

type ReconciliationParams struct {
	AccountPublicID pgtype.UUID
	Actor           string
}

type ReconciliationIssue struct {
	Type                   string      `json:"type"`
	AccountPublicID        pgtype.UUID `json:"account_public_id,omitempty"`
	TransactionID          pgtype.UUID `json:"transaction_id,omitempty"`
	Reference              string      `json:"reference,omitempty"`
	Currency               string      `json:"currency,omitempty"`
	CachedBalance          int64       `json:"cached_balance,omitempty"`
	LedgerBalance          int64       `json:"ledger_balance,omitempty"`
	PostingCount           int64       `json:"posting_count,omitempty"`
	PostingTotal           int64       `json:"posting_total,omitempty"`
	DuplicateReversalCount int64       `json:"duplicate_reversal_count,omitempty"`
}

type ReconciliationReport struct {
	RunID           pgtype.UUID           `json:"run_id"`
	AccountPublicID pgtype.UUID           `json:"account_public_id,omitempty"`
	CheckedAt       time.Time             `json:"checked_at"`
	Consistent      bool                  `json:"consistent"`
	Issues          []ReconciliationIssue `json:"issues"`
	AuditLog        AuditLog              `json:"-"`
}

func (store *SQLStore) Reconcile(
	ctx context.Context,
	arg ReconciliationParams,
) (ReconciliationReport, error) {
	runID := pgtype.UUID{Bytes: uuid.New(), Valid: true}
	report := ReconciliationReport{
		RunID:           runID,
		AccountPublicID: arg.AccountPublicID,
		CheckedAt:       time.Now().UTC(),
		Issues:          make([]ReconciliationIssue, 0),
	}

	var customerAccountID pgtype.Int8
	if arg.AccountPublicID.Valid {
		account, err := store.GetAccountByPublicID(ctx, arg.AccountPublicID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ReconciliationReport{}, ErrAccountNotFound
		}
		if err != nil {
			return ReconciliationReport{}, err
		}
		customerAccountID = pgtype.Int8{Int64: account.ID, Valid: true}
	}

	accountIssues, err := store.ListAccountReconciliationIssues(
		ctx,
		arg.AccountPublicID,
	)
	if err != nil {
		return ReconciliationReport{}, err
	}
	for _, issue := range accountIssues {
		issueType := ReconciliationIssueBalanceDrift
		if !issue.CustomerLedgerAccountID.Valid {
			issueType = ReconciliationIssueMissingLedgerAccount
		}
		report.Issues = append(report.Issues, ReconciliationIssue{
			Type:            issueType,
			AccountPublicID: issue.AccountPublicID,
			Currency:        issue.Currency,
			CachedBalance:   issue.CachedBalance,
			LedgerBalance:   issue.LedgerBalance,
		})
	}

	transactionIssues, err := store.ListTransactionReconciliationIssues(
		ctx,
		customerAccountID,
	)
	if err != nil {
		return ReconciliationReport{}, err
	}
	for _, issue := range transactionIssues {
		if issue.PostingCount < 2 {
			report.Issues = append(report.Issues, ReconciliationIssue{
				Type:          ReconciliationIssueInsufficientPostings,
				TransactionID: issue.TransactionID,
				Reference:     issue.Reference,
				PostingCount:  issue.PostingCount,
				PostingTotal:  issue.PostingTotal,
			})
		}
		if issue.PostingTotal != 0 {
			report.Issues = append(report.Issues, ReconciliationIssue{
				Type:          ReconciliationIssueUnbalancedTransaction,
				TransactionID: issue.TransactionID,
				Reference:     issue.Reference,
				PostingCount:  issue.PostingCount,
				PostingTotal:  issue.PostingTotal,
			})
		}
	}

	reversalIssues, err := store.ListDuplicateReversalIssues(ctx, customerAccountID)
	if err != nil {
		return ReconciliationReport{}, err
	}
	for _, issue := range reversalIssues {
		report.Issues = append(report.Issues, ReconciliationIssue{
			Type:                   ReconciliationIssueDuplicateTransactionRev,
			TransactionID:          issue.OriginalTransactionID,
			Reference:              issue.OriginalReference,
			DuplicateReversalCount: issue.ReversalCount,
		})
	}

	report.Consistent = len(report.Issues) == 0
	metadata, err := json.Marshal(map[string]any{
		"account_public_id": uuidStringOrEmpty(arg.AccountPublicID),
		"consistent":        report.Consistent,
		"issue_count":       len(report.Issues),
	})
	if err != nil {
		return ReconciliationReport{}, fmt.Errorf("marshal reconciliation audit: %w", err)
	}
	report.AuditLog, err = store.CreateAuditLog(ctx, CreateAuditLogParams{
		EntityType:    "reconciliation",
		EntityID:      runID,
		CorrelationID: runID,
		EventType:     "reconciliation_completed",
		Actor:         arg.Actor,
		ToState:       textValue(reconciliationState(report.Consistent)),
		Metadata:      metadata,
	})
	if err != nil {
		return ReconciliationReport{}, err
	}
	return report, nil
}

func reconciliationState(consistent bool) string {
	if consistent {
		return "consistent"
	}
	return "drift_detected"
}

func uuidStringOrEmpty(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return uuid.UUID(value.Bytes).String()
}
