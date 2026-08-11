package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
)

const emailVerificationPurpose = "email-verification-opaque-v1"

// EmailVerificationMaker derives opaque, retry-stable tokens from random email
// job UUIDs. Only a SHA-256 token hash is persisted by the worker.
type EmailVerificationMaker interface {
	Create(jobID string) (string, error)
	Hash(value string) ([]byte, error)
}

type HMACEmailVerificationMaker struct {
	secretKey []byte
}

func NewEmailVerificationMaker(secretKey string) (EmailVerificationMaker, error) {
	if len(secretKey) < minSecretKeySize {
		return nil, fmt.Errorf("invalid key size, must be at least %d characters", minSecretKeySize)
	}

	derivedKey := sha256.Sum256([]byte(secretKey + ":" + emailVerificationPurpose))
	return &HMACEmailVerificationMaker{secretKey: derivedKey[:]}, nil
}

func (maker *HMACEmailVerificationMaker) Create(jobID string) (string, error) {
	id, err := uuid.Parse(jobID)
	if err != nil {
		return "", ErrInvalidToken
	}

	mac := hmac.New(sha256.New, maker.secretKey)
	_, _ = mac.Write(id[:])
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (maker *HMACEmailVerificationMaker) Hash(value string) ([]byte, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(decoded) != sha256.Size {
		return nil, ErrInvalidToken
	}
	digest := sha256.Sum256([]byte(value))
	return digest[:], nil
}
