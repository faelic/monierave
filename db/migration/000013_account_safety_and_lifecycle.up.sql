ALTER TABLE accounts
  ALTER COLUMN balance SET DEFAULT 0,
  ADD COLUMN public_id uuid NOT NULL DEFAULT gen_random_uuid(),
  ADD COLUMN status varchar NOT NULL DEFAULT 'active',
  ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now(),
  ADD COLUMN closed_at timestamptz,
  ADD CONSTRAINT accounts_public_id_key UNIQUE (public_id),
  ADD CONSTRAINT accounts_status_check
    CHECK (status IN ('active', 'frozen', 'closed')),
  ADD CONSTRAINT accounts_closed_at_check
    CHECK (
      (status = 'closed' AND closed_at IS NOT NULL AND balance = 0)
      OR (status <> 'closed' AND closed_at IS NULL)
    );

CREATE INDEX accounts_owner_status_idx
  ON accounts (owner, status, created_at, id);

COMMENT ON COLUMN accounts.public_id IS
  'Stable, non-sequential account identifier exposed through public APIs.';

COMMENT ON COLUMN accounts.status IS
  'Account lifecycle state. Frozen accounts may receive funds but cannot send; closed accounts cannot transact.';
