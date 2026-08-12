-- name: GetOwnedTransactionByReference :one
WITH owned_activity AS (
  SELECT
    transaction.id,
    transaction.reference,
    transaction.transaction_type,
    transaction.status,
    transaction.currency,
    transaction.amount,
    transaction.narration,
    transaction.created_at,
    transaction.posted_at,
    account.public_id AS account_public_id,
    posting.amount AS signed_amount,
    CASE WHEN posting.amount > 0 THEN 'incoming' ELSE 'outgoing' END AS direction,
    coalesce(counterparty.kind, '') AS counterparty_kind,
    CASE
      WHEN counterparty.kind = 'customer' THEN counterparty.account_number
      WHEN counterparty.kind = 'settlement' THEN 'Monierave'
      ELSE ''
    END AS counterparty
  FROM banking_transactions AS transaction
  JOIN ledger_postings AS posting
    ON posting.transaction_id = transaction.id
  JOIN ledger_accounts AS ledger
    ON ledger.id = posting.ledger_account_id
  JOIN accounts AS account
    ON account.id = ledger.customer_account_id
   AND account.owner = sqlc.arg(username)
  LEFT JOIN LATERAL (
    SELECT other_ledger.kind, other_account.account_number
    FROM ledger_postings AS other_posting
    JOIN ledger_accounts AS other_ledger
      ON other_ledger.id = other_posting.ledger_account_id
    LEFT JOIN accounts AS other_account
      ON other_account.id = other_ledger.customer_account_id
    WHERE other_posting.transaction_id = transaction.id
      AND other_posting.ledger_account_id <> posting.ledger_account_id
    ORDER BY other_posting.id
    LIMIT 1
  ) AS counterparty ON true
  WHERE transaction.reference = sqlc.arg(reference)
)
SELECT * FROM owned_activity
LIMIT 1;

-- name: ListOwnedAccountTransactions :many
WITH target_ledger AS (
  SELECT ledger.id
  FROM accounts AS account
  JOIN ledger_accounts AS ledger
    ON ledger.customer_account_id = account.id
   AND ledger.kind = 'customer'
  WHERE account.public_id = sqlc.arg(account_public_id)
    AND account.owner = sqlc.arg(username)
),
account_activity AS (
  SELECT
    transaction.id,
    transaction.reference,
    transaction.transaction_type,
    transaction.status,
    transaction.currency,
    transaction.amount,
    transaction.narration,
    transaction.created_at,
    transaction.posted_at,
    posting.amount AS signed_amount,
    CASE WHEN posting.amount > 0 THEN 'incoming' ELSE 'outgoing' END AS direction,
    coalesce(counterparty.kind, '') AS counterparty_kind,
    CASE
      WHEN counterparty.kind = 'customer' THEN counterparty.account_number
      WHEN counterparty.kind = 'settlement' THEN 'Monierave'
      ELSE ''
    END AS counterparty,
    sum(posting.amount) OVER (
      ORDER BY transaction.created_at, transaction.id
      ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
    )::bigint AS balance_after
  FROM target_ledger
  JOIN ledger_postings AS posting
    ON posting.ledger_account_id = target_ledger.id
  JOIN banking_transactions AS transaction
    ON transaction.id = posting.transaction_id
  LEFT JOIN LATERAL (
    SELECT other_ledger.kind, other_account.account_number
    FROM ledger_postings AS other_posting
    JOIN ledger_accounts AS other_ledger
      ON other_ledger.id = other_posting.ledger_account_id
    LEFT JOIN accounts AS other_account
      ON other_account.id = other_ledger.customer_account_id
    WHERE other_posting.transaction_id = transaction.id
      AND other_posting.ledger_account_id <> posting.ledger_account_id
    ORDER BY other_posting.id
    LIMIT 1
  ) AS counterparty ON true
),
filtered_activity AS (
  SELECT *
  FROM account_activity
  WHERE (sqlc.narg(from_time)::timestamptz IS NULL
      OR created_at >= sqlc.narg(from_time))
    AND (sqlc.narg(to_time)::timestamptz IS NULL
      OR created_at < sqlc.narg(to_time))
    AND (sqlc.narg(transaction_type)::varchar IS NULL
      OR transaction_type = sqlc.narg(transaction_type))
    AND (sqlc.narg(transaction_status)::varchar IS NULL
      OR status = sqlc.narg(transaction_status))
    AND (sqlc.narg(direction)::varchar IS NULL
      OR direction = sqlc.narg(direction))
    AND (
      sqlc.narg(cursor_created_at)::timestamptz IS NULL
      OR (created_at, id) < (
        sqlc.narg(cursor_created_at),
        sqlc.narg(cursor_id)::uuid
      )
    )
)
SELECT *
FROM filtered_activity
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_limit);

-- name: GetOwnedAccountStatementBalances :one
WITH target_ledger AS (
  SELECT ledger.id
  FROM accounts AS account
  JOIN ledger_accounts AS ledger
    ON ledger.customer_account_id = account.id
   AND ledger.kind = 'customer'
  WHERE account.public_id = sqlc.arg(account_public_id)
    AND account.owner = sqlc.arg(username)
)
SELECT
  coalesce(sum(posting.amount) FILTER (
    WHERE sqlc.narg(from_time)::timestamptz IS NOT NULL
      AND transaction.created_at < sqlc.narg(from_time)
  ), 0)::bigint AS opening_balance,
  coalesce(sum(posting.amount) FILTER (
    WHERE sqlc.narg(to_time)::timestamptz IS NULL
      OR transaction.created_at < sqlc.narg(to_time)
  ), 0)::bigint AS closing_balance
FROM target_ledger
JOIN ledger_postings AS posting
  ON posting.ledger_account_id = target_ledger.id
JOIN banking_transactions AS transaction
  ON transaction.id = posting.transaction_id;

-- name: GetOwnedAccountMoneyMovement :many
WITH target_ledger AS (
  SELECT ledger.id
  FROM accounts AS account
  JOIN ledger_accounts AS ledger
    ON ledger.customer_account_id = account.id
   AND ledger.kind = 'customer'
  WHERE account.public_id = sqlc.arg(account_public_id)
    AND account.owner = sqlc.arg(username)
),
periods AS (
  SELECT generate_series(
    CASE
      WHEN sqlc.arg(bucket_interval)::text = 'week'
        THEN date_trunc('week', sqlc.arg(from_time)::timestamptz)
      ELSE date_trunc('day', sqlc.arg(from_time)::timestamptz)
    END,
    CASE
      WHEN sqlc.arg(bucket_interval)::text = 'week'
        THEN date_trunc('week', sqlc.arg(to_time)::timestamptz - interval '1 microsecond')
      ELSE date_trunc('day', sqlc.arg(to_time)::timestamptz - interval '1 microsecond')
    END,
    CASE
      WHEN sqlc.arg(bucket_interval)::text = 'week' THEN interval '1 week'
      ELSE interval '1 day'
    END
  )::timestamptz AS bucket_start
),
movement AS (
  SELECT
    (CASE
      WHEN sqlc.arg(bucket_interval)::text = 'week'
        THEN date_trunc('week', transaction.posted_at)
      ELSE date_trunc('day', transaction.posted_at)
    END)::timestamptz AS bucket_start,
    posting.amount
  FROM target_ledger
  JOIN ledger_postings AS posting
    ON posting.ledger_account_id = target_ledger.id
  JOIN banking_transactions AS transaction
    ON transaction.id = posting.transaction_id
  WHERE transaction.status IN ('posted', 'reversed')
    AND transaction.posted_at >= sqlc.arg(from_time)
    AND transaction.posted_at < sqlc.arg(to_time)
)
SELECT
  periods.bucket_start,
  coalesce(sum(movement.amount) FILTER (WHERE movement.amount > 0), 0)::bigint AS incoming,
  coalesce(abs(sum(movement.amount) FILTER (WHERE movement.amount < 0)), 0)::bigint AS outgoing
FROM periods
LEFT JOIN movement USING (bucket_start)
GROUP BY periods.bucket_start
ORDER BY periods.bucket_start;
