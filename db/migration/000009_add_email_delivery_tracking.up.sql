ALTER TABLE users
  ADD COLUMN email_verified_at timestamptz,
  ADD COLUMN email_deliverability_status varchar NOT NULL DEFAULT 'unknown',
  ADD COLUMN email_deliverability_updated_at timestamptz,
  ADD COLUMN email_bounced_at timestamptz,
  ADD CONSTRAINT users_email_deliverability_status_check
    CHECK (email_deliverability_status IN ('unknown', 'pending', 'deliverable', 'undeliverable'));

ALTER TABLE email_jobs
  ADD COLUMN delivery_status varchar NOT NULL DEFAULT 'pending',
  ADD COLUMN delivery_event_at timestamptz,
  ADD COLUMN accepted_at timestamptz,
  ADD COLUMN delivered_at timestamptz,
  ADD COLUMN bounced_at timestamptz,
  ADD COLUMN bounce_type varchar,
  ADD COLUMN bounce_subtype varchar,
  ADD COLUMN bounce_message text,
  ADD CONSTRAINT email_jobs_delivery_status_check
    CHECK (
      delivery_status IN (
        'pending',
        'accepted',
        'delivered',
        'delayed',
        'bounced',
        'failed',
        'suppressed',
        'complained'
      )
    );

UPDATE email_jobs
SET
  delivery_status = 'accepted',
  accepted_at = sent_at
WHERE status = 'sent';

CREATE UNIQUE INDEX email_jobs_provider_message_id_idx
  ON email_jobs (provider_message_id)
  WHERE provider_message_id IS NOT NULL;

CREATE TABLE email_delivery_events (
  webhook_id varchar PRIMARY KEY,
  email_job_id uuid,
  provider_message_id varchar NOT NULL,
  event_type varchar NOT NULL,
  occurred_at timestamptz NOT NULL,
  received_at timestamptz NOT NULL DEFAULT now(),
  payload jsonb NOT NULL,
  CONSTRAINT email_delivery_events_email_job_id_fkey
    FOREIGN KEY (email_job_id) REFERENCES email_jobs (id)
);

CREATE INDEX email_delivery_events_job_idx
  ON email_delivery_events (email_job_id, occurred_at DESC);

CREATE INDEX email_delivery_events_provider_message_idx
  ON email_delivery_events (provider_message_id, occurred_at DESC);

COMMENT ON COLUMN email_jobs.status IS
  'Worker execution state. sent means the email provider accepted the API request.';

COMMENT ON COLUMN email_jobs.delivery_status IS
  'Recipient delivery state reported asynchronously by the email provider.';

COMMENT ON TABLE email_delivery_events IS
  'Idempotent append-only record of verified email provider webhook events.';
