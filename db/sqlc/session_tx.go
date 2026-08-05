package db

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrSessionExpired    = errors.New("session expired")
	ErrSessionRevoked    = errors.New("session revoked")
	ErrSessionMismatch   = errors.New("session does not match token")
	ErrRefreshTokenReuse = errors.New("refresh token reuse detected")
)

type RotateRefreshTokenTxParams struct {
	SessionID          pgtype.UUID
	Username           string
	PresentedTokenHash []byte
	PresentedRefreshID pgtype.UUID
	NewTokenHash       []byte
	NewRefreshID       pgtype.UUID
}

func (store *SQLStore) CreateSessionTx(
	ctx context.Context,
	arg CreateSessionParams,
) (Session, error) {
	var result Session
	err := store.execTx(ctx, func(q *Queries) error {
		var err error
		result, err = q.CreateSession(ctx, arg)
		if err != nil {
			return err
		}
		return createSessionAudit(
			ctx,
			q,
			result,
			"session_created",
			"user",
			"",
			"active",
		)
	})
	return result, err
}

func (store *SQLStore) ValidateSession(
	ctx context.Context,
	id pgtype.UUID,
	username string,
) error {
	session, err := store.GetSession(ctx, id)
	if err != nil {
		return err
	}
	if session.Username != username {
		return ErrSessionMismatch
	}
	if session.RevokedAt.Valid {
		return ErrSessionRevoked
	}
	if !time.Now().Before(session.ExpiresAt.Time) {
		return ErrSessionExpired
	}
	return nil
}

func (store *SQLStore) RotateRefreshTokenTx(
	ctx context.Context,
	arg RotateRefreshTokenTxParams,
) (Session, error) {
	var result Session
	var reused bool

	err := store.execTx(ctx, func(q *Queries) error {
		session, err := q.GetSessionForUpdate(ctx, arg.SessionID)
		if err != nil {
			return err
		}
		if session.Username != arg.Username {
			return ErrSessionMismatch
		}
		if session.RevokedAt.Valid {
			return ErrSessionRevoked
		}
		if !time.Now().Before(session.ExpiresAt.Time) {
			return ErrSessionExpired
		}

		hashMatches := len(session.RefreshTokenHash) == len(arg.PresentedTokenHash) &&
			subtle.ConstantTimeCompare(session.RefreshTokenHash, arg.PresentedTokenHash) == 1
		idMatches := session.RefreshTokenID == arg.PresentedRefreshID
		if !hashMatches || !idMatches {
			reused = true
			if _, err := q.RevokeSession(ctx, RevokeSessionParams{
				ID:            arg.SessionID,
				RevokedReason: textValue("refresh_token_reuse"),
			}); err != nil {
				return err
			}
			return createSessionAudit(
				ctx,
				q,
				session,
				"refresh_token_reuse_detected",
				"security",
				"active",
				"revoked",
			)
		}

		result, err = q.RotateSessionRefreshToken(ctx, RotateSessionRefreshTokenParams{
			ID:               arg.SessionID,
			RefreshTokenHash: arg.NewTokenHash,
			RefreshTokenID:   arg.NewRefreshID,
		})
		if err != nil {
			return err
		}
		return createSessionAudit(
			ctx,
			q,
			result,
			"refresh_token_rotated",
			"user",
			"active",
			"active",
		)
	})
	if err != nil {
		return Session{}, err
	}
	if reused {
		return Session{}, ErrRefreshTokenReuse
	}
	return result, nil
}

func (store *SQLStore) RevokeSessionTx(
	ctx context.Context,
	id pgtype.UUID,
	reason string,
	actor string,
) error {
	return store.execTx(ctx, func(q *Queries) error {
		session, err := q.RevokeSession(ctx, RevokeSessionParams{
			ID:            id,
			RevokedReason: textValue(reason),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		return createSessionAudit(
			ctx,
			q,
			session,
			"session_logged_out",
			actor,
			"active",
			"revoked",
		)
	})
}

func (store *SQLStore) RevokeAllUserSessionsTx(
	ctx context.Context,
	username string,
	reason string,
	eventType string,
) error {
	return store.execTx(ctx, func(q *Queries) error {
		sessions, err := q.RevokeAllUserSessions(ctx, RevokeAllUserSessionsParams{
			Username:      username,
			RevokedReason: textValue(reason),
		})
		if err != nil {
			return err
		}
		for _, session := range sessions {
			if err := createSessionAudit(
				ctx,
				q,
				session,
				eventType,
				"user",
				"active",
				"revoked",
			); err != nil {
				return err
			}
		}
		return nil
	})
}

func createSessionAudit(
	ctx context.Context,
	q *Queries,
	session Session,
	eventType string,
	actor string,
	fromState string,
	toState string,
) error {
	metadata, err := json.Marshal(map[string]string{
		"user_agent": session.UserAgent,
		"client_ip":  session.ClientIp,
	})
	if err != nil {
		return err
	}
	_, err = q.CreateAuditLog(ctx, CreateAuditLogParams{
		EntityType:    "session",
		EntityID:      session.ID,
		CorrelationID: session.ID,
		EventType:     eventType,
		Actor:         actor,
		FromState:     textValue(fromState),
		ToState:       textValue(toState),
		Metadata:      metadata,
	})
	return err
}

func textValue(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}
