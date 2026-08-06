-- name: ListAccountReconciliationIssues :many
SELECT
  account.public_id AS account_public_id,
  account.currency,
  account.balance AS cached_balance,
  customer_ledger.id AS customer_ledger_account_id,
  coalesce(sum(posting.amount), 0)::bigint AS ledger_balance
FROM accounts AS account
LEFT JOIN ledger_accounts AS customer_ledger
  ON customer_ledger.customer_account_id = account.id
 AND customer_ledger.kind = 'customer'
LEFT JOIN ledger_postings AS posting
  ON posting.ledger_account_id = customer_ledger.id
WHERE sqlc.narg(account_public_id)::uuid IS NULL
   OR account.public_id = sqlc.narg(account_public_id)
GROUP BY
  account.id,
  account.public_id,
  account.currency,
  account.balance,
  customer_ledger.id
HAVING customer_ledger.id IS NULL
    OR account.balance <> coalesce(sum(posting.amount), 0)::bigint
ORDER BY account.public_id;

-- name: ListTransactionReconciliationIssues :many
SELECT
  transaction.id AS transaction_id,
  transaction.reference,
  transaction.status,
  count(posting.id)::bigint AS posting_count,
  coalesce(sum(posting.amount), 0)::bigint AS posting_total
FROM banking_transactions AS transaction
LEFT JOIN ledger_postings AS posting
  ON posting.transaction_id = transaction.id
WHERE transaction.status IN ('posted', 'reversed')
  AND (
    sqlc.narg(customer_account_id)::bigint IS NULL
    OR EXISTS (
      SELECT 1
      FROM ledger_postings AS scoped_posting
      JOIN ledger_accounts AS scoped_ledger
        ON scoped_ledger.id = scoped_posting.ledger_account_id
      WHERE scoped_posting.transaction_id = transaction.id
        AND scoped_ledger.customer_account_id = sqlc.narg(customer_account_id)
    )
  )
GROUP BY
  transaction.id,
  transaction.reference,
  transaction.status
HAVING count(posting.id) < 2
    OR coalesce(sum(posting.amount), 0)::bigint <> 0
ORDER BY transaction.created_at, transaction.id;

-- name: ListDuplicateReversalIssues :many
SELECT
  original.id AS original_transaction_id,
  original.reference AS original_reference,
  count(reversal.id)::bigint AS reversal_count
FROM banking_transactions AS original
JOIN banking_transactions AS reversal
  ON reversal.reversal_of = original.id
WHERE sqlc.narg(customer_account_id)::bigint IS NULL
   OR EXISTS (
     SELECT 1
     FROM ledger_postings AS scoped_posting
     JOIN ledger_accounts AS scoped_ledger
       ON scoped_ledger.id = scoped_posting.ledger_account_id
     WHERE scoped_posting.transaction_id = original.id
       AND scoped_ledger.customer_account_id = sqlc.narg(customer_account_id)
   )
GROUP BY original.id, original.reference
HAVING count(reversal.id) > 1
ORDER BY original.created_at, original.id;
