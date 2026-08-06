package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	db "github.com/faelic/monierave/db/sqlc"
	"github.com/faelic/monierave/mailer"
	"github.com/faelic/monierave/observability"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

const (
	TaskSendEmail       = "task:send_email"
	TaskSendVerifyEmail = "task:send_verify_email"
)

type PayloadSendEmail struct {
	JobID         string `json:"job_id"`
	EventID       string `json:"event_id,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	EventType     string `json:"event_type,omitempty"`
}

type PayloadSendVerifyEmail = PayloadSendEmail

func (distributor *RedisTaskDistributor) DistributeTaskSendEmail(
	ctx context.Context,
	payload *PayloadSendEmail,
	opts ...asynq.Option,
) error {
	// serialize payload
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload %w", err)
	}

	// create a new task and enqueue task
	task := asynq.NewTask(TaskSendVerifyEmail, jsonPayload, opts...)
	info, err := distributor.client.EnqueueContext(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	log.Info().Str("type", task.Type()).Bytes("payload", task.Payload()).Str("queue", info.Queue).Int("max_retry", info.MaxRetry).Msg("enqueued send verify email task")
	return nil
}

func (distributor *RedisTaskDistributor) DistributeTaskSendVerifyEmail(
	ctx context.Context,
	payload *PayloadSendVerifyEmail,
	opts ...asynq.Option,
) error {
	return distributor.DistributeTaskSendEmail(ctx, payload, opts...)
}

func (processor *RedisTaskProcessor) ProcessTaskSendEmail(
	ctx context.Context,
	task *asynq.Task,
) error {
	var payload PayloadSendEmail
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", asynq.SkipRetry)
	}

	jobID, err := parseJobID(payload.JobID)
	if err != nil {
		return fmt.Errorf("invalid email job ID: %v: %w", err, asynq.SkipRetry)
	}

	job, err := processor.store.GetEmailJob(ctx, jobID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("email job does not exist: %w", asynq.SkipRetry)
		}
		return fmt.Errorf("get email job: %w", err)
	}

	if job.Status == db.EmailJobStatusSent {
		log.Info().Str("job_id", payload.JobID).Msg("email job was already sent")
		return nil
	}
	if job.Status == db.EmailJobStatusDeadLetter {
		return fmt.Errorf("email job is already dead-lettered: %w", asynq.SkipRetry)
	}

	job, err = processor.store.StartEmailJobAttempt(ctx, jobID)
	if err != nil {
		return fmt.Errorf("start email job attempt: %w", err)
	}

	providerMessageID, sendErr := processor.sendEmailJob(ctx, job, payload.JobID)
	if sendErr != nil {
		if mailer.IsPermanent(sendErr) || job.AttemptCount >= job.MaxAttempts {
			if _, err := processor.store.MarkEmailJobDeadLetter(ctx, db.MarkEmailJobDeadLetterParams{
				ID:        jobID,
				LastError: pgtype.Text{String: sendErr.Error(), Valid: true},
			}); err != nil {
				return fmt.Errorf("persist dead-letter state after send failure: %w", err)
			}

			if mailer.IsPermanent(sendErr) {
				return fmt.Errorf("permanent email failure: %v: %w", sendErr, asynq.SkipRetry)
			}
			return fmt.Errorf("email retries exhausted: %w", sendErr)
		}

		if _, err := processor.store.MarkEmailJobRetrying(ctx, db.MarkEmailJobRetryingParams{
			ID:        jobID,
			LastError: pgtype.Text{String: sendErr.Error(), Valid: true},
		}); err != nil {
			return fmt.Errorf("persist retrying state after send failure: %w", err)
		}
		observability.Default.RecordWorkerRetry()
		return fmt.Errorf("send verification email: %w", sendErr)
	}

	if _, err := processor.store.MarkEmailJobSent(ctx, db.MarkEmailJobSentParams{
		ID:                jobID,
		ProviderMessageID: pgtype.Text{String: providerMessageID, Valid: true},
	}); err != nil {
		return fmt.Errorf("persist sent email job: %w", err)
	}

	log.Info().
		Str("type", task.Type()).
		Str("job_id", payload.JobID).
		Str("event_id", payload.EventID).
		Str("correlation_id", payload.CorrelationID).
		Str("event_type", payload.EventType).
		Str("email", job.Recipient).
		Str("provider_message_id", providerMessageID).
		Int32("attempt", job.AttemptCount).
		Str("job_type", job.JobType).
		Msg("email accepted by provider")
	return nil
}

func (processor *RedisTaskProcessor) ProcessTaskSendVerifyEmail(
	ctx context.Context,
	task *asynq.Task,
) error {
	return processor.ProcessTaskSendEmail(ctx, task)
}

func (processor *RedisTaskProcessor) sendEmailJob(
	ctx context.Context,
	job db.EmailJob,
	jobID string,
) (string, error) {
	switch job.JobType {
	case "", db.EmailJobTypeVerifyEmail:
		emailPayload, err := processor.verificationEmailPayload(job, jobID)
		if err != nil {
			return "", mailer.NewPermanentError(fmt.Errorf(
				"create verification email link: %w",
				err,
			))
		}
		return processor.mailer.SendVerificationEmail(ctx, mailer.VerificationEmail{
			JobID:     jobID,
			Username:  job.Username,
			Recipient: job.Recipient,
			Payload:   emailPayload,
		})
	case db.EmailJobTypeFinancialNotification:
		return processor.mailer.SendFinancialNotification(
			ctx,
			mailer.FinancialNotificationEmail{
				JobID:     jobID,
				Username:  job.Username,
				Recipient: job.Recipient,
				Payload:   job.Payload,
			},
		)
	default:
		return "", mailer.NewPermanentError(fmt.Errorf(
			"unsupported email job type %q",
			job.JobType,
		))
	}
}

func (processor *RedisTaskProcessor) verificationEmailPayload(
	job db.EmailJob,
	jobID string,
) ([]byte, error) {
	if processor.emailVerificationMaker == nil {
		return job.Payload, nil
	}

	value, err := processor.emailVerificationMaker.Create(
		job.Username,
		job.Recipient,
		jobID,
		processor.emailVerificationDuration,
	)
	if err != nil {
		return nil, err
	}

	var payload map[string]any
	if len(job.Payload) > 0 {
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return nil, fmt.Errorf("decode email job payload: %w", err)
		}
	}
	if payload == nil {
		payload = make(map[string]any)
	}
	payload["verification_url"] = strings.TrimRight(processor.publicAPIURL, "/") +
		"/users/verify-email?token=" + url.QueryEscape(value)

	return json.Marshal(payload)
}

func parseJobID(value string) (pgtype.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: id, Valid: true}, nil
}
