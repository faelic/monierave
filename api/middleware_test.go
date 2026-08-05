package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/faelic/monierave/db/mock"
	db "github.com/faelic/monierave/db/sqlc"
	"github.com/faelic/monierave/db/util"
	"github.com/faelic/monierave/token"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAuthMiddleware(t *testing.T) {
	user := util.RandomOwner()

	testCases := []struct {
		name          string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "NoAuthorizationHeader",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "InvalidAuthorizationHeader",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				request.Header.Set(authorizationHeaderKey, "invalid-header")
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "UnsupportedAuthorizationType",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, "basic", user, time.Minute)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "InvalidAuthorizationToken",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				request.Header.Set(authorizationHeaderKey, "bearer invalid-token")
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "ExpiredAuthorizationToken",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user, -time.Minute)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "OK",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user, time.Minute)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				require.Equal(t, user, recorder.Body.String())
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			server := newTestServer(t, nil)

			authPath := "/auth"
			server.router.GET(authPath, authMiddleware(server.tokenMaker, server.store), func(ctx *gin.Context) {
				payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
				ctx.String(http.StatusOK, payload.Username)
			})

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, authPath, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestVerifiedAccountMiddlewareDescribesRestrictedFeatures(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)
	username := util.RandomOwner()

	store.EXPECT().
		GetUser(gomock.Any(), username).
		Return(db.User{
			Username:      username,
			AccountStatus: db.AccountStatusPending,
		}, nil)

	server.router.GET(
		"/verified-test",
		authMiddleware(server.tokenMaker, server.store),
		verifiedAccountMiddleware(store),
		func(ctx *gin.Context) {
			ctx.Status(http.StatusOK)
		},
	)
	request, err := http.NewRequest(http.MethodGet, "/verified-test", nil)
	require.NoError(t, err)
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

	var response struct {
		Error              string   `json:"error"`
		AllowedFeatures    []string `json:"allowed_features"`
		RestrictedFeatures []string `json:"restricted_features"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, ErrEmailVerificationRequired.Error(), response.Error)
	require.Contains(t, response.AllowedFeatures, "change profile details or email address")
	require.Contains(t, response.RestrictedFeatures, "send money or create transfers")
}
