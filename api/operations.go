package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/faelic/monierave/observability"
	"github.com/faelic/monierave/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const (
	requestIDHeader     = "X-Request-ID"
	correlationIDHeader = "X-Correlation-ID"
	requestIDContextKey = "request_id"
	correlationIDKey    = "correlation_id"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

type ReadinessCheck func(context.Context) error

type RateLimiter interface {
	Allow(
		ctx context.Context,
		key string,
		limit int64,
		window time.Duration,
	) (bool, time.Duration, error)
}

type redisRateLimiter struct {
	client redis.UniversalClient
	script *redis.Script
}

func NewRedisRateLimiter(client redis.UniversalClient) RateLimiter {
	return &redisRateLimiter{
		client: client,
		script: redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
local ttl = redis.call("PTTL", KEYS[1])
if current > tonumber(ARGV[2]) then
  return {0, ttl}
end
return {1, ttl}
`),
	}
}

func (limiter *redisRateLimiter) Allow(
	ctx context.Context,
	key string,
	limit int64,
	window time.Duration,
) (bool, time.Duration, error) {
	result, err := limiter.script.Run(
		ctx,
		limiter.client,
		[]string{"monierave:rate:" + key},
		window.Milliseconds(),
		limit,
	).Int64Slice()
	if err != nil {
		return false, 0, err
	}
	if len(result) != 2 {
		return false, 0, errors.New("unexpected Redis rate-limit response")
	}
	retryAfter := time.Duration(max(int64(0), result[1])) * time.Millisecond
	return result[0] == 1, retryAfter, nil
}

func (server *Server) requestContextMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestID := strings.TrimSpace(ctx.GetHeader(requestIDHeader))
		if !requestIDPattern.MatchString(requestID) {
			requestID = uuid.NewString()
		}
		correlationID, err := uuid.Parse(strings.TrimSpace(
			ctx.GetHeader(correlationIDHeader),
		))
		if err != nil {
			correlationID = uuid.New()
		}

		ctx.Set(requestIDContextKey, requestID)
		ctx.Set(correlationIDKey, correlationID)
		ctx.Header(requestIDHeader, requestID)
		ctx.Header(correlationIDHeader, correlationID.String())

		logger := log.With().
			Str("component", "api").
			Str("request_id", requestID).
			Str("correlation_id", correlationID.String()).
			Logger()
		ctx.Request = ctx.Request.WithContext(logger.WithContext(ctx.Request.Context()))
		ctx.Next()
	}
}

func (server *Server) requestLoggerMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		started := time.Now()
		ctx.Next()
		elapsed := time.Since(started)
		route := ctx.FullPath()
		if route == "" {
			route = "unmatched"
		}
		status := ctx.Writer.Status()
		server.metrics.ObserveRequest(ctx.Request.Method, route, status, elapsed)
		if route == "/transfers" {
			result := "success"
			if status >= 500 {
				result = "server_error"
			} else if status >= 400 {
				result = "rejected"
			}
			server.metrics.RecordTransfer(result)
		}

		event := log.Ctx(ctx.Request.Context()).Info()
		if status >= 500 {
			event = log.Ctx(ctx.Request.Context()).Error()
			server.metrics.RecordDatabaseError()
		} else if status >= 400 {
			event = log.Ctx(ctx.Request.Context()).Warn()
		}
		event.
			Str("method", ctx.Request.Method).
			Str("route", route).
			Int("status", status).
			Int64("duration_ms", elapsed.Milliseconds()).
			Str("client_ip", ctx.ClientIP()).
			Msg("HTTP request completed")
	}
}

func (server *Server) recoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(ctx *gin.Context, recovered any) {
		log.Ctx(ctx.Request.Context()).Error().
			Interface("panic", recovered).
			Msg("recovered HTTP panic")
		ctx.AbortWithStatusJSON(
			http.StatusInternalServerError,
			errorResponse(ctx, ErrInternalServer),
		)
	})
}

func (server *Server) rateLimitMiddleware(
	endpoint string,
	limit int64,
	window time.Duration,
	key func(*gin.Context) string,
) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if server.rateLimiter == nil {
			ctx.Next()
			return
		}
		allowed, retryAfter, err := server.rateLimiter.Allow(
			ctx.Request.Context(),
			endpoint+":"+key(ctx),
			limit,
			window,
		)
		if err != nil {
			log.Ctx(ctx.Request.Context()).Error().
				Err(err).
				Str("endpoint", endpoint).
				Msg("rate limiter unavailable")
			ctx.AbortWithStatusJSON(
				http.StatusServiceUnavailable,
				codedErrorResponse(
					ctx,
					"dependency_unavailable",
					ErrServiceUnavailable,
				),
			)
			return
		}
		if !allowed {
			server.metrics.RecordRateLimit(endpoint)
			seconds := max(1, int(retryAfter.Round(time.Second).Seconds()))
			ctx.Header("Retry-After", strconv.Itoa(seconds))
			ctx.AbortWithStatusJSON(
				http.StatusTooManyRequests,
				codedErrorResponse(ctx, "rate_limited", ErrRateLimited),
			)
			return
		}
		ctx.Next()
	}
}

func clientIPRateLimitKey(ctx *gin.Context) string {
	return ctx.ClientIP()
}

func authenticatedRateLimitKey(ctx *gin.Context) string {
	payload, ok := ctx.Get(authorizationPayloadKey)
	if !ok {
		return ctx.ClientIP()
	}
	return payload.(*token.Payload).Username
}

func (server *Server) live(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "alive"})
}

func (server *Server) ready(ctx *gin.Context) {
	checks := map[string]ReadinessCheck{
		"postgres": server.databaseReady,
		"redis":    server.redisReady,
	}
	statuses := make(gin.H, len(checks))
	ready := true
	for name, check := range checks {
		if check == nil {
			statuses[name] = "not_configured"
			continue
		}
		checkCtx, cancel := context.WithTimeout(ctx.Request.Context(), 2*time.Second)
		err := check(checkCtx)
		cancel()
		if err != nil {
			ready = false
			statuses[name] = "unavailable"
			continue
		}
		statuses[name] = "ready"
	}
	status := http.StatusOK
	state := "ready"
	if !ready {
		status = http.StatusServiceUnavailable
		state = "not_ready"
	}
	ctx.JSON(status, gin.H{
		"status":       state,
		"dependencies": statuses,
	})
}

func (server *Server) metricsHandler() http.Handler {
	return server.metrics.Handler(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		values, err := server.store.GetFinancialOperationalMetrics(ctx)
		if err != nil {
			return fmt.Errorf("read operational metrics: %w", err)
		}
		server.metrics.SetOperationalGauges(
			values.OutboxLagSeconds,
			values.EmailDlqSize,
			values.ReconciliationDrift,
			values.WorkerRetries,
		)
		return nil
	})
}

func requestID(ctx *gin.Context) string {
	value, _ := ctx.Get(requestIDContextKey)
	requestID, _ := value.(string)
	return requestID
}

func correlationID(ctx *gin.Context) uuid.UUID {
	value, _ := ctx.Get(correlationIDKey)
	id, _ := value.(uuid.UUID)
	if id == uuid.Nil {
		return uuid.New()
	}
	return id
}

type ServerOption func(*Server)

func WithOperationalDependencies(
	databaseReady ReadinessCheck,
	redisReady ReadinessCheck,
	limiter RateLimiter,
	metrics *observability.Registry,
) ServerOption {
	return func(server *Server) {
		server.databaseReady = databaseReady
		server.redisReady = redisReady
		server.rateLimiter = limiter
		if metrics != nil {
			server.metrics = metrics
		}
	}
}

func stableErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrInternalServer):
		return "internal_error"
	case errors.Is(err, ErrUnauthorized):
		return "unauthorized"
	case errors.Is(err, ErrForbidden):
		return "forbidden"
	case errors.Is(err, ErrInvalidCredentials):
		return "invalid_credentials"
	case errors.Is(err, ErrRateLimited):
		return "rate_limited"
	case errors.Is(err, ErrServiceUnavailable):
		return "dependency_unavailable"
	}
	value := strings.ToLower(err.Error())
	value = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(value, "_")
	value = strings.Trim(value, "_")
	if value == "" {
		return "request_failed"
	}
	if len(value) > 64 {
		return "request_failed"
	}
	return value
}
