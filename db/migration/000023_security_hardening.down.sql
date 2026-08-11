DROP INDEX IF EXISTS email_jobs_verification_token_hash_idx;

ALTER TABLE email_jobs
  DROP CONSTRAINT IF EXISTS email_jobs_verification_token_pair_check,
  DROP CONSTRAINT IF EXISTS email_jobs_verification_token_hash_length_check,
  DROP COLUMN IF EXISTS verification_token_expires_at,
  DROP COLUMN IF EXISTS verification_token_hash;
