ALTER TABLE email_delivery_events
  DROP CONSTRAINT email_delivery_events_email_job_id_fkey;

DROP TRIGGER email_delivery_events_append_only ON email_delivery_events;

CREATE TRIGGER email_delivery_events_append_only
BEFORE UPDATE OR DELETE OR TRUNCATE ON email_delivery_events
FOR EACH STATEMENT
EXECUTE FUNCTION prevent_email_delivery_event_mutation();

ALTER TABLE users
  ADD CONSTRAINT users_username_length_check
    CHECK (char_length(username) BETWEEN 3 AND 32),
  ADD CONSTRAINT users_full_name_length_check
    CHECK (char_length(btrim(full_name)) BETWEEN 1 AND 100),
  ADD CONSTRAINT users_email_length_check
    CHECK (char_length(email) BETWEEN 3 AND 254);

COMMENT ON COLUMN email_delivery_events.email_job_id IS
  'Immutable historical job UUID. No foreign key is used so retained events survive email-job cleanup without mutation.';

