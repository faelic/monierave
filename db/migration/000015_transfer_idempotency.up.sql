CREATE TABLE idempotency_keys (
  id bigserial PRIMARY KEY,
  username varchar NOT NULL,
  operation varchar NOT NULL,
  idempotency_key varchar NOT NULL,
  request_hash bytea NOT NULL,
  transaction_id uuid,
  response_status smallint,
  result_snapshot jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz NOT NULL DEFAULT (now() + interval '24 hours'),
  CONSTRAINT idempotency_keys_username_fkey
    FOREIGN KEY (username) REFERENCES users (username),
  CONSTRAINT idempotency_keys_transaction_id_fkey
    FOREIGN KEY (transaction_id) REFERENCES banking_transactions (id),
  CONSTRAINT idempotency_keys_owner_operation_key
    UNIQUE (username, operation, idempotency_key),
  CONSTRAINT idempotency_keys_operation_check
    CHECK (operation IN ('internal_transfer')),
  CONSTRAINT idempotency_keys_key_length_check
    CHECK (length(idempotency_key) BETWEEN 1 AND 128),
  CONSTRAINT idempotency_keys_request_hash_length_check
    CHECK (octet_length(request_hash) = 32),
  CONSTRAINT idempotency_keys_completion_check
    CHECK (
      (transaction_id IS NULL AND response_status IS NULL AND result_snapshot IS NULL)
      OR
      (transaction_id IS NOT NULL AND response_status IS NOT NULL AND result_snapshot IS NOT NULL)
    ),
  CONSTRAINT idempotency_keys_expiry_check
    CHECK (expires_at > created_at)
);

CREATE INDEX idempotency_keys_expires_at_idx
  ON idempotency_keys (expires_at);

COMMENT ON TABLE idempotency_keys IS
  'Short-lived deduplication records for safely retrying money-moving requests.';

COMMENT ON COLUMN idempotency_keys.request_hash IS
  'SHA-256 hash of the normalized request payload; prevents reuse with different instructions.';

COMMENT ON COLUMN idempotency_keys.result_snapshot IS
  'Original committed domain result returned for identical retries.';
