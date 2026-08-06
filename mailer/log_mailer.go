package mailer

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
)

// LogMailer is a development adapter. Production must provide a real provider
// that honors VerificationEmail.JobID as an idempotency key.
type LogMailer struct{}

func NewLogMailer() Mailer {
	return &LogMailer{}
}

func (m *LogMailer) SendVerificationEmail(
	_ context.Context,
	message VerificationEmail,
) (string, error) {
	providerMessageID := fmt.Sprintf("log-%s", message.JobID)
	log.Info().
		Str("job_id", message.JobID).
		Str("recipient", message.Recipient).
		Str("provider_message_id", providerMessageID).
		Msg("development mailer accepted verification email")
	return providerMessageID, nil
}

func (m *LogMailer) SendFinancialNotification(
	_ context.Context,
	message FinancialNotificationEmail,
) (string, error) {
	providerMessageID := fmt.Sprintf("log-%s", message.JobID)
	log.Info().
		Str("job_id", message.JobID).
		Str("recipient", message.Recipient).
		Str("provider_message_id", providerMessageID).
		Msg("development mailer accepted financial notification")
	return providerMessageID, nil
}
