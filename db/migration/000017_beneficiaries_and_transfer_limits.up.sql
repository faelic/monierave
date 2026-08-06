CREATE TABLE beneficiaries (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  owner varchar NOT NULL,
  destination_account_id bigint NOT NULL,
  nickname varchar NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT beneficiaries_owner_fkey
    FOREIGN KEY (owner) REFERENCES users (username),
  CONSTRAINT beneficiaries_destination_account_id_fkey
    FOREIGN KEY (destination_account_id) REFERENCES accounts (id),
  CONSTRAINT beneficiaries_owner_destination_key
    UNIQUE (owner, destination_account_id),
  CONSTRAINT beneficiaries_nickname_check
    CHECK (length(nickname) BETWEEN 1 AND 50 AND nickname = btrim(nickname))
);

CREATE INDEX beneficiaries_owner_created_idx
  ON beneficiaries (owner, created_at DESC, id DESC);

COMMENT ON TABLE beneficiaries IS
  'User-owned saved transfer destinations. Account bigint IDs remain internal.';
