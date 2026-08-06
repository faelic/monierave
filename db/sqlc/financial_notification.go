package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const (
	EmailJobTypeFinancialNotification = "financial_notification"

	DomainEventTransactionPosted   = "transaction.posted"
	DomainEventTransactionFailed   = "transaction.failed"
	DomainEventTransactionReversed = "transaction.reversed"
	DomainEventAccountFrozen       = "account.frozen"
	DomainEventAccountUnfrozen     = "account.unfrozen"
	DomainEventAccountClosed       = "account.closed"
)

type FinancialNotificationPayload struct {
	EventID       string    `json:"event_id"`
	CorrelationID string    `json:"correlation_id"`
	EventType     string    `json:"event_type"`
	Reference     string    `json:"reference,omitempty"`
	Amount        int64     `json:"amount,omitempty"`
	Currency      string    `json:"currency,omitempty"`
	Direction     string    `json:"direction,omitempty"`
	OccurredAt    time.Time `json:"occurred_at"`
	AccountStatus string    `json:"account_status,omitempty"`
	Reason        string    `json:"reason,omitempty"`
}

type financialNotificationParams struct {
	Username      string
	EventType     string
	EntityType    string
	EntityID      pgtype.UUID
	CorrelationID pgtype.UUID
	Reference     string
	Amount        int64
	Currency      string
	Direction     string
	OccurredAt    time.Time
	AccountStatus string
	Reason        string
}

func createFinancialNotification(
	ctx context.Context,
	q *Queries,
	arg financialNotificationParams,
) (EmailJob, OutboxEvent, error) {
	user, err := q.GetUser(ctx, arg.Username)
	if err != nil {
		return EmailJob{}, OutboxEvent{}, err
	}
	eventID := newPGUUID()
	jobID := newPGUUID()
	occurredAt := arg.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	payload, err := json.Marshal(FinancialNotificationPayload{
		EventID:       uuidString(eventID),
		CorrelationID: uuidString(arg.CorrelationID),
		EventType:     arg.EventType,
		Reference:     arg.Reference,
		Amount:        arg.Amount,
		Currency:      arg.Currency,
		Direction:     arg.Direction,
		OccurredAt:    occurredAt,
		AccountStatus: arg.AccountStatus,
		Reason:        arg.Reason,
	})
	if err != nil {
		return EmailJob{}, OutboxEvent{}, fmt.Errorf(
			"marshal financial notification payload: %w",
			err,
		)
	}
	job, err := q.CreateEmailJob(ctx, CreateEmailJobParams{
		ID:          jobID,
		JobType:     EmailJobTypeFinancialNotification,
		Username:    user.Username,
		Recipient:   user.Email,
		Payload:     payload,
		MaxAttempts: DefaultEmailMaxAttempts,
	})
	if err != nil {
		return EmailJob{}, OutboxEvent{}, err
	}
	eventPayload, err := json.Marshal(outboxEmailPayload{
		JobID:         uuidString(jobID),
		EventID:       uuidString(eventID),
		CorrelationID: uuidString(arg.CorrelationID),
	})
	if err != nil {
		return EmailJob{}, OutboxEvent{}, fmt.Errorf(
			"marshal financial outbox payload: %w",
			err,
		)
	}
	event, err := q.CreateOutboxEvent(ctx, CreateOutboxEventParams{
		ID:            eventID,
		EmailJobID:    jobID,
		EventType:     arg.EventType,
		Payload:       eventPayload,
		CorrelationID: arg.CorrelationID,
		EntityType:    arg.EntityType,
		EntityID:      arg.EntityID,
	})
	return job, event, err
}
