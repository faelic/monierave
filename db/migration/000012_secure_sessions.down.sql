DELETE FROM sessions;

DROP INDEX IF EXISTS sessions_active_username_idx;
DROP INDEX IF EXISTS sessions_refresh_token_hash_idx;

ALTER TABLE sessions
  DROP COLUMN revoked_reason,
  DROP COLUMN revoked_at,
  DROP COLUMN last_refreshed_at,
  DROP COLUMN refresh_token_id,
  DROP COLUMN refresh_token_hash,
  ADD COLUMN refresh_token varchar NOT NULL,
  ADD COLUMN is_blocked boolean NOT NULL DEFAULT false;

CREATE UNIQUE INDEX sessions_refresh_token_idx
  ON sessions (refresh_token);
