package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	db "github.com/faelic/monierave/db/sqlc"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

type RelayConfig struct {
	InstanceID      string
	BatchSize       int32
	PollInterval    time.Duration
	ClaimLease      time.Duration
	TaskTimeout     time.Duration
	TaskRetention   time.Duration
	OutboxRetention time.Duration
	EmailRetention  time.Duration
	CleanupInterval time.Duration
}

type OutboxRelay struct {
	store       db.Store
	distributor TaskDistributor
	config      RelayConfig
}

func NewOutboxRelay(
	store db.Store,
	distributor TaskDistributor,
	config RelayConfig,
) *OutboxRelay {
	if config.InstanceID == "" {
		config.InstanceID = uuid.NewString()
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 50
	}
	if config.PollInterval <= 0 {
		config.PollInterval = time.Second
	}
	if config.ClaimLease <= 0 {
		config.ClaimLease = 30 * time.Second
	}
	if config.TaskTimeout <= 0 {
		config.TaskTimeout = 30 * time.Second
	}
	if config.TaskRetention <= 0 {
		config.TaskRetention = 7 * 24 * time.Hour
	}
	if config.OutboxRetention <= 0 {
		config.OutboxRetention = 7 * 24 * time.Hour
	}
	if config.EmailRetention <= 0 {
		config.EmailRetention = 90 * 24 * time.Hour
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 24 * time.Hour
	}

	return &OutboxRelay{
		store:       store,
		distributor: distributor,
		config:      config,
	}
}

func (relay *OutboxRelay) Run(ctx context.Context) error {
	pollTicker := time.NewTicker(relay.config.PollInterval)
	defer pollTicker.Stop()

	cleanupTicker := time.NewTicker(relay.config.CleanupInterval)
	defer cleanupTicker.Stop()

	for {
		if err := relay.publishBatch(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Error().Err(err).Msg("outbox relay batch failed")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-pollTicker.C:
		case <-cleanupTicker.C:
			relay.cleanup(ctx)
		}
	}
}

func (relay *OutboxRelay) publishBatch(ctx context.Context) error {
	now := time.Now().UTC()
	events, err := relay.store.ClaimOutboxEvents(ctx, db.ClaimOutboxEventsParams{
		ClaimedBy:    pgtype.Text{String: relay.config.InstanceID, Valid: true},
		ClaimedUntil: pgtype.Timestamptz{Time: now.Add(relay.config.ClaimLease), Valid: true},
		BatchSize:    relay.config.BatchSize,
	})
	if err != nil {
		return fmt.Errorf("claim outbox events: %w", err)
	}

	for _, event := range events {
		if err := relay.publishEvent(ctx, event); err != nil {
			log.Error().
				Err(err).
				Str("outbox_event_id", uuid.UUID(event.ID.Bytes).String()).
				Msg("failed to publish outbox event")
		}
	}
	return nil
}

func (relay *OutboxRelay) publishEvent(ctx context.Context, event db.OutboxEvent) error {
	var payload PayloadSendVerifyEmail
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return relay.release(ctx, event, fmt.Errorf("decode outbox payload: %w", err))
	}

	job, err := relay.store.GetEmailJob(ctx, event.EmailJobID)
	if err != nil {
		return relay.release(ctx, event, fmt.Errorf("load email job: %w", err))
	}

	options := []asynq.Option{
		asynq.TaskID(payload.JobID),
		asynq.Queue(QueueCritical),
		asynq.MaxRetry(int(job.MaxAttempts - 1)),
		asynq.Timeout(relay.config.TaskTimeout),
		asynq.Retention(relay.config.TaskRetention),
	}
	err = relay.distributor.DistributeTaskSendVerifyEmail(ctx, &payload, options...)
	if err != nil && !errors.Is(err, asynq.ErrTaskIDConflict) {
		return relay.release(ctx, event, err)
	}

	if _, err := relay.store.MarkOutboxEventPublished(ctx, event.ID); err != nil {
		return fmt.Errorf("mark outbox event published: %w", err)
	}

	if _, err := relay.store.MarkEmailJobQueued(ctx, event.EmailJobID); err != nil {
		log.Error().
			Err(err).
			Str("job_id", payload.JobID).
			Msg("task was published but email job queued status was not persisted")
	}

	log.Info().
		Str("job_id", payload.JobID).
		Str("outbox_event_id", uuid.UUID(event.ID.Bytes).String()).
		Int32("publish_attempt", event.PublishAttempts).
		Msg("outbox event published")
	return nil
}

func (relay *OutboxRelay) release(
	ctx context.Context,
	event db.OutboxEvent,
	publishErr error,
) error {
	_, err := relay.store.ReleaseOutboxEvent(ctx, db.ReleaseOutboxEventParams{
		ID: event.ID,
		AvailableAt: pgtype.Timestamptz{
			Time:  time.Now().UTC().Add(outboxRetryDelay(event.PublishAttempts)),
			Valid: true,
		},
		LastError: pgtype.Text{String: publishErr.Error(), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("release outbox event after %v: %w", publishErr, err)
	}
	return publishErr
}

func outboxRetryDelay(attempt int32) time.Duration {
	exponent := math.Min(float64(attempt), 8)
	base := time.Duration(math.Pow(2, exponent)) * time.Second
	if base > 5*time.Minute {
		base = 5 * time.Minute
	}
	return base + time.Duration(rand.Int63n(int64(base/5)+1))
}

func (relay *OutboxRelay) cleanup(ctx context.Context) {
	now := time.Now().UTC()
	outboxDeleted, outboxErr := relay.store.DeleteExpiredPublishedOutboxEvents(
		ctx,
		pgtype.Timestamptz{Time: now.Add(-relay.config.OutboxRetention), Valid: true},
	)
	jobsDeleted, jobsErr := relay.store.DeleteExpiredSentEmailJobs(
		ctx,
		pgtype.Timestamptz{Time: now.Add(-relay.config.EmailRetention), Valid: true},
	)
	usersDisabled, usersErr := relay.store.DisableExpiredPendingUsers(ctx)

	log.Info().
		Err(outboxErr).
		Err(jobsErr).
		Err(usersErr).
		Int64("outbox_events_deleted", outboxDeleted).
		Int64("email_jobs_deleted", jobsDeleted).
		Int64("pending_registrations_disabled", usersDisabled).
		Msg("outbox retention cleanup completed")
}
