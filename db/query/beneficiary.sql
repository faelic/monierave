-- name: CreateBeneficiary :one
INSERT INTO beneficiaries (
  owner,
  destination_account_id,
  nickname
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: ListOwnedBeneficiaries :many
SELECT
  beneficiary.id,
  beneficiary.nickname,
  account.public_id AS destination_account_public_id,
  account.currency,
  account.status AS destination_account_status,
  beneficiary.created_at,
  beneficiary.updated_at
FROM beneficiaries AS beneficiary
JOIN accounts AS account
  ON account.id = beneficiary.destination_account_id
WHERE beneficiary.owner = $1
ORDER BY beneficiary.created_at DESC, beneficiary.id DESC
LIMIT $2
OFFSET $3;

-- name: UpdateOwnedBeneficiaryNickname :one
WITH updated AS (
  UPDATE beneficiaries
  SET
    nickname = sqlc.arg(nickname),
    updated_at = now()
  WHERE beneficiaries.id = sqlc.arg(beneficiary_id)
    AND beneficiaries.owner = sqlc.arg(owner)
  RETURNING *
)
SELECT
  updated.id,
  updated.nickname,
  account.public_id AS destination_account_public_id,
  account.currency,
  account.status AS destination_account_status,
  updated.created_at,
  updated.updated_at
FROM updated
JOIN accounts AS account
  ON account.id = updated.destination_account_id;

-- name: DeleteOwnedBeneficiary :execrows
DELETE FROM beneficiaries
WHERE id = sqlc.arg(beneficiary_id)
  AND owner = sqlc.arg(owner);
