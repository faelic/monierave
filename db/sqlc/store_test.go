package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/faelic/monierave/db/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createTransferTestAccount(t *testing.T) Account {
	user := createRandomUser(t)

	account, err := testStore.CreateAccountTx(context.Background(), CreateAccountParams{
		Owner:    user.Username,
		Currency: "USD",
	})
	require.NoError(t, err)
	return fundAccount(t, account, util.RandomMoney())
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
	require.Equal(t, EmailDeliverabilityPending, result.User.EmailDeliverabilityStatus)
	require.Equal(t, AccountStatusPending, result.User.AccountStatus)
	require.True(t, result.User.RegistrationExpiresAt.Valid)
	require.WithinDuration(
		t,
		time.Now().Add(7*24*time.Hour),
		result.User.RegistrationExpiresAt.Time,
		time.Minute,
	)
	require.Equal(t, EmailJobTypeVerifyEmail, result.EmailJob.JobType)
	require.Equal(t, arg.Username, result.EmailJob.Username)
	require.Equal(t, arg.Email, result.EmailJob.Recipient)
	require.Equal(t, "pending", result.EmailJob.Status)
	require.Equal(t, EmailDeliveryStatusPending, result.EmailJob.DeliveryStatus)
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

func TestVerifyUserEmailTxActivatesMatchingCurrentAddress(t *testing.T) {
	created := createUserWithEmailJob(t)

	user, err := testStore.VerifyUserEmailTx(context.Background(), VerifyUserEmailTxParams{
		Username: created.User.Username,
		Email:    created.User.Email,
		JobID:    created.EmailJob.ID,
	})
	require.NoError(t, err)
	require.Equal(t, AccountStatusActive, user.AccountStatus)
	require.True(t, user.EmailVerifiedAt.Valid)
	require.Equal(t, EmailDeliverabilityDeliverable, user.EmailDeliverabilityStatus)
	require.False(t, user.RegistrationExpiresAt.Valid)

	auditLogs, err := testQueries.ListAuditLogsByJob(
		context.Background(),
		ListAuditLogsByJobParams{EntityID: created.EmailJob.ID, Limit: 30},
	)
	require.NoError(t, err)
	requireAuditEvent(t, auditLogs, "email_verified")
}

func TestVerifyUserEmailTxRejectsOldAddress(t *testing.T) {
	created := createUserWithEmailJob(t)
	_, err := testStore.UpdateUserTx(context.Background(), UpdateUserTxParams{
		UpdateUserParams: UpdateUserParams{
			Username: created.User.Username,
			Email:    pgtype.Text{String: util.RandomEmail(), Valid: true},
		},
	})
	require.NoError(t, err)

	_, err = testStore.VerifyUserEmailTx(context.Background(), VerifyUserEmailTxParams{
		Username: created.User.Username,
		Email:    created.User.Email,
		JobID:    created.EmailJob.ID,
	})
	require.ErrorIs(t, err, ErrEmailVerificationAddressStale)
}

func TestRequestEmailVerificationTxCooldownAndDisabledRecovery(t *testing.T) {
	created := createUserWithEmailJob(t)

	_, err := testStore.RequestEmailVerificationTx(
		context.Background(),
		created.User.Username,
	)
	require.ErrorIs(t, err, ErrEmailVerificationCooldown)

	_, err = testDB.Exec(
		context.Background(),
		`UPDATE email_jobs SET created_at = now() - interval '2 minutes' WHERE id = $1`,
		created.EmailJob.ID,
	)
	require.NoError(t, err)
	_, err = testDB.Exec(
		context.Background(),
		`UPDATE users
		 SET account_status = 'disabled', registration_expires_at = now() - interval '1 day'
		 WHERE username = $1`,
		created.User.Username,
	)
	require.NoError(t, err)

	result, err := testStore.RequestEmailVerificationTx(
		context.Background(),
		created.User.Username,
	)
	require.NoError(t, err)
	require.Equal(t, AccountStatusPending, result.User.AccountStatus)
	require.True(t, result.User.RegistrationExpiresAt.Time.After(time.Now()))
	require.NotEqual(t, created.EmailJob.ID, result.EmailJob.ID)
	require.Equal(t, result.EmailJob.ID, result.OutboxEvent.EmailJobID)
}

func TestRequestEmailVerificationTxDoesNotExtendPendingDeadline(t *testing.T) {
	created := createUserWithEmailJob(t)
	originalExpiry := created.User.RegistrationExpiresAt.Time
	_, err := testDB.Exec(
		context.Background(),
		`UPDATE email_jobs SET created_at = now() - interval '2 minutes' WHERE id = $1`,
		created.EmailJob.ID,
	)
	require.NoError(t, err)

	result, err := testStore.RequestEmailVerificationTx(
		context.Background(),
		created.User.Username,
	)
	require.NoError(t, err)
	require.WithinDuration(
		t,
		originalExpiry,
		result.User.RegistrationExpiresAt.Time,
		time.Second,
	)
}

func TestDisableExpiredPendingUser(t *testing.T) {
	created := createUserWithEmailJob(t)
	_, err := testDB.Exec(
		context.Background(),
		`UPDATE users SET registration_expires_at = now() - interval '1 second' WHERE username = $1`,
		created.User.Username,
	)
	require.NoError(t, err)

	user, err := testQueries.DisableExpiredPendingUser(
		context.Background(),
		created.User.Username,
	)
	require.NoError(t, err)
	require.Equal(t, AccountStatusDisabled, user.AccountStatus)
}

func TestUpdateUserTxCreatesVerificationJobWhenEmailChanges(t *testing.T) {
	created := createUserWithEmailJob(t)
	newEmail := util.RandomEmail()

	result, err := testStore.UpdateUserTx(context.Background(), UpdateUserTxParams{
		UpdateUserParams: UpdateUserParams{
			Username: created.User.Username,
			Email:    pgtype.Text{String: newEmail, Valid: true},
		},
	})
	require.NoError(t, err)
	require.True(t, result.EmailChanged)
	require.Equal(t, newEmail, result.User.Email)
	require.Equal(t, EmailDeliverabilityPending, result.User.EmailDeliverabilityStatus)
	require.False(t, result.User.EmailVerifiedAt.Valid)
	require.Equal(t, newEmail, result.EmailJob.Recipient)
	require.Equal(t, result.EmailJob.ID, result.OutboxEvent.EmailJobID)
}

func TestProcessPermanentBounceUpdatesJobAndCurrentUser(t *testing.T) {
	created := createUserWithEmailJob(t)
	providerMessageID := "resend-" + util.RandomString(12)
	markEmailJobAccepted(t, created.EmailJob.ID, providerMessageID)
	occurredAt := time.Now().UTC()

	arg := ProcessEmailDeliveryEventParams{
		WebhookID:         "webhook-" + util.RandomString(12),
		EventType:         "email.bounced",
		ProviderMessageID: providerMessageID,
		JobID:             created.EmailJob.ID,
		OccurredAt:        occurredAt,
		Payload:           []byte(`{"type":"email.bounced"}`),
		DeliveryStatus:    EmailDeliveryStatusBounced,
		BounceType:        "Permanent",
		BounceSubtype:     "General",
		BounceMessage:     "mailbox does not exist",
	}

	result, err := testStore.ProcessEmailDeliveryEventTx(context.Background(), arg)
	require.NoError(t, err)
	require.True(t, result.JobMatched)
	require.True(t, result.StateUpdated)
	require.True(t, result.UserStateUpdated)
	require.Equal(t, EmailDeliveryStatusBounced, result.EmailJob.DeliveryStatus)
	require.Equal(t, "Permanent", result.EmailJob.BounceType.String)
	require.Equal(t, "General", result.EmailJob.BounceSubtype.String)
	require.Equal(t, "mailbox does not exist", result.EmailJob.BounceMessage.String)

	user, err := testQueries.GetUser(context.Background(), created.User.Username)
	require.NoError(t, err)
	require.Equal(t, EmailDeliverabilityUndeliverable, user.EmailDeliverabilityStatus)
	require.WithinDuration(t, occurredAt, user.EmailBouncedAt.Time, time.Second)

	storedEvent, err := testQueries.GetEmailDeliveryEvent(context.Background(), arg.WebhookID)
	require.NoError(t, err)
	require.Equal(t, created.EmailJob.ID, storedEvent.EmailJobID)

	_, err = testDB.Exec(
		context.Background(),
		"UPDATE email_delivery_events SET event_type = 'tampered' WHERE webhook_id = $1",
		arg.WebhookID,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "append-only")

	auditLogs, err := testQueries.ListAuditLogsByJob(
		context.Background(),
		ListAuditLogsByJobParams{EntityID: created.EmailJob.ID, Limit: 30},
	)
	require.NoError(t, err)
	requireAuditEvent(t, auditLogs, "email.bounced")

	duplicate := arg
	duplicate.DeliveryStatus = EmailDeliveryStatusDelivered
	duplicate.OccurredAt = occurredAt.Add(time.Minute)
	duplicateResult, err := testStore.ProcessEmailDeliveryEventTx(context.Background(), duplicate)
	require.NoError(t, err)
	require.True(t, duplicateResult.Duplicate)

	job, err := testQueries.GetEmailJob(context.Background(), created.EmailJob.ID)
	require.NoError(t, err)
	require.Equal(t, EmailDeliveryStatusBounced, job.DeliveryStatus)
}

func TestProcessEmailDeliveryEventDoesNotRegressOrChangeNewAddress(t *testing.T) {
	created := createUserWithEmailJob(t)
	providerMessageID := "resend-" + util.RandomString(12)
	markEmailJobAccepted(t, created.EmailJob.ID, providerMessageID)

	changed, err := testStore.UpdateUserTx(context.Background(), UpdateUserTxParams{
		UpdateUserParams: UpdateUserParams{
			Username: created.User.Username,
			Email:    pgtype.Text{String: util.RandomEmail(), Valid: true},
		},
	})
	require.NoError(t, err)
	require.True(t, changed.EmailChanged)

	deliveredAt := time.Now().UTC()
	_, err = testStore.ProcessEmailDeliveryEventTx(
		context.Background(),
		ProcessEmailDeliveryEventParams{
			WebhookID:         "webhook-delivered-" + util.RandomString(8),
			EventType:         "email.delivered",
			ProviderMessageID: providerMessageID,
			JobID:             created.EmailJob.ID,
			OccurredAt:        deliveredAt,
			Payload:           []byte(`{"type":"email.delivered"}`),
			DeliveryStatus:    EmailDeliveryStatusDelivered,
		},
	)
	require.NoError(t, err)

	_, err = testStore.ProcessEmailDeliveryEventTx(
		context.Background(),
		ProcessEmailDeliveryEventParams{
			WebhookID:         "webhook-sent-" + util.RandomString(8),
			EventType:         "email.sent",
			ProviderMessageID: providerMessageID,
			JobID:             created.EmailJob.ID,
			OccurredAt:        deliveredAt.Add(-time.Minute),
			Payload:           []byte(`{"type":"email.sent"}`),
			DeliveryStatus:    EmailDeliveryStatusAccepted,
		},
	)
	require.NoError(t, err)

	job, err := testQueries.GetEmailJob(context.Background(), created.EmailJob.ID)
	require.NoError(t, err)
	require.Equal(t, EmailDeliveryStatusDelivered, job.DeliveryStatus)

	user, err := testQueries.GetUser(context.Background(), created.User.Username)
	require.NoError(t, err)
	require.Equal(t, changed.User.Email, user.Email)
	require.Equal(t, EmailDeliverabilityPending, user.EmailDeliverabilityStatus)
	require.False(t, user.EmailBouncedAt.Valid)
}

func createUserWithEmailJob(t *testing.T) CreateUserTxResult {
	t.Helper()
	hashedPassword, err := util.HashPassword(util.RandomString(8))
	require.NoError(t, err)

	result, err := testStore.CreateUserTx(context.Background(), CreateUserParams{
		Username:       util.RandomOwner(),
		HashedPassword: hashedPassword,
		FullName:       util.RandomOwner(),
		Email:          util.RandomEmail(),
	})
	require.NoError(t, err)
	return result
}

func markEmailJobAccepted(t *testing.T, jobID pgtype.UUID, providerMessageID string) {
	t.Helper()
	_, err := testQueries.MarkEmailJobSent(context.Background(), MarkEmailJobSentParams{
		ID:                jobID,
		ProviderMessageID: pgtype.Text{String: providerMessageID, Valid: true},
	})
	require.NoError(t, err)
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
			String: "provider-message-" + util.RandomString(12),
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
				FromAccountPublicID: account1.PublicID,
				ToAccountPublicID:   account2.PublicID,
				Amount:              amount,
				Currency:            account1.Currency,
				Username:            account1.Owner,
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

		transaction := result.Transaction
		require.Equal(t, BankingTransactionTypeInternalTransfer, transaction.TransactionType)
		require.Equal(t, BankingTransactionStatusPosted, transaction.Status)
		require.Equal(t, amount, transaction.Amount)
		require.True(t, transaction.ID.Valid)
		require.NotEmpty(t, transaction.Reference)
		require.True(t, transaction.PostedAt.Valid)

		stored, err := testQueries.GetBankingTransaction(context.Background(), transaction.ID)
		require.NoError(t, err)
		require.Equal(t, transaction.ID, stored.ID)

		require.Len(t, result.Postings, 2)
		require.Equal(t, -amount, result.Postings[0].Amount)
		require.Equal(t, amount, result.Postings[1].Amount)
		total, err := testQueries.GetLedgerPostingTotal(context.Background(), transaction.ID)
		require.NoError(t, err)
		require.Zero(t, total)

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
		fromAccount := account1
		toAccount := account2

		if i%2 == 1 {
			fromAccount = account2
			toAccount = account1
		}

		go func(fromAccount, toAccount Account) {
			ctx := context.Background()
			_, err := testStore.TransferTx(ctx, TransferTxParams{
				FromAccountPublicID: fromAccount.PublicID,
				ToAccountPublicID:   toAccount.PublicID,
				Amount:              amount,
				Currency:            fromAccount.Currency,
				Username:            fromAccount.Owner,
			})

			errs <- err
		}(fromAccount, toAccount)
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
		FromAccountPublicID: account1.PublicID,
		ToAccountPublicID:   account2.PublicID,
		Amount:              amount,
		Currency:            account1.Currency,
		Username:            account1.Owner,
	})

	require.Error(t, err)
	require.ErrorIs(t, err, ErrInsufficientBalance)
	require.Empty(t, result.Transaction)
	require.Empty(t, result.Postings)

	updatedAccount1, getErr := testQueries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, getErr)
	require.Equal(t, account1.Balance, updatedAccount1.Balance)

	updatedAccount2, getErr := testQueries.GetAccount(context.Background(), account2.ID)
	require.NoError(t, getErr)
	require.Equal(t, account2.Balance, updatedAccount2.Balance)
}

func TestTransferTxAccountLifecycle(t *testing.T) {
	testCases := []struct {
		name          string
		fromStatus    string
		toStatus      string
		expectedError error
	}{
		{
			name:          "FrozenSource",
			fromStatus:    FinancialAccountStatusFrozen,
			toStatus:      FinancialAccountStatusActive,
			expectedError: ErrAccountFrozen,
		},
		{
			name:       "FrozenDestinationMayReceive",
			fromStatus: FinancialAccountStatusActive,
			toStatus:   FinancialAccountStatusFrozen,
		},
		{
			name:          "ClosedSource",
			fromStatus:    FinancialAccountStatusClosed,
			toStatus:      FinancialAccountStatusActive,
			expectedError: ErrAccountClosed,
		},
		{
			name:          "ClosedDestination",
			fromStatus:    FinancialAccountStatusActive,
			toStatus:      FinancialAccountStatusClosed,
			expectedError: ErrAccountClosed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fromAccount := createTransferTestAccount(t)
			toAccount := createTransferTestAccount(t)
			var err error
			fromAccount = setTransferTestAccountStatus(t, fromAccount, tc.fromStatus)
			toAccount = setTransferTestAccountStatus(t, toAccount, tc.toStatus)

			result, err := testStore.TransferTx(context.Background(), TransferTxParams{
				FromAccountPublicID: fromAccount.PublicID,
				ToAccountPublicID:   toAccount.PublicID,
				Amount:              10,
				Currency:            fromAccount.Currency,
				Username:            fromAccount.Owner,
			})
			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError)
				require.Empty(t, result)
				return
			}
			require.NoError(t, err)
			require.Equal(t, int64(10), result.Transaction.Amount)
			require.Len(t, result.Postings, 2)
		})
	}
}

func setTransferTestAccountStatus(t *testing.T, account Account, status string) Account {
	t.Helper()
	if status == FinancialAccountStatusClosed {
		_, err := testStore.WithdrawTx(context.Background(), WithdrawTxParams{
			AccountPublicID: account.PublicID,
			Amount:          account.Balance,
			Narration:       "Prepare account closure in test",
		})
		require.NoError(t, err)
		closed, err := testStore.CloseAccountTx(context.Background(), CloseAccountTxParams{
			PublicID: account.PublicID,
			Username: account.Owner,
		})
		require.NoError(t, err)
		return closed
	}

	updated, err := testQueries.SetAccountStatus(context.Background(), SetAccountStatusParams{
		ID: account.ID, Status: status,
	})
	require.NoError(t, err)
	return updated
}

func TestCloseAccountTxSerializesWithIncomingTransfer(t *testing.T) {
	fromAccount := createTransferTestAccount(t)
	toUser := createRandomUser(t)
	toAccount, err := testStore.CreateAccountTx(context.Background(), CreateAccountParams{
		Owner: toUser.Username, Currency: fromAccount.Currency,
	})
	require.NoError(t, err)

	start := make(chan struct{})
	transferResult := make(chan error, 1)
	closeResult := make(chan error, 1)

	go func() {
		<-start
		_, err := testStore.TransferTx(context.Background(), TransferTxParams{
			FromAccountPublicID: fromAccount.PublicID,
			ToAccountPublicID:   toAccount.PublicID,
			Amount:              10,
			Currency:            fromAccount.Currency,
			Username:            fromAccount.Owner,
		})
		transferResult <- err
	}()
	go func() {
		<-start
		_, err := testStore.CloseAccountTx(context.Background(), CloseAccountTxParams{
			PublicID: toAccount.PublicID,
			Username: toAccount.Owner,
		})
		closeResult <- err
	}()

	close(start)
	transferErr := <-transferResult
	closeErr := <-closeResult

	if transferErr == nil {
		require.ErrorIs(t, closeErr, ErrAccountBalanceNotZero)
	} else {
		require.ErrorIs(t, transferErr, ErrAccountClosed)
		require.NoError(t, closeErr)
	}

	finalAccount, err := testQueries.GetAccount(context.Background(), toAccount.ID)
	require.NoError(t, err)
	if finalAccount.Status == FinancialAccountStatusClosed {
		require.Zero(t, finalAccount.Balance)
	} else {
		require.Equal(t, int64(10)+toAccount.Balance, finalAccount.Balance)
	}
}
