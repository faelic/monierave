ALTER TABLE accounts
  ADD COLUMN account_number varchar(10),
  ADD CONSTRAINT accounts_account_number_key UNIQUE (account_number);

DO $$
DECLARE
  account_row record;
  candidate varchar(10);
BEGIN
  FOR account_row IN SELECT id FROM accounts ORDER BY id LOOP
    LOOP
      candidate := substring(
        regexp_replace(
          gen_random_uuid()::text || gen_random_uuid()::text,
          '[^0-9]',
          '',
          'g'
        )
        FROM 1 FOR 10
      );

      IF char_length(candidate) <> 10 THEN
        CONTINUE;
      END IF;

      BEGIN
        UPDATE accounts
        SET account_number = candidate
        WHERE id = account_row.id;
        EXIT;
      EXCEPTION
        WHEN unique_violation THEN
          NULL;
      END;
    END LOOP;
  END LOOP;
END
$$;

ALTER TABLE accounts
  ALTER COLUMN account_number SET NOT NULL,
  ADD CONSTRAINT accounts_account_number_format_check
    CHECK (account_number ~ '^[0-9]{10}$');

COMMENT ON COLUMN accounts.account_number IS
  'Customer-facing routing identifier. It is not an authentication secret.';

COMMENT ON COLUMN accounts.balance IS
  'Current posted balance derived from posted ledger activity. Holds and card authorizations are not modeled.';
