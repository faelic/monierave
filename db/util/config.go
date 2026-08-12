package util

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Environment               string        `mapstructure:"APP_ENV"`
	DBSource                  string        `mapstructure:"DB_SOURCE"`
	DBTestSource              string        `mapstructure:"DB_TEST_SOURCE"`
	ServerAddress             string        `mapstructure:"SERVER_ADDRESS"`
	SecretKey                 string        `mapstructure:"SECRET_KEY"`
	AccessTokenDuration       time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration      time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	RefreshCookieName         string        `mapstructure:"REFRESH_COOKIE_NAME"`
	DeviceCookieName          string        `mapstructure:"DEVICE_COOKIE_NAME"`
	RefreshCookieDomain       string        `mapstructure:"REFRESH_COOKIE_DOMAIN"`
	RefreshCookieSecure       bool          `mapstructure:"REFRESH_COOKIE_SECURE"`
	RefreshCookieSameSite     string        `mapstructure:"REFRESH_COOKIE_SAME_SITE"`
	AllowedOrigins            string        `mapstructure:"ALLOWED_ORIGINS"`
	RedisURL                  string        `mapstructure:"REDIS_URL"`
	RedisAddress              string        `mapstructure:"REDIS_ADDRESS"`
	DBMaxConns                int32         `mapstructure:"DB_MAX_CONNS"`
	DBMinConns                int32         `mapstructure:"DB_MIN_CONNS"`
	DBMaxConnLifetime         time.Duration `mapstructure:"DB_MAX_CONN_LIFETIME"`
	DBMaxConnIdleTime         time.Duration `mapstructure:"DB_MAX_CONN_IDLE_TIME"`
	DBConnectTimeout          time.Duration `mapstructure:"DB_CONNECT_TIMEOUT"`
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
	LogLevel                  string        `mapstructure:"LOG_LEVEL"`
	PasswordBreachCheckURL    string        `mapstructure:"PASSWORD_BREACH_CHECK_URL"`
	PasswordBreachTimeout     time.Duration `mapstructure:"PASSWORD_BREACH_CHECK_TIMEOUT"`
	PasswordBreachCacheTTL    time.Duration `mapstructure:"PASSWORD_BREACH_CACHE_TTL"`
	OperationsToken           string        `mapstructure:"OPERATIONS_TOKEN"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.SetConfigFile(filepath.Join(path, "app.env"))
	viper.AutomaticEnv()

	_ = viper.BindEnv("APP_ENV")
	_ = viper.BindEnv("DB_SOURCE")
	_ = viper.BindEnv("DB_TEST_SOURCE")
	_ = viper.BindEnv("SERVER_ADDRESS")
	_ = viper.BindEnv("SECRET_KEY")
	_ = viper.BindEnv("ACCESS_TOKEN_DURATION")
	_ = viper.BindEnv("REFRESH_TOKEN_DURATION")
	_ = viper.BindEnv("REFRESH_COOKIE_NAME")
	_ = viper.BindEnv("DEVICE_COOKIE_NAME")
	_ = viper.BindEnv("REFRESH_COOKIE_DOMAIN")
	_ = viper.BindEnv("REFRESH_COOKIE_SECURE")
	_ = viper.BindEnv("REFRESH_COOKIE_SAME_SITE")
	_ = viper.BindEnv("ALLOWED_ORIGINS")
	_ = viper.BindEnv("REDIS_URL")
	_ = viper.BindEnv("REDIS_ADDRESS")
	_ = viper.BindEnv("DB_MAX_CONNS")
	_ = viper.BindEnv("DB_MIN_CONNS")
	_ = viper.BindEnv("DB_MAX_CONN_LIFETIME")
	_ = viper.BindEnv("DB_MAX_CONN_IDLE_TIME")
	_ = viper.BindEnv("DB_CONNECT_TIMEOUT")
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
	_ = viper.BindEnv("LOG_LEVEL")
	_ = viper.BindEnv("PASSWORD_BREACH_CHECK_URL")
	_ = viper.BindEnv("PASSWORD_BREACH_CHECK_TIMEOUT")
	_ = viper.BindEnv("PASSWORD_BREACH_CACHE_TTL")
	_ = viper.BindEnv("OPERATIONS_TOKEN")

	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("MAILER_PROVIDER", "log")
	viper.SetDefault("REFRESH_COOKIE_NAME", "monierave_refresh")
	viper.SetDefault("DEVICE_COOKIE_NAME", "monierave_device")
	viper.SetDefault("REFRESH_COOKIE_SECURE", true)
	viper.SetDefault("REFRESH_COOKIE_SAME_SITE", "lax")
	viper.SetDefault("DB_MAX_CONNS", 4)
	viper.SetDefault("DB_MIN_CONNS", 0)
	viper.SetDefault("DB_MAX_CONN_LIFETIME", 30*time.Minute)
	viper.SetDefault("DB_MAX_CONN_IDLE_TIME", 5*time.Minute)
	viper.SetDefault("DB_CONNECT_TIMEOUT", 5*time.Second)
	viper.SetDefault("WORKER_CONCURRENCY", 10)
	viper.SetDefault("RELAY_BATCH_SIZE", 50)
	viper.SetDefault("RELAY_POLL_INTERVAL", time.Second)
	viper.SetDefault("RELAY_CLAIM_LEASE", 30*time.Second)
	viper.SetDefault("PUBLIC_API_URL", "http://localhost:8080")
	viper.SetDefault("EMAIL_VERIFICATION_DURATION", 24*time.Hour)
	viper.SetDefault("ENFORCE_EMAIL_VERIFICATION", true)
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("PASSWORD_BREACH_CHECK_URL", "https://api.pwnedpasswords.com/range")
	viper.SetDefault("PASSWORD_BREACH_CHECK_TIMEOUT", 2*time.Second)
	viper.SetDefault("PASSWORD_BREACH_CACHE_TTL", 24*time.Hour)

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
	if err := validateDatabasePool(config); err != nil {
		return err
	}
	if err := validateRedis(config); err != nil {
		return err
	}
	if config.SecretKey == "" {
		return fmt.Errorf("SECRET_KEY is required")
	}
	if len(config.SecretKey) < 32 {
		return fmt.Errorf("SECRET_KEY must contain at least 32 characters")
	}
	if config.AccessTokenDuration <= 0 {
		return fmt.Errorf("ACCESS_TOKEN_DURATION must be greater than 0")
	}
	if config.RefreshTokenDuration <= 0 {
		return fmt.Errorf("REFRESH_TOKEN_DURATION must be greater than 0")
	}
	if config.RefreshCookieName == "" {
		return fmt.Errorf("REFRESH_COOKIE_NAME is required")
	}
	if config.DeviceCookieName == "" {
		return fmt.Errorf("DEVICE_COOKIE_NAME is required")
	}
	if config.DeviceCookieName == config.RefreshCookieName {
		return fmt.Errorf("DEVICE_COOKIE_NAME must differ from REFRESH_COOKIE_NAME")
	}
	switch config.RefreshCookieSameSite {
	case "strict", "lax", "none":
	default:
		return fmt.Errorf("REFRESH_COOKIE_SAME_SITE must be strict, lax, or none")
	}
	if config.RefreshCookieSameSite == "none" && !config.RefreshCookieSecure {
		return fmt.Errorf("REFRESH_COOKIE_SECURE must be true when SameSite is none")
	}
	for _, origin := range strings.Split(config.AllowedOrigins, ",") {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Host == "" ||
			(parsed.Scheme != "http" && parsed.Scheme != "https") {
			return fmt.Errorf("ALLOWED_ORIGINS must contain absolute HTTP or HTTPS origins")
		}
	}
	if config.PasswordBreachCheckURL == "" {
		return fmt.Errorf("PASSWORD_BREACH_CHECK_URL is required")
	}
	breachURL, err := url.Parse(config.PasswordBreachCheckURL)
	if err != nil || breachURL.Host == "" ||
		(breachURL.Scheme != "http" && breachURL.Scheme != "https") {
		return fmt.Errorf("PASSWORD_BREACH_CHECK_URL must be an absolute HTTP or HTTPS URL")
	}
	if config.PasswordBreachTimeout <= 0 {
		return fmt.Errorf("PASSWORD_BREACH_CHECK_TIMEOUT must be greater than 0")
	}
	if config.PasswordBreachCacheTTL <= 0 {
		return fmt.Errorf("PASSWORD_BREACH_CACHE_TTL must be greater than 0")
	}
	if strings.EqualFold(strings.TrimSpace(config.Environment), "production") &&
		len(strings.TrimSpace(config.ResendWebhookSecret)) < 32 {
		return fmt.Errorf("RESEND_WEBHOOK_SECRET must contain at least 32 characters in production")
	}
	if err := validateProductionSecurity(config); err != nil {
		return err
	}
	return nil
}

func validateProductionSecurity(config Config) error {
	if !strings.EqualFold(strings.TrimSpace(config.Environment), "production") {
		return nil
	}
	if !config.RefreshCookieSecure {
		return fmt.Errorf("REFRESH_COOKIE_SECURE must be true in production")
	}
	if !config.EnforceEmailVerification {
		return fmt.Errorf("ENFORCE_EMAIL_VERIFICATION must be true in production")
	}
	if len(config.OperationsToken) < 32 {
		return fmt.Errorf("OPERATIONS_TOKEN must contain at least 32 characters in production")
	}
	if strings.TrimSpace(config.AllowedOrigins) == "" {
		return fmt.Errorf("ALLOWED_ORIGINS must contain at least one HTTPS origin in production")
	}
	for _, origin := range strings.Split(config.AllowedOrigins, ",") {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Scheme != "https" {
			return fmt.Errorf("ALLOWED_ORIGINS must use HTTPS in production")
		}
	}
	databaseURL, err := url.Parse(config.DBSource)
	if err != nil {
		return fmt.Errorf("DB_SOURCE must be a valid URL in production")
	}
	switch strings.ToLower(databaseURL.Query().Get("sslmode")) {
	case "require", "verify-ca", "verify-full":
	default:
		return fmt.Errorf("DB_SOURCE must require TLS in production")
	}
	redisURL, err := url.Parse(config.RedisURL)
	if err != nil || redisURL.Host == "" || redisURL.Scheme != "rediss" {
		return fmt.Errorf("REDIS_URL must use TLS (rediss://) in production")
	}
	return nil
}

func ValidateRelayConfig(config Config) error {
	if config.DBSource == "" {
		return fmt.Errorf("DB_SOURCE is required")
	}
	if err := validateDatabasePool(config); err != nil {
		return err
	}
	if err := validateRedis(config); err != nil {
		return err
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
	return validateProductionSecurity(config)
}

func ValidateWorkerConfig(config Config) error {
	if config.DBSource == "" {
		return fmt.Errorf("DB_SOURCE is required")
	}
	if err := validateDatabasePool(config); err != nil {
		return err
	}
	if err := validateRedis(config); err != nil {
		return err
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
	if strings.EqualFold(config.Environment, "production") {
		publicURL, err := url.Parse(config.PublicAPIURL)
		if err != nil || publicURL.Scheme != "https" {
			return fmt.Errorf("PUBLIC_API_URL must use HTTPS in production")
		}
		if config.MailerProvider != "resend" {
			return fmt.Errorf("MAILER_PROVIDER must be resend in production")
		}
	}
	publicURL, err := url.Parse(config.PublicAPIURL)
	if err != nil || publicURL.Host == "" ||
		(publicURL.Scheme != "http" && publicURL.Scheme != "https") {
		return fmt.Errorf("PUBLIC_API_URL must be an absolute HTTP or HTTPS URL")
	}
	if config.EmailVerificationDuration <= 0 {
		return fmt.Errorf("EMAIL_VERIFICATION_DURATION must be greater than 0")
	}
	return validateProductionSecurity(config)
}

func validateRedis(config Config) error {
	if strings.TrimSpace(config.RedisURL) == "" && strings.TrimSpace(config.RedisAddress) == "" {
		return fmt.Errorf("REDIS_URL or REDIS_ADDRESS is required")
	}
	if config.RedisURL == "" {
		return nil
	}
	redisURL, err := url.Parse(config.RedisURL)
	if err != nil || redisURL.Host == "" ||
		(redisURL.Scheme != "redis" && redisURL.Scheme != "rediss") {
		return fmt.Errorf("REDIS_URL must be a valid redis:// or rediss:// URL")
	}
	return nil
}

func validateDatabasePool(config Config) error {
	if config.DBMaxConns <= 0 {
		return fmt.Errorf("DB_MAX_CONNS must be greater than 0")
	}
	if config.DBMinConns < 0 {
		return fmt.Errorf("DB_MIN_CONNS must not be negative")
	}
	if config.DBMinConns > config.DBMaxConns {
		return fmt.Errorf("DB_MIN_CONNS must not exceed DB_MAX_CONNS")
	}
	if config.DBMaxConnLifetime <= 0 {
		return fmt.Errorf("DB_MAX_CONN_LIFETIME must be greater than 0")
	}
	if config.DBMaxConnIdleTime <= 0 {
		return fmt.Errorf("DB_MAX_CONN_IDLE_TIME must be greater than 0")
	}
	if config.DBConnectTimeout <= 0 {
		return fmt.Errorf("DB_CONNECT_TIMEOUT must be greater than 0")
	}
	return nil
}
