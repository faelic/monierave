package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedisOptionsFromAddress(t *testing.T) {
	redisOptions, asynqOptions, err := RedisOptions(Config{
		RedisAddress: "redis:6379",
	})

	require.NoError(t, err)
	require.Equal(t, "redis:6379", redisOptions.Addr)
	require.Equal(t, "redis:6379", asynqOptions.Addr)
	require.Nil(t, redisOptions.TLSConfig)
	require.Nil(t, asynqOptions.TLSConfig)
}

func TestRedisOptionsFromAuthenticatedTLSURL(t *testing.T) {
	config := Config{
		RedisAddress: "ignored:6379",
		RedisURL:     "rediss://default:secret@cache.example.com:6380/2",
	}

	redisOptions, asynqOptions, err := RedisOptions(config)

	require.NoError(t, err)
	require.Equal(t, "cache.example.com:6380", redisOptions.Addr)
	require.Equal(t, "default", redisOptions.Username)
	require.Equal(t, "secret", redisOptions.Password)
	require.Equal(t, 2, redisOptions.DB)
	require.NotNil(t, redisOptions.TLSConfig)
	require.Equal(t, "cache.example.com", redisOptions.TLSConfig.ServerName)
	require.Equal(t, redisOptions.Addr, asynqOptions.Addr)
	require.Equal(t, redisOptions.Username, asynqOptions.Username)
	require.Equal(t, redisOptions.Password, asynqOptions.Password)
	require.Equal(t, redisOptions.DB, asynqOptions.DB)
	require.NotNil(t, asynqOptions.TLSConfig)
	require.Equal(t, "cache.example.com", asynqOptions.TLSConfig.ServerName)
	require.Equal(t, "rediss://cache.example.com:6380/2", RedactedRedisTarget(config))
}

func TestRedisOptionsRejectsInvalidURL(t *testing.T) {
	_, _, err := RedisOptions(Config{RedisURL: "https://cache.example.com"})

	require.ErrorContains(t, err, "parse REDIS_URL")
}

func TestValidateDatabasePool(t *testing.T) {
	valid := withDatabasePool(Config{})
	require.NoError(t, validateDatabasePool(valid))

	invalid := valid
	invalid.DBMinConns = invalid.DBMaxConns + 1
	require.ErrorContains(t, validateDatabasePool(invalid), "must not exceed")
}
