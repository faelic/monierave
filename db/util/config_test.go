package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidateWorkerConfig(t *testing.T) {
	validConfig := Config{
		DBSource:                  "postgresql://localhost/database",
		RedisAddress:              "localhost:6379",
		MailerProvider:            "log",
		WorkerConcurrency:         10,
		PublicAPIURL:              "http://localhost:8080",
		EmailVerificationDuration: 24 * time.Hour,
	}

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
	err := ValidateAPIConfig(Config{
		DBSource:              "postgresql://localhost/database",
		RedisAddress:          "localhost:6379",
		ServerAddress:         "0.0.0.0:8080",
		SecretKey:             "12345678901234567890123456789012",
		AccessTokenDuration:   time.Minute,
		RefreshTokenDuration:  time.Hour,
		RefreshCookieName:     "monierave_refresh",
		DeviceCookieName:      "monierave_device",
		RefreshCookieSameSite: "lax",
	})

	require.NoError(t, err)
}

func TestValidateAPIConfigRejectsSharedSessionCookieName(t *testing.T) {
	config := Config{
		DBSource:              "postgresql://localhost/database",
		RedisAddress:          "localhost:6379",
		ServerAddress:         "0.0.0.0:8080",
		SecretKey:             "12345678901234567890123456789012",
		AccessTokenDuration:   time.Minute,
		RefreshTokenDuration:  time.Hour,
		RefreshCookieName:     "monierave_session",
		DeviceCookieName:      "monierave_session",
		RefreshCookieSameSite: "lax",
	}

	err := ValidateAPIConfig(config)

	require.ErrorContains(t, err, "must differ")
}
