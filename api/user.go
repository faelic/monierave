package api

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	db "github.com/faelic/monierave/db/sqlc"
	"github.com/faelic/monierave/db/util"
	"github.com/faelic/monierave/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type createUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"full_name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

type userResponse struct {
	Username                  string             `json:"username"`
	FullName                  string             `json:"full_name"`
	Email                     string             `json:"email"`
	EmailVerifiedAt           pgtype.Timestamptz `json:"email_verified_at"`
	EmailDeliverabilityStatus string             `json:"email_deliverability_status"`
	EmailBouncedAt            pgtype.Timestamptz `json:"email_bounced_at"`
	AccountStatus             string             `json:"account_status"`
	RegistrationExpiresAt     pgtype.Timestamptz `json:"registration_expires_at"`
	PasswordChangedAt         pgtype.Timestamptz `json:"password_changed_at"`
	CreatedAt                 pgtype.Timestamptz `json:"created_at"`
}

func newUserResponse(user db.User) userResponse {
	return userResponse{
		Username:                  user.Username,
		FullName:                  user.FullName,
		Email:                     user.Email,
		EmailVerifiedAt:           user.EmailVerifiedAt,
		EmailDeliverabilityStatus: user.EmailDeliverabilityStatus,
		EmailBouncedAt:            user.EmailBouncedAt,
		AccountStatus:             user.AccountStatus,
		RegistrationExpiresAt:     user.RegistrationExpiresAt,
		PasswordChangedAt:         user.PasswordChangedAt,
		CreatedAt:                 user.CreatedAt,
	}
}

func (server *Server) CreateUser(ctx *gin.Context) {
	var req createUserRequest
	var pgErr *pgconn.PgError

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(friendlyValidationError(err)))
		return
	}

	arg := db.CreateUserParams{
		Username: strings.ToLower(strings.TrimSpace(req.Username)),
		FullName: strings.TrimSpace(req.FullName),
		Email:    strings.ToLower(strings.TrimSpace(req.Email)),
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		log.Printf("failed to hash password for user %q: %v", arg.Username, err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}

	arg.HashedPassword = hashedPassword

	result, err := server.store.CreateUserTx(ctx, arg)
	if err != nil {
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				switch pgErr.ConstraintName {
				case "users_pkey":
					ctx.JSON(http.StatusForbidden, errorResponse(ErrUsernameAlreadyExists))
					return
				case "users_email_lower_idx", "users_email_key":
					ctx.JSON(http.StatusForbidden, errorResponse(ErrEmailAlreadyExists))
					return
				}
			}
		}
		log.Printf("failed to create user %q: %v", arg.Username, err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}

	rsp := newUserResponse(result.User)

	ctx.JSON(http.StatusCreated, rsp)
}

type loginUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=6"`
}

type loginUserResponse struct {
	SessionID            pgtype.UUID  `json:"session_id"`
	AccessToken          string       `json:"access_token"`
	AccessTokenExpiresAt time.Time    `json:"access_token_expires_at"`
	User                 userResponse `json:"user"`
}

func (server *Server) loginUser(ctx *gin.Context) {
	var req loginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(friendlyValidationError(err)))
		return
	}

	normalizedUsername := strings.ToLower(req.Username)

	user, err := server.store.GetUser(ctx, normalizedUsername)
	if err != nil {
		if err == pgx.ErrNoRows {
			server.recordLoginFailure(ctx, normalizedUsername, "unknown_user")
			ctx.JSON(http.StatusUnauthorized, errorResponse(ErrInvalidCredentials))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}

	if err := util.CheckPassword(req.Password, user.HashedPassword); err != nil {
		server.recordLoginFailure(ctx, normalizedUsername, "invalid_password")
		ctx.JSON(http.StatusUnauthorized, errorResponse(ErrInvalidCredentials))
		return
	}

	sessionUUID := uuid.New()
	refreshToken, refreshPayload, err := server.tokenMaker.CreateRefreshToken(
		user.Username,
		sessionUUID,
		server.config.RefreshTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateAccessToken(
		user.Username,
		sessionUUID,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}

	session, err := server.store.CreateSessionTx(ctx, db.CreateSessionParams{
		ID: pgtype.UUID{
			Bytes: sessionUUID,
			Valid: true,
		},
		Username:         refreshPayload.Username,
		RefreshTokenHash: token.Hash(refreshToken),
		RefreshTokenID: pgtype.UUID{
			Bytes: refreshPayload.ID,
			Valid: true,
		},
		UserAgent: ctx.Request.UserAgent(),
		ClientIp:  ctx.ClientIP(),
		ExpiresAt: pgtype.Timestamptz{
			Time:  refreshPayload.ExpiresAt.Time,
			Valid: true,
		},
	})
	if err != nil {
		log.Printf("failed to create session with this error %v:", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}

	server.setRefreshCookie(ctx, refreshToken, refreshPayload.ExpiresAt.Time)
	rsp := loginUserResponse{
		SessionID:            session.ID,
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiresAt.Time,
		User:                 newUserResponse(user),
	}

	ctx.JSON(http.StatusOK, rsp)
}

func (server *Server) recordLoginFailure(
	ctx *gin.Context,
	username string,
	reason string,
) {
	if err := server.store.RecordLoginFailure(ctx, db.LoginFailureAuditParams{
		Username:  username,
		ClientIP:  ctx.ClientIP(),
		UserAgent: ctx.Request.UserAgent(),
		Reason:    reason,
	}); err != nil {
		log.Printf("failed to record login failure audit: %v", err)
	}
}

type updateUserRequest struct {
	CurrentPassword *string `json:"current_password"`
	Password        *string `json:"password" binding:"omitempty,min=6"`
	FullName        *string `json:"full_name" binding:"omitempty"`
	Email           *string `json:"email" binding:"omitempty,email"`
}

func (server *Server) updateUser(ctx *gin.Context) {
	var req updateUserRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(friendlyValidationError(err)))
		return
	}

	authPayLoad := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	arg := db.UpdateUserTxParams{
		UpdateUserParams: db.UpdateUserParams{
			Username: authPayLoad.Username,
		},
	}

	if req.Password != nil {
		if req.CurrentPassword == nil {
			ctx.JSON(http.StatusBadRequest, errorResponse(ErrCurrentPasswordRequired))
			return
		}
		currentUser, err := server.store.GetUser(ctx, authPayLoad.Username)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				ctx.JSON(http.StatusNotFound, errorResponse(ErrUserNotFound))
				return
			}
			ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
			return
		}
		if err := util.CheckPassword(*req.CurrentPassword, currentUser.HashedPassword); err != nil {
			ctx.JSON(http.StatusUnauthorized, errorResponse(ErrInvalidCredentials))
			return
		}
		hashedpassword, err := util.HashPassword(*req.Password)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
			return
		}

		arg.UpdateUserParams.HashedPassword = pgtype.Text{
			String: hashedpassword,
			Valid:  true,
		}
		arg.RevokeSessions = true
	}

	if req.FullName != nil {
		arg.UpdateUserParams.FullName = pgtype.Text{
			String: strings.TrimSpace(*req.FullName),
			Valid:  true,
		}
	}

	if req.Email != nil {
		arg.UpdateUserParams.Email = pgtype.Text{
			String: strings.ToLower(strings.TrimSpace(*req.Email)),
			Valid:  true,
		}
	}

	result, err := server.store.UpdateUserTx(ctx, arg)
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				switch pgErr.ConstraintName {
				case "users_email_lower_idx", "users_email_key":
					ctx.JSON(http.StatusForbidden, errorResponse(ErrEmailAlreadyExists))
					return
				}
			}
		}

		if err == pgx.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(ErrUserNotFound))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}

	if arg.RevokeSessions {
		server.clearRefreshCookie(ctx)
	}
	ctx.JSON(http.StatusOK, newUserResponse(result.User))

}

type emailStatusResponse struct {
	Email                 string             `json:"email"`
	VerifiedAt            pgtype.Timestamptz `json:"verified_at"`
	DeliverabilityStatus  string             `json:"deliverability_status"`
	BouncedAt             pgtype.Timestamptz `json:"bounced_at"`
	LatestJob             *emailJobStatus    `json:"latest_job,omitempty"`
	AccountStatus         string             `json:"account_status"`
	RegistrationExpiresAt pgtype.Timestamptz `json:"registration_expires_at"`
	AllowedFeatures       []string           `json:"allowed_features,omitempty"`
	RestrictedFeatures    []string           `json:"restricted_features,omitempty"`
}

type emailJobStatus struct {
	ID                pgtype.UUID        `json:"id"`
	WorkerStatus      string             `json:"worker_status"`
	DeliveryStatus    string             `json:"delivery_status"`
	ProviderMessageID pgtype.Text        `json:"provider_message_id"`
	AcceptedAt        pgtype.Timestamptz `json:"accepted_at"`
	DeliveredAt       pgtype.Timestamptz `json:"delivered_at"`
	BouncedAt         pgtype.Timestamptz `json:"bounced_at"`
	BounceType        pgtype.Text        `json:"bounce_type"`
	BounceSubtype     pgtype.Text        `json:"bounce_subtype"`
	BounceMessage     pgtype.Text        `json:"bounce_message"`
}

func (server *Server) getUserEmailStatus(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	user, err := server.store.GetUser(ctx, authPayload.Username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, errorResponse(ErrUserNotFound))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}

	response := emailStatusResponse{
		Email:                 user.Email,
		VerifiedAt:            user.EmailVerifiedAt,
		DeliverabilityStatus:  user.EmailDeliverabilityStatus,
		BouncedAt:             user.EmailBouncedAt,
		AccountStatus:         user.AccountStatus,
		RegistrationExpiresAt: user.RegistrationExpiresAt,
	}
	if user.AccountStatus != db.AccountStatusActive || !user.EmailVerifiedAt.Valid {
		response.AllowedFeatures = unverifiedAllowedFeatures
		response.RestrictedFeatures = unverifiedRestrictedFeatures
	}

	job, err := server.store.GetLatestEmailJobForCurrentAddress(ctx, authPayload.Username)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}
	if err == nil {
		response.LatestJob = &emailJobStatus{
			ID:                job.ID,
			WorkerStatus:      job.Status,
			DeliveryStatus:    job.DeliveryStatus,
			ProviderMessageID: job.ProviderMessageID,
			AcceptedAt:        job.AcceptedAt,
			DeliveredAt:       job.DeliveredAt,
			BouncedAt:         job.BouncedAt,
			BounceType:        job.BounceType,
			BounceSubtype:     job.BounceSubtype,
			BounceMessage:     job.BounceMessage,
		}
	}

	ctx.JSON(http.StatusOK, response)
}

type verifyEmailRequest struct {
	Token string `form:"token" binding:"required"`
}

func (server *Server) verifyUserEmail(ctx *gin.Context) {
	var req verifyEmailRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(ErrInvalidToken))
		return
	}

	payload, err := server.emailVerificationMaker.Verify(req.Token)
	if err != nil {
		if errors.Is(err, token.ErrExpiredToken) {
			ctx.JSON(http.StatusGone, errorResponse(ErrExpiredToken))
			return
		}
		ctx.JSON(http.StatusBadRequest, errorResponse(ErrInvalidToken))
		return
	}

	jobID, err := uuid.Parse(payload.JobID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(ErrInvalidToken))
		return
	}
	user, err := server.store.VerifyUserEmailTx(ctx, db.VerifyUserEmailTxParams{
		Username: payload.Username,
		Email:    payload.Email,
		JobID:    pgtype.UUID{Bytes: jobID, Valid: true},
	})
	if err != nil {
		switch {
		case errors.Is(err, db.ErrEmailVerificationAddressStale),
			errors.Is(err, db.ErrEmailVerificationJobMismatch):
			ctx.JSON(http.StatusConflict, errorResponse(ErrInvalidToken))
		case errors.Is(err, db.ErrRegistrationDisabled):
			ctx.JSON(http.StatusGone, errorResponse(ErrRegistrationExpired))
		default:
			ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "email verified successfully",
		"user":    newUserResponse(user),
	})
}

func (server *Server) resendUserEmailVerification(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	result, err := server.store.RequestEmailVerificationTx(ctx, authPayload.Username)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrEmailAlreadyVerified):
			ctx.JSON(http.StatusConflict, errorResponse(ErrEmailAlreadyVerified))
		case errors.Is(err, db.ErrEmailVerificationCooldown):
			ctx.JSON(http.StatusTooManyRequests, errorResponse(ErrVerificationCooldown))
		case errors.Is(err, pgx.ErrNoRows):
			ctx.JSON(http.StatusNotFound, errorResponse(ErrUserNotFound))
		default:
			ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		}
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{
		"message": "verification email queued",
		"job_id":  result.EmailJob.ID,
	})
}
