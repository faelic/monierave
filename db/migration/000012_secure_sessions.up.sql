-- Existing refresh tokens are stored in plaintext and cannot be migrated safely.
-- Invalidating development sessions is safer than preserving those credentials.
DELETE FROM sessions;

ALTER TABLE sessions
  DROP COLUMN refresh_token,
  DROP COLUMN is_blocked,
  ADD COLUMN refresh_token_hash bytea NOT NULL,
  ADD COLUMN refresh_token_id uuid NOT NULL,
  ADD COLUMN last_refreshed_at timestamptz,
  ADD COLUMN revoked_at timestamptz,
  ADD COLUMN revoked_reason varchar;

CREATE UNIQUE INDEX sessions_refresh_token_hash_idx
  ON sessions (refresh_token_hash);

CREATE INDEX sessions_active_username_idx
  ON sessions (username, expires_at)
  WHERE revoked_at IS NULL;

COMMENT ON COLUMN sessions.refresh_token_hash IS
  'SHA-256 hash of the current refresh token; raw refresh tokens are never persisted.';

COMMENT ON COLUMN sessions.revoked_at IS
  'Non-null means every access and refresh token bound to this session is revoked.';
