package token

import (
	"testing"
	"time"

	"github.com/faelic/monierave/db/util"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestJWTMakerCreatesTypedSessionTokens(t *testing.T) {
	maker, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)

	username := util.RandomOwner()
	sessionID := uuid.New()
	issuedAt := time.Now()

	access, accessPayload, err := maker.CreateAccessToken(username, sessionID, time.Minute)
	require.NoError(t, err)
	verifiedAccess, err := maker.VerifyAccessToken(access)
	require.NoError(t, err)
	require.Equal(t, accessPayload.ID, verifiedAccess.ID)
	require.Equal(t, sessionID, verifiedAccess.SessionID)
	require.Equal(t, username, verifiedAccess.Username)
	require.Equal(t, TypeAccess, verifiedAccess.TokenType)
	require.WithinDuration(t, issuedAt, verifiedAccess.IssuedAt.Time, time.Second)

	refresh, _, err := maker.CreateRefreshToken(username, sessionID, time.Hour)
	require.NoError(t, err)
	verifiedRefresh, err := maker.VerifyRefreshToken(refresh)
	require.NoError(t, err)
	require.Equal(t, TypeRefresh, verifiedRefresh.TokenType)

	_, err = maker.VerifyRefreshToken(access)
	require.ErrorIs(t, err, ErrInvalidToken)
	_, err = maker.VerifyAccessToken(refresh)
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTMakerRejectsExpiredToken(t *testing.T) {
	maker, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)

	value, payload, err := maker.CreateAccessToken(
		util.RandomOwner(),
		uuid.New(),
		-time.Minute,
	)
	require.NoError(t, err)
	require.NotEmpty(t, value)
	require.NotNil(t, payload)
	_, err = maker.VerifyAccessToken(value)
	require.ErrorIs(t, err, ErrExpiredToken)
}

func TestJWTMakerRejectsAlgNone(t *testing.T) {
	payload, err := NewPayload(
		util.RandomOwner(),
		uuid.New(),
		TypeAccess,
		tokenIssuer,
		accessTokenAudience,
		time.Minute,
	)
	require.NoError(t, err)

	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, payload)
	value, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	maker, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)
	_, err = maker.VerifyAccessToken(value)
	require.ErrorIs(t, err, ErrInvalidToken)
}
