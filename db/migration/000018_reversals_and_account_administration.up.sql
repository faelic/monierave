ALTER TABLE banking_transactions
ADD CONSTRAINT banking_transactions_reversal_shape_check
CHECK (
  (
    transaction_type = 'reversal'
    AND reversal_of IS NOT NULL
    AND reversal_of <> id
  )
  OR
  (
    transaction_type <> 'reversal'
    AND reversal_of IS NULL
  )
);

COMMENT ON CONSTRAINT banking_transactions_reversal_shape_check
ON banking_transactions IS
  'Only reversal transactions may reference an original transaction.';
