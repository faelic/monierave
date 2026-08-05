package token

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	minSecretKeySize     = 32
	tokenIssuer          = "monierave"
	accessTokenAudience  = "monierave-api"
	refreshTokenAudience = "monierave-refresh"
)

type JWTMaker struct {
	accessKey  []byte
	refreshKey []byte
}

func NewJWTMaker(secretKey string) (Maker, error) {
	if len(secretKey) < minSecretKeySize {
		return nil, fmt.Errorf("invalid key size, must be at least %d characters", minSecretKeySize)
	}

	accessKey := sha256.Sum256([]byte(secretKey + ":" + accessTokenAudience))
	refreshKey := sha256.Sum256([]byte(secretKey + ":" + refreshTokenAudience))
	return &JWTMaker{accessKey: accessKey[:], refreshKey: refreshKey[:]}, nil
}

func (maker *JWTMaker) CreateAccessToken(
	username string,
	sessionID uuid.UUID,
	duration time.Duration,
) (string, *Payload, error) {
	return maker.createToken(
		username,
		sessionID,
		TypeAccess,
		accessTokenAudience,
		duration,
		maker.accessKey,
	)
}

func (maker *JWTMaker) CreateRefreshToken(
	username string,
	sessionID uuid.UUID,
	duration time.Duration,
) (string, *Payload, error) {
	return maker.createToken(
		username,
		sessionID,
		TypeRefresh,
		refreshTokenAudience,
		duration,
		maker.refreshKey,
	)
}

func (maker *JWTMaker) VerifyAccessToken(value string) (*Payload, error) {
	return maker.verifyToken(
		value,
		TypeAccess,
		accessTokenAudience,
		maker.accessKey,
	)
}

func (maker *JWTMaker) VerifyRefreshToken(value string) (*Payload, error) {
	return maker.verifyToken(
		value,
		TypeRefresh,
		refreshTokenAudience,
		maker.refreshKey,
	)
}

func (maker *JWTMaker) createToken(
	username string,
	sessionID uuid.UUID,
	tokenType Type,
	audience string,
	duration time.Duration,
	key []byte,
) (string, *Payload, error) {
	payload, err := NewPayload(
		username,
		sessionID,
		tokenType,
		tokenIssuer,
		audience,
		duration,
	)
	if err != nil {
		return "", nil, err
	}

	value, err := jwt.NewWithClaims(jwt.SigningMethodHS256, payload).SignedString(key)
	return value, payload, err
}

func (maker *JWTMaker) verifyToken(
	value string,
	expectedType Type,
	audience string,
	key []byte,
) (*Payload, error) {
	payload := &Payload{}
	parsed, err := jwt.ParseWithClaims(
		value,
		payload,
		func(parsed *jwt.Token) (any, error) {
			if parsed.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}
			return key, nil
		},
		jwt.WithIssuer(tokenIssuer),
		jwt.WithAudience(audience),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	if !parsed.Valid ||
		payload.ID == uuid.Nil ||
		payload.RegisteredClaims.ID != payload.ID.String() ||
		payload.SessionID == uuid.Nil ||
		payload.Username == "" ||
		payload.Subject != payload.Username ||
		payload.TokenType != expectedType {
		return nil, ErrInvalidToken
	}

	return payload, nil
}
