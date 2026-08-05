CREATE FUNCTION prevent_email_delivery_event_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  RAISE EXCEPTION 'email delivery events are append-only';
END;
$$;

CREATE TRIGGER email_delivery_events_append_only
BEFORE UPDATE OR DELETE ON email_delivery_events
FOR EACH ROW
EXECUTE FUNCTION prevent_email_delivery_event_mutation();
