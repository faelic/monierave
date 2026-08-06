package db

import (
	"context"
	"testing"

	"github.com/faelic/monierave/db/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestReconcileReportsHealthyAccountAndAuditsRun(t *testing.T) {
	account := createRandomAccount(t)
	account = fundAccount(t, account, 1_000)

	report, err := testStore.Reconcile(context.Background(), ReconciliationParams{
		AccountPublicID: account.PublicID,
		Actor:           "reconciliation-test",
	})
	require.NoError(t, err)
	require.True(t, report.Consistent)
	require.Empty(t, report.Issues)
	require.Equal(t, "reconciliation_completed", report.AuditLog.EventType)
	require.Equal(t, "consistent", report.AuditLog.ToState.String)
	require.Equal(t, "reconciliation-test", report.AuditLog.Actor)
}

func TestReconcileDetectsBalanceDriftWithoutRepairingIt(t *testing.T) {
	account := createRandomAccount(t)
	account = fundAccount(t, account, 1_000)
	driftedBalance := account.Balance + 73
	_, err := testDB.Exec(
		context.Background(),
		"UPDATE accounts SET balance = $1 WHERE id = $2",
		driftedBalance,
		account.ID,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, cleanupErr := testDB.Exec(
			context.Background(),
			"UPDATE accounts SET balance = $1 WHERE id = $2",
			account.Balance,
			account.ID,
		)
		require.NoError(t, cleanupErr)
	})

	report, err := testStore.Reconcile(context.Background(), ReconciliationParams{
		AccountPublicID: account.PublicID,
		Actor:           "reconciliation-test",
	})
	require.NoError(t, err)
	require.False(t, report.Consistent)
	require.Len(t, report.Issues, 1)
	require.Equal(t, ReconciliationIssueBalanceDrift, report.Issues[0].Type)
	require.Equal(t, driftedBalance, report.Issues[0].CachedBalance)
	require.Equal(t, account.Balance, report.Issues[0].LedgerBalance)
	require.Equal(t, "drift_detected", report.AuditLog.ToState.String)

	stored, err := testQueries.GetAccount(context.Background(), account.ID)
	require.NoError(t, err)
	require.Equal(t, driftedBalance, stored.Balance)
}

func TestReconcileAccountScopeIgnoresUnrelatedDrift(t *testing.T) {
	healthy := createRandomAccount(t)
	drifted := createRandomAccount(t)
	_, err := testDB.Exec(
		context.Background(),
		"UPDATE accounts SET balance = 99 WHERE id = $1",
		drifted.ID,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, cleanupErr := testDB.Exec(
			context.Background(),
			"UPDATE accounts SET balance = 0 WHERE id = $1",
			drifted.ID,
		)
		require.NoError(t, cleanupErr)
	})

	report, err := testStore.Reconcile(context.Background(), ReconciliationParams{
		AccountPublicID: healthy.PublicID,
		Actor:           "reconciliation-test",
	})
	require.NoError(t, err)
	require.True(t, report.Consistent)

	report, err = testStore.Reconcile(context.Background(), ReconciliationParams{
		Actor: "reconciliation-test",
	})
	require.NoError(t, err)
	require.False(t, report.Consistent)
	require.Contains(t, reconciliationIssueTypes(report.Issues), ReconciliationIssueBalanceDrift)
}

func TestReconcileDetectsMissingCustomerLedgerAccount(t *testing.T) {
	user := createRandomUser(t)
	account, err := testQueries.CreateAccount(context.Background(), CreateAccountParams{
		Owner:    user.Username,
		Currency: util.RandomCurrency(),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, cleanupErr := testDB.Exec(
			context.Background(),
			"DELETE FROM accounts WHERE id = $1",
			account.ID,
		)
		require.NoError(t, cleanupErr)
	})

	report, err := testStore.Reconcile(context.Background(), ReconciliationParams{
		AccountPublicID: account.PublicID,
		Actor:           "reconciliation-test",
	})
	require.NoError(t, err)
	require.False(t, report.Consistent)
	require.Len(t, report.Issues, 1)
	require.Equal(t, ReconciliationIssueMissingLedgerAccount, report.Issues[0].Type)
}

func TestReconcileRejectsUnknownAccount(t *testing.T) {
	_, err := testStore.Reconcile(context.Background(), ReconciliationParams{
		AccountPublicID: pgtype.UUID{Bytes: uuid.New(), Valid: true},
		Actor:           "reconciliation-test",
	})
	require.ErrorIs(t, err, ErrAccountNotFound)
}

func reconciliationIssueTypes(issues []ReconciliationIssue) []string {
	types := make([]string, 0, len(issues))
	for _, issue := range issues {
		types = append(types, issue.Type)
	}
	return types
}
