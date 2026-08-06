DROP INDEX IF EXISTS outbox_events_type_created_idx;
DROP INDEX IF EXISTS outbox_events_entity_idx;
DROP INDEX IF EXISTS outbox_events_correlation_idx;

DROP TRIGGER IF EXISTS outbox_events_audit_trigger ON outbox_events;
DROP FUNCTION IF EXISTS audit_domain_outbox_event_changes();

CREATE TRIGGER outbox_events_audit_trigger
AFTER INSERT OR UPDATE ON outbox_events
FOR EACH ROW
EXECUTE FUNCTION audit_outbox_event_changes();

ALTER TABLE outbox_events
DROP COLUMN IF EXISTS entity_id,
DROP COLUMN IF EXISTS entity_type,
DROP COLUMN IF EXISTS correlation_id;
