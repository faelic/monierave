DROP TRIGGER IF EXISTS audit_logs_immutable_trigger ON audit_logs;
DROP FUNCTION IF EXISTS prevent_audit_log_mutation();

DROP TRIGGER IF EXISTS outbox_events_audit_trigger ON outbox_events;
DROP TRIGGER IF EXISTS email_jobs_audit_trigger ON email_jobs;
DROP FUNCTION IF EXISTS audit_outbox_event_changes();
DROP FUNCTION IF EXISTS audit_email_job_changes();

DROP TABLE IF EXISTS audit_logs;
