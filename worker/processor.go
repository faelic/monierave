package worker

import (
	"context"
	"encoding/json"
	"math/rand"
	"time"

	db "github.com/faelic/monierave/db/sqlc"
	"github.com/faelic/monierave/mailer"
	"github.com/faelic/monierave/token"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
)

const (
	QueueCritical = "critical"
	QueueDefault  = "default"
)

type TaskProcessor interface {
	Run() error
	Shutdown()
	ProcessTaskSendEmail(ctx context.Context, task *asynq.Task) error
	ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error
}

type RedisTaskProcessor struct {
	server                    *asynq.Server
	store                     db.Store
	mailer                    mailer.Mailer
	emailVerificationMaker    token.EmailVerificationMaker
	publicAPIURL              string
	emailVerificationDuration time.Duration
}

func NewRedisTaskProcessor(
	redisOpt asynq.RedisClientOpt,
	store db.Store,
	emailMailer mailer.Mailer,
	concurrency int,
	emailVerificationMaker token.EmailVerificationMaker,
	publicAPIURL string,
	emailVerificationDuration time.Duration,
) TaskProcessor {
	if concurrency <= 0 {
		concurrency = 10
	}

	ConfigureRedisLogging()

	processor := &RedisTaskProcessor{
		store:                     store,
		mailer:                    emailMailer,
		emailVerificationMaker:    emailVerificationMaker,
		publicAPIURL:              publicAPIURL,
		emailVerificationDuration: emailVerificationDuration,
	}

	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: concurrency,
			Queues: map[string]int{
				QueueCritical: 10,
				QueueDefault:  5,
			},
			RetryDelayFunc:  retryDelay,
			ErrorHandler:    asynq.ErrorHandlerFunc(processor.handleError),
			Logger:          NewAsynqLogger(),
			ShutdownTimeout: 30 * time.Second,
		},
	)

	processor.server = server
	return processor
}

func (processor *RedisTaskProcessor) Run() error {
	mux := asynq.NewServeMux()

	// mux maps task type to their corresponding handler
	mux.HandleFunc(TaskSendEmail, processor.ProcessTaskSendEmail)
	mux.HandleFunc(TaskSendVerifyEmail, processor.ProcessTaskSendVerifyEmail)
	return processor.server.Run(mux)
}

func (processor *RedisTaskProcessor) Shutdown() {
	processor.server.Shutdown()
}

var retrySchedule = []time.Duration{
	time.Minute,
	5 * time.Minute,
	15 * time.Minute,
	30 * time.Minute,
	time.Hour,
	2 * time.Hour,
	3 * time.Hour,
	4 * time.Hour,
	5 * time.Hour,
}

func retryDelay(retryCount int, _ error, _ *asynq.Task) time.Duration {
	index := retryCount - 1
	if index < 0 {
		index = 0
	}
	if index >= len(retrySchedule) {
		index = len(retrySchedule) - 1
	}

	base := retrySchedule[index]
	jitter := time.Duration(rand.Int63n(int64(base/5) + 1))
	return base + jitter
}

func (processor *RedisTaskProcessor) handleError(
	ctx context.Context,
	task *asynq.Task,
	taskErr error,
) {
	retried, retryOK := asynq.GetRetryCount(ctx)
	maxRetry, maxRetryOK := asynq.GetMaxRetry(ctx)
	if !retryOK || !maxRetryOK || retried < maxRetry {
		return
	}

	var payload PayloadSendEmail
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		log.Error().Err(err).Msg("failed to read exhausted email task payload")
		return
	}

	jobID, err := parseJobID(payload.JobID)
	if err != nil {
		log.Error().Err(err).Str("job_id", payload.JobID).Msg("invalid exhausted email job ID")
		return
	}

	_, err = processor.store.MarkEmailJobDeadLetter(ctx, db.MarkEmailJobDeadLetterParams{
		ID:        jobID,
		LastError: pgtype.Text{String: taskErr.Error(), Valid: true},
	})
	if err != nil {
		log.Error().
			Err(err).
			Str("job_id", payload.JobID).
			Msg("failed to persist exhausted email job in dead-letter state")
		return
	}

	log.Error().
		Err(taskErr).
		Str("job_id", payload.JobID).
		Str("event_id", payload.EventID).
		Str("correlation_id", payload.CorrelationID).
		Str("event_type", payload.EventType).
		Int("attempt", retried+1).
		Msg("email job exhausted retries and was dead-lettered")
}
