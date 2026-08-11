package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mockdb "github.com/faelic/monierave/db/mock"
	db "github.com/faelic/monierave/db/sqlc"
	"github.com/faelic/monierave/db/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateAccountAPI(t *testing.T) {
	account := randomAccount()
	account.Balance = 0

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().
		CreateAccountTx(gomock.Any(), db.CreateAccountTxParams{
			Owner:    account.Owner,
			Currency: account.Currency,
		}).
		Return(account, nil)

	server := newTestServer(t, store)
	body, err := json.Marshal(map[string]any{
		"currency": account.Currency,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewReader(body))
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, account.Owner, time.Minute)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusCreated, recorder.Code)
	requireAccountResponse(t, recorder.Body, account)
	require.NotContains(t, recorder.Body.String(), `"owner"`)
	require.NotContains(t, recorder.Body.String(), fmt.Sprintf(`"id":%d`, account.ID))
}

func TestCreateAccountRejectsClientSuppliedBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	body, err := json.Marshal(map[string]any{
		"currency": util.RandomCurrency(),
		"balance":  999_999,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewReader(body))
	addAuthorization(
		t,
		request,
		server.tokenMaker,
		authorizationTypeBearer,
		util.RandomOwner(),
		time.Minute,
	)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestCreateAccountRejectsClientSuppliedAccountNumber(t *testing.T) {
	store := mockdb.NewMockStore(gomock.NewController(t))
	server := newTestServer(t, store)
	body := bytes.NewBufferString(`{"currency":"USD","account_number":"4839201756"}`)
	request := httptest.NewRequest(http.MethodPost, "/accounts", body)
	addAuthorization(
		t,
		request,
		server.tokenMaker,
		authorizationTypeBearer,
		util.RandomOwner(),
		time.Minute,
	)
	recorder := httptest.NewRecorder()

	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestResolveRecipientAPI(t *testing.T) {
	accountNumber := "4839201756"
	resolved := db.ResolveReceivableAccountRow{
		AccountNumber: accountNumber,
		AccountName:   "Favour Útútú",
		Currency:      "USD",
		Status:        db.FinancialAccountStatusFrozen,
	}

	testCases := []struct {
		name          string
		requestNumber string
		result        db.ResolveReceivableAccountRow
		storeError    error
		statusCode    int
	}{
		{
			name:          "FrozenRecipientCanReceive",
			requestNumber: accountNumber,
			result:        resolved,
			statusCode:    http.StatusOK,
		},
		{
			name:          "MissingRecipient",
			requestNumber: accountNumber,
			storeError:    pgx.ErrNoRows,
			statusCode:    http.StatusNotFound,
		},
		{
			name:          "MalformedRecipient",
			requestNumber: "not-an-account",
			statusCode:    http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := mockdb.NewMockStore(gomock.NewController(t))
			if accountNumberPattern.MatchString(tc.requestNumber) {
				store.EXPECT().
					ResolveReceivableAccount(gomock.Any(), tc.requestNumber).
					Return(tc.result, tc.storeError)
			}
			server := newTestServer(t, store)
			body, err := json.Marshal(resolveRecipientRequest{
				AccountNumber: tc.requestNumber,
			})
			require.NoError(t, err)
			request := httptest.NewRequest(
				http.MethodPost,
				"/accounts/resolve",
				bytes.NewReader(body),
			)
			addAuthorization(
				t,
				request,
				server.tokenMaker,
				authorizationTypeBearer,
				util.RandomOwner(),
				time.Minute,
			)
			recorder := httptest.NewRecorder()

			server.router.ServeHTTP(recorder, request)

			require.Equal(t, tc.statusCode, recorder.Code)
			if tc.statusCode == http.StatusOK {
				var response recipientResolutionResponse
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
				require.Equal(t, "******1756", response.AccountNumber)
				require.Equal(t, "F***** Ú****", response.AccountName)
				require.Equal(t, "USD", response.Currency)
				require.True(t, response.CanReceive)
				require.NotContains(t, recorder.Body.String(), accountNumber)
				require.NotContains(t, recorder.Body.String(), `"balance"`)
				require.NotContains(t, recorder.Body.String(), `"email"`)
				require.NotContains(t, recorder.Body.String(), `"username"`)
				require.NotContains(t, recorder.Body.String(), `"id"`)
			} else {
				var response map[string]any
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
				require.Equal(t, "recipient_not_found", response["code"])
			}
		})
	}
}

func TestRecipientMasking(t *testing.T) {
	require.Equal(t, "******1756", maskAccountNumber("4839201756"))
	require.Equal(t, "F*****", maskName("Favour"))
	require.Equal(t, "F***** U****", maskName("  Favour\tUtutu  "))
	require.Equal(t, "李* É*****", maskName("李雷 Élodie"))
	require.Equal(t, "A", maskName("A"))
}

func TestResolveRecipientRateLimit(t *testing.T) {
	limiter := &stubRateLimiter{retryAfter: time.Minute}
	server := newOperationalTestServer(t, limiter, nil, nil)
	username := util.RandomOwner()
	request := httptest.NewRequest(
		http.MethodPost,
		"/accounts/resolve",
		strings.NewReader(`{"account_number":"4839201756"}`),
	)
	addAuthorization(
		t,
		request,
		server.tokenMaker,
		authorizationTypeBearer,
		username,
		time.Minute,
	)
	recorder := httptest.NewRecorder()

	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.Equal(t, "recipient_resolve:"+username, limiter.key)
	require.Equal(t, int64(20), limiter.limit)
	require.Equal(t, time.Minute, limiter.window)
}

func TestResolveRecipientRequiresVerifiedUser(t *testing.T) {
	username := util.RandomOwner()
	store := mockdb.NewMockStore(gomock.NewController(t))
	store.EXPECT().
		GetUser(gomock.Any(), username).
		Return(db.User{
			Username:      username,
			AccountStatus: db.AccountStatusPending,
			RegistrationExpiresAt: pgtype.Timestamptz{
				Time:  time.Now().Add(time.Hour),
				Valid: true,
			},
		}, nil)
	config := util.Config{
		SecretKey:                util.RandomString(32),
		AccessTokenDuration:      time.Minute,
		RefreshTokenDuration:     time.Minute,
		DeviceCookieName:         "monierave_device",
		EnforceEmailVerification: true,
	}
	server, err := NewServer(config, testSessionStore{Store: store})
	require.NoError(t, err)
	request := httptest.NewRequest(
		http.MethodPost,
		"/accounts/resolve",
		strings.NewReader(`{"account_number":"4839201756"}`),
	)
	addAuthorization(
		t,
		request,
		server.tokenMaker,
		authorizationTypeBearer,
		username,
		time.Minute,
	)
	recorder := httptest.NewRecorder()

	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(
		t,
		recorder.Body.String(),
		"email_verification_is_required_for_financial_features",
	)
}

func TestGetAccountAPIUsesOwnedPublicID(t *testing.T) {
	account := randomAccount()
	arg := db.GetOwnedAccountByPublicIDParams{
		PublicID: account.PublicID,
		Owner:    account.Owner,
	}

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetOwnedAccountByPublicID(gomock.Any(), arg).Return(account, nil)

	server := newTestServer(t, store)
	request := httptest.NewRequest(
		http.MethodGet,
		"/accounts/"+publicUUID(account.PublicID),
		nil,
	)
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, account.Owner, time.Minute)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	requireAccountResponse(t, recorder.Body, account)
}

func TestGetAccountAPIHidesForeignAccounts(t *testing.T) {
	account := randomAccount()
	otherUser := util.RandomOwner()

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().
		GetOwnedAccountByPublicID(gomock.Any(), db.GetOwnedAccountByPublicIDParams{
			PublicID: account.PublicID,
			Owner:    otherUser,
		}).
		Return(db.Account{}, pgx.ErrNoRows)

	server := newTestServer(t, store)
	request := httptest.NewRequest(
		http.MethodGet,
		"/accounts/"+publicUUID(account.PublicID),
		nil,
	)
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, otherUser, time.Minute)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestListAccountsAPIReturnsSafeDTOs(t *testing.T) {
	owner := util.RandomOwner()
	accounts := []db.Account{randomAccount(), randomAccount()}
	for i := range accounts {
		accounts[i].Owner = owner
	}

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().
		ListAccount(gomock.Any(), db.ListAccountParams{
			Owner:  owner,
			Limit:  5,
			Offset: 0,
		}).
		Return(accounts, nil)

	server := newTestServer(t, store)
	request := httptest.NewRequest(http.MethodGet, "/accounts?page_id=1&page_size=5", nil)
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, owner, time.Minute)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response []accountResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response, 2)
	require.Equal(t, publicUUID(accounts[0].PublicID), response[0].ID)
	require.NotContains(t, recorder.Body.String(), `"owner"`)
}

func TestCloseAccountAPI(t *testing.T) {
	account := randomAccount()
	account.Balance = 0
	closed := account
	closed.Status = db.FinancialAccountStatusClosed
	closed.ClosedAt = timestamp(time.Now())

	testCases := []struct {
		name       string
		storeError error
		statusCode int
	}{
		{name: "OK", statusCode: http.StatusOK},
		{name: "NotFound", storeError: db.ErrAccountNotFound, statusCode: http.StatusNotFound},
		{name: "ForeignOwner", storeError: db.ErrAccountNotOwned, statusCode: http.StatusNotFound},
		{name: "NonZeroBalance", storeError: db.ErrAccountBalanceNotZero, statusCode: http.StatusConflict},
		{name: "AlreadyClosed", storeError: db.ErrAccountClosed, statusCode: http.StatusConflict},
		{name: "InternalError", storeError: sql.ErrConnDone, statusCode: http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			store.EXPECT().
				CloseAccountTx(gomock.Any(), db.CloseAccountTxParams{
					PublicID: account.PublicID,
					Username: account.Owner,
				}).
				Return(closed, tc.storeError)

			server := newTestServer(t, store)
			request := httptest.NewRequest(
				http.MethodPost,
				"/accounts/"+publicUUID(account.PublicID)+"/close",
				nil,
			)
			addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, account.Owner, time.Minute)
			recorder := httptest.NewRecorder()
			server.router.ServeHTTP(recorder, request)

			require.Equal(t, tc.statusCode, recorder.Code)
			if tc.storeError == nil {
				requireAccountResponse(t, recorder.Body, closed)
			}
		})
	}
}

func TestAccountAPIRemovesBalanceUpdateAndHardDelete(t *testing.T) {
	account := randomAccount()
	server := newTestServer(t, mockdb.NewMockStore(gomock.NewController(t)))
	url := "/accounts/" + publicUUID(account.PublicID)

	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		request := httptest.NewRequest(method, url, strings.NewReader(`{"balance":1000}`))
		addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, account.Owner, time.Minute)
		recorder := httptest.NewRecorder()
		server.router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusNotFound, recorder.Code)
	}
}

func TestAccountAPIRejectsInvalidPublicID(t *testing.T) {
	server := newTestServer(t, mockdb.NewMockStore(gomock.NewController(t)))
	request := httptest.NewRequest(http.MethodGet, "/accounts/not-a-uuid", nil)
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, util.RandomOwner(), time.Minute)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func randomAccount() db.Account {
	now := time.Now().UTC().Truncate(time.Microsecond)
	publicID := uuid.New()
	return db.Account{
		ID:            int64(rand.IntN(1000) + 1),
		PublicID:      pgtype.UUID{Bytes: publicID, Valid: true},
		AccountNumber: fmt.Sprintf("%010d", rand.Int64N(10_000_000_000)),
		Owner:         util.RandomOwner(),
		Balance:       util.RandomMoney(),
		Currency:      util.RandomCurrency(),
		Status:        db.FinancialAccountStatusActive,
		CreatedAt:     timestamp(now),
		UpdatedAt:     timestamp(now),
	}
}

func timestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func publicUUID(value pgtype.UUID) string {
	return uuid.UUID(value.Bytes).String()
}

func requireAccountResponse(t *testing.T, body *bytes.Buffer, account db.Account) {
	t.Helper()
	var response accountResponse
	require.NoError(t, json.Unmarshal(body.Bytes(), &response))
	require.Equal(t, publicUUID(account.PublicID), response.ID)
	require.Equal(t, account.AccountNumber, response.AccountNumber)
	require.Equal(t, account.Balance, response.Balance)
	require.Equal(t, account.Currency, response.Currency)
	require.Equal(t, account.Status, response.Status)
}
