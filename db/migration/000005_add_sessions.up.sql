CREATE TABLE "sessions" (
  "id" uuid PRIMARY KEY,
  "username" varchar NOT NULL,
  "refresh_token" varchar NOT NULL,
  "user_agent" varchar NOT NULL,
  "client_ip" varchar NOT NULL,
  "is_blocked" boolean NOT NULL DEFAULT false,
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE "sessions"
ADD CONSTRAINT sessions_username_fkey
FOREIGN KEY ("username")
REFERENCES "users" ("username")
DEFERRABLE INITIALLY IMMEDIATE;

CREATE INDEX sessions_username_idx ON "sessions" ("username");
CREATE UNIQUE INDEX sessions_refresh_token_idx ON "sessions" ("refresh_token");