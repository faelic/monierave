package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	mockdb "github.com/faelic/monierave/db/mock"
	db "github.com/faelic/monierave/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateTransferAPI(t *testing.T) {
	fromAccount := randomAccount()
	toAccount := randomAccount()
	toAccount.Currency = fromAccount.Currency
	amount := int64(100)
	transactionID := uuid.New()
	result := db.TransferTxResult{
		Transaction: db.BankingTransaction{
			ID:              pgtype.UUID{Bytes: transactionID, Valid: true},
			Reference:       "TXN-" + strings.ToUpper(strings.ReplaceAll(transactionID.String(), "-", "")),
			TransactionType: db.BankingTransactionTypeInternalTransfer,
			Status:          db.BankingTransactionStatusPosted,
			Currency:        fromAccount.Currency,
			Amount:          amount,
			CreatedAt:       timestamp(time.Now()),
			PostedAt:        timestamp(time.Now()),
		},
		FromAccount: fromAccount,
		ToAccount:   toAccount,
	}

	testCases := []struct {
		name       string
		changeBody func(map[string]any)
		key        string
		omitKey    bool
		replayed   bool
		storeError error
		statusCode int
		errorCode  string
	}{
		{name: "OK", statusCode: http.StatusCreated},
		{name: "Replay", replayed: true, statusCode: http.StatusCreated},
		{
			name:       "IdempotencyConflict",
			storeError: db.ErrIdempotencyConflict,
			statusCode: http.StatusConflict,
			errorCode:  "idempotency_conflict",
		},
		{name: "AccountNotFound", storeError: db.ErrAccountNotFound, statusCode: http.StatusNotFound, errorCode: "account_not_found"},
		{name: "ForeignSource", storeError: db.ErrAccountNotOwned, statusCode: http.StatusNotFound, errorCode: "account_not_found"},
		{name: "InsufficientBalance", storeError: db.ErrInsufficientBalance, statusCode: http.StatusBadRequest, errorCode: "insufficient_funds"},
		{name: "CurrencyMismatch", storeError: db.ErrCurrencyMismatch, statusCode: http.StatusBadRequest, errorCode: "currency_mismatch"},
		{name: "SameAccount", storeError: db.ErrSameAccount, statusCode: http.StatusBadRequest, errorCode: "same_account"},
		{name: "FrozenSource", storeError: db.ErrAccountFrozen, statusCode: http.StatusConflict, errorCode: "account_frozen"},
		{name: "ClosedAccount", storeError: db.ErrAccountClosed, statusCode: http.StatusConflict, errorCode: "account_closed"},
		{name: "PerTransferLimit", storeError: db.ErrPerTransferLimitExceeded, statusCode: http.StatusConflict, errorCode: "per_transfer_limit_exceeded"},
		{name: "DailyTransferLimit", storeError: db.ErrDailyTransferLimitExceeded, statusCode: http.StatusConflict, errorCode: "daily_transfer_limit_exceeded"},
		{name: "InternalError", storeError: errors.New("database unavailable"), statusCode: http.StatusInternalServerError},
		{
			name: "InvalidSourceUUID",
			changeBody: func(body map[string]any) {
				body["from_account_id"] = "invalid"
			},
			statusCode: http.StatusBadRequest,
		},
		{
			name: "InvalidAmount",
			changeBody: func(body map[string]any) {
				body["amount"] = 0
			},
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "MissingIdempotencyKey",
			omitKey:    true,
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "InvalidIdempotencyKey",
			key:        "spaces are not allowed",
			statusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]any{
				"from_account_id": publicUUID(fromAccount.PublicID),
				"to_account_id":   publicUUID(toAccount.PublicID),
				"amount":          amount,
				"currency":        fromAccount.Currency,
				"narration":       "Lunch repayment",
			}
			if tc.changeBody != nil {
				tc.changeBody(body)
			}

			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			key := tc.key
			if key == "" {
				key = "transfer-test-key"
			}
			callsStore := tc.changeBody == nil && !tc.omitKey &&
				idempotencyKeyPattern.MatchString(key)
			if callsStore {
				requestHash, err := hashTransferRequest(
					transferRequest{
						FromAccountID: publicUUID(fromAccount.PublicID),
						ToAccountID:   publicUUID(toAccount.PublicID),
						Amount:        amount,
						Currency:      fromAccount.Currency,
						Narration:     "Lunch repayment",
					},
					fromAccount.PublicID,
					toAccount.PublicID,
				)
				require.NoError(t, err)
				storeResult := result
				storeResult.Replayed = tc.replayed
				store.EXPECT().
					IdempotentTransferTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						arg db.IdempotentTransferTxParams,
					) (db.TransferTxResult, error) {
						require.Equal(t, fromAccount.PublicID, arg.FromAccountPublicID)
						require.Equal(t, toAccount.PublicID, arg.ToAccountPublicID)
						require.Equal(t, amount, arg.Amount)
						require.Equal(t, fromAccount.Currency, arg.Currency)
						require.Equal(t, fromAccount.Owner, arg.Username)
						require.Equal(t, "Lunch repayment", arg.Narration)
						require.True(t, arg.CorrelationID.Valid)
						require.Equal(t, key, arg.IdempotencyKey)
						require.Equal(t, requestHash, arg.RequestHash)
						return storeResult, tc.storeError
					})
			}

			server := newTestServer(t, store)
			data, err := json.Marshal(body)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewReader(data))
			if !tc.omitKey {
				request.Header.Set(idempotencyKeyHeader, key)
			}
			addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, fromAccount.Owner, time.Minute)
			recorder := httptest.NewRecorder()
			server.router.ServeHTTP(recorder, request)

			require.Equal(t, tc.statusCode, recorder.Code)
			if tc.storeError == nil && callsStore {
				var response transferResponse
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
				require.Equal(t, publicUUID(fromAccount.PublicID), response.FromAccount.ID)
				require.Equal(t, publicUUID(toAccount.PublicID), response.ToAccount.ID)
				require.Equal(t, amount, response.Transaction.Amount)
				require.Equal(t, result.Transaction.Reference, response.Transaction.Reference)
				require.Equal(t, db.BankingTransactionStatusPosted, response.Transaction.Status)
				require.NotContains(t, recorder.Body.String(), `"owner"`)
				require.Equal(
					t,
					strconv.FormatBool(tc.replayed),
					recorder.Header().Get("Idempotency-Replayed"),
				)
			}
			if tc.errorCode != "" {
				var response map[string]any
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
				require.Equal(t, tc.errorCode, response["code"])
				require.NotEmpty(t, response["message"])
			}
		})
	}
}
