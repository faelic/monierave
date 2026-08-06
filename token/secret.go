package token

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const DeviceSecretBytes = 32

func NewDeviceSecret() (string, error) {
	value := make([]byte, DeviceSecretBytes)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate device secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
