DROP INDEX IF EXISTS accounts_owner_status_idx;

ALTER TABLE accounts
  ALTER COLUMN balance DROP DEFAULT,
  DROP CONSTRAINT IF EXISTS accounts_closed_at_check,
  DROP CONSTRAINT IF EXISTS accounts_status_check,
  DROP CONSTRAINT IF EXISTS accounts_public_id_key,
  DROP COLUMN IF EXISTS closed_at,
  DROP COLUMN IF EXISTS updated_at,
  DROP COLUMN IF EXISTS status,
  DROP COLUMN IF EXISTS public_id;
