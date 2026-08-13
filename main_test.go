package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	mockdb "github.com/faelic/monierave/db/mock"
	db "github.com/faelic/monierave/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSuperviseRuntimeRolesStopsAllComponentsWhenOneFails(t *testing.T) {
	expectedErr := errors.New("redis unavailable")
	siblingStopped := make(chan struct{})

	err := superviseRuntimeRoles(context.Background(), []runtimeRole{
		{
			name: "relay",
			run: func(context.Context) error {
				return expectedErr
			},
		},
		{
			name: "api",
			run: func(ctx context.Context) error {
				<-ctx.Done()
				close(siblingStopped)
				return nil
			},
		},
	})

	require.ErrorIs(t, err, expectedErr)
	require.Contains(t, err.Error(), "relay runtime stopped")
	select {
	case <-siblingStopped:
	case <-time.After(time.Second):
		t.Fatal("sibling component was not stopped")
	}
}

func TestSuperviseRuntimeRolesStopsCleanlyWithParentContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})

	go func() {
		<-started
		cancel()
	}()

	err := superviseRuntimeRoles(ctx, []runtimeRole{
		{
			name: "api",
			run: func(ctx context.Context) error {
				close(started)
				<-ctx.Done()
				return nil
			},
		},
	})

	require.NoError(t, err)
}

func TestSuperviseRuntimeRolesRejectsUnexpectedCleanExit(t *testing.T) {
	err := superviseRuntimeRoles(context.Background(), []runtimeRole{
		{name: "worker", run: func(context.Context) error { return nil }},
	})

	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "worker runtime stopped unexpectedly"))
}

func TestRunBankingAccountControlMapsOperatorFlags(t *testing.T) {
	accountID := uuid.New()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().SetAccountStatusTx(
		gomock.Any(),
		gomock.Any(),
	).DoAndReturn(func(
		_ context.Context,
		arg db.AccountStatusTxParams,
	) (db.AccountStatusTxResult, error) {
		require.Equal(t, accountID, uuid.UUID(arg.AccountPublicID.Bytes))
		require.Equal(t, db.FinancialAccountStatusFrozen, arg.TargetStatus)
		require.Equal(t, "ops@example.com", arg.Actor)
		require.Equal(t, "Fraud review", arg.Reason)
		require.True(t, arg.CorrelationID.Valid)
		return db.AccountStatusTxResult{
			Account: db.Account{
				PublicID: arg.AccountPublicID,
				Status:   db.FinancialAccountStatusFrozen,
			},
			AuditLog: db.AuditLog{ID: 42},
		}, nil
	})

	output := captureStdout(t, func() error {
		return runBankingAccountControl(
			context.Background(),
			store,
			"freeze",
			[]string{
				"--account", accountID.String(),
				"--reason", "Fraud review",
				"--actor", "ops@example.com",
			},
		)
	})
	var response accountControlOutput
	require.NoError(t, json.Unmarshal(output, &response))
	require.Equal(t, accountID.String(), response.AccountID)
	require.Equal(t, db.FinancialAccountStatusFrozen, response.Status)
	require.Equal(t, int64(42), response.AuditLogID)
	require.NotEmpty(t, response.CorrelationID)
}

func TestRunBankingReversalMapsOperatorFlags(t *testing.T) {
	originalID := uuid.New()
	reversalID := uuid.New()
	accountID := uuid.New()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().ReverseTransactionTx(
		gomock.Any(),
		gomock.Any(),
	).DoAndReturn(func(
		_ context.Context,
		arg db.ReverseTransactionTxParams,
	) (db.ReverseTransactionTxResult, error) {
		require.Equal(t, originalID, uuid.UUID(arg.TransactionID.Bytes))
		require.Equal(t, "ops@example.com", arg.Actor)
		require.Equal(t, "Duplicate transfer", arg.Reason)
		require.True(t, arg.CorrelationID.Valid)
		return db.ReverseTransactionTxResult{
			Original: db.BankingTransaction{
				ID:        pgtype.UUID{Bytes: originalID, Valid: true},
				Reference: "TXN-ORIGINAL",
				Status:    db.BankingTransactionStatusReversed,
			},
			Reversal: db.BankingTransaction{
				ID:        pgtype.UUID{Bytes: reversalID, Valid: true},
				Reference: "TXN-REVERSAL",
			},
			Accounts: []db.Account{
				{
					PublicID: pgtype.UUID{Bytes: accountID, Valid: true},
					Balance:  1_000,
					Currency: "USD",
					Status:   db.FinancialAccountStatusActive,
				},
			},
			AuditLog: db.AuditLog{ID: 99},
		}, nil
	})

	output := captureStdout(t, func() error {
		return runBankingReversal(
			context.Background(),
			store,
			[]string{
				"--transaction", originalID.String(),
				"--reason", "Duplicate transfer",
				"--actor", "ops@example.com",
			},
		)
	})
	var response reversalOutput
	require.NoError(t, json.Unmarshal(output, &response))
	require.Equal(t, originalID.String(), response.OriginalTransactionID)
	require.Equal(t, reversalID.String(), response.ReversalTransactionID)
	require.Equal(t, int64(99), response.AuditLogID)
	require.Len(t, response.Accounts, 1)
	require.Equal(t, accountID.String(), response.Accounts[0].AccountID)
}

func TestRunBankingReconcileReportsDriftAndReturnsError(t *testing.T) {
	accountID := uuid.New()
	runID := uuid.New()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().
		Reconcile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			_ context.Context,
			arg db.ReconciliationParams,
		) (db.ReconciliationReport, error) {
			require.Equal(t, accountID, uuid.UUID(arg.AccountPublicID.Bytes))
			require.Equal(t, "reconciliation-operator", arg.Actor)
			return db.ReconciliationReport{
				RunID:           pgtype.UUID{Bytes: runID, Valid: true},
				AccountPublicID: arg.AccountPublicID,
				Consistent:      false,
				Issues: []db.ReconciliationIssue{
					{Type: db.ReconciliationIssueBalanceDrift},
				},
				AuditLog: db.AuditLog{ID: 71},
			}, nil
		})

	output, err := captureStdoutResult(func() error {
		return runBankingReconcile(context.Background(), store, []string{
			"--account", accountID.String(),
			"--actor", "reconciliation-operator",
		})
	})
	require.ErrorIs(t, err, errReconciliationDrift)
	var response reconciliationOutput
	require.NoError(t, json.Unmarshal(output, &response))
	require.False(t, response.Consistent)
	require.Equal(t, 1, response.IssueCount)
	require.Equal(t, int64(71), response.AuditLogID)
}

func TestRunBankingTransactionAuditUsesSafeLedgerLabels(t *testing.T) {
	transactionID := uuid.New()
	accountID := uuid.New()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().
		GetBankingTransaction(gomock.Any(), pgtype.UUID{Bytes: transactionID, Valid: true}).
		Return(db.BankingTransaction{
			ID:              pgtype.UUID{Bytes: transactionID, Valid: true},
			Reference:       "TXN-AUDIT",
			TransactionType: db.BankingTransactionTypeDeposit,
			Status:          db.BankingTransactionStatusPosted,
			Currency:        "USD",
			Amount:          500,
		}, nil)
	store.EXPECT().
		ListFinancialAuditPostings(gomock.Any(), pgtype.UUID{Bytes: transactionID, Valid: true}).
		Return([]db.ListFinancialAuditPostingsRow{
			{
				Amount:          500,
				Kind:            "customer",
				Currency:        "USD",
				AccountPublicID: pgtype.UUID{Bytes: accountID, Valid: true},
				CreatedAt:       pgtype.Timestamptz{},
			},
		}, nil)
	store.EXPECT().
		ListAuditLogsByEntity(gomock.Any(), db.ListAuditLogsByEntityParams{
			EntityType: "banking_transaction",
			EntityID:   pgtype.UUID{Bytes: transactionID, Valid: true},
		}).
		Return([]db.AuditLog{}, nil)

	output := captureStdout(t, func() error {
		return runBankingAudit(context.Background(), store, []string{
			"--transaction", transactionID.String(),
		})
	})
	var response transactionAuditOutput
	require.NoError(t, json.Unmarshal(output, &response))
	require.Equal(t, transactionID.String(), response.TransactionID)
	require.Len(t, response.Postings, 1)
	require.Equal(t, accountID.String(), response.Postings[0].AccountID)
	require.NotContains(t, string(output), "ledger_account_id")
}

func TestRunBankingAccountAuditComparesCachedAndLedgerBalances(t *testing.T) {
	accountID := uuid.New()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().
		GetAccountByPublicID(gomock.Any(), pgtype.UUID{Bytes: accountID, Valid: true}).
		Return(db.Account{
			ID:       19,
			PublicID: pgtype.UUID{Bytes: accountID, Valid: true},
			Owner:    "audit-owner",
			Currency: "USD",
			Status:   db.FinancialAccountStatusActive,
			Balance:  700,
		}, nil)
	store.EXPECT().
		GetCustomerLedgerAccount(
			gomock.Any(),
			pgtype.Int8{Int64: 19, Valid: true},
		).
		Return(db.LedgerAccount{ID: 31}, nil)
	store.EXPECT().
		GetLedgerAccountBalance(gomock.Any(), int64(31)).
		Return(int64(700), nil)
	store.EXPECT().
		ListAccountFinancialAuditLogs(
			gomock.Any(),
			pgtype.UUID{Bytes: accountID, Valid: true},
		).
		Return([]db.AuditLog{}, nil)

	output := captureStdout(t, func() error {
		return runBankingAudit(context.Background(), store, []string{
			"--account", accountID.String(),
		})
	})
	var response accountAuditOutput
	require.NoError(t, json.Unmarshal(output, &response))
	require.Equal(t, accountID.String(), response.AccountID)
	require.True(t, response.LedgerExists)
	require.Equal(t, response.CachedBalance, response.LedgerBalance)
	require.NotContains(t, string(output), `"id":19`)
}

func TestDefaultOperatorActorUsesEnvironment(t *testing.T) {
	t.Setenv("MONIERAVE_OPERATOR", "operator@example.com")
	require.Equal(t, "operator@example.com", defaultOperatorActor())
}

func captureStdout(t *testing.T, fn func() error) []byte {
	t.Helper()
	output, err := captureStdoutResult(fn)
	require.NoError(t, err)
	return output
}

func captureStdoutResult(fn func() error) ([]byte, error) {
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	original := os.Stdout
	os.Stdout = writer
	defer func() {
		os.Stdout = original
	}()

	callErr := fn()
	if err := writer.Close(); err != nil {
		return nil, errors.Join(callErr, err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.Join(callErr, err)
	}
	if err := reader.Close(); err != nil {
		return nil, errors.Join(callErr, err)
	}
	return output, callErr
}
