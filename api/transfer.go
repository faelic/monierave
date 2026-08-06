package api

import (
	"errors"
	"net/http"
	"time"

	db "github.com/faelic/monierave/db/sqlc"
	"github.com/faelic/monierave/token"
	"github.com/gin-gonic/gin"
)

type transferRequest struct {
	FromAccountID string `json:"from_account_id" binding:"required"`
	ToAccountID   string `json:"to_account_id" binding:"required"`
	Amount        int64  `json:"amount" binding:"required,gt=0"`
	Currency      string `json:"currency" binding:"required,currency"`
}

type transferResponse struct {
	FromAccount accountResponse `json:"from_account"`
	ToAccount   accountResponse `json:"to_account"`
	Amount      int64           `json:"amount"`
	CreatedAt   time.Time       `json:"created_at"`
}

func (server *Server) createTransfer(ctx *gin.Context) {
	var req transferRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	fromPublicID, err := parsePublicID(req.FromAccountID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(ErrInvalidAccountID))
		return
	}
	toPublicID, err := parsePublicID(req.ToAccountID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(ErrInvalidAccountID))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	result, err := server.store.TransferTx(ctx, db.TransferTxParams{
		FromAccountPublicID: fromPublicID,
		ToAccountPublicID:   toPublicID,
		Amount:              req.Amount,
		Currency:            req.Currency,
		Username:            authPayload.Username,
	})
	switch {
	case errors.Is(err, db.ErrAccountNotFound), errors.Is(err, db.ErrAccountNotOwned):
		ctx.JSON(http.StatusNotFound, errorResponse(ErrAccountNotFound))
		return
	case errors.Is(err, db.ErrInsufficientBalance):
		ctx.JSON(http.StatusBadRequest, errorResponse(ErrInsufficientBalance))
		return
	case errors.Is(err, db.ErrCurrencyMismatch):
		ctx.JSON(http.StatusBadRequest, errorResponse(ErrCurrencyMismatch))
		return
	case errors.Is(err, db.ErrSameAccount):
		ctx.JSON(http.StatusBadRequest, errorResponse(ErrSameAccount))
		return
	case errors.Is(err, db.ErrAccountFrozen):
		ctx.JSON(http.StatusConflict, errorResponse(ErrAccountFrozen))
		return
	case errors.Is(err, db.ErrAccountClosed):
		ctx.JSON(http.StatusConflict, errorResponse(ErrAccountClosed))
		return
	case err != nil:
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrTransferFailed))
		return
	}

	ctx.JSON(http.StatusCreated, transferResponse{
		FromAccount: newAccountResponse(result.FromAccount),
		ToAccount:   newAccountResponse(result.ToAccount),
		Amount:      result.Transfer.Amount,
		CreatedAt:   result.Transfer.CreatedAt.Time,
	})
}
