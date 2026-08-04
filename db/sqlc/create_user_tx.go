package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrEmailJobNotDeadLetter = errors.New("email job is not dead-lettered")

const (
	EmailJobTypeVerifyEmail   = "verify_email"
	OutboxEventTypeEmailReady = "email_job.ready"
	DefaultEmailMaxAttempts   = int32(10)

	EmailJobStatusPending    = "pending"
	EmailJobStatusQueued     = "queued"
	EmailJobStatusProcessing = "processing"
	EmailJobStatusRetrying   = "retrying"
	EmailJobStatusSent       = "sent"
	EmailJobStatusDeadLetter = "dead_letter"
)

type CreateUserTxResult struct {
	User        User        `json:"user"`
	EmailJob    EmailJob    `json:"email_job"`
	OutboxEvent OutboxEvent `json:"outbox_event"`
}

type emailJobPayload struct {
	Username  string `json:"username"`
	Recipient string `json:"recipient"`
}

type outboxEmailPayload struct {
	JobID string `json:"job_id"`
}

func newPGUUID() pgtype.UUID {
	return pgtype.UUID{Bytes: uuid.New(), Valid: true}
}

func uuidString(id pgtype.UUID) string {
	return uuid.UUID(id.Bytes).String()
}

func (store *SQLStore) CreateUserTx(ctx context.Context, arg CreateUserParams) (CreateUserTxResult, error) {
	var result CreateUserTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error
		result.User, err = q.CreateUser(ctx, arg)
		if err != nil {
			return err
		}

		jobID := newPGUUID()
		jobPayload, err := json.Marshal(emailJobPayload{
			Username:  result.User.Username,
			Recipient: result.User.Email,
		})
		if err != nil {
			return fmt.Errorf("marshal email job payload: %w", err)
		}

		result.EmailJob, err = q.CreateEmailJob(ctx, CreateEmailJobParams{
			ID:          jobID,
			JobType:     EmailJobTypeVerifyEmail,
			Username:    result.User.Username,
			Recipient:   result.User.Email,
			Payload:     jobPayload,
			MaxAttempts: DefaultEmailMaxAttempts,
		})
		if err != nil {
			return err
		}

		eventPayload, err := json.Marshal(outboxEmailPayload{JobID: uuidString(jobID)})
		if err != nil {
			return fmt.Errorf("marshal outbox payload: %w", err)
		}

		result.OutboxEvent, err = q.CreateOutboxEvent(ctx, CreateOutboxEventParams{
			ID:         newPGUUID(),
			EmailJobID: jobID,
			EventType:  OutboxEventTypeEmailReady,
			Payload:    eventPayload,
		})
		return err
	})

	return result, err
}

type ReplayEmailJobTxResult struct {
	EmailJob    EmailJob    `json:"email_job"`
	OutboxEvent OutboxEvent `json:"outbox_event"`
}

func (store *SQLStore) ReplayEmailJobTx(
	ctx context.Context,
	jobID pgtype.UUID,
) (ReplayEmailJobTxResult, error) {
	var result ReplayEmailJobTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		original, err := q.GetEmailJob(ctx, jobID)
		if err != nil {
			return err
		}
		if original.Status != EmailJobStatusDeadLetter {
			return ErrEmailJobNotDeadLetter
		}

		newJobID := newPGUUID()
		result.EmailJob, err = q.CreateEmailJob(ctx, CreateEmailJobParams{
			ID:          newJobID,
			ParentJobID: original.ID,
			JobType:     original.JobType,
			Username:    original.Username,
			Recipient:   original.Recipient,
			Payload:     original.Payload,
			MaxAttempts: original.MaxAttempts,
		})
		if err != nil {
			return err
		}

		eventPayload, err := json.Marshal(outboxEmailPayload{JobID: uuidString(newJobID)})
		if err != nil {
			return fmt.Errorf("marshal replay outbox payload: %w", err)
		}

		result.OutboxEvent, err = q.CreateOutboxEvent(ctx, CreateOutboxEventParams{
			ID:         newPGUUID(),
			EmailJobID: newJobID,
			EventType:  OutboxEventTypeEmailReady,
			Payload:    eventPayload,
		})
		if err != nil {
			return err
		}

		auditMetadata, err := json.Marshal(map[string]string{
			"replay_job_id":          uuidString(newJobID),
			"replay_outbox_event_id": uuidString(result.OutboxEvent.ID),
		})
		if err != nil {
			return fmt.Errorf("marshal replay audit metadata: %w", err)
		}

		_, err = q.CreateAuditLog(ctx, CreateAuditLogParams{
			EntityType:    "email_job",
			EntityID:      original.ID,
			CorrelationID: newJobID,
			EventType:     "email_job_replayed",
			Actor:         "operator_cli",
			FromState: pgtype.Text{
				String: EmailJobStatusDeadLetter,
				Valid:  true,
			},
			ToState: pgtype.Text{
				String: EmailJobStatusDeadLetter,
				Valid:  true,
			},
			Metadata: auditMetadata,
		})
		return err
	})

	return result, err
}
