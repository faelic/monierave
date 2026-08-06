package db

import (
	"context"
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

	session, err := testStore.CreateSessionTx(context.Background(), CreateSessionParams{
		ID:               sessionID,
		Username:         created.User.Username,
		RefreshTokenHash: token.Hash(firstToken),
		RefreshTokenID:   firstRefreshID,
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
			SessionID:          sessionID,
			Username:           created.User.Username,
			PresentedTokenHash: token.Hash(firstToken),
			PresentedRefreshID: firstRefreshID,
			NewTokenHash:       token.Hash(secondToken),
			NewRefreshID:       secondRefreshID,
		},
	)
	require.NoError(t, err)
	require.Equal(t, secondRefreshID, rotated.RefreshTokenID)
	require.Equal(t, token.Hash(secondToken), rotated.RefreshTokenHash)

	_, err = testStore.RotateRefreshTokenTx(
		context.Background(),
		RotateRefreshTokenTxParams{
			SessionID:          sessionID,
			Username:           created.User.Username,
			PresentedTokenHash: token.Hash(firstToken),
			PresentedRefreshID: firstRefreshID,
			NewTokenHash:       token.Hash("attacker-token"),
			NewRefreshID:       pgUUID(uuid.New()),
		},
	)
	require.ErrorIs(t, err, ErrRefreshTokenReuse)

	revoked, err := testQueries.GetSession(context.Background(), sessionID)
	require.NoError(t, err)
	require.True(t, revoked.RevokedAt.Valid)
	require.Equal(t, "refresh_token_reuse", revoked.RevokedReason.String)
	require.ErrorIs(
		t,
		testStore.ValidateSession(context.Background(), sessionID, created.User.Username),
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

func pgUUID(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: value, Valid: true}
}
