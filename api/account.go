package api

import (
	"errors"
	"net/http"
	"time"

	db "github.com/faelic/monierave/db/sqlc"
	"github.com/faelic/monierave/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type accountResponse struct {
	ID        string     `json:"id"`
	Balance   int64      `json:"balance"`
	Currency  string     `json:"currency"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at"`
}

func newAccountResponse(account db.Account) accountResponse {
	var closedAt *time.Time
	if account.ClosedAt.Valid {
		value := account.ClosedAt.Time
		closedAt = &value
	}

	return accountResponse{
		ID:        uuid.UUID(account.PublicID.Bytes).String(),
		Balance:   account.Balance,
		Currency:  account.Currency,
		Status:    account.Status,
		CreatedAt: account.CreatedAt.Time,
		UpdatedAt: account.UpdatedAt.Time,
		ClosedAt:  closedAt,
	}
}

func newAccountResponses(accounts []db.Account) []accountResponse {
	responses := make([]accountResponse, 0, len(accounts))
	for _, account := range accounts {
		responses = append(responses, newAccountResponse(account))
	}
	return responses
}

func parsePublicID(value string) (pgtype.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: id, Valid: true}, nil
}

func publicUUIDString(value pgtype.UUID) string {
	return uuid.UUID(value.Bytes).String()
}

type createAccountRequest struct {
	Currency string `json:"currency" binding:"required,currency"`
}

func (server *Server) createAccount(ctx *gin.Context) {
	var req createAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	account, err := server.store.CreateAccountTx(ctx, db.CreateAccountParams{
		Owner:    authPayload.Username,
		Currency: req.Currency,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "23505") {
			ctx.JSON(http.StatusConflict, errorResponse(ErrAccountAlreadyExists))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}

	ctx.JSON(http.StatusCreated, newAccountResponse(account))
}

type accountURIRequest struct {
	PublicID string `uri:"public_id" binding:"required"`
}

func (server *Server) getAccount(ctx *gin.Context) {
	publicID, ok := bindAccountPublicID(ctx)
	if !ok {
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	account, err := server.store.GetOwnedAccountByPublicID(ctx, db.GetOwnedAccountByPublicIDParams{
		PublicID: publicID,
		Owner:    authPayload.Username,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		ctx.JSON(http.StatusNotFound, errorResponse(ErrAccountNotFound))
		return
	}
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}

	ctx.JSON(http.StatusOK, newAccountResponse(account))
}

type listAccountRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=100"`
}

func (server *Server) listAccount(ctx *gin.Context) {
	var req listAccountRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	accounts, err := server.store.ListAccount(ctx, db.ListAccountParams{
		Owner:  authPayload.Username,
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}

	ctx.JSON(http.StatusOK, newAccountResponses(accounts))
}

func (server *Server) closeAccount(ctx *gin.Context) {
	publicID, ok := bindAccountPublicID(ctx)
	if !ok {
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	account, err := server.store.CloseAccountTx(ctx, db.CloseAccountTxParams{
		PublicID: publicID,
		Username: authPayload.Username,
	})
	switch {
	case errors.Is(err, db.ErrAccountNotFound), errors.Is(err, db.ErrAccountNotOwned):
		ctx.JSON(http.StatusNotFound, errorResponse(ErrAccountNotFound))
		return
	case errors.Is(err, db.ErrAccountBalanceNotZero):
		ctx.JSON(http.StatusConflict, errorResponse(ErrAccountBalanceNotZero))
		return
	case errors.Is(err, db.ErrAccountClosed):
		ctx.JSON(http.StatusConflict, errorResponse(ErrAccountAlreadyClosed))
		return
	case err != nil:
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}

	ctx.JSON(http.StatusOK, newAccountResponse(account))
}

func bindAccountPublicID(ctx *gin.Context) (pgtype.UUID, bool) {
	var req accountURIRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return pgtype.UUID{}, false
	}

	publicID, err := parsePublicID(req.PublicID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(ErrInvalidAccountID))
		return pgtype.UUID{}, false
	}
	return publicID, true
}
