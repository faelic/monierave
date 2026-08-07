package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	mockdb "github.com/faelic/monierave/db/mock"
	db "github.com/faelic/monierave/db/sqlc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestResendWebhookProcessesVerifiedBounce(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	secretBytes := []byte("local-webhook-test-secret")
	server.config.ResendWebhookSecret = "whsec_" + base64.StdEncoding.EncodeToString(secretBytes)
	webhookID := "msg_test_bounce"
	createdAt := time.Now().UTC().Truncate(time.Millisecond)
	jobID := "ceeb4e49-0b44-4944-9034-a12cbc32aaad"
	providerMessageID := "1e78e6f2-05a2-441f-bca9-148f12d4ffc2"
	payload := map[string]any{
		"created_at": createdAt.Format(time.RFC3339Nano),
		"type":       "email.bounced",
		"data": map[string]any{
			"email_id": providerMessageID,
			"tags":     map[string]string{"job_id": jobID},
			"bounce": map[string]any{
				"type":    "Permanent",
				"subType": "General",
				"message": "mailbox does not exist",
			},
		},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	store.EXPECT().
		ProcessEmailDeliveryEventTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, arg db.ProcessEmailDeliveryEventParams) (db.ProcessEmailDeliveryEventResult, error) {
			require.Equal(t, webhookID, arg.WebhookID)
			require.Equal(t, "email.bounced", arg.EventType)
			require.Equal(t, providerMessageID, arg.ProviderMessageID)
			require.Equal(t, jobID, fmt.Sprintf("%x-%x-%x-%x-%x",
				arg.JobID.Bytes[0:4],
				arg.JobID.Bytes[4:6],
				arg.JobID.Bytes[6:8],
				arg.JobID.Bytes[8:10],
				arg.JobID.Bytes[10:16],
			))
			require.Equal(t, db.EmailDeliveryStatusBounced, arg.DeliveryStatus)
			require.Equal(t, "Permanent", arg.BounceType)
			require.Equal(t, "General", arg.BounceSubtype)
			require.Equal(t, "email provider reported a bounce", arg.BounceMessage)

			var normalized normalizedResendWebhook
			require.NoError(t, json.Unmarshal(arg.Payload, &normalized))
			require.Equal(t, "email.bounced", normalized.EventType)
			require.Equal(t, providerMessageID, normalized.ProviderMessageID)
			require.Equal(t, jobID, normalized.JobID)
			require.Equal(t, createdAt, normalized.OccurredAt)
			require.Equal(t, fmt.Sprintf("%x", sha256.Sum256(body)), normalized.RawPayloadSHA256)
			require.NotNil(t, normalized.Bounce)
			require.Equal(t, "Permanent", normalized.Bounce.Type)
			require.Equal(t, "General", normalized.Bounce.Subtype)
			require.NotContains(t, string(arg.Payload), "mailbox does not exist")
			return db.ProcessEmailDeliveryEventResult{JobMatched: true, StateUpdated: true}, nil
		})

	request := signedWebhookRequest(t, body, webhookID, secretBytes)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestResendWebhookRejectsInvalidSignature(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)
	server.config.ResendWebhookSecret = "whsec_" +
		base64.StdEncoding.EncodeToString([]byte("expected-secret"))

	store.EXPECT().ProcessEmailDeliveryEventTx(gomock.Any(), gomock.Any()).Times(0)

	request := signedWebhookRequest(
		t,
		[]byte(`{"type":"email.bounced"}`),
		"msg_invalid",
		[]byte("wrong-secret"),
	)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestResendWebhookPersistsIgnoredEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)
	secretBytes := []byte("local-webhook-test-secret")
	server.config.ResendWebhookSecret = "whsec_" +
		base64.StdEncoding.EncodeToString(secretBytes)
	webhookID := "msg_domain_event"
	body, err := json.Marshal(map[string]any{
		"created_at": time.Now().UTC().Format(time.RFC3339Nano),
		"type":       "domain.updated",
		"data":       map[string]any{},
	})
	require.NoError(t, err)

	store.EXPECT().
		ProcessEmailDeliveryEventTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			_ any,
			arg db.ProcessEmailDeliveryEventParams,
		) (db.ProcessEmailDeliveryEventResult, error) {
			require.Equal(t, webhookID, arg.ProviderMessageID)
			require.Equal(t, "domain.updated", arg.EventType)
			require.Empty(t, arg.DeliveryStatus)
			require.False(t, arg.JobID.Valid)
			return db.ProcessEmailDeliveryEventResult{}, nil
		})

	request := signedWebhookRequest(t, body, webhookID, secretBytes)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
}

func signedWebhookRequest(
	t *testing.T,
	body []byte,
	webhookID string,
	secret []byte,
) *http.Request {
	t.Helper()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	content := webhookID + "." + timestamp + "." + string(body)
	mac := hmac.New(sha256.New, secret)
	_, err := mac.Write([]byte(content))
	require.NoError(t, err)
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	request, err := http.NewRequest(
		http.MethodPost,
		"/webhooks/resend",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	request.Header.Set("svix-id", webhookID)
	request.Header.Set("svix-timestamp", timestamp)
	request.Header.Set("svix-signature", "v1,"+signature)
	return request
}
