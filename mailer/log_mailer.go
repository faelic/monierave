package mailer

import (
	"context"
	"encoding/json"
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
		Str("provider_message_id", providerMessageID).
		Msg("development mailer accepted verification email")
	return providerMessageID, nil
}

func (m *LogMailer) SendFinancialNotification(
	_ context.Context,
	message FinancialNotificationEmail,
) (string, error) {
	providerMessageID := fmt.Sprintf("log-%s", message.JobID)
	var payload struct {
		EventID       string `json:"event_id"`
		CorrelationID string `json:"correlation_id"`
		EventType     string `json:"event_type"`
	}
	_ = json.Unmarshal(message.Payload, &payload)
	log.Info().
		Str("job_id", message.JobID).
		Str("event_id", payload.EventID).
		Str("correlation_id", payload.CorrelationID).
		Str("event_type", payload.EventType).
		Str("provider_message_id", providerMessageID).
		Msg("development mailer accepted financial notification")
	return providerMessageID, nil
}
