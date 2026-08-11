package token

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestEmailVerificationTokenIsOpaqueAndDeterministicPerJob(t *testing.T) {
	maker, err := NewEmailVerificationMaker("12345678901234567890123456789012")
	require.NoError(t, err)

	jobID := uuid.NewString()
	value, err := maker.Create(jobID)
	require.NoError(t, err)
	require.NotContains(t, value, jobID)
	require.False(t, strings.Contains(value, "@"))

	repeated, err := maker.Create(jobID)
	require.NoError(t, err)
	require.Equal(t, value, repeated)

	other, err := maker.Create(uuid.NewString())
	require.NoError(t, err)
	require.NotEqual(t, value, other)
}

func TestEmailVerificationTokenHashRejectsMalformedValues(t *testing.T) {
	maker, err := NewEmailVerificationMaker("12345678901234567890123456789012")
	require.NoError(t, err)

	value, err := maker.Create(uuid.NewString())
	require.NoError(t, err)
	firstHash, err := maker.Hash(value)
	require.NoError(t, err)
	secondHash, err := maker.Hash(value)
	require.NoError(t, err)
	require.Len(t, firstHash, 32)
	require.True(t, bytes.Equal(firstHash, secondHash))

	_, err = maker.Hash("not-a-token")
	require.ErrorIs(t, err, ErrInvalidToken)
	_, err = maker.Create("not-a-uuid")
	require.ErrorIs(t, err, ErrInvalidToken)
}
