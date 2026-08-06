package worker

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"testing"
	"time"

	mockdb "github.com/faelic/monierave/db/mock"
	db "github.com/faelic/monierave/db/sqlc"
	"github.com/faelic/monierave/mailer"
	"github.com/faelic/monierave/token"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type stubMailer struct {
	messageID string
	err       error
	calls     int
	message   mailer.VerificationEmail
	financial mailer.FinancialNotificationEmail
}

func (m *stubMailer) SendFinancialNotification(
	_ context.Context,
	message mailer.FinancialNotificationEmail,
) (string, error) {
	m.calls++
	m.financial = message
	return m.messageID, m.err
}

func (m *stubMailer) SendVerificationEmail(
	_ context.Context,
	message mailer.VerificationEmail,
) (string, error) {
	m.calls++
	m.message = message
	return m.messageID, m.err
}

func newEmailTask(t *testing.T, id uuid.UUID) *asynq.Task {
	t.Helper()
	payload, err := json.Marshal(PayloadSendVerifyEmail{JobID: id.String()})
	require.NoError(t, err)
	return asynq.NewTask(TaskSendVerifyEmail, payload)
}

func newGenericEmailTask(t *testing.T, id uuid.UUID) *asynq.Task {
	t.Helper()
	payload, err := json.Marshal(PayloadSendEmail{JobID: id.String()})
	require.NoError(t, err)
	return asynq.NewTask(TaskSendEmail, payload)
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func TestProcessTaskSendVerifyEmailSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	emailMailer := &stubMailer{messageID: "provider-123"}
	processor := &RedisTaskProcessor{store: store, mailer: emailMailer}

	id := uuid.New()
	job := db.EmailJob{
		ID:           pgUUID(id),
		Username:     "favour",
		Recipient:    "favour@example.com",
		Payload:      []byte(`{"name":"Favour"}`),
		Status:       "queued",
		AttemptCount: 0,
		MaxAttempts:  10,
	}
	started := job
	started.Status = "processing"
	started.AttemptCount = 1

	store.EXPECT().GetEmailJob(gomock.Any(), pgUUID(id)).Return(job, nil)
	store.EXPECT().StartEmailJobAttempt(gomock.Any(), pgUUID(id)).Return(started, nil)
	store.EXPECT().
		MarkEmailJobSent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.MarkEmailJobSentParams) (db.EmailJob, error) {
			require.Equal(t, pgUUID(id), arg.ID)
			require.Equal(t, "provider-123", arg.ProviderMessageID.String)
			return started, nil
		})

	err := processor.ProcessTaskSendVerifyEmail(context.Background(), newEmailTask(t, id))
	require.NoError(t, err)
	require.Equal(t, 1, emailMailer.calls)
	require.Equal(t, id.String(), emailMailer.message.JobID)
	require.Equal(t, job.Recipient, emailMailer.message.Recipient)
}

func TestProcessTaskSendFinancialNotificationSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	emailMailer := &stubMailer{messageID: "financial-provider-123"}
	processor := &RedisTaskProcessor{store: store, mailer: emailMailer}

	id := uuid.New()
	job := db.EmailJob{
		ID:          pgUUID(id),
		JobType:     db.EmailJobTypeFinancialNotification,
		Username:    "favour",
		Recipient:   "favour@example.com",
		Payload:     []byte(`{"event_type":"transaction.posted","reference":"TXN-123"}`),
		Status:      db.EmailJobStatusQueued,
		MaxAttempts: 10,
	}
	started := job
	started.Status = db.EmailJobStatusProcessing
	started.AttemptCount = 1

	store.EXPECT().GetEmailJob(gomock.Any(), pgUUID(id)).Return(job, nil)
	store.EXPECT().StartEmailJobAttempt(gomock.Any(), pgUUID(id)).Return(started, nil)
	store.EXPECT().
		MarkEmailJobSent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			_ context.Context,
			arg db.MarkEmailJobSentParams,
		) (db.EmailJob, error) {
			require.Equal(t, "financial-provider-123", arg.ProviderMessageID.String)
			return started, nil
		})

	err := processor.ProcessTaskSendEmail(
		context.Background(),
		newGenericEmailTask(t, id),
	)
	require.NoError(t, err)
	require.Equal(t, 1, emailMailer.calls)
	require.Equal(t, id.String(), emailMailer.financial.JobID)
	require.Equal(t, job.Payload, []byte(emailMailer.financial.Payload))
	require.Empty(t, emailMailer.message.JobID)
}

func TestProcessTaskSendFinancialNotificationPermanentFailureGoesToDLQ(
	t *testing.T,
) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	sendErr := mailer.NewPermanentError(errors.New("invalid financial payload"))
	emailMailer := &stubMailer{err: sendErr}
	processor := &RedisTaskProcessor{store: store, mailer: emailMailer}

	id := uuid.New()
	job := db.EmailJob{
		ID:          pgUUID(id),
		JobType:     db.EmailJobTypeFinancialNotification,
		Username:    "favour",
		Recipient:   "favour@example.com",
		Payload:     []byte(`{"event_type":"unsupported"}`),
		Status:      db.EmailJobStatusQueued,
		MaxAttempts: 10,
	}
	started := job
	started.Status = db.EmailJobStatusProcessing
	started.AttemptCount = 1

	store.EXPECT().GetEmailJob(gomock.Any(), pgUUID(id)).Return(job, nil)
	store.EXPECT().StartEmailJobAttempt(gomock.Any(), pgUUID(id)).Return(started, nil)
	store.EXPECT().
		MarkEmailJobDeadLetter(gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			_ context.Context,
			arg db.MarkEmailJobDeadLetterParams,
		) (db.EmailJob, error) {
			require.Equal(t, pgUUID(id), arg.ID)
			require.Equal(t, "permanent email delivery failure", arg.LastError.String)
			return started, nil
		})

	err := processor.ProcessTaskSendEmail(
		context.Background(),
		newGenericEmailTask(t, id),
	)
	require.Error(t, err)
	require.ErrorIs(t, err, asynq.SkipRetry)
	require.Equal(t, 1, emailMailer.calls)
}

func TestProcessTaskSendVerifyEmailAddsSignedVerificationURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	emailMailer := &stubMailer{messageID: "provider-123"}
	verificationMaker, err := token.NewEmailVerificationMaker(
		"12345678901234567890123456789012",
	)
	require.NoError(t, err)
	processor := &RedisTaskProcessor{
		store:                     store,
		mailer:                    emailMailer,
		emailVerificationMaker:    verificationMaker,
		publicAPIURL:              "https://api.example.com/",
		emailVerificationDuration: 24 * time.Hour,
	}

	id := uuid.New()
	job := db.EmailJob{
		ID:           pgUUID(id),
		Username:     "favour",
		Recipient:    "favour@example.com",
		Payload:      []byte(`{"username":"favour"}`),
		Status:       db.EmailJobStatusQueued,
		AttemptCount: 0,
		MaxAttempts:  10,
	}
	started := job
	started.Status = db.EmailJobStatusProcessing
	started.AttemptCount = 1

	store.EXPECT().GetEmailJob(gomock.Any(), pgUUID(id)).Return(job, nil)
	store.EXPECT().StartEmailJobAttempt(gomock.Any(), pgUUID(id)).Return(started, nil)
	store.EXPECT().MarkEmailJobSent(gomock.Any(), gomock.Any()).Return(started, nil)

	err = processor.ProcessTaskSendVerifyEmail(context.Background(), newEmailTask(t, id))
	require.NoError(t, err)

	var payload struct {
		VerificationURL       string `json:"verification_url"`
		VerificationExpiresAt string `json:"verification_expires_at"`
	}
	require.NoError(t, json.Unmarshal(emailMailer.message.Payload, &payload))
	expiresAt, err := time.Parse(time.RFC3339, payload.VerificationExpiresAt)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now().Add(24*time.Hour), expiresAt, time.Second)
	parsedURL, err := url.Parse(payload.VerificationURL)
	require.NoError(t, err)
	require.Equal(t, "https://api.example.com/users/verify-email", parsedURL.Scheme+"://"+parsedURL.Host+parsedURL.Path)

	verificationPayload, err := verificationMaker.Verify(parsedURL.Query().Get("token"))
	require.NoError(t, err)
	require.Equal(t, job.Username, verificationPayload.Username)
	require.Equal(t, job.Recipient, verificationPayload.Email)
	require.Equal(t, id.String(), verificationPayload.JobID)
}

func TestProcessTaskSendVerifyEmailAlreadySent(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	emailMailer := &stubMailer{}
	processor := &RedisTaskProcessor{store: store, mailer: emailMailer}

	id := uuid.New()
	store.EXPECT().GetEmailJob(gomock.Any(), pgUUID(id)).Return(db.EmailJob{
		ID:     pgUUID(id),
		Status: "sent",
	}, nil)

	err := processor.ProcessTaskSendVerifyEmail(context.Background(), newEmailTask(t, id))
	require.NoError(t, err)
	require.Zero(t, emailMailer.calls)
}

func TestRetryScheduleStaysWithinResendIdempotencyWindow(t *testing.T) {
	var maximumElapsed time.Duration
	for _, delay := range retrySchedule {
		maximumElapsed += delay + delay/5
	}

	require.LessOrEqual(t, maximumElapsed, 20*time.Hour)
}

func TestProcessTaskSendVerifyEmailTransientFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	sendErr := errors.New("provider unavailable")
	emailMailer := &stubMailer{err: sendErr}
	processor := &RedisTaskProcessor{store: store, mailer: emailMailer}

	id := uuid.New()
	job := db.EmailJob{
		ID:          pgUUID(id),
		Status:      "queued",
		MaxAttempts: 10,
	}
	started := job
	started.Status = "processing"
	started.AttemptCount = 1

	store.EXPECT().GetEmailJob(gomock.Any(), pgUUID(id)).Return(job, nil)
	store.EXPECT().StartEmailJobAttempt(gomock.Any(), pgUUID(id)).Return(started, nil)
	store.EXPECT().
		MarkEmailJobRetrying(gomock.Any(), gomock.Any()).
		Return(started, nil)

	err := processor.ProcessTaskSendVerifyEmail(context.Background(), newEmailTask(t, id))
	require.EqualError(t, err, "temporary email delivery failure")
	require.False(t, errors.Is(err, asynq.SkipRetry))
}

func TestProcessTaskSendVerifyEmailPermanentFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	sendErr := mailer.NewPermanentError(errors.New("invalid recipient"))
	emailMailer := &stubMailer{err: sendErr}
	processor := &RedisTaskProcessor{store: store, mailer: emailMailer}

	id := uuid.New()
	job := db.EmailJob{
		ID:          pgUUID(id),
		Status:      "queued",
		MaxAttempts: 10,
	}
	started := job
	started.Status = "processing"
	started.AttemptCount = 1

	store.EXPECT().GetEmailJob(gomock.Any(), pgUUID(id)).Return(job, nil)
	store.EXPECT().StartEmailJobAttempt(gomock.Any(), pgUUID(id)).Return(started, nil)
	store.EXPECT().
		MarkEmailJobDeadLetter(gomock.Any(), gomock.Any()).
		Return(started, nil)

	err := processor.ProcessTaskSendVerifyEmail(context.Background(), newEmailTask(t, id))
	require.Error(t, err)
	require.ErrorIs(t, err, asynq.SkipRetry)
}

func TestProcessTaskSendVerifyEmailFinalAttempt(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	sendErr := errors.New("provider unavailable")
	emailMailer := &stubMailer{err: sendErr}
	processor := &RedisTaskProcessor{store: store, mailer: emailMailer}

	id := uuid.New()
	job := db.EmailJob{
		ID:           pgUUID(id),
		Status:       "retrying",
		AttemptCount: 9,
		MaxAttempts:  10,
	}
	started := job
	started.Status = "processing"
	started.AttemptCount = 10

	store.EXPECT().GetEmailJob(gomock.Any(), pgUUID(id)).Return(job, nil)
	store.EXPECT().StartEmailJobAttempt(gomock.Any(), pgUUID(id)).Return(started, nil)
	store.EXPECT().
		MarkEmailJobDeadLetter(gomock.Any(), gomock.Any()).
		Return(started, nil)

	err := processor.ProcessTaskSendVerifyEmail(context.Background(), newEmailTask(t, id))
	require.EqualError(t, err, "email delivery retries exhausted")
	require.False(t, errors.Is(err, asynq.SkipRetry))
}

func TestProcessTaskSendVerifyEmailSentPersistenceFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	emailMailer := &stubMailer{messageID: "provider-123"}
	processor := &RedisTaskProcessor{store: store, mailer: emailMailer}

	id := uuid.New()
	job := db.EmailJob{
		ID:          pgUUID(id),
		Status:      "queued",
		MaxAttempts: 10,
	}
	started := job
	started.Status = "processing"
	started.AttemptCount = 1
	persistErr := errors.New("database unavailable")

	store.EXPECT().GetEmailJob(gomock.Any(), pgUUID(id)).Return(job, nil)
	store.EXPECT().StartEmailJobAttempt(gomock.Any(), pgUUID(id)).Return(started, nil)
	store.EXPECT().
		MarkEmailJobSent(gomock.Any(), gomock.Any()).
		Return(db.EmailJob{}, persistErr)

	err := processor.ProcessTaskSendVerifyEmail(context.Background(), newEmailTask(t, id))
	require.ErrorIs(t, err, persistErr)
	require.Equal(t, 1, emailMailer.calls)
}
