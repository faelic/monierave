package token

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestEmailVerificationToken(t *testing.T) {
	maker, err := NewEmailVerificationMaker("12345678901234567890123456789012")
	require.NoError(t, err)

	value, err := maker.Create(
		"favour",
		"favour@example.com",
		"ceeb4e49-0b44-4944-9034-a12cbc32aaad",
		time.Minute,
	)
	require.NoError(t, err)

	payload, err := maker.Verify(value)
	require.NoError(t, err)
	require.Equal(t, "favour", payload.Username)
	require.Equal(t, "favour@example.com", payload.Email)
	require.Equal(t, "ceeb4e49-0b44-4944-9034-a12cbc32aaad", payload.JobID)
}

func TestEmailVerificationTokenRejectsExpiredAndAccessTokens(t *testing.T) {
	secret := "12345678901234567890123456789012"
	maker, err := NewEmailVerificationMaker(secret)
	require.NoError(t, err)

	expired, err := maker.Create("favour", "favour@example.com", "job-id", -time.Minute)
	require.Error(t, err)
	require.Empty(t, expired)

	accessMaker, err := NewJWTMaker(secret)
	require.NoError(t, err)
	accessToken, _, err := accessMaker.CreateAccessToken("favour", uuid.New(), time.Minute)
	require.NoError(t, err)
	_, err = maker.Verify(accessToken)
	require.ErrorIs(t, err, ErrInvalidToken)
}
