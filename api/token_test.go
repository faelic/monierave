package api

import (
	"bytes"
	"context"
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
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRenewAccessTokenRotatesRefreshCookie(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)
	username := util.RandomOwner()
	sessionID := uuid.New()
	current, currentPayload, err := server.tokenMaker.CreateRefreshToken(
		username,
		sessionID,
		time.Hour,
	)
	require.NoError(t, err)
	deviceToken := "test-device-token"

	store.EXPECT().
		GetSession(gomock.Any(), pgtype.UUID{Bytes: sessionID, Valid: true}).
		Return(db.Session{
			ID:        pgtype.UUID{Bytes: sessionID, Valid: true},
			Username:  username,
			ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
		}, nil)
	store.EXPECT().
		RotateRefreshTokenTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.RotateRefreshTokenTxParams) (db.Session, error) {
			require.Equal(t, token.Hash(current), arg.PresentedTokenHash)
			require.Equal(t, token.Hash(deviceToken), arg.PresentedDeviceHash)
			require.Equal(t, currentPayload.ID, uuid.UUID(arg.PresentedRefreshID.Bytes))
			require.NotEqual(t, arg.PresentedRefreshID, arg.NewRefreshID)
			return db.Session{}, nil
		})

	request := httptest.NewRequest(http.MethodPost, "/tokens/renew_access", nil)
	request.AddCookie(&http.Cookie{Name: server.config.RefreshCookieName, Value: current})
	request.AddCookie(&http.Cookie{Name: server.config.DeviceCookieName, Value: deviceToken})
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 1)
	require.Equal(t, server.config.RefreshCookieName, cookies[0].Name)
	require.NotEqual(t, current, cookies[0].Value)
	require.True(t, cookies[0].HttpOnly)
}

func TestLogoutCurrentSessionRevokesSessionAndClearsCookie(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)
	sessionID := uuid.New()
	value, _, err := server.tokenMaker.CreateRefreshToken(
		util.RandomOwner(),
		sessionID,
		time.Hour,
	)
	require.NoError(t, err)
	deviceToken := "test-device-token"

	store.EXPECT().
		RevokeSessionTx(
			gomock.Any(),
			pgtype.UUID{Bytes: sessionID, Valid: true},
			token.Hash(deviceToken),
			"user_logout",
			"user",
		).
		Return(nil)

	request := httptest.NewRequest(http.MethodPost, "/sessions/logout", nil)
	request.AddCookie(&http.Cookie{Name: server.config.RefreshCookieName, Value: value})
	request.AddCookie(&http.Cookie{Name: server.config.DeviceCookieName, Value: deviceToken})
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 2)
	for _, cookie := range cookies {
		require.Less(t, cookie.MaxAge, 0)
	}
}

func TestLogoutAllSessionsRevokesEveryUserSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)
	username := util.RandomOwner()

	store.EXPECT().
		RevokeAllUserSessionsTx(
			gomock.Any(),
			username,
			"user_logout_all",
			"all_sessions_logged_out",
		).
		Return(nil)

	request := httptest.NewRequest(http.MethodPost, "/sessions/logout-all", nil)
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

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.Len(t, recorder.Result().Cookies(), 2)
	for _, cookie := range recorder.Result().Cookies() {
		require.Less(t, cookie.MaxAge, 0)
	}
}

func TestRevokedSessionRejectsValidAccessToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	maker, err := token.NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)
	username := util.RandomOwner()
	sessionID := uuid.New()
	value, _, err := maker.CreateAccessToken(username, sessionID, time.Minute)
	require.NoError(t, err)

	store.EXPECT().
		ValidateSession(
			gomock.Any(),
			pgtype.UUID{Bytes: sessionID, Valid: true},
			username,
			token.Hash(""),
		).
		Return(db.ErrSessionRevoked)

	router := gin.New()
	router.GET("/protected", authMiddleware(maker, store, "monierave_device"), func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set(authorizationHeaderKey, "Bearer "+value)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestAccessTokenRequiresMatchingDeviceCookie(t *testing.T) {
	tests := []struct {
		name         string
		deviceCookie string
		storeError   error
		statusCode   int
	}{
		{
			name:         "matching device",
			deviceCookie: "correct-device",
			statusCode:   http.StatusOK,
		},
		{
			name:       "missing device",
			storeError: db.ErrDeviceMismatch,
			statusCode: http.StatusUnauthorized,
		},
		{
			name:         "wrong device",
			deviceCookie: "wrong-device",
			storeError:   db.ErrDeviceMismatch,
			statusCode:   http.StatusUnauthorized,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			maker, err := token.NewJWTMaker(util.RandomString(32))
			require.NoError(t, err)
			username := util.RandomOwner()
			sessionID := uuid.New()
			value, _, err := maker.CreateAccessToken(username, sessionID, time.Minute)
			require.NoError(t, err)

			store.EXPECT().
				ValidateSession(
					gomock.Any(),
					pgtype.UUID{Bytes: sessionID, Valid: true},
					username,
					token.Hash(test.deviceCookie),
				).
				Return(test.storeError)

			router := gin.New()
			router.GET(
				"/protected",
				authMiddleware(maker, store, "monierave_device"),
				func(ctx *gin.Context) {
					ctx.Status(http.StatusOK)
				},
			)
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set(authorizationHeaderKey, "Bearer "+value)
			if test.deviceCookie != "" {
				request.AddCookie(&http.Cookie{
					Name:  "monierave_device",
					Value: test.deviceCookie,
				})
			}
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			require.Equal(t, test.statusCode, recorder.Code)
		})
	}
}

func TestRenewAccessTokenRejectsMissingDeviceCookie(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)
	refreshToken, _, err := server.tokenMaker.CreateRefreshToken(
		util.RandomOwner(),
		uuid.New(),
		time.Hour,
	)
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodPost, "/tokens/renew_access", nil)
	request.AddCookie(&http.Cookie{
		Name:  server.config.RefreshCookieName,
		Value: refreshToken,
	})
	recorder := httptest.NewRecorder()

	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Len(t, recorder.Result().Cookies(), 2)
	for _, cookie := range recorder.Result().Cookies() {
		require.Less(t, cookie.MaxAge, 0)
	}
}

func TestPasswordChangeRequiresCurrentPasswordAndRevokesSessions(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)
	currentPassword := util.RandomString(10)
	user, err := randomUser(currentPassword)
	require.NoError(t, err)

	store.EXPECT().
		GetUser(gomock.Any(), user.Username).
		Return(user, nil)
	store.EXPECT().
		UpdateUserTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.UpdateUserTxParams) (db.UpdateUserTxResult, error) {
			require.True(t, arg.RevokeSessions)
			require.True(t, arg.HashedPassword.Valid)
			require.NoError(
				t,
				util.CheckPassword("new-secure-password", arg.HashedPassword.String),
			)
			return db.UpdateUserTxResult{User: user}, nil
		})

	body, err := json.Marshal(gin.H{
		"current_password": currentPassword,
		"password":         "new-secure-password",
	})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	addAuthorization(
		t,
		request,
		server.tokenMaker,
		authorizationTypeBearer,
		user.Username,
		time.Minute,
	)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Len(t, recorder.Result().Cookies(), 2)
	for _, cookie := range recorder.Result().Cookies() {
		require.Less(t, cookie.MaxAge, 0)
	}
}

func TestEmailChangeRequiresCurrentPasswordAndRevokesSessions(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)
	currentPassword := util.RandomString(10)
	user, err := randomUser(currentPassword)
	require.NoError(t, err)

	store.EXPECT().GetUser(gomock.Any(), user.Username).Return(user, nil)
	store.EXPECT().
		UpdateUserTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.UpdateUserTxParams) (db.UpdateUserTxResult, error) {
			require.True(t, arg.RevokeSessions)
			require.Equal(t, "email_changed", arg.SessionRevocationReason)
			require.Equal(t, "sessions_revoked_email_change", arg.SessionAuditEvent)
			require.Equal(t, "new-address@example.com", arg.Email.String)
			updated := user
			updated.Email = arg.Email.String
			return db.UpdateUserTxResult{User: updated, EmailChanged: true}, nil
		})

	body, err := json.Marshal(gin.H{
		"current_password": currentPassword,
		"email":            "new-address@example.com",
	})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
	recorder := httptest.NewRecorder()

	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Len(t, recorder.Result().Cookies(), 2)
	for _, cookie := range recorder.Result().Cookies() {
		require.Less(t, cookie.MaxAge, 0)
	}
}

func TestEmailChangeRejectsMissingCurrentPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)
	user, err := randomUser(util.RandomString(10))
	require.NoError(t, err)
	store.EXPECT().GetUser(gomock.Any(), user.Username).Return(user, nil)

	body := bytes.NewBufferString(`{"email":"new-address@example.com"}`)
	request := httptest.NewRequest(http.MethodPatch, "/users/me", body)
	request.Header.Set("Content-Type", "application/json")
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
	recorder := httptest.NewRecorder()

	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"code":"current_password_required"`)
}

func TestCredentialedCORSAllowsOnlyConfiguredOrigin(t *testing.T) {
	server := newTestServer(t, nil)

	allowed := httptest.NewRequest(http.MethodOptions, "/tokens/renew_access", nil)
	allowed.Header.Set("Origin", "http://localhost:3000")
	allowedRecorder := httptest.NewRecorder()
	server.router.ServeHTTP(allowedRecorder, allowed)
	require.Equal(t, http.StatusNoContent, allowedRecorder.Code)
	require.Equal(
		t,
		"http://localhost:3000",
		allowedRecorder.Header().Get("Access-Control-Allow-Origin"),
	)
	require.Equal(
		t,
		"true",
		allowedRecorder.Header().Get("Access-Control-Allow-Credentials"),
	)

	blocked := httptest.NewRequest(http.MethodOptions, "/tokens/renew_access", nil)
	blocked.Header.Set("Origin", "https://attacker.example")
	blockedRecorder := httptest.NewRecorder()
	server.router.ServeHTTP(blockedRecorder, blocked)
	require.Equal(t, http.StatusForbidden, blockedRecorder.Code)
	require.Empty(t, blockedRecorder.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSDoesNotBlockOrdinaryCrossOriginGET(t *testing.T) {
	server := newTestServer(t, nil)

	request := httptest.NewRequest(http.MethodGet, "/livez", nil)
	request.Header.Set("Origin", "https://mail.google.com")
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Empty(t, recorder.Header().Get("Access-Control-Allow-Origin"))
}

func TestTrustedBrowserOriginProtectsPublicMutations(t *testing.T) {
	server := newTestServer(t, nil)

	for _, origin := range []string{"https://attacker.example", "null"} {
		request := httptest.NewRequest(http.MethodPost, "/users/login", nil)
		request.Header.Set("Origin", origin)
		recorder := httptest.NewRecorder()
		server.router.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusForbidden, recorder.Code)
		require.Empty(t, recorder.Header().Get("Access-Control-Allow-Origin"))
	}

	request := httptest.NewRequest(http.MethodPost, "/users/login", nil)
	request.Host = "public.ngrok-free.dev"
	request.Header.Set("Origin", "https://public.ngrok-free.dev")
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.NotEqual(t, http.StatusForbidden, recorder.Code)
}
