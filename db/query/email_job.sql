-- name: CreateEmailJob :one
INSERT INTO email_jobs (
  id,
  parent_job_id,
  job_type,
  username,
  recipient,
  payload,
  max_attempts
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetEmailJob :one
SELECT * FROM email_jobs
WHERE id = $1
LIMIT 1;

-- name: GetEmailJobByProviderMessageID :one
SELECT * FROM email_jobs
WHERE provider_message_id = $1
LIMIT 1;

-- name: GetLatestEmailJobForCurrentAddress :one
SELECT email_jobs.*
FROM email_jobs
JOIN users ON users.username = email_jobs.username
WHERE email_jobs.username = $1
  AND lower(email_jobs.recipient) = lower(users.email)
ORDER BY email_jobs.created_at DESC
LIMIT 1;

-- name: MarkEmailJobQueued :one
UPDATE email_jobs
SET
  status = CASE WHEN status = 'pending' THEN 'queued' ELSE status END,
  queued_at = CASE WHEN status = 'pending' THEN now() ELSE queued_at END,
  updated_at = now()
WHERE id = $1
RETURNING *;

-- name: StartEmailJobAttempt :one
UPDATE email_jobs
SET
  status = 'processing',
  attempt_count = attempt_count + 1,
  processing_at = now(),
  last_attempt_at = now(),
  updated_at = now()
WHERE id = $1
  AND status IN ('pending', 'queued', 'processing', 'retrying')
RETURNING *;

-- name: MarkEmailJobRetrying :one
UPDATE email_jobs
SET
  status = 'retrying',
  last_error = $2,
  updated_at = now()
WHERE id = $1
  AND status <> 'sent'
RETURNING *;

-- name: MarkEmailJobSent :one
UPDATE email_jobs
SET
  status = 'sent',
  provider_message_id = $2,
  delivery_status = CASE
    WHEN delivery_event_at IS NULL THEN 'accepted'
    ELSE delivery_status
  END,
  accepted_at = COALESCE(accepted_at, now()),
  last_error = NULL,
  sent_at = now(),
  updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateEmailJobDelivery :one
UPDATE email_jobs
SET
  provider_message_id = COALESCE(provider_message_id, sqlc.arg(provider_message_id)),
  delivery_status = sqlc.arg(delivery_status),
  delivery_event_at = sqlc.arg(occurred_at),
  accepted_at = CASE
    WHEN sqlc.arg(delivery_status)::varchar = 'accepted'
      THEN COALESCE(accepted_at, sqlc.arg(occurred_at))
    ELSE accepted_at
  END,
  delivered_at = CASE
    WHEN sqlc.arg(delivery_status)::varchar = 'delivered'
      THEN sqlc.arg(occurred_at)
    ELSE delivered_at
  END,
  bounced_at = CASE
    WHEN sqlc.arg(delivery_status)::varchar IN ('bounced', 'suppressed')
      THEN sqlc.arg(occurred_at)
    ELSE bounced_at
  END,
  bounce_type = CASE
    WHEN sqlc.arg(delivery_status)::varchar = 'bounced'
      THEN sqlc.narg(bounce_type)
    ELSE bounce_type
  END,
  bounce_subtype = CASE
    WHEN sqlc.arg(delivery_status)::varchar = 'bounced'
      THEN sqlc.narg(bounce_subtype)
    ELSE bounce_subtype
  END,
  bounce_message = CASE
    WHEN sqlc.arg(delivery_status)::varchar = 'bounced'
      THEN sqlc.narg(bounce_message)
    ELSE bounce_message
  END,
  updated_at = now()
WHERE id = sqlc.arg(id)
  AND (
    delivery_event_at IS NULL
    OR delivery_event_at <= sqlc.arg(occurred_at)
  )
RETURNING *;

-- name: MarkEmailJobDeadLetter :one
UPDATE email_jobs
SET
  status = 'dead_letter',
  last_error = $2,
  dead_lettered_at = now(),
  updated_at = now()
WHERE id = $1
  AND status <> 'sent'
RETURNING *;

-- name: ListDeadLetterEmailJobs :many
SELECT * FROM email_jobs
WHERE status = 'dead_letter'
ORDER BY dead_lettered_at DESC
LIMIT $1;

-- name: DeleteExpiredSentEmailJobs :execrows
DELETE FROM email_jobs
WHERE status = 'sent'
  AND sent_at < $1;
