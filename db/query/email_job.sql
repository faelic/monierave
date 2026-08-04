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
  last_error = NULL,
  sent_at = now(),
  updated_at = now()
WHERE id = $1
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
