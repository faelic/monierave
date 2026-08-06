package observability

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPrometheusMetricsHandler(t *testing.T) {
	registry := NewRegistry()
	registry.ObserveRequest(http.MethodPost, "/transfers", http.StatusCreated, 25*time.Millisecond)
	registry.RecordTransfer("success")
	registry.RecordRateLimit("login")
	registry.RecordDatabaseError()
	registry.RecordWorkerRetry()
	registry.SetOperationalGauges(12.5, 3, 2, 7)

	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	recorder := httptest.NewRecorder()
	registry.Handler(nil).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "text/plain")
	body := recorder.Body.String()
	require.Contains(t, body, `monierave_http_requests_total{method="POST",route="/transfers",status="201"} 1`)
	require.Contains(t, body, `monierave_transfer_results_total{result="success"} 1`)
	require.Contains(t, body, `monierave_rate_limited_requests_total{endpoint="login"} 1`)
	require.Contains(t, body, "monierave_database_errors_total 1")
	require.Contains(t, body, "monierave_worker_retries_total 7")
	require.Contains(t, body, "monierave_outbox_lag_seconds 12.5")
	require.Contains(t, body, "monierave_email_dlq_size 3")
	require.Contains(t, body, "monierave_reconciliation_drift_total 2")
}

func TestMetricsRefreshFailureIsCounted(t *testing.T) {
	registry := NewRegistry()
	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	recorder := httptest.NewRecorder()

	registry.Handler(func() error {
		return errors.New("database unavailable")
	}).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "monierave_database_errors_total 1")
}
