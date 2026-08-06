package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	db "github.com/faelic/monierave/db/sqlc"
	"github.com/faelic/monierave/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	defaultTransactionPageSize = int32(20)
	maxTransactionPageSize     = int32(100)
)

type transactionHistoryQuery struct {
	Cursor    string `form:"cursor"`
	PageSize  int32  `form:"page_size" binding:"omitempty,min=1,max=100"`
	From      string `form:"from"`
	To        string `form:"to"`
	Type      string `form:"type" binding:"omitempty,oneof=deposit withdrawal internal_transfer reversal"`
	Status    string `form:"status" binding:"omitempty,oneof=pending posted failed reversed"`
	Direction string `form:"direction" binding:"omitempty,oneof=incoming outgoing"`
}

type transactionCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        uuid.UUID `json:"id"`
}

type transactionHistoryResponse struct {
	Transactions []transactionResponse `json:"transactions"`
	NextCursor   string                `json:"next_cursor,omitempty"`
}

type accountStatementResponse struct {
	AccountID      string                `json:"account_id"`
	Currency       string                `json:"currency"`
	From           *time.Time            `json:"from"`
	To             *time.Time            `json:"to"`
	OpeningBalance int64                 `json:"opening_balance"`
	ClosingBalance int64                 `json:"closing_balance"`
	Transactions   []transactionResponse `json:"transactions"`
	NextCursor     string                `json:"next_cursor,omitempty"`
}

type transactionResponse struct {
	ID               string     `json:"id"`
	Reference        string     `json:"reference"`
	AccountID        string     `json:"account_id"`
	Type             string     `json:"type"`
	Status           string     `json:"status"`
	Currency         string     `json:"currency"`
	Amount           int64      `json:"amount"`
	Direction        string     `json:"direction"`
	Narration        string     `json:"narration"`
	CounterpartyType string     `json:"counterparty_type"`
	Counterparty     string     `json:"counterparty"`
	BalanceAfter     *int64     `json:"balance_after,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	PostedAt         *time.Time `json:"posted_at"`
}

func (server *Server) getTransaction(ctx *gin.Context) {
	reference := strings.TrimSpace(ctx.Param("reference"))
	if reference == "" {
		ctx.JSON(http.StatusBadRequest, errorResponse(ErrTransactionNotFound))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	row, err := server.store.GetOwnedTransactionByReference(
		ctx,
		db.GetOwnedTransactionByReferenceParams{
			Username:  authPayload.Username,
			Reference: reference,
		},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		ctx.JSON(http.StatusNotFound, errorResponse(ErrTransactionNotFound))
		return
	}
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}

	ctx.JSON(http.StatusOK, transactionResponseFromDetail(row))
}

func (server *Server) listAccountTransactions(ctx *gin.Context) {
	account, query, params, ok := server.bindTransactionHistoryRequest(ctx)
	if !ok {
		return
	}

	rows, err := server.store.ListOwnedAccountTransactions(ctx, params)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}
	transactions, nextCursor, err := transactionPage(rows, query.PageSize, account.PublicID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}

	ctx.JSON(http.StatusOK, transactionHistoryResponse{
		Transactions: transactions,
		NextCursor:   nextCursor,
	})
}

func (server *Server) getAccountStatement(ctx *gin.Context) {
	account, query, params, ok := server.bindTransactionHistoryRequest(ctx)
	if !ok {
		return
	}

	rows, err := server.store.ListOwnedAccountTransactions(ctx, params)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}
	balances, err := server.store.GetOwnedAccountStatementBalances(
		ctx,
		db.GetOwnedAccountStatementBalancesParams{
			FromTime:        params.FromTime,
			ToTime:          params.ToTime,
			AccountPublicID: params.AccountPublicID,
			Username:        params.Username,
		},
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}
	transactions, nextCursor, err := transactionPage(rows, query.PageSize, account.PublicID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return
	}

	ctx.JSON(http.StatusOK, accountStatementResponse{
		AccountID:      publicUUIDString(account.PublicID),
		Currency:       account.Currency,
		From:           optionalTime(params.FromTime),
		To:             optionalTime(params.ToTime),
		OpeningBalance: balances.OpeningBalance,
		ClosingBalance: balances.ClosingBalance,
		Transactions:   transactions,
		NextCursor:     nextCursor,
	})
}

func (server *Server) bindTransactionHistoryRequest(
	ctx *gin.Context,
) (db.Account, transactionHistoryQuery, db.ListOwnedAccountTransactionsParams, bool) {
	var query transactionHistoryQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return db.Account{}, query, db.ListOwnedAccountTransactionsParams{}, false
	}
	if query.PageSize == 0 {
		query.PageSize = defaultTransactionPageSize
	}

	publicID, ok := bindAccountPublicID(ctx)
	if !ok {
		return db.Account{}, query, db.ListOwnedAccountTransactionsParams{}, false
	}
	fromTime, toTime, err := parseTransactionDateRange(query.From, query.To)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(ErrInvalidDateRange))
		return db.Account{}, query, db.ListOwnedAccountTransactionsParams{}, false
	}
	cursorTime, cursorID, err := decodeTransactionCursor(query.Cursor)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(ErrInvalidCursor))
		return db.Account{}, query, db.ListOwnedAccountTransactionsParams{}, false
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	account, err := server.store.GetOwnedAccountByPublicID(
		ctx,
		db.GetOwnedAccountByPublicIDParams{
			PublicID: publicID,
			Owner:    authPayload.Username,
		},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		ctx.JSON(http.StatusNotFound, errorResponse(ErrAccountNotFound))
		return db.Account{}, query, db.ListOwnedAccountTransactionsParams{}, false
	}
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(ErrInternalServer))
		return db.Account{}, query, db.ListOwnedAccountTransactionsParams{}, false
	}

	return account, query, db.ListOwnedAccountTransactionsParams{
		PageLimit:         query.PageSize + 1,
		AccountPublicID:   publicID,
		Username:          authPayload.Username,
		FromTime:          fromTime,
		ToTime:            toTime,
		TransactionType:   optionalText(query.Type),
		TransactionStatus: optionalText(query.Status),
		Direction:         optionalText(query.Direction),
		CursorCreatedAt:   cursorTime,
		CursorID:          cursorID,
	}, true
}

func transactionPage(
	rows []db.ListOwnedAccountTransactionsRow,
	pageSize int32,
	accountPublicID pgtype.UUID,
) ([]transactionResponse, string, error) {
	hasNextPage := len(rows) > int(pageSize)
	if hasNextPage {
		rows = rows[:pageSize]
	}
	transactions := make([]transactionResponse, 0, len(rows))
	for _, row := range rows {
		transactions = append(
			transactions,
			transactionResponseFromList(row, accountPublicID),
		)
	}
	if !hasNextPage || len(rows) == 0 {
		return transactions, "", nil
	}
	last := rows[len(rows)-1]
	cursor, err := encodeTransactionCursor(transactionCursor{
		CreatedAt: last.CreatedAt.Time,
		ID:        uuid.UUID(last.ID.Bytes),
	})
	return transactions, cursor, err
}

func transactionResponseFromList(
	row db.ListOwnedAccountTransactionsRow,
	accountPublicID pgtype.UUID,
) transactionResponse {
	balanceAfter := row.BalanceAfter
	return transactionResponse{
		ID:               publicUUIDString(row.ID),
		Reference:        row.Reference,
		AccountID:        publicUUIDString(accountPublicID),
		Type:             row.TransactionType,
		Status:           row.Status,
		Currency:         row.Currency,
		Amount:           row.Amount,
		Direction:        row.Direction,
		Narration:        row.Narration,
		CounterpartyType: publicCounterpartyType(row.CounterpartyKind),
		Counterparty:     row.Counterparty,
		BalanceAfter:     &balanceAfter,
		CreatedAt:        row.CreatedAt.Time,
		PostedAt:         optionalTime(row.PostedAt),
	}
}

func transactionResponseFromDetail(
	row db.GetOwnedTransactionByReferenceRow,
) transactionResponse {
	return transactionResponse{
		ID:               publicUUIDString(row.ID),
		Reference:        row.Reference,
		AccountID:        publicUUIDString(row.AccountPublicID),
		Type:             row.TransactionType,
		Status:           row.Status,
		Currency:         row.Currency,
		Amount:           row.Amount,
		Direction:        row.Direction,
		Narration:        row.Narration,
		CounterpartyType: publicCounterpartyType(row.CounterpartyKind),
		Counterparty:     row.Counterparty,
		CreatedAt:        row.CreatedAt.Time,
		PostedAt:         optionalTime(row.PostedAt),
	}
}

func publicCounterpartyType(kind string) string {
	if kind == "settlement" {
		return "system"
	}
	return kind
}

func optionalText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

func optionalTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func parseTransactionDateRange(
	fromValue string,
	toValue string,
) (pgtype.Timestamptz, pgtype.Timestamptz, error) {
	from, err := parseTransactionBoundary(fromValue, false)
	if err != nil {
		return pgtype.Timestamptz{}, pgtype.Timestamptz{}, err
	}
	to, err := parseTransactionBoundary(toValue, true)
	if err != nil {
		return pgtype.Timestamptz{}, pgtype.Timestamptz{}, err
	}
	if from.Valid && to.Valid && !from.Time.Before(to.Time) {
		return pgtype.Timestamptz{}, pgtype.Timestamptz{}, ErrInvalidDateRange
	}
	return from, to, nil
}

func parseTransactionBoundary(
	value string,
	endOfDate bool,
) (pgtype.Timestamptz, error) {
	if value == "" {
		return pgtype.Timestamptz{}, nil
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		if endOfDate {
			parsed = parsed.AddDate(0, 0, 1)
		}
		return pgtype.Timestamptz{Time: parsed, Valid: true}, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return pgtype.Timestamptz{}, err
	}
	return pgtype.Timestamptz{Time: parsed, Valid: true}, nil
}

func encodeTransactionCursor(cursor transactionCursor) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeTransactionCursor(
	value string,
) (pgtype.Timestamptz, pgtype.UUID, error) {
	if value == "" {
		return pgtype.Timestamptz{}, pgtype.UUID{}, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return pgtype.Timestamptz{}, pgtype.UUID{}, err
	}
	var cursor transactionCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return pgtype.Timestamptz{}, pgtype.UUID{}, err
	}
	if cursor.CreatedAt.IsZero() || cursor.ID == uuid.Nil {
		return pgtype.Timestamptz{}, pgtype.UUID{}, ErrInvalidCursor
	}
	return pgtype.Timestamptz{
			Time: cursor.CreatedAt, Valid: true,
		}, pgtype.UUID{
			Bytes: cursor.ID, Valid: true,
		}, nil
}
