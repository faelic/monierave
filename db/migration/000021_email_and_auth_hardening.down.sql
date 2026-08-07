ALTER TABLE users
  DROP CONSTRAINT IF EXISTS users_username_length_check,
  DROP CONSTRAINT IF EXISTS users_full_name_length_check,
  DROP CONSTRAINT IF EXISTS users_email_length_check;

DROP TRIGGER email_delivery_events_append_only ON email_delivery_events;

UPDATE email_delivery_events AS event
SET email_job_id = NULL
WHERE email_job_id IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM email_jobs AS job
    WHERE job.id = event.email_job_id
  );

ALTER TABLE email_delivery_events
  ADD CONSTRAINT email_delivery_events_email_job_id_fkey
    FOREIGN KEY (email_job_id) REFERENCES email_jobs (id) ON DELETE SET NULL;

CREATE TRIGGER email_delivery_events_append_only
BEFORE UPDATE OR DELETE ON email_delivery_events
FOR EACH ROW
EXECUTE FUNCTION prevent_email_delivery_event_mutation();

COMMENT ON COLUMN email_delivery_events.email_job_id IS NULL;
