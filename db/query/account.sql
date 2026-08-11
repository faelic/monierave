-- name: CreateAccount :one
INSERT INTO accounts (
  owner,
  balance,
  currency,
  account_number
) VALUES (
  $1, 0, $2, $3
)
RETURNING *;

-- name: GetAccount :one
SELECT * FROM accounts
WHERE id = $1 LIMIT 1;

-- name: GetAccountByPublicID :one
SELECT * FROM accounts
WHERE public_id = $1 LIMIT 1;

-- name: GetAccountByAccountNumber :one
SELECT * FROM accounts
WHERE account_number = $1 LIMIT 1;

-- name: ResolveReceivableAccount :one
SELECT
  account.account_number,
  owner.full_name AS account_name,
  account.currency,
  account.status
FROM accounts AS account
JOIN users AS owner
  ON owner.username = account.owner
WHERE account.account_number = $1
  AND account.status IN ('active', 'frozen')
LIMIT 1;

-- name: GetOwnedAccountByPublicID :one
SELECT * FROM accounts
WHERE public_id = $1
  AND owner = $2
LIMIT 1;

-- name: GetAccountForUpdate :one
SELECT * FROM accounts
WHERE id = $1 LIMIT 1
FOR NO KEY UPDATE;

-- name: GetAccountByPublicIDForUpdate :one
SELECT * FROM accounts
WHERE public_id = $1 LIMIT 1
FOR NO KEY UPDATE;

-- name: ListAccount :many
SELECT * FROM accounts
WHERE owner = $1
ORDER BY created_at, id
LIMIT $2
OFFSET $3;

-- name: AddAccountBalanceInternal :one
UPDATE accounts
SET
  balance = balance + sqlc.arg(amount),
  updated_at = now()
WHERE id = sqlc.arg(id)
  AND balance + sqlc.arg(amount) >= 0
RETURNING *;

-- name: CloseAccount :one
UPDATE accounts
SET
  status = 'closed',
  closed_at = now(),
  updated_at = now()
WHERE id = $1
  AND balance = 0
  AND status <> 'closed'
RETURNING *;

-- name: SetAccountStatus :one
UPDATE accounts
SET
  status = $2,
  closed_at = NULL,
  updated_at = now()
WHERE id = $1
  AND $2 IN ('active', 'frozen')
RETURNING *;
