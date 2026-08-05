package util

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	DBSource                  string        `mapstructure:"DB_SOURCE"`
	ServerAddress             string        `mapstructure:"SERVER_ADDRESS"`
	SecretKey                 string        `mapstructure:"SECRET_KEY"`
	AccessTokenDuration       time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration      time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	RedisAddress              string        `mapstructure:"REDIS_ADDRESS"`
	MailerProvider            string        `mapstructure:"MAILER_PROVIDER"`
	WorkerConcurrency         int           `mapstructure:"WORKER_CONCURRENCY"`
	RelayBatchSize            int32         `mapstructure:"RELAY_BATCH_SIZE"`
	RelayPollInterval         time.Duration `mapstructure:"RELAY_POLL_INTERVAL"`
	RelayClaimLease           time.Duration `mapstructure:"RELAY_CLAIM_LEASE"`
	ResendAPIKey              string        `mapstructure:"RESEND_API_KEY"`
	EmailFrom                 string        `mapstructure:"EMAIL_FROM"`
	ResendWebhookSecret       string        `mapstructure:"RESEND_WEBHOOK_SECRET"`
	PublicAPIURL              string        `mapstructure:"PUBLIC_API_URL"`
	EmailVerificationDuration time.Duration `mapstructure:"EMAIL_VERIFICATION_DURATION"`
	EnforceEmailVerification  bool          `mapstructure:"ENFORCE_EMAIL_VERIFICATION"`
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
	_ = viper.BindEnv("RESEND_API_KEY")
	_ = viper.BindEnv("EMAIL_FROM")
	_ = viper.BindEnv("RESEND_WEBHOOK_SECRET")
	_ = viper.BindEnv("PUBLIC_API_URL")
	_ = viper.BindEnv("EMAIL_VERIFICATION_DURATION")
	_ = viper.BindEnv("ENFORCE_EMAIL_VERIFICATION")

	viper.SetDefault("MAILER_PROVIDER", "log")
	viper.SetDefault("WORKER_CONCURRENCY", 10)
	viper.SetDefault("RELAY_BATCH_SIZE", 50)
	viper.SetDefault("RELAY_POLL_INTERVAL", time.Second)
	viper.SetDefault("RELAY_CLAIM_LEASE", 30*time.Second)
	viper.SetDefault("PUBLIC_API_URL", "http://localhost:8080")
	viper.SetDefault("EMAIL_VERIFICATION_DURATION", 24*time.Hour)
	viper.SetDefault("ENFORCE_EMAIL_VERIFICATION", true)

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
	switch config.MailerProvider {
	case "log":
	case "resend":
		if config.ResendAPIKey == "" {
			return fmt.Errorf("RESEND_API_KEY is required when MAILER_PROVIDER=resend")
		}
		if config.EmailFrom == "" {
			return fmt.Errorf("EMAIL_FROM is required when MAILER_PROVIDER=resend")
		}
	default:
		return fmt.Errorf("unsupported MAILER_PROVIDER %q", config.MailerProvider)
	}
	if config.WorkerConcurrency <= 0 {
		return fmt.Errorf("WORKER_CONCURRENCY must be greater than 0")
	}
	if config.PublicAPIURL == "" {
		return fmt.Errorf("PUBLIC_API_URL is required")
	}
	publicURL, err := url.Parse(config.PublicAPIURL)
	if err != nil || publicURL.Host == "" ||
		(publicURL.Scheme != "http" && publicURL.Scheme != "https") {
		return fmt.Errorf("PUBLIC_API_URL must be an absolute HTTP or HTTPS URL")
	}
	if config.EmailVerificationDuration <= 0 {
		return fmt.Errorf("EMAIL_VERIFICATION_DURATION must be greater than 0")
	}
	return nil
}
