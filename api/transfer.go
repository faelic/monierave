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
	Narration     string `json:"narration" binding:"omitempty,max=255"`
}

type bankingTransactionResponse struct {
	ID              string     `json:"id"`
	Reference       string     `json:"reference"`
	TransactionType string     `json:"type"`
	Status          string     `json:"status"`
	Currency        string     `json:"currency"`
	Amount          int64      `json:"amount"`
	Narration       string     `json:"narration"`
	CreatedAt       time.Time  `json:"created_at"`
	PostedAt        *time.Time `json:"posted_at"`
}

func newBankingTransactionResponse(
	transaction db.BankingTransaction,
) bankingTransactionResponse {
	var postedAt *time.Time
	if transaction.PostedAt.Valid {
		value := transaction.PostedAt.Time
		postedAt = &value
	}
	return bankingTransactionResponse{
		ID:              publicUUIDString(transaction.ID),
		Reference:       transaction.Reference,
		TransactionType: transaction.TransactionType,
		Status:          transaction.Status,
		Currency:        transaction.Currency,
		Amount:          transaction.Amount,
		Narration:       transaction.Narration,
		CreatedAt:       transaction.CreatedAt.Time,
		PostedAt:        postedAt,
	}
}

type transferResponse struct {
	Transaction bankingTransactionResponse `json:"transaction"`
	FromAccount accountResponse            `json:"from_account"`
	ToAccount   accountResponse            `json:"to_account"`
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
		Narration:           req.Narration,
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
		Transaction: newBankingTransactionResponse(result.Transaction),
		FromAccount: newAccountResponse(result.FromAccount),
		ToAccount:   newAccountResponse(result.ToAccount),
	})
}
