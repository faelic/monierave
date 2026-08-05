ALTER TABLE email_delivery_events
  DROP CONSTRAINT email_delivery_events_email_job_id_fkey,
  ADD CONSTRAINT email_delivery_events_email_job_id_fkey
    FOREIGN KEY (email_job_id) REFERENCES email_jobs (id);

ALTER TABLE users
  DROP CONSTRAINT IF EXISTS users_account_status_check,
  DROP COLUMN IF EXISTS registration_expires_at,
  DROP COLUMN IF EXISTS account_status;
