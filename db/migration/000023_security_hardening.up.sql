ALTER TABLE email_jobs
  ADD COLUMN verification_token_hash bytea,
  ADD COLUMN verification_token_expires_at timestamptz,
  ADD CONSTRAINT email_jobs_verification_token_hash_length_check
    CHECK (
      verification_token_hash IS NULL
      OR octet_length(verification_token_hash) = 32
    ),
  ADD CONSTRAINT email_jobs_verification_token_pair_check
    CHECK (
      (verification_token_hash IS NULL) =
      (verification_token_expires_at IS NULL)
    );

CREATE UNIQUE INDEX email_jobs_verification_token_hash_idx
  ON email_jobs (verification_token_hash)
  WHERE verification_token_hash IS NOT NULL;

COMMENT ON COLUMN email_jobs.verification_token_hash IS
  'SHA-256 hash of the opaque email-verification token. Raw tokens are never persisted.';

COMMENT ON COLUMN email_jobs.verification_token_expires_at IS
  'Server-side expiration for the opaque email-verification token.';
