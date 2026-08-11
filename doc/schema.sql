-- SQL dump generated using DBML (dbml.dbdiagram.io)
-- Database: PostgreSQL
-- Generated at: 2026-08-05T15:31:07.488Z

CREATE TYPE "Currency" AS ENUM (
  'USD',
  'EUR'
);

CREATE TABLE "users" (
  "username" varchar PRIMARY KEY,
  "hashed_password" varchar NOT NULL,
  "full_name" varchar NOT NULL,
  "email" varchar NOT NULL,
  "password_changed_at" timestamptz NOT NULL DEFAULT (TIMESTAMPTZ '0001-01-01 00:00:00Z'),
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "email_verified_at" timestamptz,
  "email_deliverability_status" varchar NOT NULL DEFAULT 'unknown',
  "email_deliverability_updated_at" timestamptz,
  "email_bounced_at" timestamptz,
  "account_status" varchar NOT NULL DEFAULT 'active',
  "registration_expires_at" timestamptz,
  CHECK (char_length(username) BETWEEN 3 AND 32),
  CHECK (char_length(btrim(full_name)) BETWEEN 1 AND 100),
  CHECK (char_length(email) BETWEEN 3 AND 254),
  CHECK (email_deliverability_status IN ('unknown', 'pending', 'deliverable', 'undeliverable')),
  CHECK (account_status IN ('pending', 'active', 'disabled'))
);

CREATE TABLE "accounts" (
  "id" bigserial PRIMARY KEY,
  "public_id" uuid UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "account_number" varchar(10) UNIQUE NOT NULL,
  "owner" varchar NOT NULL,
  "balance" bigint NOT NULL DEFAULT 0,
  "currency" varchar NOT NULL,
  "status" varchar NOT NULL DEFAULT 'active',
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now()),
  "closed_at" timestamptz,
  CHECK (account_number ~ '^[0-9]{10}$'),
  CHECK (balance >= 0),
  CHECK (status IN ('active', 'frozen', 'closed')),
  CHECK ((status = 'closed' AND closed_at IS NOT NULL AND balance = 0) OR (status <> 'closed' AND closed_at IS NULL))
);

CREATE TABLE "ledger_accounts" (
  "id" bigserial PRIMARY KEY,
  "public_id" uuid UNIQUE NOT NULL DEFAULT (gen_random_uuid()),
  "customer_account_id" bigint UNIQUE,
  "code" varchar UNIQUE,
  "kind" varchar NOT NULL,
  "currency" varchar NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  CHECK (kind IN ('customer', 'settlement')),
  CHECK ((kind = 'customer' AND customer_account_id IS NOT NULL AND code IS NULL) OR (kind = 'settlement' AND customer_account_id IS NULL AND code IS NOT NULL))
);

CREATE TABLE "banking_transactions" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "reference" varchar UNIQUE NOT NULL,
  "transaction_type" varchar NOT NULL,
  "status" varchar NOT NULL DEFAULT 'pending',
  "currency" varchar NOT NULL,
  "amount" bigint NOT NULL,
  "narration" varchar NOT NULL DEFAULT '',
  "initiated_by" varchar NOT NULL,
  "reversal_of" uuid UNIQUE,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "posted_at" timestamptz,
  "failed_at" timestamptz,
  "reversed_at" timestamptz,
  CHECK (transaction_type IN ('deposit', 'withdrawal', 'internal_transfer', 'reversal')),
  CHECK (status IN ('pending', 'posted', 'failed', 'reversed')),
  CHECK (amount > 0),
  CHECK ((transaction_type = 'reversal' AND reversal_of IS NOT NULL AND reversal_of <> id) OR (transaction_type <> 'reversal' AND reversal_of IS NULL)),
  CHECK ((status = 'pending' AND posted_at IS NULL AND failed_at IS NULL AND reversed_at IS NULL) OR (status = 'posted' AND posted_at IS NOT NULL AND failed_at IS NULL AND reversed_at IS NULL) OR (status = 'failed' AND posted_at IS NULL AND failed_at IS NOT NULL AND reversed_at IS NULL) OR (status = 'reversed' AND posted_at IS NOT NULL AND failed_at IS NULL AND reversed_at IS NOT NULL))
);

CREATE TABLE "ledger_postings" (
  "id" bigserial PRIMARY KEY,
  "transaction_id" uuid NOT NULL,
  "ledger_account_id" bigint NOT NULL,
  "amount" bigint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  UNIQUE ("transaction_id", "ledger_account_id"),
  CHECK (amount <> 0)
);

CREATE TABLE "idempotency_keys" (
  "id" bigserial PRIMARY KEY,
  "username" varchar NOT NULL,
  "operation" varchar NOT NULL,
  "idempotency_key" varchar NOT NULL,
  "request_hash" bytea NOT NULL,
  "transaction_id" uuid,
  "response_status" smallint,
  "result_snapshot" jsonb,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "expires_at" timestamptz NOT NULL DEFAULT (now() + interval '24 hours'),
  UNIQUE ("username", "operation", "idempotency_key"),
  CHECK (operation IN ('internal_transfer')),
  CHECK (length(idempotency_key) BETWEEN 1 AND 128),
  CHECK (octet_length(request_hash) = 32),
  CHECK ((transaction_id IS NULL AND response_status IS NULL AND result_snapshot IS NULL) OR (transaction_id IS NOT NULL AND response_status IS NOT NULL AND result_snapshot IS NOT NULL)),
  CHECK (expires_at > created_at)
);

CREATE TABLE "beneficiaries" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "owner" varchar NOT NULL,
  "destination_account_id" bigint NOT NULL,
  "nickname" varchar NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now()),
  UNIQUE ("owner", "destination_account_id"),
  CHECK (length(nickname) BETWEEN 1 AND 50 AND nickname = btrim(nickname))
);

CREATE TABLE "sessions" (
  "id" uuid PRIMARY KEY,
  "username" varchar NOT NULL,
  "refresh_token_hash" bytea NOT NULL,
  "refresh_token_id" uuid NOT NULL,
  "device_token_hash" bytea NOT NULL,
  "user_agent" varchar NOT NULL,
  "client_ip" varchar NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "last_refreshed_at" timestamptz,
  "revoked_at" timestamptz,
  "revoked_reason" varchar
);

CREATE TABLE "email_jobs" (
  "id" uuid PRIMARY KEY,
  "parent_job_id" uuid,
  "job_type" varchar NOT NULL,
  "username" varchar NOT NULL,
  "recipient" varchar NOT NULL,
  "payload" jsonb NOT NULL DEFAULT ('{}'::jsonb),
  "status" varchar NOT NULL DEFAULT 'pending',
  "attempt_count" integer NOT NULL DEFAULT 0,
  "max_attempts" integer NOT NULL DEFAULT 10,
  "provider_message_id" varchar,
  "last_error" text,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now()),
  "queued_at" timestamptz,
  "processing_at" timestamptz,
  "last_attempt_at" timestamptz,
  "sent_at" timestamptz,
  "dead_lettered_at" timestamptz,
  "delivery_status" varchar NOT NULL DEFAULT 'pending',
  "delivery_event_at" timestamptz,
  "accepted_at" timestamptz,
  "delivered_at" timestamptz,
  "bounced_at" timestamptz,
  "bounce_type" varchar,
  "bounce_subtype" varchar,
  "bounce_message" text,
  "verification_token_hash" bytea,
  "verification_token_expires_at" timestamptz,
  CHECK (status IN ('pending', 'queued', 'processing', 'retrying', 'sent', 'dead_letter')),
  CHECK (attempt_count >= 0),
  CHECK (max_attempts > 0),
  CHECK (delivery_status IN ('pending', 'accepted', 'delivered', 'delayed', 'bounced', 'failed', 'suppressed', 'complained')),
  CHECK ((verification_token_hash IS NULL) = (verification_token_expires_at IS NULL)),
  CHECK (verification_token_hash IS NULL OR octet_length(verification_token_hash) = 32)
);

CREATE TABLE "outbox_events" (
  "id" uuid PRIMARY KEY,
  "email_job_id" uuid UNIQUE NOT NULL,
  "event_type" varchar NOT NULL,
  "payload" jsonb NOT NULL,
  "correlation_id" uuid NOT NULL,
  "entity_type" varchar NOT NULL,
  "entity_id" uuid NOT NULL,
  "status" varchar NOT NULL DEFAULT 'pending',
  "publish_attempts" integer NOT NULL DEFAULT 0,
  "available_at" timestamptz NOT NULL DEFAULT (now()),
  "claimed_by" varchar,
  "claimed_until" timestamptz,
  "last_error" text,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "published_at" timestamptz,
  CHECK (status IN ('pending', 'publishing', 'published')),
  CHECK (publish_attempts >= 0)
);

CREATE TABLE "audit_logs" (
  "id" bigserial PRIMARY KEY,
  "entity_type" varchar NOT NULL,
  "entity_id" uuid NOT NULL,
  "correlation_id" uuid NOT NULL,
  "event_type" varchar NOT NULL,
  "actor" varchar NOT NULL,
  "from_state" varchar,
  "to_state" varchar,
  "message" text,
  "metadata" jsonb NOT NULL DEFAULT ('{}'::jsonb),
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "email_delivery_events" (
  "webhook_id" varchar PRIMARY KEY,
  "email_job_id" uuid,
  "provider_message_id" varchar NOT NULL,
  "event_type" varchar NOT NULL,
  "occurred_at" timestamptz NOT NULL,
  "received_at" timestamptz NOT NULL DEFAULT (now()),
  "payload" jsonb NOT NULL
);

CREATE UNIQUE INDEX "users_email_lower_idx" ON "users" ((lower(email)));

CREATE INDEX "accounts_owner_idx" ON "accounts" ("owner");

CREATE UNIQUE INDEX "accounts_owner_currency_key" ON "accounts" ("owner", "currency");

CREATE INDEX "accounts_owner_status_idx" ON "accounts" ("owner", "status", "created_at", "id");

CREATE INDEX "ledger_accounts_kind_currency_idx" ON "ledger_accounts" ("kind", "currency", "id");

CREATE INDEX "banking_transactions_created_at_idx" ON "banking_transactions" ("created_at" DESC, "id");

CREATE INDEX "banking_transactions_type_status_idx" ON "banking_transactions" ("transaction_type", "status", "created_at" DESC);

CREATE INDEX "ledger_postings_account_created_idx" ON "ledger_postings" ("ledger_account_id", "created_at", "id");

CREATE INDEX "ledger_postings_account_transaction_idx" ON "ledger_postings" ("ledger_account_id", "transaction_id");

CREATE INDEX "idempotency_keys_expires_at_idx" ON "idempotency_keys" ("expires_at");

CREATE INDEX "beneficiaries_owner_created_idx" ON "beneficiaries" ("owner", "created_at" DESC, "id" DESC);

CREATE INDEX "sessions_username_idx" ON "sessions" ("username");

CREATE UNIQUE INDEX "sessions_refresh_token_hash_idx" ON "sessions" ("refresh_token_hash");

CREATE UNIQUE INDEX "sessions_device_token_hash_idx" ON "sessions" ("device_token_hash");

CREATE UNIQUE INDEX "sessions_one_active_per_user_idx" ON "sessions" ("username")
WHERE "revoked_at" IS NULL;

CREATE INDEX "email_jobs_status_created_at_idx" ON "email_jobs" ("status", "created_at");

CREATE INDEX "email_jobs_username_idx" ON "email_jobs" ("username");

CREATE INDEX "email_jobs_parent_job_id_idx" ON "email_jobs" ("parent_job_id")
WHERE "parent_job_id" IS NOT NULL;

CREATE UNIQUE INDEX "email_jobs_provider_message_id_idx" ON "email_jobs" ("provider_message_id")
WHERE "provider_message_id" IS NOT NULL;

CREATE UNIQUE INDEX "email_jobs_verification_token_hash_idx" ON "email_jobs" ("verification_token_hash")
WHERE "verification_token_hash" IS NOT NULL;

CREATE INDEX "outbox_events_claimable_idx" ON "outbox_events" ("available_at", "created_at")
WHERE "status" IN ('pending', 'publishing');

CREATE INDEX "outbox_events_correlation_idx" ON "outbox_events" ("correlation_id", "created_at", "id");

CREATE INDEX "outbox_events_entity_idx" ON "outbox_events" ("entity_type", "entity_id", "created_at", "id");

CREATE INDEX "outbox_events_type_created_idx" ON "outbox_events" ("event_type", "created_at", "id");

CREATE INDEX "audit_logs_entity_idx" ON "audit_logs" ("entity_type", "entity_id", "created_at", "id");

CREATE INDEX "audit_logs_correlation_idx" ON "audit_logs" ("correlation_id", "created_at", "id");

CREATE INDEX "audit_logs_created_at_idx" ON "audit_logs" ("created_at" DESC, "id" DESC);

CREATE INDEX "email_delivery_events_job_idx" ON "email_delivery_events" ("email_job_id", "occurred_at" DESC);

CREATE INDEX "email_delivery_events_provider_message_idx" ON "email_delivery_events" ("provider_message_id", "occurred_at" DESC);

COMMENT ON TABLE "users" IS 'Application users and their email verification lifecycle.';

COMMENT ON COLUMN "accounts"."public_id" IS 'Stable, non-sequential account identifier exposed through public APIs.';

COMMENT ON COLUMN "accounts"."status" IS 'Frozen accounts may receive funds but cannot send; closed accounts cannot transact.';

COMMENT ON COLUMN "sessions"."refresh_token_hash" IS 'SHA-256 hash of the current refresh token; raw refresh tokens are never persisted.';

COMMENT ON COLUMN "sessions"."device_token_hash" IS 'SHA-256 hash of the browser-bound device secret; raw device secrets are never persisted.';

COMMENT ON COLUMN "sessions"."revoked_at" IS 'Non-null means every access and refresh token bound to this session is revoked.';

COMMENT ON COLUMN "users"."account_status" IS 'Pending accounts may only manage email verification; active accounts may use financial features; disabled registrations exceeded the verification window.';

COMMENT ON COLUMN "users"."registration_expires_at" IS 'Unverified registrations remain recoverable for seven days. Verification tokens expire independently after 24 hours.';

COMMENT ON TABLE "ledger_accounts" IS 'Customer and system accounts used by the double-entry ledger.';

COMMENT ON TABLE "banking_transactions" IS 'Business-level financial events whose details are immutable after creation.';

COMMENT ON TABLE "ledger_postings" IS 'Append-only signed movements. Every posted transaction must sum to zero.';

COMMENT ON TABLE "idempotency_keys" IS 'Short-lived deduplication records for safely retrying money-moving requests.';

COMMENT ON COLUMN "idempotency_keys"."request_hash" IS 'SHA-256 hash of the normalized request payload.';

COMMENT ON COLUMN "idempotency_keys"."result_snapshot" IS 'Original committed domain result returned for identical retries.';

COMMENT ON TABLE "beneficiaries" IS 'User-owned saved transfer destinations. Account bigint IDs remain internal.';

COMMENT ON TABLE "email_jobs" IS 'Durable email work items. Dead-letter jobs remain here with status dead_letter and may be replayed using parent_job_id.';

COMMENT ON COLUMN "email_jobs"."status" IS 'Worker execution state. Sent means the email provider accepted the API request.';

COMMENT ON COLUMN "email_jobs"."delivery_status" IS 'Recipient delivery state reported asynchronously by the email provider.';

COMMENT ON COLUMN "email_jobs"."verification_token_hash" IS 'SHA-256 hash of the opaque verification token. The raw token is never stored.';

COMMENT ON TABLE "outbox_events" IS 'Transactional outbox records committed in the same database transaction as application state.';

COMMENT ON TABLE "audit_logs" IS 'Append-only audit history. Intentionally has no foreign keys so records survive entity retention cleanup.';

COMMENT ON TABLE "email_delivery_events" IS 'Immutable verified provider events. email_job_id intentionally has no foreign key so retained history survives email-job cleanup with its original UUID.';

ALTER TABLE "accounts" ADD FOREIGN KEY ("owner") REFERENCES "users" ("username") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "ledger_accounts" ADD FOREIGN KEY ("customer_account_id") REFERENCES "accounts" ("id");

ALTER TABLE "banking_transactions" ADD FOREIGN KEY ("reversal_of") REFERENCES "banking_transactions" ("id");

ALTER TABLE "ledger_postings" ADD FOREIGN KEY ("transaction_id") REFERENCES "banking_transactions" ("id");

ALTER TABLE "ledger_postings" ADD FOREIGN KEY ("ledger_account_id") REFERENCES "ledger_accounts" ("id");

ALTER TABLE "idempotency_keys" ADD FOREIGN KEY ("username") REFERENCES "users" ("username");

ALTER TABLE "idempotency_keys" ADD FOREIGN KEY ("transaction_id") REFERENCES "banking_transactions" ("id");

ALTER TABLE "beneficiaries" ADD FOREIGN KEY ("owner") REFERENCES "users" ("username");

ALTER TABLE "beneficiaries" ADD FOREIGN KEY ("destination_account_id") REFERENCES "accounts" ("id");

ALTER TABLE "sessions" ADD FOREIGN KEY ("username") REFERENCES "users" ("username") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "email_jobs" ADD FOREIGN KEY ("parent_job_id") REFERENCES "email_jobs" ("id");

ALTER TABLE "email_jobs" ADD FOREIGN KEY ("username") REFERENCES "users" ("username");

ALTER TABLE "outbox_events" ADD FOREIGN KEY ("email_job_id") REFERENCES "email_jobs" ("id");

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
      entity_type, entity_id, correlation_id, event_type, actor, to_state, metadata
    ) VALUES (
      'email_job', NEW.id, NEW.id, audit_event, audit_actor, NEW.status, audit_metadata
    );
    RETURN NEW;
  END IF;

  IF NEW.attempt_count > OLD.attempt_count AND NEW.status = 'processing' THEN
    audit_event := 'email_attempt_started';
    audit_actor := 'worker';
    audit_metadata := jsonb_build_object(
      'attempt', NEW.attempt_count,
      'max_attempts', NEW.max_attempts
    );
  ELSIF NEW.status IS DISTINCT FROM OLD.status THEN
    audit_actor := CASE WHEN NEW.status = 'queued' THEN 'relay' ELSE 'worker' END;

    CASE NEW.status
      WHEN 'queued' THEN audit_event := 'email_job_queued';
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
      ELSE audit_event := 'email_status_changed';
    END CASE;
  END IF;

  IF audit_event IS NOT NULL THEN
    INSERT INTO audit_logs (
      entity_type, entity_id, correlation_id, event_type, actor,
      from_state, to_state, message, metadata
    ) VALUES (
      'email_job', NEW.id, NEW.id, audit_event, audit_actor,
      OLD.status, NEW.status, audit_message, audit_metadata
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
      entity_type, entity_id, correlation_id, event_type, actor, to_state, metadata
    ) VALUES (
      'outbox_event', NEW.id, NEW.email_job_id, 'outbox_event_created',
      'application', NEW.status, jsonb_build_object('event_type', NEW.event_type)
    );
    RETURN NEW;
  END IF;

  IF NEW.status = 'publishing' AND NEW.publish_attempts > OLD.publish_attempts THEN
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
        audit_metadata := jsonb_build_object('publish_attempt', NEW.publish_attempts);
      ELSE audit_event := 'outbox_status_changed';
    END CASE;
  END IF;

  IF audit_event IS NOT NULL THEN
    INSERT INTO audit_logs (
      entity_type, entity_id, correlation_id, event_type, actor,
      from_state, to_state, message, metadata
    ) VALUES (
      'outbox_event', NEW.id, NEW.email_job_id, audit_event, 'relay',
      OLD.status, NEW.status, audit_message, audit_metadata
    );
  END IF;

  RETURN NEW;
END;
$$;

CREATE FUNCTION prevent_audit_log_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  RAISE EXCEPTION 'audit_logs is append-only; % is not allowed', TG_OP;
END;
$$;

CREATE FUNCTION prevent_email_delivery_event_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  RAISE EXCEPTION 'email delivery events are append-only';
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

CREATE TRIGGER audit_logs_immutable_trigger
BEFORE UPDATE OR DELETE OR TRUNCATE ON audit_logs
FOR EACH STATEMENT
EXECUTE FUNCTION prevent_audit_log_mutation();

CREATE TRIGGER email_delivery_events_append_only
BEFORE UPDATE OR DELETE OR TRUNCATE ON email_delivery_events
FOR EACH STATEMENT
EXECUTE FUNCTION prevent_email_delivery_event_mutation();
