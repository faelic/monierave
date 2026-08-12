package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidateWorkerConfig(t *testing.T) {
	validConfig := withDatabasePool(Config{
		DBSource:                  "postgresql://localhost/database",
		RedisAddress:              "localhost:6379",
		MailerProvider:            "log",
		WorkerConcurrency:         10,
		PublicAPIURL:              "http://localhost:8080",
		EmailVerificationDuration: 24 * time.Hour,
	})

	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{
			name: "log provider",
		},
		{
			name: "resend provider",
			mutate: func(config *Config) {
				config.MailerProvider = "resend"
				config.ResendAPIKey = "re_test"
				config.EmailFrom = "Monierave <no-reply@example.com>"
			},
		},
		{
			name: "missing Resend API key",
			mutate: func(config *Config) {
				config.MailerProvider = "resend"
				config.EmailFrom = "no-reply@example.com"
			},
			wantErr: "RESEND_API_KEY is required",
		},
		{
			name: "missing sender",
			mutate: func(config *Config) {
				config.MailerProvider = "resend"
				config.ResendAPIKey = "re_test"
			},
			wantErr: "EMAIL_FROM is required",
		},
		{
			name: "unsupported provider",
			mutate: func(config *Config) {
				config.MailerProvider = "smtp"
			},
			wantErr: "unsupported MAILER_PROVIDER",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := validConfig
			if test.mutate != nil {
				test.mutate(&config)
			}

			err := ValidateWorkerConfig(config)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestValidateAPIConfigDoesNotRequireWebhookSecret(t *testing.T) {
	err := ValidateAPIConfig(withDatabasePool(Config{
		DBSource:               "postgresql://localhost/database",
		RedisAddress:           "localhost:6379",
		ServerAddress:          "0.0.0.0:8080",
		SecretKey:              "12345678901234567890123456789012",
		AccessTokenDuration:    time.Minute,
		RefreshTokenDuration:   time.Hour,
		RefreshCookieName:      "monierave_refresh",
		DeviceCookieName:       "monierave_device",
		RefreshCookieSameSite:  "lax",
		PasswordBreachCheckURL: "https://api.pwnedpasswords.com/range",
		PasswordBreachTimeout:  2 * time.Second,
		PasswordBreachCacheTTL: 24 * time.Hour,
	}))

	require.NoError(t, err)
}

func TestValidateAPIConfigRejectsSharedSessionCookieName(t *testing.T) {
	config := withDatabasePool(Config{
		DBSource:               "postgresql://localhost/database",
		RedisAddress:           "localhost:6379",
		ServerAddress:          "0.0.0.0:8080",
		SecretKey:              "12345678901234567890123456789012",
		AccessTokenDuration:    time.Minute,
		RefreshTokenDuration:   time.Hour,
		RefreshCookieName:      "monierave_session",
		DeviceCookieName:       "monierave_session",
		RefreshCookieSameSite:  "lax",
		PasswordBreachCheckURL: "https://api.pwnedpasswords.com/range",
		PasswordBreachTimeout:  2 * time.Second,
		PasswordBreachCacheTTL: 24 * time.Hour,
	})

	err := ValidateAPIConfig(config)

	require.ErrorContains(t, err, "must differ")
}

func TestValidateAPIConfigFailsClosedInProduction(t *testing.T) {
	config := withDatabasePool(Config{
		Environment:              "production",
		DBSource:                 "postgresql://db.example/monierave?sslmode=verify-full",
		RedisURL:                 "rediss://default:secret@redis.example:6379/0",
		ServerAddress:            "0.0.0.0:8080",
		SecretKey:                "12345678901234567890123456789012",
		OperationsToken:          "operations-token-123456789012345",
		AccessTokenDuration:      time.Minute,
		RefreshTokenDuration:     time.Hour,
		RefreshCookieName:        "monierave_refresh",
		DeviceCookieName:         "monierave_device",
		RefreshCookieSecure:      true,
		RefreshCookieSameSite:    "lax",
		AllowedOrigins:           "https://app.example.com",
		EnforceEmailVerification: true,
		ResendWebhookSecret:      "whsec_12345678901234567890123456789012",
		PasswordBreachCheckURL:   "https://api.pwnedpasswords.com/range",
		PasswordBreachTimeout:    2 * time.Second,
		PasswordBreachCacheTTL:   24 * time.Hour,
	})
	require.NoError(t, ValidateAPIConfig(config))

	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{"insecure database", func(c *Config) { c.DBSource = "postgresql://db.example/monierave?sslmode=disable" }, "require TLS"},
		{"insecure origin", func(c *Config) { c.AllowedOrigins = "http://app.example.com" }, "use HTTPS"},
		{"missing origin", func(c *Config) { c.AllowedOrigins = "" }, "at least one HTTPS origin"},
		{"missing operations token", func(c *Config) { c.OperationsToken = "" }, "OPERATIONS_TOKEN"},
		{"missing webhook secret", func(c *Config) { c.ResendWebhookSecret = "" }, "RESEND_WEBHOOK_SECRET"},
		{"insecure cookie", func(c *Config) { c.RefreshCookieSecure = false }, "REFRESH_COOKIE_SECURE"},
		{"verification disabled", func(c *Config) { c.EnforceEmailVerification = false }, "ENFORCE_EMAIL_VERIFICATION"},
		{"insecure Redis", func(c *Config) { c.RedisURL = "redis://redis.example:6379" }, "REDIS_URL must use TLS"},
		{"missing Redis URL", func(c *Config) { c.RedisURL = ""; c.RedisAddress = "redis.example:6379" }, "REDIS_URL must use TLS"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := config
			test.mutate(&candidate)
			require.ErrorContains(t, ValidateAPIConfig(candidate), test.wantErr)
		})
	}
}

func withDatabasePool(config Config) Config {
	config.DBMaxConns = 4
	config.DBMinConns = 0
	config.DBMaxConnLifetime = 30 * time.Minute
	config.DBMaxConnIdleTime = 5 * time.Minute
	config.DBConnectTimeout = 5 * time.Second
	return config
}
