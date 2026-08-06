package db

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/faelic/monierave/token"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestRotateRefreshTokenTxDetectsReuseAndRevokesSession(t *testing.T) {
	created := createUserWithEmailJob(t)
	sessionID := pgUUID(uuid.New())
	firstRefreshID := pgUUID(uuid.New())
	firstToken := "first-refresh-token-" + uuid.NewString()
	deviceToken := "device-token-" + uuid.NewString()

	session, err := testStore.CreateExclusiveSessionTx(context.Background(), CreateSessionParams{
		ID:               sessionID,
		Username:         created.User.Username,
		RefreshTokenHash: token.Hash(firstToken),
		RefreshTokenID:   firstRefreshID,
		DeviceTokenHash:  token.Hash(deviceToken),
		UserAgent:        "session-test",
		ClientIp:         "127.0.0.1",
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(time.Hour),
			Valid: true,
		},
	})
	require.NoError(t, err)
	require.False(t, session.RevokedAt.Valid)

	secondToken := "second-refresh-token-" + uuid.NewString()
	secondRefreshID := pgUUID(uuid.New())
	rotated, err := testStore.RotateRefreshTokenTx(
		context.Background(),
		RotateRefreshTokenTxParams{
			SessionID:           sessionID,
			Username:            created.User.Username,
			PresentedTokenHash:  token.Hash(firstToken),
			PresentedRefreshID:  firstRefreshID,
			PresentedDeviceHash: token.Hash(deviceToken),
			NewTokenHash:        token.Hash(secondToken),
			NewRefreshID:        secondRefreshID,
		},
	)
	require.NoError(t, err)
	require.Equal(t, secondRefreshID, rotated.RefreshTokenID)
	require.Equal(t, token.Hash(secondToken), rotated.RefreshTokenHash)

	_, err = testStore.RotateRefreshTokenTx(
		context.Background(),
		RotateRefreshTokenTxParams{
			SessionID:           sessionID,
			Username:            created.User.Username,
			PresentedTokenHash:  token.Hash(firstToken),
			PresentedRefreshID:  firstRefreshID,
			PresentedDeviceHash: token.Hash(deviceToken),
			NewTokenHash:        token.Hash("attacker-token"),
			NewRefreshID:        pgUUID(uuid.New()),
		},
	)
	require.ErrorIs(t, err, ErrRefreshTokenReuse)

	revoked, err := testQueries.GetSession(context.Background(), sessionID)
	require.NoError(t, err)
	require.True(t, revoked.RevokedAt.Valid)
	require.Equal(t, "refresh_token_reuse", revoked.RevokedReason.String)
	require.ErrorIs(
		t,
		testStore.ValidateSession(
			context.Background(),
			sessionID,
			created.User.Username,
			token.Hash(deviceToken),
		),
		ErrSessionRevoked,
	)

	logs, err := testQueries.ListAuditLogsByJob(
		context.Background(),
		ListAuditLogsByJobParams{EntityID: sessionID, Limit: 20},
	)
	require.NoError(t, err)
	requireAuditEvent(t, logs, "session_created")
	requireAuditEvent(t, logs, "login_succeeded")
	requireAuditEvent(t, logs, "refresh_token_rotated")
	requireAuditEvent(t, logs, "refresh_token_reuse_detected")
}

func TestCreateExclusiveSessionTxRevokesPreviousSession(t *testing.T) {
	created := createUserWithEmailJob(t)
	firstDevice := "first-device-" + uuid.NewString()
	first, err := testStore.CreateExclusiveSessionTx(
		context.Background(),
		newSessionParams(created.User.Username, firstDevice),
	)
	require.NoError(t, err)

	secondDevice := "second-device-" + uuid.NewString()
	second, err := testStore.CreateExclusiveSessionTx(
		context.Background(),
		newSessionParams(created.User.Username, secondDevice),
	)
	require.NoError(t, err)
	require.NotEqual(t, first.ID, second.ID)

	storedFirst, err := testQueries.GetSession(context.Background(), first.ID)
	require.NoError(t, err)
	require.True(t, storedFirst.RevokedAt.Valid)
	require.Equal(t, "replaced_by_new_login", storedFirst.RevokedReason.String)
	require.ErrorIs(
		t,
		testStore.ValidateSession(
			context.Background(),
			first.ID,
			created.User.Username,
			token.Hash(firstDevice),
		),
		ErrSessionRevoked,
	)
	require.NoError(t, testStore.ValidateSession(
		context.Background(),
		second.ID,
		created.User.Username,
		token.Hash(secondDevice),
	))

	logs, err := testQueries.ListAuditLogsByJob(
		context.Background(),
		ListAuditLogsByJobParams{EntityID: first.ID, Limit: 20},
	)
	require.NoError(t, err)
	requireAuditEvent(t, logs, "session_replaced_by_login")
}

func TestCreateExclusiveSessionTxRollsBackRevocationWhenCreateFails(t *testing.T) {
	created := createUserWithEmailJob(t)
	deviceToken := "rollback-device-" + uuid.NewString()
	first, err := testStore.CreateExclusiveSessionTx(
		context.Background(),
		newSessionParams(created.User.Username, deviceToken),
	)
	require.NoError(t, err)

	conflicting := newSessionParams(created.User.Username, deviceToken)
	_, err = testStore.CreateExclusiveSessionTx(context.Background(), conflicting)
	require.Error(t, err)

	stored, err := testQueries.GetSession(context.Background(), first.ID)
	require.NoError(t, err)
	require.False(t, stored.RevokedAt.Valid)
	require.NoError(t, testStore.ValidateSession(
		context.Background(),
		first.ID,
		created.User.Username,
		token.Hash(deviceToken),
	))
}

func TestConcurrentExclusiveLoginsLeaveOneActiveSession(t *testing.T) {
	created := createUserWithEmailJob(t)
	const loginCount = 5
	results := make(chan Session, loginCount)
	errs := make(chan error, loginCount)
	var wait sync.WaitGroup

	for index := 0; index < loginCount; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			session, err := testStore.CreateExclusiveSessionTx(
				context.Background(),
				newSessionParams(created.User.Username, uuid.NewString()),
			)
			results <- session
			errs <- err
		}()
	}
	wait.Wait()
	close(results)
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	require.Len(t, results, loginCount)

	sessions, err := testQueries.ListSessions(
		context.Background(),
		created.User.Username,
	)
	require.NoError(t, err)
	active := 0
	for _, session := range sessions {
		if !session.RevokedAt.Valid {
			active++
		}
	}
	require.Equal(t, 1, active)
}

func TestSessionDeviceMismatchDoesNotRevokeSession(t *testing.T) {
	created := createUserWithEmailJob(t)
	deviceToken := "correct-device-" + uuid.NewString()
	refreshToken := "refresh-" + uuid.NewString()
	refreshID := pgUUID(uuid.New())
	params := newSessionParams(created.User.Username, deviceToken)
	params.RefreshTokenHash = token.Hash(refreshToken)
	params.RefreshTokenID = refreshID
	session, err := testStore.CreateExclusiveSessionTx(context.Background(), params)
	require.NoError(t, err)

	require.ErrorIs(t, testStore.ValidateSession(
		context.Background(),
		session.ID,
		created.User.Username,
		token.Hash("wrong-device"),
	), ErrDeviceMismatch)

	_, err = testStore.RotateRefreshTokenTx(
		context.Background(),
		RotateRefreshTokenTxParams{
			SessionID:           session.ID,
			Username:            created.User.Username,
			PresentedTokenHash:  token.Hash(refreshToken),
			PresentedRefreshID:  refreshID,
			PresentedDeviceHash: token.Hash("wrong-device"),
			NewTokenHash:        token.Hash("new-refresh"),
			NewRefreshID:        pgUUID(uuid.New()),
		},
	)
	require.ErrorIs(t, err, ErrDeviceMismatch)

	err = testStore.RevokeSessionTx(
		context.Background(),
		session.ID,
		token.Hash("wrong-device"),
		"user_logout",
		"user",
	)
	require.ErrorIs(t, err, ErrDeviceMismatch)

	stored, err := testQueries.GetSession(context.Background(), session.ID)
	require.NoError(t, err)
	require.False(t, stored.RevokedAt.Valid)
}

func newSessionParams(username, deviceToken string) CreateSessionParams {
	return CreateSessionParams{
		ID:               pgUUID(uuid.New()),
		Username:         username,
		RefreshTokenHash: token.Hash("refresh-" + uuid.NewString()),
		RefreshTokenID:   pgUUID(uuid.New()),
		DeviceTokenHash:  token.Hash(deviceToken),
		UserAgent:        "session-test",
		ClientIp:         "127.0.0.1",
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(time.Hour),
			Valid: true,
		},
	}
}

func pgUUID(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: value, Valid: true}
}
