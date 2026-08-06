package db

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type financialAuditParams struct {
	EntityType    string
	EntityID      pgtype.UUID
	CorrelationID pgtype.UUID
	EventType     string
	Actor         string
	FromState     string
	ToState       string
	Message       string
	Metadata      map[string]any
}

type LoginFailureAuditParams struct {
	Username  string
	ClientIP  string
	UserAgent string
	Reason    string
}

func (store *SQLStore) RecordLoginFailure(
	ctx context.Context,
	arg LoginFailureAuditParams,
) error {
	correlationID := auditCorrelation(pgtype.UUID{})
	actor := arg.Username
	if actor == "" {
		actor = "anonymous"
	}
	_, err := createFinancialAudit(ctx, store.Queries, financialAuditParams{
		EntityType:    "authentication_attempt",
		EntityID:      correlationID,
		CorrelationID: correlationID,
		EventType:     "login_failed",
		Actor:         actor,
		ToState:       "rejected",
		Message:       arg.Reason,
		Metadata: map[string]any{
			"attempted_username": arg.Username,
			"client_ip":          arg.ClientIP,
			"user_agent":         arg.UserAgent,
		},
	})
	return err
}

func createFinancialAudit(
	ctx context.Context,
	q *Queries,
	arg financialAuditParams,
) (AuditLog, error) {
	metadata := arg.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return AuditLog{}, fmt.Errorf("marshal %s audit metadata: %w", arg.EventType, err)
	}
	return q.CreateAuditLog(ctx, CreateAuditLogParams{
		EntityType:    arg.EntityType,
		EntityID:      arg.EntityID,
		CorrelationID: arg.CorrelationID,
		EventType:     arg.EventType,
		Actor:         arg.Actor,
		FromState:     textValue(arg.FromState),
		ToState:       textValue(arg.ToState),
		Message:       textValue(arg.Message),
		Metadata:      encoded,
	})
}

func auditCorrelation(value pgtype.UUID) pgtype.UUID {
	if value.Valid {
		return value
	}
	return pgtype.UUID{Bytes: uuid.New(), Valid: true}
}

func transactionAuditMetadata(transaction BankingTransaction) map[string]any {
	return map[string]any{
		"amount":      transaction.Amount,
		"currency":    transaction.Currency,
		"reference":   transaction.Reference,
		"type":        transaction.TransactionType,
		"reversal_of": uuidStringOrEmpty(transaction.ReversalOf),
	}
}
