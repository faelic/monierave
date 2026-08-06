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

-- name: GetBankingTransactionForUpdate :one
SELECT * FROM banking_transactions
WHERE id = $1
LIMIT 1
FOR UPDATE;

-- name: MarkBankingTransactionReversed :one
UPDATE banking_transactions
SET
  status = 'reversed',
  reversed_at = now()
WHERE id = $1
  AND status = 'posted'
RETURNING *;

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

-- name: ListFinancialAuditPostings :many
SELECT
  posting.amount,
  ledger.kind,
  ledger.code,
  ledger.currency,
  account.public_id AS account_public_id,
  posting.created_at
FROM ledger_postings AS posting
JOIN ledger_accounts AS ledger
  ON ledger.id = posting.ledger_account_id
LEFT JOIN accounts AS account
  ON account.id = ledger.customer_account_id
WHERE posting.transaction_id = $1
ORDER BY posting.id;

-- name: ListReversalSourcePostings :many
SELECT
  posting.ledger_account_id,
  posting.amount,
  ledger.kind,
  ledger.customer_account_id
FROM ledger_postings AS posting
JOIN ledger_accounts AS ledger
  ON ledger.id = posting.ledger_account_id
WHERE posting.transaction_id = $1
ORDER BY posting.id;

-- name: GetLedgerPostingTotal :one
SELECT coalesce(sum(amount), 0)::bigint
FROM ledger_postings
WHERE transaction_id = $1;

-- name: GetLedgerAccountBalance :one
SELECT coalesce(sum(amount), 0)::bigint
FROM ledger_postings
WHERE ledger_account_id = $1;

-- name: GetDailyOutgoingTransferTotal :one
SELECT coalesce(sum(transaction.amount), 0)::bigint
FROM ledger_postings AS posting
JOIN banking_transactions AS transaction
  ON transaction.id = posting.transaction_id
WHERE posting.ledger_account_id = $1
  AND posting.amount < 0
  AND transaction.transaction_type = 'internal_transfer'
  AND transaction.status IN ('posted', 'reversed')
  AND transaction.created_at >= date_trunc(
    'day',
    now() AT TIME ZONE 'UTC'
  ) AT TIME ZONE 'UTC';
