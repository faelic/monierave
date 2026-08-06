-- name: DeleteExpiredIdempotencyKey :exec
DELETE FROM idempotency_keys
WHERE username = $1
  AND operation = $2
  AND idempotency_key = $3
  AND expires_at <= now();

-- name: CreateIdempotencyKey :one
INSERT INTO idempotency_keys (
  username,
  operation,
  idempotency_key,
  request_hash
) VALUES (
  $1, $2, $3, $4
)
ON CONFLICT (username, operation, idempotency_key) DO NOTHING
RETURNING *;

-- name: GetIdempotencyKeyForUpdate :one
SELECT * FROM idempotency_keys
WHERE username = $1
  AND operation = $2
  AND idempotency_key = $3
FOR UPDATE;

-- name: CompleteIdempotencyKey :one
UPDATE idempotency_keys
SET
  transaction_id = $4,
  response_status = $5,
  result_snapshot = $6
WHERE username = $1
  AND operation = $2
  AND idempotency_key = $3
  AND transaction_id IS NULL
RETURNING *;

-- name: CountIdempotencyKeysByTransaction :one
SELECT count(*) FROM idempotency_keys
WHERE transaction_id = $1;

-- name: DeleteExpiredIdempotencyKeys :execrows
DELETE FROM idempotency_keys
WHERE expires_at <= now();
