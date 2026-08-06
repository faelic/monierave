-- Existing sessions cannot be bound to a device secret that was never issued.
-- This project has no production users, so invalidating development sessions is safest.
DELETE FROM sessions;

DROP INDEX IF EXISTS sessions_active_username_idx;

ALTER TABLE sessions
ADD COLUMN device_token_hash bytea NOT NULL;

CREATE UNIQUE INDEX sessions_device_token_hash_idx
  ON sessions (device_token_hash);

CREATE UNIQUE INDEX sessions_one_active_per_user_idx
  ON sessions (username)
  WHERE revoked_at IS NULL;

COMMENT ON COLUMN sessions.device_token_hash IS
  'SHA-256 hash of the browser-bound device secret; raw device secrets are never persisted.';

COMMENT ON INDEX sessions_one_active_per_user_idx IS
  'Enforces the strict policy that each user has at most one unrevoked session.';
