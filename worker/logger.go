package worker

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type AsynqLogger struct {
	logger zerolog.Logger
}

var _ asynq.Logger = (*AsynqLogger)(nil)

type RedisLogger struct {
	logger zerolog.Logger
}

var configureRedisLogger sync.Once

func NewAsynqLogger() *AsynqLogger {
	logger := log.With().Str("component", "asynq").Logger()
	return &AsynqLogger{logger: logger}
}

func (logger *AsynqLogger) print(level zerolog.Level, args ...interface{}) {
	logger.logger.WithLevel(level).Msg(fmt.Sprint(args...))
}

func (logger *AsynqLogger) Debug(args ...interface{}) {
	logger.print(zerolog.DebugLevel, args...)
}

func (logger *AsynqLogger) Info(args ...interface{}) {
	logger.print(zerolog.InfoLevel, args...)
}

func (logger *AsynqLogger) Warn(args ...interface{}) {
	logger.print(zerolog.WarnLevel, args...)
}

func (logger *AsynqLogger) Error(args ...interface{}) {
	logger.print(zerolog.ErrorLevel, args...)
}

func (logger *AsynqLogger) Fatal(args ...interface{}) {
	logger.logger.Fatal().Msg(fmt.Sprint(args...))
}

func newRedisLogger() *RedisLogger {
	logger := log.With().Str("component", "redis").Logger()
	return &RedisLogger{logger: logger}
}

func (logger *RedisLogger) Printf(_ context.Context, format string, args ...interface{}) {
	message := strings.TrimSpace(fmt.Sprintf(format, args...))
	logger.logger.Warn().Msg(message)
}

func configureRedisLogging() {
	configureRedisLogger.Do(func() {
		redis.SetLogger(newRedisLogger())
	})
}
