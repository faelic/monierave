package token

import (
	"time"

	"github.com/google/uuid"
)

// maker is an interface for managing tokens
type Maker interface {
	CreateAccessToken(
		username string,
		sessionID uuid.UUID,
		duration time.Duration,
	) (string, *Payload, error)
	CreateRefreshToken(
		username string,
		sessionID uuid.UUID,
		duration time.Duration,
	) (string, *Payload, error)
	VerifyAccessToken(value string) (*Payload, error)
	VerifyRefreshToken(value string) (*Payload, error)
}
