CREATE TABLE audit_logs (
  id bigserial PRIMARY KEY,
  entity_type varchar NOT NULL,
  entity_id uuid NOT NULL,
  correlation_id uuid NOT NULL,
  event_type varchar NOT NULL,
  actor varchar NOT NULL,
  from_state varchar,
  to_state varchar,
  message text,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);

COMMENT ON TABLE audit_logs IS
  'Append-only audit history. Intentionally has no foreign keys so records survive entity retention cleanup.';

CREATE INDEX audit_logs_entity_idx
  ON audit_logs (entity_type, entity_id, created_at, id);

CREATE INDEX audit_logs_correlation_idx
  ON audit_logs (correlation_id, created_at, id);

CREATE INDEX audit_logs_created_at_idx
  ON audit_logs (created_at DESC, id DESC);

CREATE FUNCTION audit_email_job_changes()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  audit_event varchar;
  audit_actor varchar;
  audit_message text;
  audit_metadata jsonb := '{}'::jsonb;
BEGIN
  IF TG_OP = 'INSERT' THEN
    IF NEW.parent_job_id IS NULL THEN
      audit_event := 'email_job_created';
      audit_actor := 'api';
    ELSE
      audit_event := 'email_job_replay_created';
      audit_actor := 'operator_cli';
      audit_metadata := jsonb_build_object('parent_job_id', NEW.parent_job_id);
    END IF;

    INSERT INTO audit_logs (
      entity_type,
      entity_id,
      correlation_id,
      event_type,
      actor,
      to_state,
      metadata
    ) VALUES (
      'email_job',
      NEW.id,
      NEW.id,
      audit_event,
      audit_actor,
      NEW.status,
      audit_metadata
    );
    RETURN NEW;
  END IF;

  IF NEW.attempt_count > OLD.attempt_count
     AND NEW.status = 'processing' THEN
    audit_event := 'email_attempt_started';
    audit_actor := 'worker';
    audit_metadata := jsonb_build_object(
      'attempt', NEW.attempt_count,
      'max_attempts', NEW.max_attempts
    );
  ELSIF NEW.status IS DISTINCT FROM OLD.status THEN
    audit_actor := CASE
      WHEN NEW.status = 'queued' THEN 'relay'
      ELSE 'worker'
    END;

    CASE NEW.status
      WHEN 'queued' THEN
        audit_event := 'email_job_queued';
      WHEN 'retrying' THEN
        audit_event := 'email_send_failed';
        audit_message := NEW.last_error;
        audit_metadata := jsonb_build_object('attempt', NEW.attempt_count);
      WHEN 'sent' THEN
        audit_event := 'email_sent';
        audit_metadata := jsonb_build_object(
          'attempt', NEW.attempt_count,
          'provider_message_id', NEW.provider_message_id
        );
      WHEN 'dead_letter' THEN
        audit_event := 'email_dead_lettered';
        audit_message := NEW.last_error;
        audit_metadata := jsonb_build_object('attempt', NEW.attempt_count);
      ELSE
        audit_event := 'email_status_changed';
    END CASE;
  END IF;

  IF audit_event IS NOT NULL THEN
    INSERT INTO audit_logs (
      entity_type,
      entity_id,
      correlation_id,
      event_type,
      actor,
      from_state,
      to_state,
      message,
      metadata
    ) VALUES (
      'email_job',
      NEW.id,
      NEW.id,
      audit_event,
      audit_actor,
      OLD.status,
      NEW.status,
      audit_message,
      audit_metadata
    );
  END IF;

  RETURN NEW;
END;
$$;

CREATE FUNCTION audit_outbox_event_changes()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  audit_event varchar;
  audit_message text;
  audit_metadata jsonb := '{}'::jsonb;
BEGIN
  IF TG_OP = 'INSERT' THEN
    INSERT INTO audit_logs (
      entity_type,
      entity_id,
      correlation_id,
      event_type,
      actor,
      to_state,
      metadata
    ) VALUES (
      'outbox_event',
      NEW.id,
      NEW.email_job_id,
      'outbox_event_created',
      'application',
      NEW.status,
      jsonb_build_object('event_type', NEW.event_type)
    );
    RETURN NEW;
  END IF;

  IF NEW.status = 'publishing'
     AND NEW.publish_attempts > OLD.publish_attempts THEN
    audit_event := CASE
      WHEN OLD.status = 'publishing' THEN 'outbox_event_reclaimed'
      ELSE 'outbox_event_claimed'
    END;
    audit_metadata := jsonb_build_object(
      'publish_attempt', NEW.publish_attempts,
      'claimed_by', NEW.claimed_by,
      'claimed_until', NEW.claimed_until
    );
  ELSIF NEW.status IS DISTINCT FROM OLD.status THEN
    CASE NEW.status
      WHEN 'pending' THEN
        audit_event := 'outbox_publish_failed';
        audit_message := NEW.last_error;
        audit_metadata := jsonb_build_object(
          'publish_attempt', NEW.publish_attempts,
          'available_at', NEW.available_at
        );
      WHEN 'published' THEN
        audit_event := 'outbox_event_published';
        audit_metadata := jsonb_build_object(
          'publish_attempt', NEW.publish_attempts
        );
      ELSE
        audit_event := 'outbox_status_changed';
    END CASE;
  END IF;

  IF audit_event IS NOT NULL THEN
    INSERT INTO audit_logs (
      entity_type,
      entity_id,
      correlation_id,
      event_type,
      actor,
      from_state,
      to_state,
      message,
      metadata
    ) VALUES (
      'outbox_event',
      NEW.id,
      NEW.email_job_id,
      audit_event,
      'relay',
      OLD.status,
      NEW.status,
      audit_message,
      audit_metadata
    );
  END IF;

  RETURN NEW;
END;
$$;

CREATE TRIGGER email_jobs_audit_trigger
AFTER INSERT OR UPDATE ON email_jobs
FOR EACH ROW
EXECUTE FUNCTION audit_email_job_changes();

CREATE TRIGGER outbox_events_audit_trigger
AFTER INSERT OR UPDATE ON outbox_events
FOR EACH ROW
EXECUTE FUNCTION audit_outbox_event_changes();

-- Existing rows predate auditing, so record their current state without
-- pretending that their full historical transitions are known.
INSERT INTO audit_logs (
  entity_type,
  entity_id,
  correlation_id,
  event_type,
  actor,
  to_state,
  metadata,
  created_at
)
SELECT
  'email_job',
  id,
  id,
  'audit_snapshot_created',
  'migration',
  status,
  jsonb_build_object('attempt_count', attempt_count),
  now()
FROM email_jobs;

INSERT INTO audit_logs (
  entity_type,
  entity_id,
  correlation_id,
  event_type,
  actor,
  to_state,
  metadata,
  created_at
)
SELECT
  'outbox_event',
  id,
  email_job_id,
  'audit_snapshot_created',
  'migration',
  status,
  jsonb_build_object('publish_attempts', publish_attempts),
  now()
FROM outbox_events;

CREATE FUNCTION prevent_audit_log_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  RAISE EXCEPTION 'audit_logs is append-only; % is not allowed', TG_OP;
END;
$$;

CREATE TRIGGER audit_logs_immutable_trigger
BEFORE UPDATE OR DELETE OR TRUNCATE ON audit_logs
FOR EACH STATEMENT
EXECUTE FUNCTION prevent_audit_log_mutation();
