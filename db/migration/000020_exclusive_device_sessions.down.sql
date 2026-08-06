DELETE FROM sessions;

DROP INDEX IF EXISTS sessions_one_active_per_user_idx;
DROP INDEX IF EXISTS sessions_device_token_hash_idx;

ALTER TABLE sessions
DROP COLUMN IF EXISTS device_token_hash;

CREATE INDEX sessions_active_username_idx
  ON sessions (username, expires_at)
  WHERE revoked_at IS NULL;
