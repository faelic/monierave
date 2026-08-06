package db

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestTransferCommitsFinancialEventsWithoutRedisOrMailer(t *testing.T) {
	from := createRandomAccount(t)
	to := createAccountWithCurrency(t, from.Currency)
	from = fundAccount(t, from, 2_000)
	arg := idempotentTransferTestParams(from, to, 500, "notification-outage")
	arg.CorrelationID = pgtype.UUID{Bytes: uuid.New(), Valid: true}

	result, err := testStore.IdempotentTransferTx(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, BankingTransactionStatusPosted, result.Transaction.Status)

	events, err := testQueries.ListOutboxEventsByEntity(
		context.Background(),
		ListOutboxEventsByEntityParams{
			EntityType: "banking_transaction",
			EntityID:   result.Transaction.ID,
		},
	)
	require.NoError(t, err)
	require.Len(t, events, 2)

	directions := make(map[string]string, 2)
	for _, event := range events {
		require.NotEqual(t, uuid.Nil, uuid.UUID(event.ID.Bytes))
		require.Equal(t, DomainEventTransactionPosted, event.EventType)
		require.Equal(t, arg.CorrelationID, event.CorrelationID)
		require.Equal(t, "pending", event.Status)

		job, err := testQueries.GetEmailJob(context.Background(), event.EmailJobID)
		require.NoError(t, err)
		require.Equal(t, EmailJobTypeFinancialNotification, job.JobType)
		require.Equal(t, EmailJobStatusPending, job.Status)
		var payload FinancialNotificationPayload
		require.NoError(t, json.Unmarshal(job.Payload, &payload))
		require.Equal(t, uuidString(event.ID), payload.EventID)
		require.Equal(t, uuidString(arg.CorrelationID), payload.CorrelationID)
		require.Equal(t, result.Transaction.Reference, payload.Reference)
		require.Equal(t, result.Transaction.Amount, payload.Amount)
		require.Equal(t, result.Transaction.Currency, payload.Currency)
		require.False(t, payload.OccurredAt.IsZero())
		directions[job.Username] = payload.Direction

		auditLogs, err := testQueries.ListAuditLogsByJob(
			context.Background(),
			ListAuditLogsByJobParams{
				EntityID: event.ID,
				Limit:    10,
			},
		)
		require.NoError(t, err)
		require.NotEmpty(t, auditLogs)
		require.Equal(t, arg.CorrelationID, auditLogs[0].CorrelationID)
	}
	require.Equal(t, "outgoing", directions[from.Owner])
	require.Equal(t, "incoming", directions[to.Owner])
	require.NotEqual(t, events[0].ID, events[1].ID)
	require.NotEqual(t, events[0].EmailJobID, events[1].EmailJobID)
}

func TestFailedTransferCreatesSanitizedDomainEvent(t *testing.T) {
	from := createRandomAccount(t)
	to := createAccountWithCurrency(t, from.Currency)
	arg := idempotentTransferTestParams(from, to, 1, "failed-notification")
	arg.CorrelationID = pgtype.UUID{Bytes: uuid.New(), Valid: true}

	_, err := testStore.IdempotentTransferTx(context.Background(), arg)
	require.ErrorIs(t, err, ErrInsufficientBalance)
	events, err := testQueries.ListOutboxEventsByEntity(
		context.Background(),
		ListOutboxEventsByEntityParams{
			EntityType: "transfer_attempt",
			EntityID:   arg.CorrelationID,
		},
	)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, DomainEventTransactionFailed, events[0].EventType)

	job, err := testQueries.GetEmailJob(context.Background(), events[0].EmailJobID)
	require.NoError(t, err)
	var payload FinancialNotificationPayload
	require.NoError(t, json.Unmarshal(job.Payload, &payload))
	require.Equal(t, "Insufficient funds", payload.Reason)
	require.NotContains(t, string(job.Payload), "password")
	require.NotContains(t, string(job.Payload), "token")
}

func TestAccountAndReversalActionsCreateDomainEvents(t *testing.T) {
	account := createRandomAccount(t)
	freeze := operatorAccountStatusParams(
		account.PublicID,
		FinancialAccountStatusFrozen,
		"Security review",
	)
	_, err := testStore.SetAccountStatusTx(context.Background(), freeze)
	require.NoError(t, err)
	unfreeze := operatorAccountStatusParams(
		account.PublicID,
		FinancialAccountStatusActive,
		"Security review completed",
	)
	_, err = testStore.SetAccountStatusTx(context.Background(), unfreeze)
	require.NoError(t, err)
	_, err = testStore.CloseAccountTx(context.Background(), CloseAccountTxParams{
		PublicID: account.PublicID,
		Username: account.Owner,
	})
	require.NoError(t, err)
	accountEvents, err := testQueries.ListOutboxEventsByEntity(
		context.Background(),
		ListOutboxEventsByEntityParams{
			EntityType: "account",
			EntityID:   account.PublicID,
		},
	)
	require.NoError(t, err)
	require.Len(t, accountEvents, 3)
	require.Equal(t, DomainEventAccountFrozen, accountEvents[0].EventType)
	require.Equal(t, freeze.CorrelationID, accountEvents[0].CorrelationID)
	require.Equal(t, DomainEventAccountUnfrozen, accountEvents[1].EventType)
	require.Equal(t, unfreeze.CorrelationID, accountEvents[1].CorrelationID)
	require.Equal(t, DomainEventAccountClosed, accountEvents[2].EventType)
	require.Equal(t, account.PublicID, accountEvents[2].CorrelationID)

	fixture := createReversalTransferFixture(t, 1_000, 400)
	reversalArg := operatorReversalParams(
		fixture.transfer.Transaction.ID,
		"Duplicate payment",
	)
	_, err = testStore.ReverseTransactionTx(context.Background(), reversalArg)
	require.NoError(t, err)
	transactionEvents, err := testQueries.ListOutboxEventsByEntity(
		context.Background(),
		ListOutboxEventsByEntityParams{
			EntityType: "banking_transaction",
			EntityID:   fixture.transfer.Transaction.ID,
		},
	)
	require.NoError(t, err)
	var reversalEvents []OutboxEvent
	for _, event := range transactionEvents {
		if event.EventType == DomainEventTransactionReversed {
			reversalEvents = append(reversalEvents, event)
		}
	}
	require.Len(t, reversalEvents, 2)
	for _, event := range reversalEvents {
		require.Equal(t, reversalArg.CorrelationID, event.CorrelationID)
	}
}

func TestReplayFinancialNotificationPreservesDomainMetadata(t *testing.T) {
	account := createRandomAccount(t)
	result, err := testStore.DepositTx(context.Background(), DepositTxParams{
		AccountPublicID: account.PublicID,
		Amount:          500,
		Narration:       "Replay test funding",
		Actor:           "replay-test",
		CorrelationID:   pgtype.UUID{Bytes: uuid.New(), Valid: true},
	})
	require.NoError(t, err)
	events, err := testQueries.ListOutboxEventsByEntity(
		context.Background(),
		ListOutboxEventsByEntityParams{
			EntityType: "banking_transaction",
			EntityID:   result.Transaction.ID,
		},
	)
	require.NoError(t, err)
	require.Len(t, events, 1)
	original := events[0]
	_, err = testQueries.MarkEmailJobDeadLetter(
		context.Background(),
		MarkEmailJobDeadLetterParams{
			ID: original.EmailJobID,
			LastError: pgtype.Text{
				String: "provider unavailable",
				Valid:  true,
			},
		},
	)
	require.NoError(t, err)

	replayed, err := testStore.ReplayEmailJobTx(
		context.Background(),
		original.EmailJobID,
	)
	require.NoError(t, err)
	require.NotEqual(t, original.ID, replayed.OutboxEvent.ID)
	require.NotEqual(t, original.EmailJobID, replayed.EmailJob.ID)
	require.Equal(t, original.EventType, replayed.OutboxEvent.EventType)
	require.Equal(t, original.CorrelationID, replayed.OutboxEvent.CorrelationID)
	require.Equal(t, original.EntityType, replayed.OutboxEvent.EntityType)
	require.Equal(t, original.EntityID, replayed.OutboxEvent.EntityID)
	require.Equal(t, EmailJobTypeFinancialNotification, replayed.EmailJob.JobType)
}

func TestFinancialStateRollsBackWhenDurableEventCannotBeStored(t *testing.T) {
	account := createRandomAccount(t)
	_, err := testDB.Exec(context.Background(), `
		CREATE FUNCTION reject_test_financial_outbox()
		RETURNS trigger
		LANGUAGE plpgsql
		AS $$
		BEGIN
		  IF NEW.event_type = 'transaction.posted' THEN
		    RAISE EXCEPTION 'simulated outbox persistence failure';
		  END IF;
		  RETURN NEW;
		END;
		$$;

		CREATE TRIGGER reject_test_financial_outbox_trigger
		BEFORE INSERT ON outbox_events
		FOR EACH ROW
		EXECUTE FUNCTION reject_test_financial_outbox();
	`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, cleanupErr := testDB.Exec(context.Background(), `
			DROP TRIGGER IF EXISTS reject_test_financial_outbox_trigger
			  ON outbox_events;
			DROP FUNCTION IF EXISTS reject_test_financial_outbox();
		`)
		require.NoError(t, cleanupErr)
	})

	_, err = testStore.DepositTx(context.Background(), DepositTxParams{
		AccountPublicID: account.PublicID,
		Amount:          700,
		Narration:       "Must roll back with outbox",
	})
	require.ErrorContains(t, err, "simulated outbox persistence failure")

	stored, err := testQueries.GetAccount(context.Background(), account.ID)
	require.NoError(t, err)
	require.Zero(t, stored.Balance)
	ledger, err := customerLedgerAccount(context.Background(), testQueries, account.ID)
	require.NoError(t, err)
	balance, err := testQueries.GetLedgerAccountBalance(context.Background(), ledger.ID)
	require.NoError(t, err)
	require.Zero(t, balance)
}
