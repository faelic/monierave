package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetFinancialOperationalMetrics(t *testing.T) {
	metrics, err := testQueries.GetFinancialOperationalMetrics(context.Background())

	require.NoError(t, err)
	require.GreaterOrEqual(t, metrics.OutboxLagSeconds, float64(0))
	require.GreaterOrEqual(t, metrics.EmailDlqSize, int64(0))
	require.GreaterOrEqual(t, metrics.WorkerRetries, int64(0))
	require.GreaterOrEqual(t, metrics.ReconciliationDrift, int64(0))
}
