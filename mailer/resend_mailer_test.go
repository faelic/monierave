package mailer

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/resend/resend-go/v3"
	"github.com/stretchr/testify/require"
)

type stubResendSender struct {
	response *resend.SendEmailResponse
	err      error
	params   *resend.SendEmailRequest
	options  *resend.SendEmailOptions
	calls    int
}

func (sender *stubResendSender) SendWithOptions(
	_ context.Context,
	params *resend.SendEmailRequest,
	options *resend.SendEmailOptions,
) (*resend.SendEmailResponse, error) {
	sender.calls++
	sender.params = params
	sender.options = options
	return sender.response, sender.err
}

func TestResendMailerSendsIdempotentEmail(t *testing.T) {
	sender := &stubResendSender{
		response: &resend.SendEmailResponse{Id: "resend-message-id"},
	}
	emailMailer := &ResendMailer{
		sender: sender,
		from:   "Monierave <no-reply@example.com>",
	}

	messageID, err := emailMailer.SendVerificationEmail(context.Background(), VerificationEmail{
		JobID:     "322b4f16-60b4-4ee7-909d-2232c43fc4f7",
		Username:  "<Favour>",
		Recipient: "favour@example.com",
		Payload:   jsonPayload(t, map[string]string{}),
	})

	require.NoError(t, err)
	require.Equal(t, "resend-message-id", messageID)
	require.Equal(t, 1, sender.calls)
	require.Equal(t, "Monierave <no-reply@example.com>", sender.params.From)
	require.Equal(t, []string{"favour@example.com"}, sender.params.To)
	require.Equal(t, resendEmailSubject, sender.params.Subject)
	require.Contains(t, sender.params.Html, "&lt;Favour&gt;")
	require.NotContains(t, sender.params.Html, "<Favour>")
	require.Equal(t, "verify-email/322b4f16-60b4-4ee7-909d-2232c43fc4f7", sender.options.IdempotencyKey)
}

func TestResendMailerIncludesVerificationURL(t *testing.T) {
	sender := &stubResendSender{
		response: &resend.SendEmailResponse{Id: "resend-message-id"},
	}
	emailMailer := &ResendMailer{sender: sender, from: "no-reply@example.com"}

	_, err := emailMailer.SendVerificationEmail(context.Background(), VerificationEmail{
		JobID:     "job-id",
		Username:  "Favour",
		Recipient: "favour@example.com",
		Payload: jsonPayload(t, map[string]string{
			"verification_url": "https://example.com/verify?token=secret&source=email",
		}),
	})

	require.NoError(t, err)
	require.Equal(t, resendVerificationSubject, sender.params.Subject)
	require.Equal(t, 2, strings.Count(
		sender.params.Html,
		`href="https://example.com/verify?token=secret&amp;source=email"`,
	))
	require.Contains(t, sender.params.Html, "copy and paste this link")
	require.Contains(t, sender.params.Html, "expires in 24 hours")
	require.Contains(
		t,
		sender.params.Text,
		"https://example.com/verify?token=secret&source=email",
	)
	require.Contains(t, sender.params.Text, "copy and paste")
}

func TestResendMailerSendsIdempotentFinancialNotification(t *testing.T) {
	sender := &stubResendSender{
		response: &resend.SendEmailResponse{Id: "financial-message-id"},
	}
	emailMailer := &ResendMailer{sender: sender, from: "no-reply@example.com"}
	jobID := "b54658bc-8e05-48d8-8737-d1c009f91028"

	messageID, err := emailMailer.SendFinancialNotification(
		context.Background(),
		FinancialNotificationEmail{
			JobID:     jobID,
			Username:  "<Favour>",
			Recipient: "favour@example.com",
			Payload: jsonPayload(t, financialPayload{
				EventType:  "transaction.posted",
				Reference:  "TXN-SAFE-REFERENCE",
				Amount:     5_000,
				Currency:   "USD",
				Direction:  "outgoing",
				OccurredAt: time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC),
			}),
		},
	)

	require.NoError(t, err)
	require.Equal(t, "financial-message-id", messageID)
	require.Equal(t, "financial-notification/"+jobID, sender.options.IdempotencyKey)
	require.Contains(t, sender.params.Subject, "Transaction posted")
	require.Contains(t, sender.params.Html, "TXN-SAFE-REFERENCE")
	require.Contains(t, sender.params.Html, "5000 USD")
	require.Contains(t, sender.params.Html, "outgoing")
	require.Contains(t, sender.params.Html, "&lt;Favour&gt;")
	require.NotContains(t, sender.params.Html, "password")
	require.NotContains(t, sender.params.Html, "token")
	require.Equal(t, "transaction_posted", sender.params.Tags[0].Value)
	require.Equal(t, jobID, sender.params.Tags[1].Value)
}

func TestResendMailerRejectsUnsupportedFinancialEvent(t *testing.T) {
	sender := &stubResendSender{}
	emailMailer := &ResendMailer{sender: sender, from: "no-reply@example.com"}
	_, err := emailMailer.SendFinancialNotification(
		context.Background(),
		FinancialNotificationEmail{
			JobID:     "job-id",
			Username:  "Favour",
			Recipient: "favour@example.com",
			Payload: jsonPayload(t, financialPayload{
				EventType:  "user.password_changed",
				OccurredAt: time.Now(),
			}),
		},
	)
	require.Error(t, err)
	require.True(t, IsPermanent(err))
	require.Zero(t, sender.calls)
}

func TestResendMailerReturnsRetryableProviderError(t *testing.T) {
	sender := &stubResendSender{err: errors.New("temporary provider failure")}
	emailMailer := &ResendMailer{sender: sender, from: "no-reply@example.com"}

	_, err := emailMailer.SendVerificationEmail(context.Background(), VerificationEmail{
		JobID:     "job-id",
		Username:  "Favour",
		Recipient: "favour@example.com",
		Payload:   jsonPayload(t, map[string]string{}),
	})

	require.Error(t, err)
	require.False(t, IsPermanent(err))
}

func TestResendMailerRejectsInvalidMessage(t *testing.T) {
	tests := []struct {
		name      string
		recipient string
		payload   []byte
	}{
		{
			name:      "invalid recipient",
			recipient: "not-an-email",
			payload:   jsonPayload(t, map[string]string{}),
		},
		{
			name:      "invalid payload",
			recipient: "favour@example.com",
			payload:   []byte("{"),
		},
		{
			name:      "relative verification URL",
			recipient: "favour@example.com",
			payload: jsonPayload(t, map[string]string{
				"verification_url": "/users/verify-email?token=secret",
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sender := &stubResendSender{}
			emailMailer := &ResendMailer{sender: sender, from: "no-reply@example.com"}

			_, err := emailMailer.SendVerificationEmail(context.Background(), VerificationEmail{
				JobID:     "job-id",
				Username:  "Favour",
				Recipient: test.recipient,
				Payload:   test.payload,
			})

			require.Error(t, err)
			require.True(t, IsPermanent(err))
			require.Zero(t, sender.calls)
		})
	}
}

func TestNewResendMailerValidatesConfiguration(t *testing.T) {
	_, err := NewResendMailer("", "no-reply@example.com")
	require.Error(t, err)

	_, err = NewResendMailer("re_test", "invalid")
	require.Error(t, err)

	emailMailer, err := NewResendMailer("re_test", "Monierave <no-reply@example.com>")
	require.NoError(t, err)
	require.NotNil(t, emailMailer)
}

func jsonPayload(t *testing.T, value any) []byte {
	t.Helper()
	payload, err := json.Marshal(value)
	require.NoError(t, err)
	return payload
}
