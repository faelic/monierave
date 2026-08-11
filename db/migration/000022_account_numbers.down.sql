COMMENT ON COLUMN accounts.balance IS NULL;

ALTER TABLE accounts
  DROP CONSTRAINT IF EXISTS accounts_account_number_format_check,
  DROP CONSTRAINT IF EXISTS accounts_account_number_key,
  DROP COLUMN IF EXISTS account_number;
