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
	Username          string             `json:"username"`
	FullName          string             `json:"full_name"`
	Email             string             `json:"email"`
	PasswordChangedAt pgtype.Timestamptz `json:"password_changed_at"`
	CreatedAt         pgtype.Timestamptz `json:"created_at"`
}

func newUserResponse(user db.User) userResponse {
	return userResponse{
		Username:          user.Username,
		FullName:          user.FullName,
		Email:             user.Email,
		PasswordChangedAt: user.PasswordChangedAt,
		CreatedAt:         user.CreatedAt,
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
	SessionID             pgtype.UUID  `json:"session_id"`
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	User                  userResponse `json:"user"`
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
			ctx.JSON(http.StatusUnauthorized, errorResponse(ErrInvalidCredentials))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}

	if err := util.CheckPassword(req.Password, user.HashedPassword); err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(ErrInvalidCredentials))
		return
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(user.Username, server.config.AccessTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}

	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		server.config.RefreshTokenDuration,
	)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}

	session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
		ID: pgtype.UUID{
			Bytes: refreshPayload.ID,
			Valid: true,
		},
		Username:     refreshPayload.Username,
		RefreshToken: refreshToken,
		UserAgent:    ctx.Request.UserAgent(),
		ClientIp:     ctx.ClientIP(),
		IsBlocked:    false,
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

	rsp := loginUserResponse{
		SessionID:             session.ID,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiresAt.Time,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiresAt.Time,
		User:                  newUserResponse(user),
	}

	ctx.JSON(http.StatusOK, rsp)
}

type updateUserRequest struct {
	Password *string `json:"password" binding:"omitempty,min=6"`
	FullName *string `json:"full_name" binding:"omitempty"`
	Email    *string `json:"email" binding:"omitempty,email"`
}

func (server *Server) updateUser(ctx *gin.Context) {
	var req updateUserRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(friendlyValidationError(err)))
		return
	}

	authPayLoad := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	arg := db.UpdateUserParams{
		Username: authPayLoad.Username,
	}

	if req.Password != nil {
		hashedpassword, err := util.HashPassword(*req.Password)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
			return
		}

		arg.HashedPassword = pgtype.Text{
			String: hashedpassword,
			Valid:  true,
		}
	}

	if req.FullName != nil {
		arg.FullName = pgtype.Text{
			String: *req.FullName,
			Valid:  true,
		}
	}

	if req.Email != nil {
		arg.Email = pgtype.Text{
			String: strings.ToLower(*req.Email),
			Valid:  true,
		}
	}

	user, err := server.store.UpdateUser(ctx, arg)
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

	ctx.JSON(http.StatusOK, newUserResponse(user))

}
