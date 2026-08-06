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

-- name: ListAuditLogsByEntity :many
SELECT * FROM audit_logs
WHERE entity_type = $1
  AND entity_id = $2
ORDER BY created_at, id;

-- name: ListAccountFinancialAuditLogs :many
SELECT DISTINCT audit.*
FROM audit_logs AS audit
LEFT JOIN ledger_postings AS posting
  ON audit.entity_type = 'banking_transaction'
 AND posting.transaction_id = audit.entity_id
LEFT JOIN ledger_accounts AS ledger
  ON ledger.id = posting.ledger_account_id
LEFT JOIN accounts AS account
  ON account.id = ledger.customer_account_id
  WHERE (
    audit.entity_type = 'account'
    AND audit.entity_id = sqlc.arg(account_public_id)
  )
  OR account.public_id = sqlc.arg(account_public_id)
  OR (
    audit.entity_type = 'transfer_attempt'
    AND (
      audit.metadata->>'from_account_id' = sqlc.arg(account_public_id)::text
      OR audit.metadata->>'to_account_id' = sqlc.arg(account_public_id)::text
    )
  )
ORDER BY audit.created_at, audit.id;

-- name: ListRecentAuditLogs :many
SELECT * FROM audit_logs
ORDER BY created_at DESC, id DESC
LIMIT $1;
