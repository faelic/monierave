package worker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNormalizeProcessorConfigUsesQuotaFriendlyDefaults(t *testing.T) {
	config := normalizeProcessorConfig(ProcessorConfig{})

	require.Equal(t, 1, config.Concurrency)
	require.Equal(t, 15*time.Second, config.TaskCheckInterval)
	require.Equal(t, 2*time.Minute, config.HealthCheckInterval)
	require.Equal(t, time.Minute, config.DelayedTaskCheckInterval)
	require.Equal(t, time.Hour, config.JanitorInterval)
}

func TestNormalizeProcessorConfigPreservesExplicitValues(t *testing.T) {
	expected := ProcessorConfig{
		Concurrency:              3,
		TaskCheckInterval:        20 * time.Second,
		HealthCheckInterval:      3 * time.Minute,
		DelayedTaskCheckInterval: 90 * time.Second,
		JanitorInterval:          2 * time.Hour,
	}

	require.Equal(t, expected, normalizeProcessorConfig(expected))
}
