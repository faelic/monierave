DROP TRIGGER IF EXISTS email_delivery_events_append_only ON email_delivery_events;
DROP FUNCTION IF EXISTS prevent_email_delivery_event_mutation();
