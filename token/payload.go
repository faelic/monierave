package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Payload contains the payload data of the token
type Payload struct {
	ID        uuid.UUID `json:"token_id"`
	SessionID uuid.UUID `json:"sid"`
	Username  string    `json:"username"`
	TokenType Type      `json:"token_type"`
	jwt.RegisteredClaims
}

type Type string

const (
	TypeAccess  Type = "access"
	TypeRefresh Type = "refresh"
)

// different types of errors returned by verifying token
var (
	ErrInvalidToken = errors.New("token is not valid")
	ErrExpiredToken = errors.New("token has expired")
)

// Payload creates a new payload with a specific username and duration
func NewPayload(
	username string,
	sessionID uuid.UUID,
	tokenType Type,
	issuer string,
	audience string,
	duration time.Duration,
) (*Payload, error) {
	if username == "" || sessionID == uuid.Nil || duration == 0 {
		return nil, ErrInvalidToken
	}
	tokenID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	payload := &Payload{
		ID:        tokenID,
		SessionID: sessionID,
		Username:  username,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID.String(),
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
		},
	}
	return payload, nil
}
