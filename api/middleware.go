package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	db "github.com/faelic/monierave/db/sqlc"
	"github.com/faelic/monierave/token"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

const (
	authorizationHeaderKey  = "Authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "authorization_payload"
)

func authMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authorizationHeader := ctx.GetHeader(authorizationHeaderKey)

		if len(authorizationHeader) == 0 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(ErrUnauthorized))
			return
		}

		fields := strings.Fields(authorizationHeader)

		if len(fields) != 2 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(ErrUnauthorized))
			return
		}
		authorizationType := strings.ToLower(fields[0])

		if authorizationType != authorizationTypeBearer {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(ErrUnauthorized))
			return
		}

		accessToken := fields[1]
		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(ErrUnauthorized))
			return
		}

		ctx.Set(authorizationPayloadKey, payload)
		ctx.Next()
	}
}

var unverifiedAllowedFeatures = []string{
	"login and refresh access tokens",
	"view email verification status",
	"change profile details or email address",
	"request another verification email",
	"verify the current email address",
}

var unverifiedRestrictedFeatures = []string{
	"create, view, update, or delete accounts",
	"send money or create transfers",
}

func verifiedAccountMiddleware(store db.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
		user, err := store.GetUser(ctx, payload.Username)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(ErrUnauthorized))
				return
			}
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
			return
		}

		if user.AccountStatus == db.AccountStatusActive && user.EmailVerifiedAt.Valid {
			ctx.Next()
			return
		}

		expired := user.AccountStatus == db.AccountStatusDisabled ||
			(user.AccountStatus == db.AccountStatusPending &&
				user.RegistrationExpiresAt.Valid &&
				!time.Now().Before(user.RegistrationExpiresAt.Time))
		if expired && user.AccountStatus == db.AccountStatusPending {
			_, disableErr := store.DisableExpiredPendingUser(ctx, user.Username)
			if disableErr != nil && !errors.Is(disableErr, pgx.ErrNoRows) {
				ctx.AbortWithStatusJSON(
					http.StatusInternalServerError,
					errorResponse(ErrInternalServer),
				)
				return
			}
		}

		message := ErrEmailVerificationRequired
		if expired {
			message = ErrRegistrationExpired
		}
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error":               message.Error(),
			"account_status":      user.AccountStatus,
			"allowed_features":    unverifiedAllowedFeatures,
			"restricted_features": unverifiedRestrictedFeatures,
		})
	}
}
