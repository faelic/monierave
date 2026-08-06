DROP TRIGGER IF EXISTS ledger_postings_balanced ON ledger_postings;
DROP TRIGGER IF EXISTS banking_transactions_balanced ON banking_transactions;
DROP TRIGGER IF EXISTS banking_transactions_protected_truncate ON banking_transactions;
DROP TRIGGER IF EXISTS banking_transactions_protected_rows ON banking_transactions;
DROP TRIGGER IF EXISTS ledger_postings_pending_transaction ON ledger_postings;
DROP TRIGGER IF EXISTS ledger_postings_immutable ON ledger_postings;

DROP FUNCTION IF EXISTS validate_banking_transaction_balance();
DROP FUNCTION IF EXISTS protect_banking_transaction();
DROP FUNCTION IF EXISTS enforce_pending_transaction_posting();
DROP FUNCTION IF EXISTS prevent_ledger_posting_mutation();

DROP TABLE IF EXISTS ledger_postings;
DROP TABLE IF EXISTS banking_transactions;
DROP TABLE IF EXISTS ledger_accounts;

CREATE TABLE entries (
  id bigserial PRIMARY KEY,
  account_id bigint NOT NULL,
  amount bigint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT entries_account_id_fkey
    FOREIGN KEY (account_id) REFERENCES accounts (id) DEFERRABLE INITIALLY IMMEDIATE
);

CREATE INDEX entries_account_id_idx ON entries (account_id);

CREATE TABLE transfers (
  id bigserial PRIMARY KEY,
  from_account_id bigint NOT NULL,
  to_account_id bigint NOT NULL,
  amount bigint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT transfers_from_account_id_fkey
    FOREIGN KEY (from_account_id) REFERENCES accounts (id) DEFERRABLE INITIALLY IMMEDIATE,
  CONSTRAINT transfers_to_account_id_fkey
    FOREIGN KEY (to_account_id) REFERENCES accounts (id) DEFERRABLE INITIALLY IMMEDIATE,
  CONSTRAINT transfers_amount_positive CHECK (amount > 0)
);

CREATE INDEX transfers_from_account_id_idx ON transfers (from_account_id);
CREATE INDEX transfers_to_account_id_idx ON transfers (to_account_id);
CREATE INDEX transfers_from_to_account_id_idx
  ON transfers (from_account_id, to_account_id);
