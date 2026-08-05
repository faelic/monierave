ALTER TABLE users
  ADD COLUMN account_status varchar NOT NULL DEFAULT 'active',
  ADD COLUMN registration_expires_at timestamptz,
  ADD CONSTRAINT users_account_status_check
    CHECK (account_status IN ('pending', 'active', 'disabled'));

-- Preserve access for legacy accounts that existed before verification was enforced.
UPDATE users
SET email_verified_at = COALESCE(email_verified_at, created_at)
WHERE email_deliverability_status IN ('unknown', 'deliverable');

UPDATE users
SET
  account_status = 'pending',
  registration_expires_at = now() + interval '7 days'
WHERE email_verified_at IS NULL;

ALTER TABLE email_delivery_events
  DROP CONSTRAINT email_delivery_events_email_job_id_fkey,
  ADD CONSTRAINT email_delivery_events_email_job_id_fkey
    FOREIGN KEY (email_job_id) REFERENCES email_jobs (id) ON DELETE SET NULL;

COMMENT ON COLUMN users.account_status IS
  'pending accounts may only manage email verification; active accounts may use financial features; disabled registrations exceeded the verification window.';
