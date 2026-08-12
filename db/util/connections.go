package util

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// RedisOptions returns equivalent clients for API concerns and Asynq workers.
func RedisOptions(config Config) (*redis.Options, asynq.RedisClientOpt, error) {
	if strings.TrimSpace(config.RedisURL) == "" {
		address := strings.TrimSpace(config.RedisAddress)
		if address == "" {
			return nil, asynq.RedisClientOpt{}, fmt.Errorf("REDIS_URL or REDIS_ADDRESS is required")
		}
		return &redis.Options{Addr: address}, asynq.RedisClientOpt{Addr: address}, nil
	}

	redisOptions, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		return nil, asynq.RedisClientOpt{}, fmt.Errorf("parse REDIS_URL: %w", err)
	}
	if redisOptions.TLSConfig != nil && redisOptions.TLSConfig.ServerName == "" {
		host, _, splitErr := net.SplitHostPort(redisOptions.Addr)
		if splitErr == nil {
			redisOptions.TLSConfig.ServerName = host
		}
	}

	asynqOptions := asynq.RedisClientOpt{
		Network:   redisOptions.Network,
		Addr:      redisOptions.Addr,
		Username:  redisOptions.Username,
		Password:  redisOptions.Password,
		DB:        redisOptions.DB,
		TLSConfig: redisOptions.TLSConfig,
	}
	return redisOptions, asynqOptions, nil
}

// RedactedRedisTarget describes the configured endpoint without credentials.
func RedactedRedisTarget(config Config) string {
	if config.RedisURL == "" {
		return strings.TrimSpace(config.RedisAddress)
	}
	parsed, err := url.Parse(config.RedisURL)
	if err != nil {
		return "invalid Redis URL"
	}
	database := strings.Trim(parsed.Path, "/")
	if database != "" {
		if _, err := strconv.Atoi(database); err == nil {
			return parsed.Scheme + "://" + parsed.Host + "/" + database
		}
	}
	return parsed.Scheme + "://" + parsed.Host
}
