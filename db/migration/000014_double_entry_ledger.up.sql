CREATE TABLE ledger_accounts (
  id bigserial PRIMARY KEY,
  public_id uuid NOT NULL DEFAULT gen_random_uuid(),
  customer_account_id bigint,
  code varchar,
  kind varchar NOT NULL,
  currency varchar NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ledger_accounts_public_id_key UNIQUE (public_id),
  CONSTRAINT ledger_accounts_customer_account_id_key UNIQUE (customer_account_id),
  CONSTRAINT ledger_accounts_code_key UNIQUE (code),
  CONSTRAINT ledger_accounts_customer_account_id_fkey
    FOREIGN KEY (customer_account_id) REFERENCES accounts (id),
  CONSTRAINT ledger_accounts_kind_check
    CHECK (kind IN ('customer', 'settlement')),
  CONSTRAINT ledger_accounts_shape_check
    CHECK (
      (kind = 'customer' AND customer_account_id IS NOT NULL AND code IS NULL)
      OR (kind = 'settlement' AND customer_account_id IS NULL AND code IS NOT NULL)
    )
);

CREATE INDEX ledger_accounts_kind_currency_idx
  ON ledger_accounts (kind, currency, id);

CREATE TABLE banking_transactions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  reference varchar NOT NULL UNIQUE,
  transaction_type varchar NOT NULL,
  status varchar NOT NULL DEFAULT 'pending',
  currency varchar NOT NULL,
  amount bigint NOT NULL,
  narration varchar NOT NULL DEFAULT '',
  initiated_by varchar NOT NULL,
  reversal_of uuid UNIQUE,
  created_at timestamptz NOT NULL DEFAULT now(),
  posted_at timestamptz,
  failed_at timestamptz,
  reversed_at timestamptz,
  CONSTRAINT banking_transactions_reversal_of_fkey
    FOREIGN KEY (reversal_of) REFERENCES banking_transactions (id),
  CONSTRAINT banking_transactions_type_check
    CHECK (transaction_type IN ('deposit', 'withdrawal', 'internal_transfer', 'reversal')),
  CONSTRAINT banking_transactions_status_check
    CHECK (status IN ('pending', 'posted', 'failed', 'reversed')),
  CONSTRAINT banking_transactions_amount_positive
    CHECK (amount > 0),
  CONSTRAINT banking_transactions_status_timestamp_check
    CHECK (
      (status = 'pending' AND posted_at IS NULL AND failed_at IS NULL AND reversed_at IS NULL)
      OR (status = 'posted' AND posted_at IS NOT NULL AND failed_at IS NULL AND reversed_at IS NULL)
      OR (status = 'failed' AND posted_at IS NULL AND failed_at IS NOT NULL AND reversed_at IS NULL)
      OR (status = 'reversed' AND posted_at IS NOT NULL AND failed_at IS NULL AND reversed_at IS NOT NULL)
    )
);

CREATE INDEX banking_transactions_created_at_idx
  ON banking_transactions (created_at DESC, id);

CREATE INDEX banking_transactions_type_status_idx
  ON banking_transactions (transaction_type, status, created_at DESC);

CREATE TABLE ledger_postings (
  id bigserial PRIMARY KEY,
  transaction_id uuid NOT NULL,
  ledger_account_id bigint NOT NULL,
  amount bigint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ledger_postings_transaction_id_fkey
    FOREIGN KEY (transaction_id) REFERENCES banking_transactions (id),
  CONSTRAINT ledger_postings_ledger_account_id_fkey
    FOREIGN KEY (ledger_account_id) REFERENCES ledger_accounts (id),
  CONSTRAINT ledger_postings_transaction_account_key
    UNIQUE (transaction_id, ledger_account_id),
  CONSTRAINT ledger_postings_amount_nonzero
    CHECK (amount <> 0)
);

CREATE INDEX ledger_postings_account_created_idx
  ON ledger_postings (ledger_account_id, created_at, id);

INSERT INTO ledger_accounts (code, kind, currency)
SELECT 'settlement:' || currency, 'settlement', currency
FROM (VALUES ('USD'), ('EUR')) AS supported(currency);

INSERT INTO ledger_accounts (customer_account_id, kind, currency)
SELECT id, 'customer', currency
FROM accounts;

CREATE TEMP TABLE account_opening_transactions
ON COMMIT DROP
AS
SELECT
  account.id AS account_id,
  gen_random_uuid() AS transaction_id,
  account.balance,
  account.currency,
  account.owner
FROM accounts AS account
WHERE account.balance > 0;

INSERT INTO banking_transactions (
  id,
  reference,
  transaction_type,
  status,
  currency,
  amount,
  narration,
  initiated_by,
  posted_at
)
SELECT
  transaction_id,
  'MIG-' || upper(replace(transaction_id::text, '-', '')),
  'deposit',
  'posted',
  currency,
  balance,
  'Opening balance migrated from the legacy account model',
  'migration',
  now()
FROM account_opening_transactions;

INSERT INTO ledger_postings (transaction_id, ledger_account_id, amount)
SELECT opening.transaction_id, customer_ledger.id, opening.balance
FROM account_opening_transactions AS opening
JOIN ledger_accounts AS customer_ledger
  ON customer_ledger.customer_account_id = opening.account_id
UNION ALL
SELECT opening.transaction_id, settlement_ledger.id, -opening.balance
FROM account_opening_transactions AS opening
JOIN ledger_accounts AS settlement_ledger
  ON settlement_ledger.code = 'settlement:' || opening.currency;

CREATE FUNCTION prevent_ledger_posting_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  RAISE EXCEPTION 'ledger_postings is append-only; % is not allowed', TG_OP;
END;
$$;

CREATE TRIGGER ledger_postings_immutable
BEFORE UPDATE OR DELETE OR TRUNCATE ON ledger_postings
FOR EACH STATEMENT
EXECUTE FUNCTION prevent_ledger_posting_mutation();

CREATE FUNCTION enforce_pending_transaction_posting()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  transaction_status varchar;
BEGIN
  SELECT status INTO transaction_status
  FROM banking_transactions
  WHERE id = NEW.transaction_id;

  IF transaction_status IS DISTINCT FROM 'pending' THEN
    RAISE EXCEPTION 'postings may only be added to pending transactions';
  END IF;
  RETURN NEW;
END;
$$;

CREATE TRIGGER ledger_postings_pending_transaction
BEFORE INSERT ON ledger_postings
FOR EACH ROW
EXECUTE FUNCTION enforce_pending_transaction_posting();

CREATE FUNCTION protect_banking_transaction()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF TG_OP IN ('DELETE', 'TRUNCATE') THEN
    RAISE EXCEPTION 'banking_transactions is append-only; % is not allowed', TG_OP;
  END IF;

  IF NEW.id IS DISTINCT FROM OLD.id
    OR NEW.reference IS DISTINCT FROM OLD.reference
    OR NEW.transaction_type IS DISTINCT FROM OLD.transaction_type
    OR NEW.currency IS DISTINCT FROM OLD.currency
    OR NEW.amount IS DISTINCT FROM OLD.amount
    OR NEW.narration IS DISTINCT FROM OLD.narration
    OR NEW.initiated_by IS DISTINCT FROM OLD.initiated_by
    OR NEW.reversal_of IS DISTINCT FROM OLD.reversal_of
    OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
    RAISE EXCEPTION 'banking transaction financial details are immutable';
  END IF;

  IF OLD.status = 'pending' AND NEW.status = 'posted'
    AND OLD.posted_at IS NULL AND NEW.posted_at IS NOT NULL
    AND NEW.failed_at IS NULL AND NEW.reversed_at IS NULL THEN
    RETURN NEW;
  END IF;

  IF OLD.status = 'pending' AND NEW.status = 'failed'
    AND OLD.failed_at IS NULL AND NEW.failed_at IS NOT NULL
    AND NEW.posted_at IS NULL AND NEW.reversed_at IS NULL THEN
    RETURN NEW;
  END IF;

  IF OLD.status = 'posted' AND NEW.status = 'reversed'
    AND OLD.reversed_at IS NULL AND NEW.reversed_at IS NOT NULL
    AND NEW.posted_at IS NOT NULL AND NEW.failed_at IS NULL THEN
    RETURN NEW;
  END IF;

  RAISE EXCEPTION 'invalid banking transaction state transition from % to %',
    OLD.status, NEW.status;
END;
$$;

CREATE TRIGGER banking_transactions_protected_rows
BEFORE UPDATE OR DELETE ON banking_transactions
FOR EACH ROW
EXECUTE FUNCTION protect_banking_transaction();

CREATE TRIGGER banking_transactions_protected_truncate
BEFORE TRUNCATE ON banking_transactions
FOR EACH STATEMENT
EXECUTE FUNCTION protect_banking_transaction();

CREATE FUNCTION validate_banking_transaction_balance()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  target_transaction_id uuid;
  transaction_status varchar;
  transaction_currency varchar;
  transaction_amount bigint;
  posting_count bigint;
  posting_total bigint;
  posting_absolute_total bigint;
  currency_mismatch_count bigint;
BEGIN
  IF TG_TABLE_NAME = 'banking_transactions' THEN
    target_transaction_id := NEW.id;
  ELSE
    target_transaction_id := NEW.transaction_id;
  END IF;

  SELECT status, currency, amount
  INTO transaction_status, transaction_currency, transaction_amount
  FROM banking_transactions
  WHERE id = target_transaction_id;

  IF transaction_status IN ('posted', 'reversed') THEN
    SELECT
      count(*),
      coalesce(sum(posting.amount), 0),
      coalesce(sum(abs(posting.amount)), 0),
      count(*) FILTER (WHERE ledger.currency <> transaction_currency)
    INTO
      posting_count,
      posting_total,
      posting_absolute_total,
      currency_mismatch_count
    FROM ledger_postings AS posting
    JOIN ledger_accounts AS ledger ON ledger.id = posting.ledger_account_id
    WHERE posting.transaction_id = target_transaction_id;

    IF posting_count < 2
      OR posting_total <> 0
      OR posting_absolute_total <> transaction_amount * 2
      OR currency_mismatch_count <> 0 THEN
      RAISE EXCEPTION
        'invalid ledger postings for transaction %: count %, total %, absolute total %, currency mismatches %',
        target_transaction_id,
        posting_count,
        posting_total,
        posting_absolute_total,
        currency_mismatch_count;
    END IF;
  ELSIF transaction_status = 'failed' THEN
    SELECT count(*) INTO posting_count
    FROM ledger_postings
    WHERE transaction_id = target_transaction_id;
    IF posting_count <> 0 THEN
      RAISE EXCEPTION 'failed transaction % cannot retain postings',
        target_transaction_id;
    END IF;
  END IF;

  RETURN NEW;
END;
$$;

CREATE CONSTRAINT TRIGGER banking_transactions_balanced
AFTER INSERT OR UPDATE ON banking_transactions
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION validate_banking_transaction_balance();

CREATE CONSTRAINT TRIGGER ledger_postings_balanced
AFTER INSERT ON ledger_postings
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION validate_banking_transaction_balance();

DROP TABLE entries;
DROP TABLE transfers;

COMMENT ON TABLE ledger_accounts IS
  'Customer and system accounts used by the double-entry ledger.';

COMMENT ON TABLE banking_transactions IS
  'Business-level financial events whose details are immutable after creation.';

COMMENT ON TABLE ledger_postings IS
  'Append-only signed movements. Every posted transaction must sum to zero.';
