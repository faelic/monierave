-- name: CreateCustomerLedgerAccount :one
INSERT INTO ledger_accounts (
  customer_account_id,
  kind,
  currency
) VALUES (
  $1, 'customer', $2
)
RETURNING *;

-- name: GetCustomerLedgerAccount :one
SELECT * FROM ledger_accounts
WHERE customer_account_id = $1
  AND kind = 'customer'
LIMIT 1;

-- name: GetSettlementLedgerAccount :one
SELECT * FROM ledger_accounts
WHERE code = 'settlement:' || sqlc.arg(currency)
  AND kind = 'settlement'
LIMIT 1;

-- name: CreateBankingTransaction :one
INSERT INTO banking_transactions (
  id,
  reference,
  transaction_type,
  currency,
  amount,
  narration,
  initiated_by,
  reversal_of
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: MarkBankingTransactionPosted :one
UPDATE banking_transactions
SET
  status = 'posted',
  posted_at = now()
WHERE id = $1
  AND status = 'pending'
RETURNING *;

-- name: GetBankingTransaction :one
SELECT * FROM banking_transactions
WHERE id = $1
LIMIT 1;

-- name: GetBankingTransactionByReference :one
SELECT * FROM banking_transactions
WHERE reference = $1
LIMIT 1;

-- name: CreateLedgerPosting :one
INSERT INTO ledger_postings (
  transaction_id,
  ledger_account_id,
  amount
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: ListLedgerPostingsByTransaction :many
SELECT * FROM ledger_postings
WHERE transaction_id = $1
ORDER BY id;

-- name: GetLedgerPostingTotal :one
SELECT coalesce(sum(amount), 0)::bigint
FROM ledger_postings
WHERE transaction_id = $1;

-- name: GetLedgerAccountBalance :one
SELECT coalesce(sum(amount), 0)::bigint
FROM ledger_postings
WHERE ledger_account_id = $1;
