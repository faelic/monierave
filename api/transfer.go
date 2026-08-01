package api

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	db "github.com/faelic/monierave/db/sqlc"
	"github.com/faelic/monierave/token"
	"github.com/gin-gonic/gin"
)

type transferRequest struct {
	FromAccountID int64  `json:"from_account_id" binding:"required,min=1"`
	ToAccountID   int64  `json:"to_account_id" binding:"required,min=1"`
	Amount        int64  `json:"amount" binding:"required,gt=0"`
	Currency      string `json:"currency" binding:"required,currency"`
}

func (server *Server) createTransfer(ctx *gin.Context) {
	var req transferRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	fromAccount, valid := server.validAccount(ctx, req.FromAccountID, req.Currency)
	if !valid {
		return
	}

	authPayLoad := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	if fromAccount.Owner != authPayLoad.Username {
		log.Printf("from account does not belong to the authenticated user")
		ctx.JSON(http.StatusUnauthorized, errorResponse(ErrUnauthorized))
		return
	}

	_, valid = server.validAccount(ctx, req.ToAccountID, req.Currency)
	if !valid {
		return
	}

	arg := db.TransferTxParams{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount,
	}
	result, err := server.store.TransferTx(ctx, arg)
	if err != nil {
		if errors.Is(err, db.ErrInsufficientBalance) {
			ctx.JSON(http.StatusBadRequest, errorResponse(ErrInsufficientBalance))
			return
		}
		log.Printf("transfer failed from account %d to %d: %v", req.FromAccountID, req.ToAccountID, err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrTransferFailed))
		return
	}

	ctx.JSON(http.StatusOK, result)
}

func (server *Server) validAccount(ctx *gin.Context, accountID int64, currency string) (db.Account, bool) {
	account, err := server.store.GetAccount(ctx, accountID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(ErrAccountNotFound))
			return account, false
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return account, false
	}

	if account.Currency != currency {
		log.Printf("account %d currency mismatch: %s vs %s", account.ID, account.Currency, currency)
		ctx.JSON(http.StatusBadRequest, errorResponse(ErrCurrencyMismatch))
		return account, false
	}

	return account, true
}
