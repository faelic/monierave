-- name: CreateSession :one
INSERT INTO sessions (
  id,
  username,
  refresh_token_hash,
  refresh_token_id,
  user_agent,
  client_ip,
  expires_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetSession :one
SELECT * FROM sessions
WHERE id = $1
LIMIT 1;

-- name: GetSessionForUpdate :one
SELECT * FROM sessions
WHERE id = $1
LIMIT 1
FOR UPDATE;

-- name: RotateSessionRefreshToken :one
UPDATE sessions
SET
  refresh_token_hash = sqlc.arg(refresh_token_hash),
  refresh_token_id = sqlc.arg(refresh_token_id),
  last_refreshed_at = now()
WHERE id = $1
RETURNING *;

-- name: RevokeSession :one
UPDATE sessions
SET
  revoked_at = COALESCE(revoked_at, now()),
  revoked_reason = COALESCE(revoked_reason, sqlc.arg(revoked_reason))
WHERE id = sqlc.arg(id)
  AND revoked_at IS NULL
RETURNING *;

-- name: RevokeAllUserSessions :many
UPDATE sessions
SET
  revoked_at = COALESCE(revoked_at, now()),
  revoked_reason = COALESCE(revoked_reason, sqlc.arg(revoked_reason))
WHERE username = sqlc.arg(username)
  AND revoked_at IS NULL
RETURNING *;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE id = $1;

-- name: ListSessions :many
SELECT * FROM sessions
WHERE username = $1
ORDER BY created_at DESC;
