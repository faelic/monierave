package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/faelic/monierave/db/mock"
	db "github.com/faelic/monierave/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetTransactionAPIUsesOwnershipScopedReference(t *testing.T) {
	account := randomAccount()
	transactionID := uuid.New()
	createdAt := time.Now().UTC()
	row := db.GetOwnedTransactionByReferenceRow{
		ID:               pgtype.UUID{Bytes: transactionID, Valid: true},
		Reference:        "TXN-STATEMENT-TEST",
		TransactionType:  db.BankingTransactionTypeInternalTransfer,
		Status:           db.BankingTransactionStatusPosted,
		Currency:         account.Currency,
		Amount:           500,
		Narration:        "API history test",
		CreatedAt:        timestamp(createdAt),
		PostedAt:         timestamp(createdAt),
		AccountPublicID:  account.PublicID,
		SignedAmount:     -500,
		Direction:        "outgoing",
		CounterpartyKind: "customer",
		Counterparty:     "9374028641",
	}

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetOwnedTransactionByReference(
		gomock.Any(),
		db.GetOwnedTransactionByReferenceParams{
			Username: account.Owner, Reference: row.Reference,
		},
	).Return(row, nil)

	server := newTestServer(t, store)
	request := httptest.NewRequest(
		http.MethodGet,
		"/transactions/"+row.Reference,
		nil,
	)
	addAuthorization(
		t,
		request,
		server.tokenMaker,
		authorizationTypeBearer,
		account.Owner,
		time.Minute,
	)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response transactionResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, transactionID.String(), response.ID)
	require.Equal(t, row.Reference, response.Reference)
	require.Equal(t, "outgoing", response.Direction)
	require.Equal(t, publicUUID(account.PublicID), response.AccountID)
	require.Equal(t, row.Counterparty, response.Counterparty)
	require.NotEqual(t, publicUUID(account.PublicID), response.Counterparty)
	require.NotContains(t, recorder.Body.String(), `"signed_amount"`)
}

func TestGetTransactionAPIHidesForeignTransaction(t *testing.T) {
	account := randomAccount()
	reference := "TXN-FOREIGN"

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetOwnedTransactionByReference(
		gomock.Any(),
		db.GetOwnedTransactionByReferenceParams{
			Username: account.Owner, Reference: reference,
		},
	).Return(db.GetOwnedTransactionByReferenceRow{}, pgx.ErrNoRows)

	server := newTestServer(t, store)
	request := httptest.NewRequest(
		http.MethodGet,
		"/transactions/"+reference,
		nil,
	)
	addAuthorization(
		t,
		request,
		server.tokenMaker,
		authorizationTypeBearer,
		account.Owner,
		time.Minute,
	)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestListAccountTransactionsAPIReturnsCursorPage(t *testing.T) {
	account := randomAccount()
	now := time.Now().UTC()
	newestID := uuid.New()
	olderID := uuid.New()
	rows := []db.ListOwnedAccountTransactionsRow{
		historyRow(newestID, now, 900),
		historyRow(olderID, now.Add(-time.Minute), 1_000),
	}

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetOwnedAccountByPublicID(
		gomock.Any(),
		db.GetOwnedAccountByPublicIDParams{
			PublicID: account.PublicID, Owner: account.Owner,
		},
	).Return(account, nil)
	store.EXPECT().ListOwnedAccountTransactions(
		gomock.Any(),
		db.ListOwnedAccountTransactionsParams{
			PageLimit:       2,
			AccountPublicID: account.PublicID,
			Username:        account.Owner,
		},
	).Return(rows, nil)

	server := newTestServer(t, store)
	request := httptest.NewRequest(
		http.MethodGet,
		"/accounts/"+publicUUID(account.PublicID)+"/transactions?page_size=1",
		nil,
	)
	addAuthorization(
		t,
		request,
		server.tokenMaker,
		authorizationTypeBearer,
		account.Owner,
		time.Minute,
	)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response transactionHistoryResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Transactions, 1)
	require.Equal(t, newestID.String(), response.Transactions[0].ID)
	require.NotEmpty(t, response.NextCursor)

	cursorTime, cursorID, err := decodeTransactionCursor(response.NextCursor)
	require.NoError(t, err)
	require.Equal(t, newestID, uuid.UUID(cursorID.Bytes))
	require.WithinDuration(t, now, cursorTime.Time, time.Microsecond)
}

func TestGetAccountStatementAPI(t *testing.T) {
	account := randomAccount()
	row := historyRow(uuid.New(), time.Now().UTC(), 700)
	from := "2026-08-01"
	to := "2026-08-31"
	fromTime, toTime, err := parseTransactionDateRange(from, to)
	require.NoError(t, err)

	listParams := db.ListOwnedAccountTransactionsParams{
		PageLimit:       defaultTransactionPageSize + 1,
		AccountPublicID: account.PublicID,
		Username:        account.Owner,
		FromTime:        fromTime,
		ToTime:          toTime,
	}
	balanceParams := db.GetOwnedAccountStatementBalancesParams{
		FromTime:        fromTime,
		ToTime:          toTime,
		AccountPublicID: account.PublicID,
		Username:        account.Owner,
	}

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetOwnedAccountByPublicID(
		gomock.Any(),
		db.GetOwnedAccountByPublicIDParams{
			PublicID: account.PublicID, Owner: account.Owner,
		},
	).Return(account, nil)
	store.EXPECT().ListOwnedAccountTransactions(gomock.Any(), listParams).
		Return([]db.ListOwnedAccountTransactionsRow{row}, nil)
	store.EXPECT().GetOwnedAccountStatementBalances(gomock.Any(), balanceParams).
		Return(db.GetOwnedAccountStatementBalancesRow{
			OpeningBalance: 1_000,
			ClosingBalance: 700,
		}, nil)

	server := newTestServer(t, store)
	request := httptest.NewRequest(
		http.MethodGet,
		"/accounts/"+publicUUID(account.PublicID)+
			"/statement?from="+from+"&to="+to,
		nil,
	)
	addAuthorization(
		t,
		request,
		server.tokenMaker,
		authorizationTypeBearer,
		account.Owner,
		time.Minute,
	)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response accountStatementResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, int64(1_000), response.OpeningBalance)
	require.Equal(t, int64(700), response.ClosingBalance)
	require.Equal(t, publicUUID(account.PublicID), response.AccountID)
	require.Len(t, response.Transactions, 1)
	require.Equal(t, row.BalanceAfter, *response.Transactions[0].BalanceAfter)
}

func TestTransactionHistoryQueryValidation(t *testing.T) {
	account := randomAccount()
	testCases := []struct {
		name  string
		query string
	}{
		{name: "PageTooLarge", query: "?page_size=101"},
		{name: "BadCursor", query: "?cursor=not-base64"},
		{name: "BackwardsDates", query: "?from=2026-08-05&to=2026-08-01"},
		{name: "BadType", query: "?type=fee"},
		{name: "BadDirection", query: "?direction=sideways"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			server := newTestServer(t, store)
			request := httptest.NewRequest(
				http.MethodGet,
				"/accounts/"+publicUUID(account.PublicID)+"/transactions"+tc.query,
				nil,
			)
			addAuthorization(
				t,
				request,
				server.tokenMaker,
				authorizationTypeBearer,
				account.Owner,
				time.Minute,
			)
			recorder := httptest.NewRecorder()
			server.router.ServeHTTP(recorder, request)
			require.Equal(t, http.StatusBadRequest, recorder.Code)
		})
	}
}

func TestTransactionCursorRoundTrip(t *testing.T) {
	expected := transactionCursor{
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		ID:        uuid.New(),
	}
	encoded, err := encodeTransactionCursor(expected)
	require.NoError(t, err)
	createdAt, id, err := decodeTransactionCursor(encoded)
	require.NoError(t, err)
	require.Equal(t, expected.CreatedAt, createdAt.Time)
	require.Equal(t, expected.ID, uuid.UUID(id.Bytes))
}

func TestParseTransactionDateRange(t *testing.T) {
	from, to, err := parseTransactionDateRange("2026-08-01", "2026-08-05")
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), from.Time)
	require.Equal(t, time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC), to.Time)

	_, _, err = parseTransactionDateRange("2026-08-05", "2026-08-01")
	require.Error(t, err)
}

func historyRow(
	id uuid.UUID,
	createdAt time.Time,
	balanceAfter int64,
) db.ListOwnedAccountTransactionsRow {
	return db.ListOwnedAccountTransactionsRow{
		ID:               pgtype.UUID{Bytes: id, Valid: true},
		Reference:        "TXN-" + id.String(),
		TransactionType:  db.BankingTransactionTypeInternalTransfer,
		Status:           db.BankingTransactionStatusPosted,
		Currency:         "USD",
		Amount:           100,
		Narration:        "History API test",
		CreatedAt:        timestamp(createdAt),
		PostedAt:         timestamp(createdAt),
		SignedAmount:     -100,
		Direction:        "outgoing",
		CounterpartyKind: "customer",
		Counterparty:     "9374028641",
		BalanceAfter:     balanceAfter,
	}
}
