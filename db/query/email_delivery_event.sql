-- name: CreateEmailDeliveryEvent :one
INSERT INTO email_delivery_events (
  webhook_id,
  email_job_id,
  provider_message_id,
  event_type,
  occurred_at,
  payload
) VALUES (
  $1, $2, $3, $4, $5, $6
)
ON CONFLICT (webhook_id) DO NOTHING
RETURNING *;

-- name: GetEmailDeliveryEvent :one
SELECT * FROM email_delivery_events
WHERE webhook_id = $1
LIMIT 1;

-- name: ListEmailDeliveryEventsByJob :many
SELECT * FROM email_delivery_events
WHERE email_job_id = $1
ORDER BY occurred_at DESC, received_at DESC
LIMIT $2;
