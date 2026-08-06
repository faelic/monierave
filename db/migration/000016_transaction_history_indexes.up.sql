CREATE INDEX ledger_postings_account_transaction_idx
  ON ledger_postings (ledger_account_id, transaction_id);
