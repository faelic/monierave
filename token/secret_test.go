package token

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewDeviceSecretIsRandomAndHasExpectedEntropy(t *testing.T) {
	first, err := NewDeviceSecret()
	require.NoError(t, err)
	second, err := NewDeviceSecret()
	require.NoError(t, err)
	require.NotEqual(t, first, second)

	decoded, err := base64.RawURLEncoding.DecodeString(first)
	require.NoError(t, err)
	require.Len(t, decoded, DeviceSecretBytes)
}
