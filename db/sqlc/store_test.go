package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/faelic/monierave/db/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createTransferTestAccount(t *testing.T) Account {
	user := createRandomUser(t)

	account, err := testQueries.CreateAccount(context.Background(), CreateAccountParams{
		Owner:    user.Username,
		Balance:  util.RandomMoney(),
		Currency: "USD",
	})
	require.NoError(t, err)
	require.NotEmpty(t, account)

	return account
}

func TestCreateUserTxCreatesEmailJobAndOutboxEvent(t *testing.T) {
	hashedPassword, err := util.HashPassword(util.RandomString(8))
	require.NoError(t, err)

	arg := CreateUserParams{
		Username:       util.RandomOwner(),
		HashedPassword: hashedPassword,
		FullName:       util.RandomOwner(),
		Email:          util.RandomEmail(),
	}
	result, err := testStore.CreateUserTx(context.Background(), arg)
	require.NoError(t, err)

	require.Equal(t, arg.Username, result.User.Username)
	require.Equal(t, EmailJobTypeVerifyEmail, result.EmailJob.JobType)
	require.Equal(t, arg.Username, result.EmailJob.Username)
	require.Equal(t, arg.Email, result.EmailJob.Recipient)
	require.Equal(t, "pending", result.EmailJob.Status)
	require.Equal(t, DefaultEmailMaxAttempts, result.EmailJob.MaxAttempts)
	require.Equal(t, result.EmailJob.ID, result.OutboxEvent.EmailJobID)
	require.Equal(t, OutboxEventTypeEmailReady, result.OutboxEvent.EventType)
	require.Equal(t, "pending", result.OutboxEvent.Status)

	storedJob, err := testQueries.GetEmailJob(context.Background(), result.EmailJob.ID)
	require.NoError(t, err)
	require.Equal(t, result.EmailJob.ID, storedJob.ID)

	storedEvent, err := testQueries.GetOutboxEvent(context.Background(), result.OutboxEvent.ID)
	require.NoError(t, err)
	require.Equal(t, result.OutboxEvent.ID, storedEvent.ID)

	auditLogs, err := testQueries.ListAuditLogsByJob(
		context.Background(),
		ListAuditLogsByJobParams{EntityID: result.EmailJob.ID, Limit: 20},
	)
	require.NoError(t, err)
	requireAuditEvent(t, auditLogs, "email_job_created")
	requireAuditEvent(t, auditLogs, "outbox_event_created")
}

func TestReplayEmailJobTxCreatesLinkedJob(t *testing.T) {
	hashedPassword, err := util.HashPassword(util.RandomString(8))
	require.NoError(t, err)

	created, err := testStore.CreateUserTx(context.Background(), CreateUserParams{
		Username:       util.RandomOwner(),
		HashedPassword: hashedPassword,
		FullName:       util.RandomOwner(),
		Email:          util.RandomEmail(),
	})
	require.NoError(t, err)

	_, err = testQueries.MarkEmailJobDeadLetter(context.Background(), MarkEmailJobDeadLetterParams{
		ID:        created.EmailJob.ID,
		LastError: pgtype.Text{String: "provider rejected message", Valid: true},
	})
	require.NoError(t, err)

	replayed, err := testStore.ReplayEmailJobTx(context.Background(), created.EmailJob.ID)
	require.NoError(t, err)
	require.NotEqual(t, created.EmailJob.ID, replayed.EmailJob.ID)
	require.Equal(t, created.EmailJob.ID, replayed.EmailJob.ParentJobID)
	require.Equal(t, "pending", replayed.EmailJob.Status)
	require.Equal(t, replayed.EmailJob.ID, replayed.OutboxEvent.EmailJobID)

	originalAuditLogs, err := testQueries.ListAuditLogsByJob(
		context.Background(),
		ListAuditLogsByJobParams{EntityID: created.EmailJob.ID, Limit: 20},
	)
	require.NoError(t, err)
	requireAuditEvent(t, originalAuditLogs, "email_dead_lettered")
	requireAuditEvent(t, originalAuditLogs, "email_job_replayed")

	replayAuditLogs, err := testQueries.ListAuditLogsByJob(
		context.Background(),
		ListAuditLogsByJobParams{EntityID: replayed.EmailJob.ID, Limit: 20},
	)
	require.NoError(t, err)
	requireAuditEvent(t, replayAuditLogs, "email_job_replay_created")
	requireAuditEvent(t, replayAuditLogs, "outbox_event_created")
	requireAuditEvent(t, replayAuditLogs, "email_job_replayed")
}

func TestAuditLogsAreAppendOnly(t *testing.T) {
	hashedPassword, err := util.HashPassword(util.RandomString(8))
	require.NoError(t, err)

	created, err := testStore.CreateUserTx(context.Background(), CreateUserParams{
		Username:       util.RandomOwner(),
		HashedPassword: hashedPassword,
		FullName:       util.RandomOwner(),
		Email:          util.RandomEmail(),
	})
	require.NoError(t, err)

	auditLogs, err := testQueries.ListAuditLogsByJob(
		context.Background(),
		ListAuditLogsByJobParams{EntityID: created.EmailJob.ID, Limit: 20},
	)
	require.NoError(t, err)
	require.NotEmpty(t, auditLogs)

	_, err = testDB.Exec(
		context.Background(),
		"UPDATE audit_logs SET actor = 'tampered' WHERE id = $1",
		auditLogs[0].ID,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "append-only")

	_, err = testDB.Exec(context.Background(), "TRUNCATE audit_logs")
	require.Error(t, err)
	require.Contains(t, err.Error(), "append-only")
}

func TestEmailJobLifecycleCreatesAuditHistory(t *testing.T) {
	hashedPassword, err := util.HashPassword(util.RandomString(8))
	require.NoError(t, err)

	created, err := testStore.CreateUserTx(context.Background(), CreateUserParams{
		Username:       util.RandomOwner(),
		HashedPassword: hashedPassword,
		FullName:       util.RandomOwner(),
		Email:          util.RandomEmail(),
	})
	require.NoError(t, err)

	_, err = testQueries.MarkEmailJobQueued(context.Background(), created.EmailJob.ID)
	require.NoError(t, err)
	_, err = testQueries.StartEmailJobAttempt(context.Background(), created.EmailJob.ID)
	require.NoError(t, err)
	_, err = testQueries.MarkEmailJobRetrying(context.Background(), MarkEmailJobRetryingParams{
		ID:        created.EmailJob.ID,
		LastError: pgtype.Text{String: "temporary provider failure", Valid: true},
	})
	require.NoError(t, err)
	_, err = testQueries.StartEmailJobAttempt(context.Background(), created.EmailJob.ID)
	require.NoError(t, err)
	_, err = testQueries.MarkEmailJobSent(context.Background(), MarkEmailJobSentParams{
		ID: created.EmailJob.ID,
		ProviderMessageID: pgtype.Text{
			String: "provider-message-id",
			Valid:  true,
		},
	})
	require.NoError(t, err)

	auditLogs, err := testQueries.ListAuditLogsByJob(
		context.Background(),
		ListAuditLogsByJobParams{EntityID: created.EmailJob.ID, Limit: 30},
	)
	require.NoError(t, err)
	requireAuditEvent(t, auditLogs, "email_job_queued")
	requireAuditEvent(t, auditLogs, "email_attempt_started")
	requireAuditEvent(t, auditLogs, "email_send_failed")
	requireAuditEvent(t, auditLogs, "email_sent")
}

func requireAuditEvent(t *testing.T, logs []AuditLog, eventType string) {
	t.Helper()
	for _, entry := range logs {
		if entry.EventType == eventType {
			return
		}
	}
	require.Failf(t, "missing audit event", "expected %q in %#v", eventType, logs)
}

func TestTransferTx(t *testing.T) {
	account1 := createTransferTestAccount(t)
	account2 := createTransferTestAccount(t)

	n := 5
	amount := int64(10)

	errs := make(chan error)
	results := make(chan TransferTxResult)

	for i := 0; i < n; i++ {
		go func() {
			ctx := context.Background()
			result, err := testStore.TransferTx(ctx, TransferTxParams{
				FromAccountID: account1.ID,
				ToAccountID:   account2.ID,
				Amount:        amount,
			})

			errs <- err
			results <- result
		}()
	}

	// check results
	existed := make(map[int]bool)

	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err)

		result := <-results
		require.NotEmpty(t, result)

		transfer := result.Transfer
		require.NotEmpty(t, transfer)
		require.Equal(t, account1.ID, transfer.FromAccountID)
		require.Equal(t, account2.ID, transfer.ToAccountID)
		require.Equal(t, amount, transfer.Amount)
		require.NotZero(t, transfer.ID)
		require.NotZero(t, transfer.CreatedAt)

		createdTransfer, err := testQueries.GetTransfer(context.Background(), transfer.ID)
		require.NoError(t, err)
		require.Equal(t, transfer.ID, createdTransfer.ID)

		//check entry
		fromEntry := result.FromEntry
		require.NotEmpty(t, fromEntry)
		require.Equal(t, account1.ID, fromEntry.AccountID)
		require.Equal(t, -amount, fromEntry.Amount)
		require.NotZero(t, fromEntry.ID)
		require.NotZero(t, fromEntry.CreatedAt)

		createdFromEntry, err := testQueries.GetEntry(context.Background(), fromEntry.ID)
		require.NoError(t, err)
		require.Equal(t, fromEntry.ID, createdFromEntry.ID)

		toEntry := result.ToEntry
		require.NotEmpty(t, toEntry)
		require.Equal(t, account2.ID, toEntry.AccountID)
		require.Equal(t, amount, toEntry.Amount)
		require.NotZero(t, toEntry.ID)
		require.NotZero(t, toEntry.CreatedAt)

		createdToEntry, err := testQueries.GetEntry(context.Background(), toEntry.ID)
		require.NoError(t, err)
		require.Equal(t, toEntry.ID, createdToEntry.ID)

		// check accounts
		fromAccount := result.FromAccount
		require.NotEmpty(t, fromAccount)
		require.Equal(t, account1.ID, fromAccount.ID)

		toAccount := result.ToAccount
		require.NotEmpty(t, toAccount)
		require.Equal(t, account2.ID, toAccount.ID)

		//check accounts balance
		diff1 := account1.Balance - fromAccount.Balance
		diff2 := toAccount.Balance - account2.Balance
		require.Equal(t, diff1, diff2)
		require.True(t, diff1 > 0)
		require.True(t, diff1%amount == 0)

		k := int(diff1 / amount)
		require.True(t, k >= 1 && k <= n)
		require.NotContains(t, existed, k)
		existed[k] = true
	}

	// check the final updated balances
	updatedAccount1, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.NotEmpty(t, updatedAccount1)
	require.NoError(t, err)

	updatedAccount2, err := testQueries.GetAccount(context.Background(), account2.ID)
	require.NotEmpty(t, updatedAccount2)
	require.NoError(t, err)

	require.Equal(t, account1.Balance-int64(n)*amount, updatedAccount1.Balance)
	require.Equal(t, account2.Balance+int64(n)*amount, updatedAccount2.Balance)
}

func TestTransferTxDeadlock(t *testing.T) {
	account1 := createTransferTestAccount(t)
	account2 := createTransferTestAccount(t)

	fmt.Println("before account1 balance:", account1.Balance, "before account2 balance:", account2.Balance)

	n := 10
	amount := int64(10)

	errs := make(chan error)

	for i := 0; i < n; i++ {
		fromAccountID := account1.ID
		toAccountID := account2.ID

		if i%2 == 1 {
			fromAccountID = account2.ID
			toAccountID = account1.ID
		}

		go func() {
			ctx := context.Background()
			_, err := testStore.TransferTx(ctx, TransferTxParams{
				FromAccountID: fromAccountID,
				ToAccountID:   toAccountID,
				Amount:        amount,
			})

			errs <- err
		}()
	}

	// check results

	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err)
	}

	// check the final updated balances
	updatedAccount1, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.NotEmpty(t, updatedAccount1)
	require.NoError(t, err)

	updatedAccount2, err := testQueries.GetAccount(context.Background(), account2.ID)
	require.NotEmpty(t, updatedAccount2)
	require.NoError(t, err)

	fmt.Println("after >>", updatedAccount1.Balance, updatedAccount2.Balance)
	require.Equal(t, account1.Balance, updatedAccount1.Balance)
	require.Equal(t, account2.Balance, updatedAccount2.Balance)
}

func TestTransferTxInsufficientBalance(t *testing.T) {
	account1 := createTransferTestAccount(t)
	account2 := createTransferTestAccount(t)

	amount := account1.Balance + 1

	result, err := testStore.TransferTx(context.Background(), TransferTxParams{
		FromAccountID: account1.ID,
		ToAccountID:   account2.ID,
		Amount:        amount,
	})

	require.Error(t, err)
	require.ErrorIs(t, err, ErrInsufficientBalance)
	require.Empty(t, result.Transfer)
	require.Empty(t, result.FromEntry)
	require.Empty(t, result.ToEntry)

	updatedAccount1, getErr := testQueries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, getErr)
	require.Equal(t, account1.Balance, updatedAccount1.Balance)

	updatedAccount2, getErr := testQueries.GetAccount(context.Background(), account2.ID)
	require.NoError(t, getErr)
	require.Equal(t, account2.Balance, updatedAccount2.Balance)
}
