ALTER TABLE outbox_events
ADD COLUMN correlation_id uuid,
ADD COLUMN entity_type varchar,
ADD COLUMN entity_id uuid;

UPDATE outbox_events
SET
  correlation_id = email_job_id,
  entity_type = 'email_job',
  entity_id = email_job_id;

ALTER TABLE outbox_events
ALTER COLUMN correlation_id SET NOT NULL,
ALTER COLUMN entity_type SET NOT NULL,
ALTER COLUMN entity_id SET NOT NULL;

CREATE INDEX outbox_events_correlation_idx
  ON outbox_events (correlation_id, created_at, id);

CREATE INDEX outbox_events_entity_idx
  ON outbox_events (entity_type, entity_id, created_at, id);

CREATE INDEX outbox_events_type_created_idx
  ON outbox_events (event_type, created_at, id);

DROP TRIGGER outbox_events_audit_trigger ON outbox_events;

CREATE FUNCTION audit_domain_outbox_event_changes()
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
      NEW.correlation_id,
      'outbox_event_created',
      'application',
      NEW.status,
      jsonb_build_object(
        'event_type', NEW.event_type,
        'email_job_id', NEW.email_job_id,
        'source_entity_type', NEW.entity_type,
        'source_entity_id', NEW.entity_id
      )
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
      NEW.correlation_id,
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

CREATE TRIGGER outbox_events_audit_trigger
AFTER INSERT OR UPDATE ON outbox_events
FOR EACH ROW
EXECUTE FUNCTION audit_domain_outbox_event_changes();

COMMENT ON COLUMN outbox_events.id IS
  'Unique domain-event identifier and Asynq publication source.';

COMMENT ON COLUMN outbox_events.correlation_id IS
  'Connects the originating request, financial audit, notification event, and delivery logs.';

COMMENT ON COLUMN outbox_events.entity_type IS
  'Type of aggregate that emitted the event, such as account or banking_transaction.';

COMMENT ON COLUMN outbox_events.entity_id IS
  'Public UUID of the aggregate that emitted the event.';
