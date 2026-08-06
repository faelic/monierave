package api

import (
	"encoding/json"
	"errors"
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
	DiagnosticCode []string `json:"diagnosticCode"`
	Message        string   `json:"message"`
	Subtype        string   `json:"subType"`
	Type           string   `json:"type"`
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

	if !strings.HasPrefix(event.Type, "email.") {
		log.Info().
			Str("webhook_id", ctx.GetHeader("svix-id")).
			Str("event_type", event.Type).
			Msg("verified non-email Resend webhook ignored")
		ctx.Status(http.StatusOK)
		return
	}
	if event.Data.EmailID == "" {
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

	result, err := server.store.ProcessEmailDeliveryEventTx(
		ctx,
		db.ProcessEmailDeliveryEventParams{
			WebhookID:         ctx.GetHeader("svix-id"),
			EventType:         event.Type,
			ProviderMessageID: event.Data.EmailID,
			JobID:             parseOptionalWebhookJobID(event.Data.Tags["job_id"]),
			OccurredAt:        occurredAt,
			Payload:           body,
			DeliveryStatus:    deliveryStatusForResendEvent(event.Type),
			BounceType:        event.Data.Bounce.Type,
			BounceSubtype:     event.Data.Bounce.Subtype,
			BounceMessage:     event.Data.Bounce.Message,
		},
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("webhook_id", ctx.GetHeader("svix-id")).
			Str("event_type", event.Type).
			Str("provider_message_id", event.Data.EmailID).
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
		Str("provider_message_id", event.Data.EmailID).
		Bool("duplicate", result.Duplicate).
		Bool("job_matched", result.JobMatched).
		Bool("state_updated", result.StateUpdated).
		Msg("verified Resend webhook processed")

	ctx.Status(http.StatusOK)
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
