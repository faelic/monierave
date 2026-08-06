package api

import (
	"errors"
	"net/http"
	"time"

	db "github.com/faelic/monierave/db/sqlc"
	"github.com/faelic/monierave/token"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type renewAccessTokenResponse struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
}

func (server *Server) renewAccessToken(ctx *gin.Context) {
	if !server.originAllowed(ctx) {
		ctx.JSON(http.StatusForbidden, errorResponse(ctx, ErrForbidden))
		return
	}

	currentRefreshToken, err := server.refreshCookie(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(ctx, ErrInvalidToken))
		return
	}
	deviceToken, err := server.deviceCookie(ctx)
	if err != nil {
		server.clearSessionCookies(ctx)
		ctx.JSON(http.StatusUnauthorized, errorResponse(ctx, ErrInvalidToken))
		return
	}
	refreshPayload, err := server.tokenMaker.VerifyRefreshToken(currentRefreshToken)
	if err != nil {
		server.clearSessionCookies(ctx)
		ctx.JSON(http.StatusUnauthorized, errorResponse(ctx, ErrInvalidToken))
		return
	}

	sessionID := pgtype.UUID{Bytes: refreshPayload.SessionID, Valid: true}
	session, err := server.store.GetSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			server.clearSessionCookies(ctx)
			ctx.JSON(http.StatusUnauthorized, errorResponse(ctx, ErrInvalidSession))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(ctx, ErrInternalServer))
		return
	}

	remaining := time.Until(session.ExpiresAt.Time)
	if session.RevokedAt.Valid || remaining <= 0 {
		server.clearSessionCookies(ctx)
		ctx.JSON(http.StatusUnauthorized, errorResponse(ctx, ErrInvalidSession))
		return
	}

	newRefreshToken, newRefreshPayload, err := server.tokenMaker.CreateRefreshToken(
		refreshPayload.Username,
		refreshPayload.SessionID,
		remaining,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(ctx, ErrInternalServer))
		return
	}
	accessToken, accessPayload, err := server.tokenMaker.CreateAccessToken(
		refreshPayload.Username,
		refreshPayload.SessionID,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(ctx, ErrInternalServer))
		return
	}

	_, err = server.store.RotateRefreshTokenTx(ctx, db.RotateRefreshTokenTxParams{
		SessionID:           sessionID,
		Username:            refreshPayload.Username,
		PresentedTokenHash:  token.Hash(currentRefreshToken),
		PresentedRefreshID:  pgtype.UUID{Bytes: refreshPayload.ID, Valid: true},
		PresentedDeviceHash: token.Hash(deviceToken),
		NewTokenHash:        token.Hash(newRefreshToken),
		NewRefreshID:        pgtype.UUID{Bytes: newRefreshPayload.ID, Valid: true},
	})
	if err != nil {
		server.clearSessionCookies(ctx)
		switch {
		case errors.Is(err, db.ErrRefreshTokenReuse),
			errors.Is(err, db.ErrSessionExpired),
			errors.Is(err, db.ErrSessionRevoked),
			errors.Is(err, db.ErrDeviceMismatch),
			errors.Is(err, db.ErrSessionMismatch),
			errors.Is(err, pgx.ErrNoRows):
			ctx.JSON(http.StatusUnauthorized, errorResponse(ctx, ErrInvalidSession))
		default:
			ctx.JSON(http.StatusInternalServerError, errorResponse(ctx, ErrInternalServer))
		}
		return
	}

	server.setRefreshCookie(ctx, newRefreshToken, newRefreshPayload.ExpiresAt.Time)
	ctx.JSON(http.StatusOK, renewAccessTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiresAt.Time,
	})
}

func (server *Server) logoutCurrentSession(ctx *gin.Context) {
	if !server.originAllowed(ctx) {
		ctx.JSON(http.StatusForbidden, errorResponse(ctx, ErrForbidden))
		return
	}
	value, err := server.refreshCookie(ctx)
	deviceToken, deviceErr := server.deviceCookie(ctx)
	server.clearSessionCookies(ctx)
	if err != nil || deviceErr != nil {
		ctx.Status(http.StatusNoContent)
		return
	}

	payload, err := server.tokenMaker.VerifyRefreshToken(value)
	if err != nil {
		ctx.Status(http.StatusNoContent)
		return
	}
	err = server.store.RevokeSessionTx(
		ctx,
		pgtype.UUID{Bytes: payload.SessionID, Valid: true},
		token.Hash(deviceToken),
		"user_logout",
		"user",
	)
	if err != nil {
		if errors.Is(err, db.ErrDeviceMismatch) {
			ctx.Status(http.StatusNoContent)
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(ctx, ErrInternalServer))
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (server *Server) logoutAllSessions(ctx *gin.Context) {
	payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	if err := server.store.RevokeAllUserSessionsTx(
		ctx,
		payload.Username,
		"user_logout_all",
		"all_sessions_logged_out",
	); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(ctx, ErrInternalServer))
		return
	}
	server.clearSessionCookies(ctx)
	ctx.Status(http.StatusNoContent)
}
