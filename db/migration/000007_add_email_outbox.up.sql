CREATE TABLE email_jobs (
  id uuid PRIMARY KEY,
  parent_job_id uuid,
  job_type varchar NOT NULL,
  username varchar NOT NULL,
  recipient varchar NOT NULL,
  payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  status varchar NOT NULL DEFAULT 'pending',
  attempt_count integer NOT NULL DEFAULT 0,
  max_attempts integer NOT NULL DEFAULT 10,
  provider_message_id varchar,
  last_error text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  queued_at timestamptz,
  processing_at timestamptz,
  last_attempt_at timestamptz,
  sent_at timestamptz,
  dead_lettered_at timestamptz,
  CONSTRAINT email_jobs_username_fkey
    FOREIGN KEY (username) REFERENCES users (username),
  CONSTRAINT email_jobs_parent_job_id_fkey
    FOREIGN KEY (parent_job_id) REFERENCES email_jobs (id),
  CONSTRAINT email_jobs_status_check
    CHECK (status IN ('pending', 'queued', 'processing', 'retrying', 'sent', 'dead_letter')),
  CONSTRAINT email_jobs_attempt_count_check
    CHECK (attempt_count >= 0),
  CONSTRAINT email_jobs_max_attempts_check
    CHECK (max_attempts > 0)
);

CREATE INDEX email_jobs_status_created_at_idx
  ON email_jobs (status, created_at);

CREATE INDEX email_jobs_username_idx
  ON email_jobs (username);

CREATE INDEX email_jobs_parent_job_id_idx
  ON email_jobs (parent_job_id)
  WHERE parent_job_id IS NOT NULL;

CREATE TABLE outbox_events (
  id uuid PRIMARY KEY,
  email_job_id uuid NOT NULL UNIQUE,
  event_type varchar NOT NULL,
  payload jsonb NOT NULL,
  status varchar NOT NULL DEFAULT 'pending',
  publish_attempts integer NOT NULL DEFAULT 0,
  available_at timestamptz NOT NULL DEFAULT now(),
  claimed_by varchar,
  claimed_until timestamptz,
  last_error text,
  created_at timestamptz NOT NULL DEFAULT now(),
  published_at timestamptz,
  CONSTRAINT outbox_events_email_job_id_fkey
    FOREIGN KEY (email_job_id) REFERENCES email_jobs (id),
  CONSTRAINT outbox_events_status_check
    CHECK (status IN ('pending', 'publishing', 'published')),
  CONSTRAINT outbox_events_publish_attempts_check
    CHECK (publish_attempts >= 0)
);

CREATE INDEX outbox_events_claimable_idx
  ON outbox_events (available_at, created_at)
  WHERE status IN ('pending', 'publishing');
