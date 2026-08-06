package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	mockdb "github.com/faelic/monierave/db/mock"
	db "github.com/faelic/monierave/db/sqlc"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type stubDistributor struct {
	err     error
	calls   int
	payload *PayloadSendEmail
	options []asynq.Option
}

func (d *stubDistributor) DistributeTaskSendEmail(
	_ context.Context,
	payload *PayloadSendEmail,
	options ...asynq.Option,
) error {
	d.calls++
	d.payload = payload
	d.options = options
	return d.err
}

func (d *stubDistributor) Close() error {
	return nil
}

func newOutboxEvent(t *testing.T) db.OutboxEvent {
	t.Helper()
	jobID := uuid.New()
	payload, err := json.Marshal(PayloadSendVerifyEmail{JobID: jobID.String()})
	require.NoError(t, err)
	return db.OutboxEvent{
		ID:              pgUUID(uuid.New()),
		EmailJobID:      pgUUID(jobID),
		EventType:       db.OutboxEventTypeEmailReady,
		Payload:         payload,
		CorrelationID:   pgUUID(uuid.New()),
		EntityType:      "email_job",
		EntityID:        pgUUID(jobID),
		Status:          "publishing",
		PublishAttempts: 1,
	}
}

func TestOutboxRelayPublishEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	distributor := &stubDistributor{}
	relay := NewOutboxRelay(store, distributor, RelayConfig{})
	event := newOutboxEvent(t)

	store.EXPECT().
		GetEmailJob(gomock.Any(), event.EmailJobID).
		Return(db.EmailJob{MaxAttempts: db.DefaultEmailMaxAttempts}, nil)
	store.EXPECT().
		MarkOutboxEventPublished(gomock.Any(), event.ID).
		Return(event, nil)
	store.EXPECT().
		MarkEmailJobQueued(gomock.Any(), event.EmailJobID).
		Return(db.EmailJob{}, nil)

	err := relay.publishEvent(context.Background(), event)
	require.NoError(t, err)
	require.Equal(t, 1, distributor.calls)
	require.Equal(t, uuid.UUID(event.EmailJobID.Bytes).String(), distributor.payload.JobID)
	require.Equal(t, uuid.UUID(event.ID.Bytes).String(), distributor.payload.EventID)
	require.Equal(
		t,
		uuid.UUID(event.CorrelationID.Bytes).String(),
		distributor.payload.CorrelationID,
	)
	require.Equal(t, event.EventType, distributor.payload.EventType)
	require.NotEmpty(t, distributor.options)
}

func TestOutboxRelayTreatsTaskIDConflictAsPublished(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	distributor := &stubDistributor{err: asynq.ErrTaskIDConflict}
	relay := NewOutboxRelay(store, distributor, RelayConfig{})
	event := newOutboxEvent(t)

	store.EXPECT().
		GetEmailJob(gomock.Any(), event.EmailJobID).
		Return(db.EmailJob{MaxAttempts: db.DefaultEmailMaxAttempts}, nil)
	store.EXPECT().
		MarkOutboxEventPublished(gomock.Any(), event.ID).
		Return(event, nil)
	store.EXPECT().
		MarkEmailJobQueued(gomock.Any(), event.EmailJobID).
		Return(db.EmailJob{}, nil)

	err := relay.publishEvent(context.Background(), event)
	require.NoError(t, err)
}

func TestOutboxRelayReleasesEventWhenRedisFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	redisErr := errors.New("redis unavailable")
	distributor := &stubDistributor{err: redisErr}
	relay := NewOutboxRelay(store, distributor, RelayConfig{})
	event := newOutboxEvent(t)

	store.EXPECT().
		GetEmailJob(gomock.Any(), event.EmailJobID).
		Return(db.EmailJob{MaxAttempts: db.DefaultEmailMaxAttempts}, nil)
	store.EXPECT().
		ReleaseOutboxEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.ReleaseOutboxEventParams) (db.OutboxEvent, error) {
			require.Equal(t, event.ID, arg.ID)
			require.Contains(t, arg.LastError.String, "redis unavailable")
			require.True(t, arg.AvailableAt.Valid)
			return event, nil
		})

	err := relay.publishEvent(context.Background(), event)
	require.ErrorIs(t, err, redisErr)
}

func TestOutboxRelayCleanupIncludesExpiredIdempotencyKeys(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	relay := NewOutboxRelay(store, &stubDistributor{}, RelayConfig{})

	store.EXPECT().
		DeleteExpiredPublishedOutboxEvents(gomock.Any(), gomock.Any()).
		Return(int64(2), nil)
	store.EXPECT().
		DeleteExpiredSentEmailJobs(gomock.Any(), gomock.Any()).
		Return(int64(3), nil)
	store.EXPECT().
		DisableExpiredPendingUsers(gomock.Any()).
		Return(int64(1), nil)
	store.EXPECT().
		DeleteExpiredIdempotencyKeys(gomock.Any()).
		Return(int64(4), nil)

	relay.cleanup(context.Background())
}
