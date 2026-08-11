package db

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestAccountLifecycleCreatesFinancialAudit(t *testing.T) {
	account := createRandomAccount(t)
	closed, err := testStore.CloseAccountTx(context.Background(), CloseAccountTxParams{
		PublicID: account.PublicID,
		Username: account.Owner,
	})
	require.NoError(t, err)
	require.Equal(t, FinancialAccountStatusClosed, closed.Status)

	logs, err := testQueries.ListAuditLogsByEntity(
		context.Background(),
		ListAuditLogsByEntityParams{
			EntityType: "account",
			EntityID:   account.PublicID,
		},
	)
	require.NoError(t, err)
	requireAuditEvent(t, logs, "account_created")
	requireAuditEvent(t, logs, "account_closed")
}

func TestMoneyMovementCreatesFinancialAudit(t *testing.T) {
	account := createRandomAccount(t)
	result, err := testStore.DepositTx(context.Background(), DepositTxParams{
		AccountPublicID: account.PublicID,
		Amount:          1_000,
		Narration:       "Audited funding",
		Actor:           "funding-operator",
		CorrelationID:   pgtype.UUID{Bytes: uuid.New(), Valid: true},
	})
	require.NoError(t, err)
	require.Equal(t, "operator_transaction_posted", result.AuditLog.EventType)

	logs, err := testQueries.ListAuditLogsByEntity(
		context.Background(),
		ListAuditLogsByEntityParams{
			EntityType: "banking_transaction",
			EntityID:   result.Transaction.ID,
		},
	)
	require.NoError(t, err)
	requireAuditEvent(t, logs, "operator_transaction_created")
	requireAuditEvent(t, logs, "operator_transaction_posted")
}

func TestTransferSuccessAndRejectionsCreateFinancialAudit(t *testing.T) {
	from := createRandomAccount(t)
	to := createAccountWithCurrency(t, from.Currency)
	from = fundAccount(t, from, 2_000)

	success := idempotentTransferTestParams(from, to, 500, "financial-audit")
	success.CorrelationID = pgtype.UUID{Bytes: uuid.New(), Valid: true}
	result, err := testStore.IdempotentTransferTx(context.Background(), success)
	require.NoError(t, err)
	logs, err := testQueries.ListAuditLogsByEntity(
		context.Background(),
		ListAuditLogsByEntityParams{
			EntityType: "banking_transaction",
			EntityID:   result.Transaction.ID,
		},
	)
	require.NoError(t, err)
	requireAuditEvent(t, logs, "transfer_created")
	requireAuditEvent(t, logs, "transfer_posted")

	insufficient := idempotentTransferTestParams(
		from,
		to,
		from.Balance+1,
		"insufficient-audit",
	)
	insufficient.CorrelationID = pgtype.UUID{Bytes: uuid.New(), Valid: true}
	_, err = testStore.IdempotentTransferTx(context.Background(), insufficient)
	require.ErrorIs(t, err, ErrInsufficientBalance)
	logs, err = testQueries.ListAuditLogsByEntity(
		context.Background(),
		ListAuditLogsByEntityParams{
			EntityType: "transfer_attempt",
			EntityID:   insufficient.CorrelationID,
		},
	)
	require.NoError(t, err)
	requireAuditEvent(t, logs, "transfer_rejected_insufficient_funds")
	for _, entry := range logs {
		require.NotContains(t, string(entry.Metadata), to.AccountNumber)
	}

	limit := idempotentTransferTestParams(
		from,
		to,
		USDPerTransferLimit+1,
		"limit-audit",
	)
	limit.Currency = "USD"
	if from.Currency != "USD" {
		limit.Amount = EURPerTransferLimit + 1
		limit.Currency = "EUR"
	}
	limit.CorrelationID = pgtype.UUID{Bytes: uuid.New(), Valid: true}
	_, err = testStore.IdempotentTransferTx(context.Background(), limit)
	require.ErrorIs(t, err, ErrPerTransferLimitExceeded)
	logs, err = testQueries.ListAuditLogsByEntity(
		context.Background(),
		ListAuditLogsByEntityParams{
			EntityType: "transfer_attempt",
			EntityID:   limit.CorrelationID,
		},
	)
	require.NoError(t, err)
	requireAuditEvent(t, logs, "transfer_rejected_limit")

	accountLogs, err := testQueries.ListAccountFinancialAuditLogs(
		context.Background(),
		from.PublicID,
	)
	require.NoError(t, err)
	requireAuditEvent(t, accountLogs, "transfer_posted")
	requireAuditEvent(t, accountLogs, "transfer_rejected_insufficient_funds")
	requireAuditEvent(t, accountLogs, "transfer_rejected_limit")
}

func TestRecordLoginFailureCreatesSecurityAudit(t *testing.T) {
	err := testStore.RecordLoginFailure(context.Background(), LoginFailureAuditParams{
		Username:  "unknown-user",
		ClientIP:  "127.0.0.1",
		UserAgent: "integration-test",
		Reason:    "unknown_user",
	})
	require.NoError(t, err)

	logs, err := testQueries.ListRecentAuditLogs(context.Background(), 20)
	require.NoError(t, err)
	var found bool
	for _, entry := range logs {
		if entry.EventType == "login_failed" && entry.Actor == "unknown-user" {
			found = true
			require.Equal(t, "rejected", entry.ToState.String)
			break
		}
	}
	require.True(t, found)
}

func createAccountWithCurrency(t *testing.T, currency string) Account {
	t.Helper()
	user := createRandomUser(t)
	account, err := testStore.CreateAccountTx(context.Background(), CreateAccountTxParams{
		Owner:    user.Username,
		Currency: currency,
	})
	require.NoError(t, err)
	return account
}
