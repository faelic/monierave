-- name: CreateOutboxEvent :one
INSERT INTO outbox_events (
  id,
  email_job_id,
  event_type,
  payload
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: GetOutboxEvent :one
SELECT * FROM outbox_events
WHERE id = $1
LIMIT 1;

-- name: ClaimOutboxEvents :many
WITH candidates AS (
  SELECT id
  FROM outbox_events
  WHERE available_at <= now()
    AND (
      status = 'pending'
      OR (
        status = 'publishing'
        AND claimed_until < now()
      )
    )
  ORDER BY created_at
  FOR UPDATE SKIP LOCKED
  LIMIT sqlc.arg(batch_size)
)
UPDATE outbox_events AS event
SET
  status = 'publishing',
  claimed_by = sqlc.arg(claimed_by),
  claimed_until = sqlc.arg(claimed_until),
  publish_attempts = publish_attempts + 1
FROM candidates
WHERE event.id = candidates.id
RETURNING event.*;

-- name: MarkOutboxEventPublished :one
UPDATE outbox_events
SET
  status = 'published',
  published_at = now(),
  claimed_by = NULL,
  claimed_until = NULL,
  last_error = NULL
WHERE id = $1
RETURNING *;

-- name: ReleaseOutboxEvent :one
UPDATE outbox_events
SET
  status = 'pending',
  available_at = $2,
  claimed_by = NULL,
  claimed_until = NULL,
  last_error = $3
WHERE id = $1
RETURNING *;

-- name: DeleteExpiredPublishedOutboxEvents :execrows
DELETE FROM outbox_events
WHERE status = 'published'
  AND published_at < $1;
