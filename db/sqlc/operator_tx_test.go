package db

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestSetAccountStatusTxAuditsFreezeAndUnfreeze(t *testing.T) {
	account := createRandomAccount(t)
	freeze := operatorAccountStatusParams(
		account.PublicID,
		FinancialAccountStatusFrozen,
		"Fraud review",
	)

	frozen, err := testStore.SetAccountStatusTx(context.Background(), freeze)
	require.NoError(t, err)
	require.Equal(t, FinancialAccountStatusFrozen, frozen.Account.Status)
	require.Equal(t, "account_frozen", frozen.AuditLog.EventType)
	require.Equal(t, freeze.Actor, frozen.AuditLog.Actor)
	require.Equal(t, freeze.Reason, frozen.AuditLog.Message.String)
	require.Equal(t, account.PublicID, frozen.AuditLog.EntityID)
	require.Equal(t, freeze.CorrelationID, frozen.AuditLog.CorrelationID)

	_, err = testStore.SetAccountStatusTx(context.Background(), freeze)
	require.ErrorIs(t, err, ErrAccountAlreadyFrozen)

	unfreeze := operatorAccountStatusParams(
		account.PublicID,
		FinancialAccountStatusActive,
		"Review completed",
	)
	active, err := testStore.SetAccountStatusTx(context.Background(), unfreeze)
	require.NoError(t, err)
	require.Equal(t, FinancialAccountStatusActive, active.Account.Status)
	require.Equal(t, "account_unfrozen", active.AuditLog.EventType)
	require.Equal(t, FinancialAccountStatusFrozen, active.AuditLog.FromState.String)
	require.Equal(t, FinancialAccountStatusActive, active.AuditLog.ToState.String)

	_, err = testStore.SetAccountStatusTx(context.Background(), unfreeze)
	require.ErrorIs(t, err, ErrAccountAlreadyActive)
}

func TestSetAccountStatusTxRejectsClosedAndInvalidActions(t *testing.T) {
	user := createRandomUser(t)
	account, err := testStore.CreateAccountTx(context.Background(), CreateAccountParams{
		Owner: user.Username, Currency: "USD",
	})
	require.NoError(t, err)
	_, err = testStore.CloseAccountTx(context.Background(), CloseAccountTxParams{
		PublicID: account.PublicID, Username: account.Owner,
	})
	require.NoError(t, err)

	_, err = testStore.SetAccountStatusTx(
		context.Background(),
		operatorAccountStatusParams(
			account.PublicID,
			FinancialAccountStatusFrozen,
			"Should fail",
		),
	)
	require.ErrorIs(t, err, ErrAccountClosed)

	invalid := operatorAccountStatusParams(
		account.PublicID,
		FinancialAccountStatusFrozen,
		"",
	)
	_, err = testStore.SetAccountStatusTx(context.Background(), invalid)
	require.ErrorIs(t, err, ErrInvalidOperatorAction)
}

func TestReverseInternalTransferCreatesOppositePostings(t *testing.T) {
	fixture := createReversalTransferFixture(t, 1_000, 400)
	arg := operatorReversalParams(
		fixture.transfer.Transaction.ID,
		"Duplicate customer transfer",
	)

	result, err := testStore.ReverseTransactionTx(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, BankingTransactionStatusReversed, result.Original.Status)
	require.True(t, result.Original.ReversedAt.Valid)
	require.Equal(t, BankingTransactionTypeReversal, result.Reversal.TransactionType)
	require.Equal(t, BankingTransactionStatusPosted, result.Reversal.Status)
	require.Equal(t, fixture.transfer.Transaction.ID, result.Reversal.ReversalOf)
	require.Equal(t, fixture.transfer.Transaction.Amount, result.Reversal.Amount)
	require.Len(t, result.Postings, len(fixture.transfer.Postings))

	originalAmounts := make(map[int64]int64, len(fixture.transfer.Postings))
	for _, posting := range fixture.transfer.Postings {
		originalAmounts[posting.LedgerAccountID] = posting.Amount
	}
	for _, posting := range result.Postings {
		require.Equal(t, -originalAmounts[posting.LedgerAccountID], posting.Amount)
	}

	from, err := testQueries.GetAccount(context.Background(), fixture.from.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1_000), from.Balance)
	to, err := testQueries.GetAccount(context.Background(), fixture.to.ID)
	require.NoError(t, err)
	require.Zero(t, to.Balance)

	require.Equal(t, "transaction_reversed", result.AuditLog.EventType)
	require.Equal(t, arg.Actor, result.AuditLog.Actor)
	require.Equal(t, arg.Reason, result.AuditLog.Message.String)
	require.Equal(t, fixture.transfer.Transaction.ID, result.AuditLog.EntityID)
	require.Equal(t, arg.CorrelationID, result.AuditLog.CorrelationID)
	require.Contains(t, string(result.AuditLog.Metadata), result.Reversal.Reference)

	dailyTotal, err := testQueries.GetDailyOutgoingTransferTotal(
		context.Background(),
		fixture.fromLedger.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(400), dailyTotal)
}

func TestReverseTransactionRejectsDuplicateAndReversal(t *testing.T) {
	fixture := createReversalTransferFixture(t, 1_000, 400)
	first, err := testStore.ReverseTransactionTx(
		context.Background(),
		operatorReversalParams(fixture.transfer.Transaction.ID, "First reversal"),
	)
	require.NoError(t, err)

	duplicate := operatorReversalParams(
		fixture.transfer.Transaction.ID,
		"Duplicate reversal",
	)
	_, err = testStore.ReverseTransactionTx(context.Background(), duplicate)
	require.ErrorIs(t, err, ErrTransactionAlreadyReversed)
	logs, err := testQueries.ListAuditLogsByEntity(
		context.Background(),
		ListAuditLogsByEntityParams{
			EntityType: "banking_transaction",
			EntityID:   fixture.transfer.Transaction.ID,
		},
	)
	require.NoError(t, err)
	requireAuditEvent(t, logs, "transaction_reversal_failed")

	_, err = testStore.ReverseTransactionTx(
		context.Background(),
		operatorReversalParams(first.Reversal.ID, "Reverse the reversal"),
	)
	require.ErrorIs(t, err, ErrTransactionNotReversible)
}

func TestReverseSettlementMovements(t *testing.T) {
	t.Run("Deposit", func(t *testing.T) {
		account := createRandomAccount(t)
		deposit, err := testStore.DepositTx(context.Background(), DepositTxParams{
			AccountPublicID: account.PublicID,
			Amount:          500,
			Narration:       "Deposit to reverse",
		})
		require.NoError(t, err)

		result, err := testStore.ReverseTransactionTx(
			context.Background(),
			operatorReversalParams(deposit.Transaction.ID, "Reverse deposit"),
		)
		require.NoError(t, err)
		require.Equal(t, BankingTransactionTypeReversal, result.Reversal.TransactionType)
		stored, err := testQueries.GetAccount(context.Background(), account.ID)
		require.NoError(t, err)
		require.Zero(t, stored.Balance)
	})

	t.Run("Withdrawal", func(t *testing.T) {
		account := createRandomAccount(t)
		deposit, err := testStore.DepositTx(context.Background(), DepositTxParams{
			AccountPublicID: account.PublicID,
			Amount:          500,
			Narration:       "Withdrawal reversal funding",
		})
		require.NoError(t, err)
		withdrawal, err := testStore.WithdrawTx(context.Background(), WithdrawTxParams{
			AccountPublicID: account.PublicID,
			Amount:          200,
			Narration:       "Withdrawal to reverse",
		})
		require.NoError(t, err)

		_, err = testStore.ReverseTransactionTx(
			context.Background(),
			operatorReversalParams(withdrawal.Transaction.ID, "Reverse withdrawal"),
		)
		require.NoError(t, err)
		stored, err := testQueries.GetAccount(context.Background(), account.ID)
		require.NoError(t, err)
		require.Equal(t, deposit.Account.Balance, stored.Balance)
	})
}

func TestConcurrentReversalCreatesExactlyOneTransaction(t *testing.T) {
	fixture := createReversalTransferFixture(t, 1_000, 400)
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := testStore.ReverseTransactionTx(
				context.Background(),
				operatorReversalParams(
					fixture.transfer.Transaction.ID,
					"Concurrent reversal",
				),
			)
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)

	successes := 0
	duplicates := 0
	for err := range errs {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrTransactionAlreadyReversed):
			duplicates++
		default:
			require.NoError(t, err)
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, duplicates)

	var reversalCount int64
	err := testDB.QueryRow(
		context.Background(),
		"SELECT count(*) FROM banking_transactions WHERE reversal_of = $1",
		fixture.transfer.Transaction.ID,
	).Scan(&reversalCount)
	require.NoError(t, err)
	require.Equal(t, int64(1), reversalCount)
}

func TestReverseTransactionFailsWhenRecipientSpentFunds(t *testing.T) {
	fixture := createReversalTransferFixture(t, 1_000, 400)
	_, err := testStore.WithdrawTx(context.Background(), WithdrawTxParams{
		AccountPublicID: fixture.to.PublicID,
		Amount:          400,
		Narration:       "Recipient spent transfer",
	})
	require.NoError(t, err)

	_, err = testStore.ReverseTransactionTx(
		context.Background(),
		operatorReversalParams(
			fixture.transfer.Transaction.ID,
			"Insufficient recipient funds",
		),
	)
	require.ErrorIs(t, err, ErrReversalInsufficientFunds)

	original, err := testQueries.GetBankingTransaction(
		context.Background(),
		fixture.transfer.Transaction.ID,
	)
	require.NoError(t, err)
	require.Equal(t, BankingTransactionStatusPosted, original.Status)

	var reversalCount int64
	err = testDB.QueryRow(
		context.Background(),
		"SELECT count(*) FROM banking_transactions WHERE reversal_of = $1",
		original.ID,
	).Scan(&reversalCount)
	require.NoError(t, err)
	require.Zero(t, reversalCount)
}

func TestReverseTransactionAllowsFrozenButRejectsClosedAccounts(t *testing.T) {
	t.Run("Frozen", func(t *testing.T) {
		fixture := createReversalTransferFixture(t, 1_000, 400)
		_, err := testStore.SetAccountStatusTx(
			context.Background(),
			operatorAccountStatusParams(
				fixture.to.PublicID,
				FinancialAccountStatusFrozen,
				"Freeze before reversal",
			),
		)
		require.NoError(t, err)

		result, err := testStore.ReverseTransactionTx(
			context.Background(),
			operatorReversalParams(
				fixture.transfer.Transaction.ID,
				"Reverse frozen recipient transfer",
			),
		)
		require.NoError(t, err)
		require.Equal(t, BankingTransactionStatusPosted, result.Reversal.Status)
	})

	t.Run("Closed", func(t *testing.T) {
		fixture := createReversalTransferFixture(t, 400, 400)
		_, err := testStore.CloseAccountTx(
			context.Background(),
			CloseAccountTxParams{
				PublicID: fixture.from.PublicID,
				Username: fixture.from.Owner,
			},
		)
		require.NoError(t, err)

		_, err = testStore.ReverseTransactionTx(
			context.Background(),
			operatorReversalParams(
				fixture.transfer.Transaction.ID,
				"Closed account reversal",
			),
		)
		require.ErrorIs(t, err, ErrAccountClosed)
	})
}

func TestReverseTransactionRejectsPendingOrMissingTransaction(t *testing.T) {
	pendingID, reference := newTransactionIdentity()
	_, err := testQueries.CreateBankingTransaction(
		context.Background(),
		CreateBankingTransactionParams{
			ID:              pendingID,
			Reference:       reference,
			TransactionType: BankingTransactionTypeDeposit,
			Currency:        "USD",
			Amount:          100,
			Narration:       "Pending reversal test",
			InitiatedBy:     "test",
		},
	)
	require.NoError(t, err)

	_, err = testStore.ReverseTransactionTx(
		context.Background(),
		operatorReversalParams(pendingID, "Pending transaction"),
	)
	require.ErrorIs(t, err, ErrTransactionNotPosted)

	missing := pgtype.UUID{Bytes: uuid.New(), Valid: true}
	_, err = testStore.ReverseTransactionTx(
		context.Background(),
		operatorReversalParams(missing, "Missing transaction"),
	)
	require.ErrorIs(t, err, ErrTransactionNotFound)
}

func TestDatabaseEnforcesReversalShape(t *testing.T) {
	original := createReversalTransferFixture(t, 1_000, 100).transfer.Transaction
	id, reference := newTransactionIdentity()
	_, err := testDB.Exec(
		context.Background(),
		`INSERT INTO banking_transactions (
			id, reference, transaction_type, currency, amount, narration,
			initiated_by
		) VALUES ($1, $2, 'reversal', 'USD', 100, 'invalid', 'test')`,
		id,
		reference,
	)
	require.Error(t, err)

	id, reference = newTransactionIdentity()
	_, err = testDB.Exec(
		context.Background(),
		`INSERT INTO banking_transactions (
			id, reference, transaction_type, currency, amount, narration,
			initiated_by, reversal_of
		) VALUES ($1, $2, 'deposit', 'USD', 100, 'invalid', 'test', $3)`,
		id,
		reference,
		original.ID,
	)
	require.Error(t, err)
}

type reversalTransferFixture struct {
	from       Account
	to         Account
	fromLedger LedgerAccount
	transfer   TransferTxResult
}

func createReversalTransferFixture(
	t *testing.T,
	funding int64,
	amount int64,
) reversalTransferFixture {
	t.Helper()
	from, to := createLimitTestAccounts(t, "USD", funding)
	transfer, err := testStore.TransferTx(context.Background(), TransferTxParams{
		FromAccountPublicID: from.PublicID,
		ToAccountPublicID:   to.PublicID,
		Amount:              amount,
		Currency:            "USD",
		Username:            from.Owner,
		Narration:           "Reversal fixture transfer",
	})
	require.NoError(t, err)
	fromLedger, err := customerLedgerAccount(
		context.Background(),
		testQueries,
		from.ID,
	)
	require.NoError(t, err)
	return reversalTransferFixture{
		from:       from,
		to:         to,
		fromLedger: fromLedger,
		transfer:   transfer,
	}
}

func operatorAccountStatusParams(
	accountID pgtype.UUID,
	targetStatus string,
	reason string,
) AccountStatusTxParams {
	return AccountStatusTxParams{
		AccountPublicID: accountID,
		TargetStatus:    targetStatus,
		Actor:           "operator@test",
		Reason:          reason,
		CorrelationID:   pgtype.UUID{Bytes: uuid.New(), Valid: true},
	}
}

func operatorReversalParams(
	transactionID pgtype.UUID,
	reason string,
) ReverseTransactionTxParams {
	return ReverseTransactionTxParams{
		TransactionID: transactionID,
		Actor:         "operator@test",
		Reason:        reason,
		CorrelationID: pgtype.UUID{Bytes: uuid.New(), Valid: true},
	}
}
