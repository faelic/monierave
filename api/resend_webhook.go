package api

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	db "github.com/faelic/monierave/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/resend/resend-go/v3"
	"github.com/rs/zerolog/log"
)

const maxWebhookBodySize = 1 << 20 // 1 MB

type resendWebhookEvent struct {
	Type      string          `json:"type"`
	CreatedAt string          `json:"created_at"`
	Data      resendEmailData `json:"data"`
}

type resendEmailData struct {
	EmailID string            `json:"email_id"`
	Tags    map[string]string `json:"tags"`
	Bounce  resendBounceData  `json:"bounce"`
}

type resendBounceData struct {
	Subtype string `json:"subType"`
	Type    string `json:"type"`
}

type normalizedResendWebhook struct {
	EventType         string                  `json:"event_type"`
	ProviderMessageID string                  `json:"provider_message_id"`
	JobID             string                  `json:"job_id,omitempty"`
	OccurredAt        time.Time               `json:"occurred_at"`
	Bounce            *normalizedResendBounce `json:"bounce,omitempty"`
	RawPayloadSHA256  string                  `json:"raw_payload_sha256"`
}

type normalizedResendBounce struct {
	Type    string `json:"type,omitempty"`
	Subtype string `json:"subtype,omitempty"`
}

func (server *Server) handleResendWebhook(ctx *gin.Context) {
	if server.config.ResendWebhookSecret == "" {
		ctx.JSON(
			http.StatusServiceUnavailable,
			codedErrorResponse(
				ctx,
				"webhook_not_configured",
				errors.New("webhook is not configured"),
			),
		)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(
		ctx.Writer,
		ctx.Request.Body,
		maxWebhookBodySize,
	))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, codedErrorResponse(
			ctx, "invalid_request_body", errors.New("invalid request body"),
		))
		return
	}

	client := resend.NewClient("")
	err = client.Webhooks.Verify(&resend.VerifyWebhookOptions{
		Payload: string(body),
		Headers: resend.WebhookHeaders{
			Id:        ctx.GetHeader("svix-id"),
			Timestamp: ctx.GetHeader("svix-timestamp"),
			Signature: ctx.GetHeader("svix-signature"),
		},
		WebhookSecret: server.config.ResendWebhookSecret,
	})
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, codedErrorResponse(
			ctx, "invalid_signature", errors.New("invalid signature"),
		))
		return
	}

	var event resendWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		ctx.JSON(http.StatusBadRequest, codedErrorResponse(
			ctx, "invalid_webhook_event", errors.New("invalid webhook event"),
		))
		return
	}

	isEmailEvent := strings.HasPrefix(event.Type, "email.")
	if isEmailEvent && event.Data.EmailID == "" {
		ctx.JSON(http.StatusBadRequest, codedErrorResponse(
			ctx,
			"missing_email_id",
			errors.New("email webhook is missing email_id"),
		))
		return
	}

	occurredAt, err := time.Parse(time.RFC3339Nano, event.CreatedAt)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, codedErrorResponse(
			ctx,
			"invalid_webhook_timestamp",
			errors.New("email webhook has invalid created_at"),
		))
		return
	}
	providerMessageID := event.Data.EmailID
	if providerMessageID == "" {
		// Non-email events do not necessarily carry an email ID. The verified
		// webhook ID is stable and preserves append-only observability.
		providerMessageID = ctx.GetHeader("svix-id")
	}

	normalizedPayload, err := normalizeResendWebhookPayload(
		event,
		providerMessageID,
		occurredAt,
		body,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, codedErrorResponse(
			ctx,
			"webhook_processing_failed",
			errors.New("failed to normalize webhook"),
		))
		return
	}

	result, err := server.store.ProcessEmailDeliveryEventTx(
		ctx,
		db.ProcessEmailDeliveryEventParams{
			WebhookID:         ctx.GetHeader("svix-id"),
			EventType:         event.Type,
			ProviderMessageID: providerMessageID,
			JobID:             parseOptionalWebhookJobID(event.Data.Tags["job_id"]),
			OccurredAt:        occurredAt,
			Payload:           normalizedPayload,
			DeliveryStatus:    deliveryStatusForResendEvent(event.Type),
			BounceType:        event.Data.Bounce.Type,
			BounceSubtype:     event.Data.Bounce.Subtype,
			BounceMessage:     safeDeliveryFailureMessage(event.Type),
		},
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("webhook_id", ctx.GetHeader("svix-id")).
			Str("event_type", event.Type).
			Str("provider_message_id", providerMessageID).
			Msg("failed to persist Resend webhook")
		ctx.JSON(http.StatusInternalServerError, codedErrorResponse(
			ctx,
			"webhook_processing_failed",
			errors.New("failed to process webhook"),
		))
		return
	}

	log.Info().
		Str("webhook_id", ctx.GetHeader("svix-id")).
		Str("event_type", event.Type).
		Str("provider_message_id", providerMessageID).
		Bool("duplicate", result.Duplicate).
		Bool("job_matched", result.JobMatched).
		Bool("state_updated", result.StateUpdated).
		Msg("verified Resend webhook processed")

	ctx.Status(http.StatusOK)
}

func normalizeResendWebhookPayload(
	event resendWebhookEvent,
	providerMessageID string,
	occurredAt time.Time,
	rawPayload []byte,
) ([]byte, error) {
	rawHash := sha256.Sum256(rawPayload)
	payload := normalizedResendWebhook{
		EventType:         event.Type,
		ProviderMessageID: providerMessageID,
		JobID:             event.Data.Tags["job_id"],
		OccurredAt:        occurredAt,
		RawPayloadSHA256:  fmt.Sprintf("%x", rawHash),
	}
	if event.Data.Bounce.Type != "" || event.Data.Bounce.Subtype != "" {
		payload.Bounce = &normalizedResendBounce{
			Type:    event.Data.Bounce.Type,
			Subtype: event.Data.Bounce.Subtype,
		}
	}
	return json.Marshal(payload)
}

func safeDeliveryFailureMessage(eventType string) string {
	switch eventType {
	case "email.bounced":
		return "email provider reported a bounce"
	case "email.failed":
		return "email provider reported a delivery failure"
	case "email.suppressed":
		return "email provider suppressed delivery"
	case "email.complained":
		return "recipient reported the email as unwanted"
	default:
		return ""
	}
}

func deliveryStatusForResendEvent(eventType string) string {
	switch eventType {
	case "email.sent":
		return db.EmailDeliveryStatusAccepted
	case "email.delivered":
		return db.EmailDeliveryStatusDelivered
	case "email.delivery_delayed":
		return db.EmailDeliveryStatusDelayed
	case "email.bounced":
		return db.EmailDeliveryStatusBounced
	case "email.failed":
		return db.EmailDeliveryStatusFailed
	case "email.suppressed":
		return db.EmailDeliveryStatusSuppressed
	case "email.complained":
		return db.EmailDeliveryStatusComplained
	default:
		return ""
	}
}

func parseOptionalWebhookJobID(value string) pgtype.UUID {
	id, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: id, Valid: true}
}
