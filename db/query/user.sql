-- name: CreateUser :one
INSERT INTO users (
  username,
  hashed_password,
  full_name,
  email,
  email_deliverability_status,
  account_status,
  registration_expires_at
) VALUES (
  $1, $2, $3, $4, 'pending', 'pending', now() + interval '7 days'
)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE username = $1 LIMIT 1;

-- name: GetUserForUpdate :one
SELECT * FROM users
WHERE username = $1
LIMIT 1
FOR UPDATE;

-- name: UpdateUser :one
UPDATE users
SET
  hashed_password = COALESCE(sqlc.narg(hashed_password), hashed_password),
  full_name = COALESCE(sqlc.narg(full_name), full_name),
  email = COALESCE(sqlc.narg(email), email),
  email_verified_at = CASE
    WHEN sqlc.narg(email) IS NOT NULL
      AND lower(sqlc.narg(email)::varchar) <> lower(email)
      THEN NULL
    ELSE email_verified_at
  END,
  email_deliverability_status = CASE
    WHEN sqlc.narg(email) IS NOT NULL
      AND lower(sqlc.narg(email)::varchar) <> lower(email)
      THEN 'pending'
    ELSE email_deliverability_status
  END,
  email_deliverability_updated_at = CASE
    WHEN sqlc.narg(email) IS NOT NULL
      AND lower(sqlc.narg(email)::varchar) <> lower(email)
      THEN NULL
    ELSE email_deliverability_updated_at
  END,
  email_bounced_at = CASE
    WHEN sqlc.narg(email) IS NOT NULL
      AND lower(sqlc.narg(email)::varchar) <> lower(email)
      THEN NULL
    ELSE email_bounced_at
  END,
  account_status = CASE
    WHEN sqlc.narg(email) IS NOT NULL
      AND lower(sqlc.narg(email)::varchar) <> lower(email)
      THEN 'pending'
    ELSE account_status
  END,
  registration_expires_at = CASE
    WHEN sqlc.narg(email) IS NOT NULL
      AND lower(sqlc.narg(email)::varchar) <> lower(email)
      THEN now() + interval '7 days'
    ELSE registration_expires_at
  END,
  password_changed_at = CASE
    WHEN sqlc.narg(hashed_password) IS NOT NULL THEN now()
    ELSE password_changed_at
  END
WHERE username = sqlc.arg(username)
RETURNING *;

-- name: UpdateCurrentUserEmailDeliverability :one
UPDATE users
SET
  email_deliverability_status = sqlc.arg(deliverability_status),
  email_deliverability_updated_at = sqlc.arg(occurred_at),
  email_bounced_at = CASE
    WHEN sqlc.narg(bounced_at)::timestamptz IS NOT NULL
      THEN sqlc.narg(bounced_at)
    WHEN sqlc.arg(deliverability_status)::varchar = 'deliverable'
      THEN NULL
    ELSE email_bounced_at
  END,
  email_verified_at = CASE
    WHEN sqlc.arg(deliverability_status)::varchar = 'undeliverable'
      THEN NULL
    ELSE email_verified_at
  END,
  account_status = CASE
    WHEN sqlc.arg(deliverability_status)::varchar = 'undeliverable'
      THEN 'pending'
    ELSE account_status
  END,
  registration_expires_at = CASE
    WHEN sqlc.arg(deliverability_status)::varchar = 'undeliverable'
      THEN now() + interval '7 days'
    ELSE registration_expires_at
  END
WHERE username = sqlc.arg(username)
  AND lower(email) = lower(sqlc.arg(recipient))
  AND (
    email_deliverability_updated_at IS NULL
    OR email_deliverability_updated_at < sqlc.arg(occurred_at)
  )
  AND (
    email_deliverability_status <> 'undeliverable'
    OR sqlc.arg(deliverability_status)::varchar = 'undeliverable'
  )
RETURNING *;

-- name: MarkUserEmailVerified :one
UPDATE users
SET
  email_verified_at = COALESCE(email_verified_at, now()),
  email_deliverability_status = 'deliverable',
  email_deliverability_updated_at = now(),
  email_bounced_at = NULL,
  account_status = 'active',
  registration_expires_at = NULL
WHERE username = sqlc.arg(username)
  AND lower(email) = lower(sqlc.arg(email))
RETURNING *;

-- name: RestartUserEmailVerification :one
UPDATE users
SET
  account_status = 'pending',
  registration_expires_at = CASE
    WHEN account_status = 'disabled' THEN now() + interval '7 days'
    ELSE registration_expires_at
  END,
  email_verified_at = NULL,
  email_deliverability_status = 'pending',
  email_deliverability_updated_at = NULL,
  email_bounced_at = NULL
WHERE username = $1
RETURNING *;

-- name: DisableExpiredPendingUser :one
UPDATE users
SET account_status = 'disabled'
WHERE username = $1
  AND account_status = 'pending'
  AND registration_expires_at <= now()
RETURNING *;

-- name: DisableExpiredPendingUsers :execrows
UPDATE users
SET account_status = 'disabled'
WHERE account_status = 'pending'
  AND registration_expires_at <= now();
