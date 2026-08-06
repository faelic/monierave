package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type ProcessEmailDeliveryEventParams struct {
	WebhookID         string
	EventType         string
	ProviderMessageID string
	JobID             pgtype.UUID
	OccurredAt        time.Time
	Payload           []byte
	DeliveryStatus    string
	BounceType        string
	BounceSubtype     string
	BounceMessage     string
}

type ProcessEmailDeliveryEventResult struct {
	Event            EmailDeliveryEvent
	EmailJob         EmailJob
	Duplicate        bool
	JobMatched       bool
	StateUpdated     bool
	UserStateUpdated bool
}

func (store *SQLStore) ProcessEmailDeliveryEventTx(
	ctx context.Context,
	arg ProcessEmailDeliveryEventParams,
) (ProcessEmailDeliveryEventResult, error) {
	var result ProcessEmailDeliveryEventResult

	err := store.execTx(ctx, func(q *Queries) error {
		job, matched, err := findEmailJobForDeliveryEvent(ctx, q, arg)
		if err != nil {
			return err
		}
		result.EmailJob = job
		result.JobMatched = matched

		result.Event, err = q.CreateEmailDeliveryEvent(ctx, CreateEmailDeliveryEventParams{
			WebhookID:         arg.WebhookID,
			EmailJobID:        job.ID,
			ProviderMessageID: arg.ProviderMessageID,
			EventType:         arg.EventType,
			OccurredAt:        pgtype.Timestamptz{Time: arg.OccurredAt, Valid: true},
			Payload:           arg.Payload,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			result.Duplicate = true
			return nil
		}
		if err != nil {
			return err
		}

		if !matched || arg.DeliveryStatus == "" {
			return nil
		}

		previousStatus := job.DeliveryStatus
		if !canApplyEmailDeliveryTransition(
			previousStatus,
			arg.DeliveryStatus,
			job.DeliveryEventAt,
			arg.OccurredAt,
		) {
			return nil
		}
		result.EmailJob, err = q.UpdateEmailJobDelivery(ctx, UpdateEmailJobDeliveryParams{
			ProviderMessageID: pgtype.Text{String: arg.ProviderMessageID, Valid: true},
			DeliveryStatus:    arg.DeliveryStatus,
			OccurredAt:        pgtype.Timestamptz{Time: arg.OccurredAt, Valid: true},
			BounceType:        nullableText(arg.BounceType),
			BounceSubtype:     nullableText(arg.BounceSubtype),
			BounceMessage:     nullableText(arg.BounceMessage),
			ID:                job.ID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		result.StateUpdated = true

		deliverability := userDeliverabilityForEvent(arg)
		if deliverability != "" {
			_, userErr := q.UpdateCurrentUserEmailDeliverability(
				ctx,
				UpdateCurrentUserEmailDeliverabilityParams{
					DeliverabilityStatus: deliverability,
					OccurredAt: pgtype.Timestamptz{
						Time:  arg.OccurredAt,
						Valid: true,
					},
					BouncedAt: bounceTimestamp(arg),
					Username:  job.Username,
					Recipient: job.Recipient,
				},
			)
			if userErr != nil && !errors.Is(userErr, pgx.ErrNoRows) {
				return userErr
			}
			result.UserStateUpdated = userErr == nil
		}

		return createDeliveryAuditLog(ctx, q, arg, job, previousStatus)
	})

	return result, err
}

func findEmailJobForDeliveryEvent(
	ctx context.Context,
	q *Queries,
	arg ProcessEmailDeliveryEventParams,
) (EmailJob, bool, error) {
	job, err := q.GetEmailJobByProviderMessageIDForUpdate(ctx, pgtype.Text{
		String: arg.ProviderMessageID,
		Valid:  true,
	})
	if err == nil {
		return job, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return EmailJob{}, false, err
	}
	if !arg.JobID.Valid {
		return EmailJob{}, false, nil
	}

	job, err = q.GetEmailJobForUpdate(ctx, arg.JobID)
	if errors.Is(err, pgx.ErrNoRows) {
		return EmailJob{ID: arg.JobID}, false, nil
	}
	if err != nil {
		return EmailJob{}, false, err
	}
	if job.ProviderMessageID.Valid && job.ProviderMessageID.String != arg.ProviderMessageID {
		return EmailJob{ID: arg.JobID}, false, nil
	}
	return job, true, nil
}

func canApplyEmailDeliveryTransition(
	current string,
	next string,
	currentOccurredAt pgtype.Timestamptz,
	nextOccurredAt time.Time,
) bool {
	if current == next {
		return false
	}
	if currentOccurredAt.Valid && !nextOccurredAt.After(currentOccurredAt.Time) {
		return false
	}

	switch current {
	case EmailDeliveryStatusPending:
		return isRecognizedEmailDeliveryStatus(next)
	case EmailDeliveryStatusAccepted:
		return next == EmailDeliveryStatusDelayed ||
			next == EmailDeliveryStatusDelivered ||
			isTerminalEmailDeliveryStatus(next)
	case EmailDeliveryStatusDelayed:
		return next == EmailDeliveryStatusDelivered ||
			isTerminalEmailDeliveryStatus(next)
	case EmailDeliveryStatusDelivered:
		return next == EmailDeliveryStatusComplained
	default:
		return false
	}
}

func isRecognizedEmailDeliveryStatus(status string) bool {
	switch status {
	case EmailDeliveryStatusAccepted,
		EmailDeliveryStatusDelivered,
		EmailDeliveryStatusDelayed,
		EmailDeliveryStatusBounced,
		EmailDeliveryStatusFailed,
		EmailDeliveryStatusSuppressed,
		EmailDeliveryStatusComplained:
		return true
	default:
		return false
	}
}

func isTerminalEmailDeliveryStatus(status string) bool {
	switch status {
	case EmailDeliveryStatusBounced,
		EmailDeliveryStatusFailed,
		EmailDeliveryStatusSuppressed,
		EmailDeliveryStatusComplained:
		return true
	default:
		return false
	}
}

func userDeliverabilityForEvent(arg ProcessEmailDeliveryEventParams) string {
	switch arg.DeliveryStatus {
	case EmailDeliveryStatusDelivered:
		return EmailDeliverabilityDeliverable
	case EmailDeliveryStatusSuppressed, EmailDeliveryStatusComplained:
		return EmailDeliverabilityUndeliverable
	case EmailDeliveryStatusBounced:
		if strings.EqualFold(arg.BounceType, "permanent") {
			return EmailDeliverabilityUndeliverable
		}
	}
	return ""
}

func bounceTimestamp(arg ProcessEmailDeliveryEventParams) pgtype.Timestamptz {
	if arg.DeliveryStatus == EmailDeliveryStatusBounced &&
		strings.EqualFold(arg.BounceType, "permanent") {
		return pgtype.Timestamptz{Time: arg.OccurredAt, Valid: true}
	}
	return pgtype.Timestamptz{}
}

func createDeliveryAuditLog(
	ctx context.Context,
	q *Queries,
	arg ProcessEmailDeliveryEventParams,
	job EmailJob,
	previousStatus string,
) error {
	metadata, err := json.Marshal(map[string]string{
		"webhook_id":          arg.WebhookID,
		"provider_message_id": arg.ProviderMessageID,
		"bounce_type":         arg.BounceType,
		"bounce_subtype":      arg.BounceSubtype,
	})
	if err != nil {
		return fmt.Errorf("marshal email delivery audit metadata: %w", err)
	}

	_, err = q.CreateAuditLog(ctx, CreateAuditLogParams{
		EntityType:    "email_job",
		EntityID:      job.ID,
		CorrelationID: job.ID,
		EventType:     arg.EventType,
		Actor:         "resend_webhook",
		FromState:     nullableText(previousStatus),
		ToState:       nullableText(arg.DeliveryStatus),
		Metadata:      metadata,
	})
	return err
}

func nullableText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}
