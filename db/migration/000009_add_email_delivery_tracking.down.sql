DROP TABLE IF EXISTS email_delivery_events;

DROP INDEX IF EXISTS email_jobs_provider_message_id_idx;

ALTER TABLE email_jobs
  DROP CONSTRAINT IF EXISTS email_jobs_delivery_status_check,
  DROP COLUMN IF EXISTS bounce_message,
  DROP COLUMN IF EXISTS bounce_subtype,
  DROP COLUMN IF EXISTS bounce_type,
  DROP COLUMN IF EXISTS bounced_at,
  DROP COLUMN IF EXISTS delivered_at,
  DROP COLUMN IF EXISTS accepted_at,
  DROP COLUMN IF EXISTS delivery_event_at,
  DROP COLUMN IF EXISTS delivery_status;

ALTER TABLE users
  DROP CONSTRAINT IF EXISTS users_email_deliverability_status_check,
  DROP COLUMN IF EXISTS email_bounced_at,
  DROP COLUMN IF EXISTS email_deliverability_updated_at,
  DROP COLUMN IF EXISTS email_deliverability_status,
  DROP COLUMN IF EXISTS email_verified_at;
