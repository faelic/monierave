package util

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	DBSource             string        `mapstructure:"DB_SOURCE"`
	ServerAddress        string        `mapstructure:"SERVER_ADDRESS"`
	SecretKey            string        `mapstructure:"SECRET_KEY"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	RedisAddress         string        `mapstructure:"REDIS_ADDRESS"`
	MailerProvider       string        `mapstructure:"MAILER_PROVIDER"`
	WorkerConcurrency    int           `mapstructure:"WORKER_CONCURRENCY"`
	RelayBatchSize       int32         `mapstructure:"RELAY_BATCH_SIZE"`
	RelayPollInterval    time.Duration `mapstructure:"RELAY_POLL_INTERVAL"`
	RelayClaimLease      time.Duration `mapstructure:"RELAY_CLAIM_LEASE"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.SetConfigFile(filepath.Join(path, "app.env"))
	viper.AutomaticEnv()

	_ = viper.BindEnv("DB_SOURCE")
	_ = viper.BindEnv("SERVER_ADDRESS")
	_ = viper.BindEnv("SECRET_KEY")
	_ = viper.BindEnv("ACCESS_TOKEN_DURATION")
	_ = viper.BindEnv("REFRESH_TOKEN_DURATION")
	_ = viper.BindEnv("REDIS_ADDRESS")
	_ = viper.BindEnv("MAILER_PROVIDER")
	_ = viper.BindEnv("WORKER_CONCURRENCY")
	_ = viper.BindEnv("RELAY_BATCH_SIZE")
	_ = viper.BindEnv("RELAY_POLL_INTERVAL")
	_ = viper.BindEnv("RELAY_CLAIM_LEASE")

	viper.SetDefault("MAILER_PROVIDER", "log")
	viper.SetDefault("WORKER_CONCURRENCY", 10)
	viper.SetDefault("RELAY_BATCH_SIZE", 50)
	viper.SetDefault("RELAY_POLL_INTERVAL", time.Second)
	viper.SetDefault("RELAY_CLAIM_LEASE", 30*time.Second)

	err = viper.ReadInConfig()
	if err != nil {
		var configFileNotFound viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFound) && !os.IsNotExist(err) {
			return config, err
		}
	}

	err = viper.Unmarshal(&config)
	return
}

func ValidateConfig(config Config) error {
	return ValidateAPIConfig(config)
}

func ValidateAPIConfig(config Config) error {
	if config.DBSource == "" {
		return fmt.Errorf("DB_SOURCE is required")
	}
	if config.ServerAddress == "" {
		return fmt.Errorf("SERVER_ADDRESS is required")
	}
	if config.SecretKey == "" {
		return fmt.Errorf("SECRET_KEY is required")
	}
	if config.AccessTokenDuration <= 0 {
		return fmt.Errorf("ACCESS_TOKEN_DURATION must be greater than 0")
	}
	if config.RefreshTokenDuration <= 0 {
		return fmt.Errorf("REFRESH_TOKEN_DURATION must be greater than 0")
	}
	return nil
}

func ValidateRelayConfig(config Config) error {
	if config.DBSource == "" {
		return fmt.Errorf("DB_SOURCE is required")
	}
	if config.RedisAddress == "" {
		return fmt.Errorf("REDIS_ADDRESS is required")
	}
	if config.RelayBatchSize <= 0 {
		return fmt.Errorf("RELAY_BATCH_SIZE must be greater than 0")
	}
	if config.RelayPollInterval <= 0 {
		return fmt.Errorf("RELAY_POLL_INTERVAL must be greater than 0")
	}
	if config.RelayClaimLease <= 0 {
		return fmt.Errorf("RELAY_CLAIM_LEASE must be greater than 0")
	}
	return nil
}

func ValidateWorkerConfig(config Config) error {
	if config.DBSource == "" {
		return fmt.Errorf("DB_SOURCE is required")
	}
	if config.RedisAddress == "" {
		return fmt.Errorf("REDIS_ADDRESS is required")
	}
	if config.MailerProvider == "" {
		return fmt.Errorf("MAILER_PROVIDER is required")
	}
	if config.WorkerConcurrency <= 0 {
		return fmt.Errorf("WORKER_CONCURRENCY must be greater than 0")
	}
	return nil
}
