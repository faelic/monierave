package worker

import (
	"bytes"
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestAsynqLoggerWritesStructuredLevels(t *testing.T) {
	var output bytes.Buffer
	logger := &AsynqLogger{logger: zerolog.New(&output)}

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	logged := output.String()
	require.Contains(t, logged, `"level":"debug"`)
	require.Contains(t, logged, `"level":"info"`)
	require.Contains(t, logged, `"level":"warn"`)
	require.Contains(t, logged, `"level":"error"`)
}

func TestRedisLoggerWritesStructuredWarning(t *testing.T) {
	var output bytes.Buffer
	logger := &RedisLogger{logger: zerolog.New(&output)}

	logger.Printf(context.Background(), "connection pool: %s\n", "failed")

	logged := output.String()
	require.Contains(t, logged, `"level":"warn"`)
	require.Contains(t, logged, `"message":"connection pool: failed"`)
	require.NotContains(t, logged, `\n"`)
}
