package token

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	emailVerificationIssuer   = "monierave"
	emailVerificationAudience = "email-verification"
)

type EmailVerificationPayload struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	JobID    string `json:"job_id"`
	jwt.RegisteredClaims
}

type EmailVerificationMaker interface {
	Create(
		username string,
		email string,
		jobID string,
		duration time.Duration,
	) (string, time.Time, error)
	Verify(value string) (*EmailVerificationPayload, error)
}

type JWTEmailVerificationMaker struct {
	secretKey []byte
}

func NewEmailVerificationMaker(secretKey string) (EmailVerificationMaker, error) {
	if len(secretKey) < minSecretKeySize {
		return nil, fmt.Errorf("invalid key size, must be at least %d characters", minSecretKeySize)
	}

	derivedKey := sha256.Sum256([]byte(secretKey + ":" + emailVerificationAudience))
	return &JWTEmailVerificationMaker{secretKey: derivedKey[:]}, nil
}

func (maker *JWTEmailVerificationMaker) Create(
	username string,
	email string,
	jobID string,
	duration time.Duration,
) (string, time.Time, error) {
	if username == "" || email == "" || jobID == "" || duration <= 0 {
		return "", time.Time{}, ErrInvalidToken
	}

	now := time.Now().UTC()
	expiresAt := now.Add(duration)
	payload := EmailVerificationPayload{
		Username: username,
		Email:    email,
		JobID:    jobID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    emailVerificationIssuer,
			Audience:  jwt.ClaimStrings{emailVerificationAudience},
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	value, err := jwt.NewWithClaims(jwt.SigningMethodHS256, payload).
		SignedString(maker.secretKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign email verification token: %w", err)
	}
	return value, expiresAt, nil
}

func (maker *JWTEmailVerificationMaker) Verify(value string) (*EmailVerificationPayload, error) {
	payload := &EmailVerificationPayload{}
	parsed, err := jwt.ParseWithClaims(
		value,
		payload,
		func(parsed *jwt.Token) (any, error) {
			if parsed.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}
			return maker.secretKey, nil
		},
		jwt.WithIssuer(emailVerificationIssuer),
		jwt.WithAudience(emailVerificationAudience),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	if !parsed.Valid ||
		payload.Username == "" ||
		payload.Email == "" ||
		payload.JobID == "" ||
		payload.Subject != payload.Username {
		return nil, ErrInvalidToken
	}
	return payload, nil
}
