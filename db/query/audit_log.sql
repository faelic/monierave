-- name: CreateAuditLog :one
INSERT INTO audit_logs (
  entity_type,
  entity_id,
  correlation_id,
  event_type,
  actor,
  from_state,
  to_state,
  message,
  metadata
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: ListAuditLogsByJob :many
SELECT * FROM audit_logs
WHERE entity_id = $1
   OR correlation_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2;

-- name: ListRecentAuditLogs :many
SELECT * FROM audit_logs
ORDER BY created_at DESC, id DESC
LIMIT $1;
