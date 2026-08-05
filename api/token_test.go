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
			require.Equal(t, currentPayload.ID, uuid.UUID(arg.PresentedRefreshID.Bytes))
			require.NotEqual(t, arg.PresentedRefreshID, arg.NewRefreshID)
			return db.Session{}, nil
		})

	request := httptest.NewRequest(http.MethodPost, "/tokens/renew_access", nil)
	request.AddCookie(&http.Cookie{Name: server.config.RefreshCookieName, Value: current})
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

	store.EXPECT().
		RevokeSessionTx(
			gomock.Any(),
			pgtype.UUID{Bytes: sessionID, Valid: true},
			"user_logout",
			"user",
		).
		Return(nil)

	request := httptest.NewRequest(http.MethodPost, "/sessions/logout", nil)
	request.AddCookie(&http.Cookie{Name: server.config.RefreshCookieName, Value: value})
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 1)
	require.Less(t, cookies[0].MaxAge, 0)
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
	require.Len(t, recorder.Result().Cookies(), 1)
	require.Less(t, recorder.Result().Cookies()[0].MaxAge, 0)
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
		).
		Return(db.ErrSessionRevoked)

	router := gin.New()
	router.GET("/protected", authMiddleware(maker, store), func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set(authorizationHeaderKey, "Bearer "+value)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
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
	require.Len(t, recorder.Result().Cookies(), 1)
	require.Less(t, recorder.Result().Cookies()[0].MaxAge, 0)
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
