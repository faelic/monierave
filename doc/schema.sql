-- SQL dump generated using DBML (dbml.dbdiagram.io)
-- Database: PostgreSQL
-- Generated at: 2026-08-04T16:29:50.768Z

CREATE TYPE "Currency" AS ENUM (
  'USD',
  'EUR'
);

CREATE TABLE "users" (
  "username" varchar PRIMARY KEY,
  "hashed_password" varchar NOT NULL,
  "full_name" varchar NOT NULL,
  "email" varchar NOT NULL,
  "password_changed_at" timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00Z',
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "accounts" (
  "id" bigserial PRIMARY KEY,
  "owner" varchar NOT NULL,
  "balance" bigint NOT NULL,
  "currency" varchar NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  CHECK (balance >= 0)
);

CREATE TABLE "entries" (
  "id" bigserial PRIMARY KEY,
  "account_id" bigint NOT NULL,
  "amount" bigint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "transfers" (
  "id" bigserial PRIMARY KEY,
  "from_account_id" bigint NOT NULL,
  "to_account_id" bigint NOT NULL,
  "amount" bigint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  CHECK (amount > 0)
);

CREATE TABLE "sessions" (
  "id" uuid PRIMARY KEY,
  "username" varchar NOT NULL,
  "refresh_token" varchar NOT NULL,
  "user_agent" varchar NOT NULL,
  "client_ip" varchar NOT NULL,
  "is_blocked" boolean NOT NULL DEFAULT false,
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
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
  "dead_lettered_at" timestamptz
);

CREATE TABLE "outbox_events" (
  "id" uuid PRIMARY KEY,
  "email_job_id" uuid UNIQUE NOT NULL,
  "event_type" varchar NOT NULL,
  "payload" jsonb NOT NULL,
  "status" varchar NOT NULL DEFAULT 'pending',
  "publish_attempts" integer NOT NULL DEFAULT 0,
  "available_at" timestamptz NOT NULL DEFAULT (now()),
  "claimed_by" varchar,
  "claimed_until" timestamptz,
  "last_error" text,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "published_at" timestamptz
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

CREATE UNIQUE INDEX ON "users" ((lower(email)));

CREATE INDEX ON "accounts" ("owner");

CREATE UNIQUE INDEX ON "accounts" ("owner", "currency");

CREATE INDEX ON "entries" ("account_id");

CREATE INDEX ON "transfers" ("from_account_id");

CREATE INDEX ON "transfers" ("to_account_id");

CREATE INDEX ON "transfers" ("from_account_id", "to_account_id");

CREATE INDEX ON "sessions" ("username");

CREATE UNIQUE INDEX ON "sessions" ("refresh_token");

CREATE INDEX ON "email_jobs" ("status", "created_at");

CREATE INDEX ON "email_jobs" ("username");

CREATE INDEX ON "email_jobs" ("parent_job_id");

CREATE INDEX ON "outbox_events" ("available_at", "created_at");

CREATE INDEX ON "audit_logs" ("entity_type", "entity_id", "created_at", "id");

CREATE INDEX ON "audit_logs" ("correlation_id", "created_at", "id");

CREATE INDEX ON "audit_logs" ("created_at", "id");

COMMENT ON COLUMN "entries"."amount" IS 'can be negative or positive';

COMMENT ON COLUMN "transfers"."amount" IS 'must be positive';

COMMENT ON TABLE "audit_logs" IS 'Append-only audit history. No foreign keys so records survive operational data cleanup.';

ALTER TABLE "accounts" ADD FOREIGN KEY ("owner") REFERENCES "users" ("username") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "entries" ADD FOREIGN KEY ("account_id") REFERENCES "accounts" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "transfers" ADD FOREIGN KEY ("from_account_id") REFERENCES "accounts" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "transfers" ADD FOREIGN KEY ("to_account_id") REFERENCES "accounts" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "sessions" ADD FOREIGN KEY ("username") REFERENCES "users" ("username") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "email_jobs" ADD FOREIGN KEY ("parent_job_id") REFERENCES "email_jobs" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "email_jobs" ADD FOREIGN KEY ("username") REFERENCES "users" ("username") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "outbox_events" ADD FOREIGN KEY ("email_job_id") REFERENCES "email_jobs" ("id") DEFERRABLE INITIALLY IMMEDIATE;
