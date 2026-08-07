package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/faelic/monierave/db/mock"
	"github.com/faelic/monierave/db/util"
	"github.com/faelic/monierave/observability"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type stubRateLimiter struct {
	allowed    bool
	retryAfter time.Duration
	err        error
	key        string
	limit      int64
	window     time.Duration
}

func (limiter *stubRateLimiter) Allow(
	_ context.Context,
	key string,
	limit int64,
	window time.Duration,
) (bool, time.Duration, error) {
	limiter.key = key
	limiter.limit = limit
	limiter.window = window
	return limiter.allowed, limiter.retryAfter, limiter.err
}

func (limiter *stubRateLimiter) Reset(context.Context, string) error {
	return nil
}

func newOperationalTestServer(
	t *testing.T,
	limiter RateLimiter,
	databaseReady ReadinessCheck,
	redisReady ReadinessCheck,
) *Server {
	t.Helper()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	config := util.Config{
		SecretKey:             util.RandomString(32),
		AccessTokenDuration:   time.Minute,
		RefreshTokenDuration:  time.Minute,
		RefreshCookieName:     "monierave_refresh",
		DeviceCookieName:      "monierave_device",
		RefreshCookieSameSite: "lax",
		AllowedOrigins:        "http://localhost:3000",
	}
	server, err := NewServer(
		config,
		testSessionStore{Store: store},
		WithOperationalDependencies(
			databaseReady,
			redisReady,
			limiter,
			observability.NewRegistry(),
		),
	)
	require.NoError(t, err)
	return server
}

func TestRequestContextHeadersAndErrorBody(t *testing.T) {
	server := newOperationalTestServer(t, nil, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/accounts/not-a-uuid", nil)
	request.Header.Set(requestIDHeader, "client-request-123")
	request.Header.Set(correlationIDHeader, "not-a-uuid")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Equal(t, "client-request-123", recorder.Header().Get(requestIDHeader))
	require.NotEqual(t, "not-a-uuid", recorder.Header().Get(correlationIDHeader))
	_, err := uuid.Parse(recorder.Header().Get(correlationIDHeader))
	require.NoError(t, err)
	var response map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "client-request-123", response["request_id"])
	require.Equal(t, "unauthorized", response["code"])
	require.NotEmpty(t, response["message"])
}

func TestInvalidRequestIDIsReplaced(t *testing.T) {
	server := newOperationalTestServer(t, nil, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/livez", nil)
	request.Header.Set(requestIDHeader, "invalid request id")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotEqual(t, "invalid request id", recorder.Header().Get(requestIDHeader))
	_, err := uuid.Parse(recorder.Header().Get(requestIDHeader))
	require.NoError(t, err)
}

func TestHealthEndpoints(t *testing.T) {
	t.Run("live regardless of dependencies", func(t *testing.T) {
		server := newOperationalTestServer(
			t,
			nil,
			func(context.Context) error { return errors.New("postgres unavailable") },
			func(context.Context) error { return errors.New("redis unavailable") },
		)
		request := httptest.NewRequest(http.MethodGet, "/livez", nil)
		recorder := httptest.NewRecorder()

		server.Handler().ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)
		require.JSONEq(t, `{"status":"alive"}`, recorder.Body.String())
	})

	t.Run("ready", func(t *testing.T) {
		server := newOperationalTestServer(
			t,
			nil,
			func(context.Context) error { return nil },
			func(context.Context) error { return nil },
		)
		request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		recorder := httptest.NewRecorder()

		server.Handler().ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)
		require.Contains(t, recorder.Body.String(), `"status":"ready"`)
		require.Contains(t, recorder.Body.String(), `"postgres":"ready"`)
		require.Contains(t, recorder.Body.String(), `"redis":"ready"`)
	})

	t.Run("not ready", func(t *testing.T) {
		server := newOperationalTestServer(
			t,
			nil,
			func(context.Context) error { return errors.New("postgres unavailable") },
			func(context.Context) error { return nil },
		)
		request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		recorder := httptest.NewRecorder()

		server.Handler().ServeHTTP(recorder, request)

		require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
		require.Contains(t, recorder.Body.String(), `"status":"not_ready"`)
		require.Contains(t, recorder.Body.String(), `"postgres":"unavailable"`)
	})
}

func TestLoginRateLimit(t *testing.T) {
	tests := []struct {
		name       string
		limiter    *stubRateLimiter
		statusCode int
		errorCode  string
	}{
		{
			name:       "blocked",
			limiter:    &stubRateLimiter{retryAfter: 45 * time.Second},
			statusCode: http.StatusTooManyRequests,
			errorCode:  "rate_limited",
		},
		{
			name:       "dependency unavailable",
			limiter:    &stubRateLimiter{err: errors.New("redis unavailable")},
			statusCode: http.StatusServiceUnavailable,
			errorCode:  "dependency_unavailable",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := newOperationalTestServer(t, test.limiter, nil, nil)
			request := httptest.NewRequest(http.MethodPost, "/users/login", nil)
			request.RemoteAddr = "192.0.2.10:1234"
			recorder := httptest.NewRecorder()

			server.Handler().ServeHTTP(recorder, request)

			require.Equal(t, test.statusCode, recorder.Code)
			require.Equal(t, "login:192.0.2.10", test.limiter.key)
			require.Equal(t, int64(5), test.limiter.limit)
			require.Equal(t, time.Minute, test.limiter.window)
			var response map[string]any
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
			require.Equal(t, test.errorCode, response["code"])
			require.NotEmpty(t, response["request_id"])
			if test.statusCode == http.StatusTooManyRequests {
				require.Equal(t, "45", recorder.Header().Get("Retry-After"))
			}
		})
	}
}

func TestSignupRateLimit(t *testing.T) {
	limiter := &stubRateLimiter{retryAfter: time.Minute}
	server := newOperationalTestServer(t, limiter, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/users", nil)
	request.RemoteAddr = "192.0.2.11:1234"
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.Equal(t, "signup:192.0.2.11", limiter.key)
	require.Equal(t, int64(5), limiter.limit)
	require.Equal(t, time.Hour, limiter.window)
}
