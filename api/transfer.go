package api

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	db "github.com/faelic/monierave/db/sqlc"
	"github.com/faelic/monierave/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const idempotencyKeyHeader = "Idempotency-Key"

var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

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
	idempotencyKey := ctx.GetHeader(idempotencyKeyHeader)
	if idempotencyKey == "" {
		ctx.JSON(http.StatusBadRequest, errorResponse(ErrIdempotencyKeyRequired))
		return
	}
	if !idempotencyKeyPattern.MatchString(idempotencyKey) {
		ctx.JSON(http.StatusBadRequest, errorResponse(ErrInvalidIdempotencyKey))
		return
	}

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
	requestHash, err := hashTransferRequest(req, fromPublicID, toPublicID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrTransferFailed))
		return
	}
	result, err := server.store.IdempotentTransferTx(ctx, db.IdempotentTransferTxParams{
		TransferTxParams: db.TransferTxParams{
			FromAccountPublicID: fromPublicID,
			ToAccountPublicID:   toPublicID,
			Amount:              req.Amount,
			Currency:            req.Currency,
			Username:            authPayload.Username,
			Narration:           strings.TrimSpace(req.Narration),
			CorrelationID: pgtype.UUID{
				Bytes: uuid.New(), Valid: true,
			},
		},
		IdempotencyKey: idempotencyKey,
		RequestHash:    requestHash,
	})
	switch {
	case errors.Is(err, db.ErrIdempotencyConflict):
		ctx.JSON(
			http.StatusConflict,
			codedErrorResponse("idempotency_conflict", ErrIdempotencyConflict),
		)
		return
	case errors.Is(err, db.ErrAccountNotFound), errors.Is(err, db.ErrAccountNotOwned):
		ctx.JSON(
			http.StatusNotFound,
			codedErrorResponse("account_not_found", ErrAccountNotFound),
		)
		return
	case errors.Is(err, db.ErrInsufficientBalance):
		ctx.JSON(
			http.StatusBadRequest,
			codedErrorResponse("insufficient_funds", ErrInsufficientBalance),
		)
		return
	case errors.Is(err, db.ErrCurrencyMismatch):
		ctx.JSON(
			http.StatusBadRequest,
			codedErrorResponse("currency_mismatch", ErrCurrencyMismatch),
		)
		return
	case errors.Is(err, db.ErrSameAccount):
		ctx.JSON(
			http.StatusBadRequest,
			codedErrorResponse("same_account", ErrSameAccount),
		)
		return
	case errors.Is(err, db.ErrAccountFrozen):
		ctx.JSON(
			http.StatusConflict,
			codedErrorResponse("account_frozen", ErrAccountFrozen),
		)
		return
	case errors.Is(err, db.ErrAccountClosed):
		ctx.JSON(
			http.StatusConflict,
			codedErrorResponse("account_closed", ErrAccountClosed),
		)
		return
	case errors.Is(err, db.ErrPerTransferLimitExceeded):
		ctx.JSON(
			http.StatusConflict,
			codedErrorResponse("per_transfer_limit_exceeded", ErrPerTransferLimitExceeded),
		)
		return
	case errors.Is(err, db.ErrDailyTransferLimitExceeded):
		ctx.JSON(
			http.StatusConflict,
			codedErrorResponse("daily_transfer_limit_exceeded", ErrDailyTransferLimitExceeded),
		)
		return
	case err != nil:
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrTransferFailed))
		return
	}

	ctx.Header("Idempotency-Replayed", strconv.FormatBool(result.Replayed))
	ctx.JSON(http.StatusCreated, transferResponse{
		Transaction: newBankingTransactionResponse(result.Transaction),
		FromAccount: newAccountResponse(result.FromAccount),
		ToAccount:   newAccountResponse(result.ToAccount),
	})
}

func hashTransferRequest(
	req transferRequest,
	fromPublicID pgtype.UUID,
	toPublicID pgtype.UUID,
) ([]byte, error) {
	normalized := struct {
		FromAccountID string `json:"from_account_id"`
		ToAccountID   string `json:"to_account_id"`
		Amount        int64  `json:"amount"`
		Currency      string `json:"currency"`
		Narration     string `json:"narration"`
	}{
		FromAccountID: publicUUIDString(fromPublicID),
		ToAccountID:   publicUUIDString(toPublicID),
		Amount:        req.Amount,
		Currency:      req.Currency,
		Narration:     strings.TrimSpace(req.Narration),
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(payload)
	return hash[:], nil
}
