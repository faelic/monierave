package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/faelic/monierave/db/mock"
	db "github.com/faelic/monierave/db/sqlc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateTransferAPI(t *testing.T) {
	fromAccount := randomAccount()
	toAccount := randomAccount()
	toAccount.Currency = fromAccount.Currency
	amount := int64(100)
	result := db.TransferTxResult{
		Transfer: db.Transfer{
			Amount:    amount,
			CreatedAt: timestamp(time.Now()),
		},
		FromAccount: fromAccount,
		ToAccount:   toAccount,
	}

	testCases := []struct {
		name       string
		changeBody func(map[string]any)
		storeError error
		statusCode int
	}{
		{name: "OK", statusCode: http.StatusCreated},
		{name: "AccountNotFound", storeError: db.ErrAccountNotFound, statusCode: http.StatusNotFound},
		{name: "ForeignSource", storeError: db.ErrAccountNotOwned, statusCode: http.StatusNotFound},
		{name: "InsufficientBalance", storeError: db.ErrInsufficientBalance, statusCode: http.StatusBadRequest},
		{name: "CurrencyMismatch", storeError: db.ErrCurrencyMismatch, statusCode: http.StatusBadRequest},
		{name: "SameAccount", storeError: db.ErrSameAccount, statusCode: http.StatusBadRequest},
		{name: "FrozenSource", storeError: db.ErrAccountFrozen, statusCode: http.StatusConflict},
		{name: "ClosedAccount", storeError: db.ErrAccountClosed, statusCode: http.StatusConflict},
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]any{
				"from_account_id": publicUUID(fromAccount.PublicID),
				"to_account_id":   publicUUID(toAccount.PublicID),
				"amount":          amount,
				"currency":        fromAccount.Currency,
			}
			if tc.changeBody != nil {
				tc.changeBody(body)
			}

			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			if tc.changeBody == nil {
				store.EXPECT().
					TransferTx(gomock.Any(), db.TransferTxParams{
						FromAccountPublicID: fromAccount.PublicID,
						ToAccountPublicID:   toAccount.PublicID,
						Amount:              amount,
						Currency:            fromAccount.Currency,
						Username:            fromAccount.Owner,
					}).
					Return(result, tc.storeError)
			}

			server := newTestServer(t, store)
			data, err := json.Marshal(body)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewReader(data))
			addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, fromAccount.Owner, time.Minute)
			recorder := httptest.NewRecorder()
			server.router.ServeHTTP(recorder, request)

			require.Equal(t, tc.statusCode, recorder.Code)
			if tc.storeError == nil && tc.changeBody == nil {
				var response transferResponse
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
				require.Equal(t, publicUUID(fromAccount.PublicID), response.FromAccount.ID)
				require.Equal(t, publicUUID(toAccount.PublicID), response.ToAccount.ID)
				require.Equal(t, amount, response.Amount)
				require.NotContains(t, recorder.Body.String(), `"owner"`)
			}
		})
	}
}
