#!/usr/bin/env bash

set -euo pipefail

if [[ -z "${DB_TEST_SOURCE:-}" ]]; then
  echo "DB_TEST_SOURCE is required" >&2
  exit 1
fi

pattern='^(TestDatabaseRejectsUnbalancedPostedTransaction|TestLedgerRecordsAreImmutable|TestIdempotentTransferTxConcurrentRetriesPostOnce|TestTransferTxDeadlock|TestConcurrentTransfersCannotExceedDailyLimit|TestConcurrentReversalCreatesExactlyOneTransaction|TestFinancialStateRollsBackWhenDurableEventCannotBeStored|TestReconcileReportsHealthyAccountAndAuditsRun)$'

MAILER_PROVIDER=log go test -race -count=1 ./db/sqlc -run "${pattern}"
