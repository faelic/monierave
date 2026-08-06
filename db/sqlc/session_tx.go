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
	ErrDeviceMismatch    = errors.New("session device does not match")
	ErrRefreshTokenReuse = errors.New("refresh token reuse detected")
)

type RotateRefreshTokenTxParams struct {
	SessionID           pgtype.UUID
	Username            string
	PresentedTokenHash  []byte
	PresentedRefreshID  pgtype.UUID
	PresentedDeviceHash []byte
	NewTokenHash        []byte
	NewRefreshID        pgtype.UUID
}

func (store *SQLStore) CreateExclusiveSessionTx(
	ctx context.Context,
	arg CreateSessionParams,
) (Session, error) {
	var result Session
	err := store.execTx(ctx, func(q *Queries) error {
		if _, err := q.GetUserForUpdate(ctx, arg.Username); err != nil {
			return err
		}
		replaced, err := q.RevokeAllUserSessions(ctx, RevokeAllUserSessionsParams{
			Username:      arg.Username,
			RevokedReason: textValue("replaced_by_new_login"),
		})
		if err != nil {
			return err
		}
		for _, session := range replaced {
			if err := createSessionAudit(
				ctx,
				q,
				session,
				"session_replaced_by_login",
				"user",
				"active",
				"revoked",
			); err != nil {
				return err
			}
		}

		result, err = q.CreateSession(ctx, arg)
		if err != nil {
			return err
		}
		if err := createSessionAudit(
			ctx,
			q,
			result,
			"session_created",
			"user",
			"",
			"active",
		); err != nil {
			return err
		}
		_, err = createFinancialAudit(ctx, q, financialAuditParams{
			EntityType:    "session",
			EntityID:      result.ID,
			CorrelationID: result.ID,
			EventType:     "login_succeeded",
			Actor:         result.Username,
			ToState:       "authenticated",
			Metadata: map[string]any{
				"client_ip":  result.ClientIp,
				"user_agent": result.UserAgent,
			},
		})
		return err
	})
	return result, err
}

func (store *SQLStore) ValidateSession(
	ctx context.Context,
	id pgtype.UUID,
	username string,
	deviceTokenHash []byte,
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
	if !constantTimeHashEqual(session.DeviceTokenHash, deviceTokenHash) {
		return ErrDeviceMismatch
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
		if !constantTimeHashEqual(
			session.DeviceTokenHash,
			arg.PresentedDeviceHash,
		) {
			return ErrDeviceMismatch
		}

		hashMatches := constantTimeHashEqual(
			session.RefreshTokenHash,
			arg.PresentedTokenHash,
		)
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
	deviceTokenHash []byte,
	reason string,
	actor string,
) error {
	return store.execTx(ctx, func(q *Queries) error {
		current, err := q.GetSessionForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if !constantTimeHashEqual(current.DeviceTokenHash, deviceTokenHash) {
			return ErrDeviceMismatch
		}
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

func constantTimeHashEqual(expected, actual []byte) bool {
	return len(expected) == len(actual) &&
		subtle.ConstantTimeCompare(expected, actual) == 1
}

func textValue(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}
