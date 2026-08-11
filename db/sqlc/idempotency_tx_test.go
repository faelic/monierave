package db

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/faelic/monierave/db/util"
	"github.com/stretchr/testify/require"
)

func TestIdempotentTransferTxReplaysOriginalResult(t *testing.T) {
	fromAccount := createTransferTestAccount(t)
	toAccount := createTransferTestAccount(t)
	amount := int64(25)
	arg := idempotentTransferTestParams(fromAccount, toAccount, amount, "same-request")

	first, err := testStore.IdempotentTransferTx(context.Background(), arg)
	require.NoError(t, err)
	require.False(t, first.Replayed)

	second, err := testStore.IdempotentTransferTx(context.Background(), arg)
	require.NoError(t, err)
	require.True(t, second.Replayed)
	require.Equal(t, first.Transaction.ID, second.Transaction.ID)
	require.Equal(t, first.Transaction.Reference, second.Transaction.Reference)
	require.Equal(t, first.FromAccount.Balance, second.FromAccount.Balance)
	require.Equal(t, first.ToAccount.Balance, second.ToAccount.Balance)

	updatedFrom, err := testQueries.GetAccount(context.Background(), fromAccount.ID)
	require.NoError(t, err)
	require.Equal(t, fromAccount.Balance-amount, updatedFrom.Balance)

	count, err := testQueries.CountIdempotencyKeysByTransaction(
		context.Background(),
		first.Transaction.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	postings, err := testQueries.ListLedgerPostingsByTransaction(
		context.Background(),
		first.Transaction.ID,
	)
	require.NoError(t, err)
	require.Len(t, postings, 2)
}

func TestIdempotentTransferTxRejectsChangedPayload(t *testing.T) {
	fromAccount := createTransferTestAccount(t)
	toAccount := createTransferTestAccount(t)
	firstArg := idempotentTransferTestParams(fromAccount, toAccount, 20, "first-payload")

	first, err := testStore.IdempotentTransferTx(context.Background(), firstArg)
	require.NoError(t, err)

	changedArg := firstArg
	changedArg.Amount = 30
	changedArg.RequestHash = testRequestHash("changed-payload")
	_, err = testStore.IdempotentTransferTx(context.Background(), changedArg)
	require.ErrorIs(t, err, ErrIdempotencyConflict)

	updatedFrom, err := testQueries.GetAccount(context.Background(), fromAccount.ID)
	require.NoError(t, err)
	require.Equal(t, fromAccount.Balance-firstArg.Amount, updatedFrom.Balance)

	count, err := testQueries.CountIdempotencyKeysByTransaction(
		context.Background(),
		first.Transaction.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestIdempotentTransferTxConcurrentRetriesPostOnce(t *testing.T) {
	fromAccount := createTransferTestAccount(t)
	toAccount := createTransferTestAccount(t)
	amount := int64(15)
	arg := idempotentTransferTestParams(fromAccount, toAccount, amount, "concurrent")

	const workers = 8
	results := make(chan TransferTxResult, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := testStore.IdempotentTransferTx(context.Background(), arg)
			results <- result
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}

	var transactionID [16]byte
	originals := 0
	for result := range results {
		if transactionID == ([16]byte{}) {
			transactionID = result.Transaction.ID.Bytes
		}
		require.Equal(t, transactionID, result.Transaction.ID.Bytes)
		if !result.Replayed {
			originals++
		}
	}
	require.Equal(t, 1, originals)

	updatedFrom, err := testQueries.GetAccount(context.Background(), fromAccount.ID)
	require.NoError(t, err)
	require.Equal(t, fromAccount.Balance-amount, updatedFrom.Balance)
}

func TestIdempotentTransferTxFailureDoesNotConsumeKey(t *testing.T) {
	fromAccount := createTransferTestAccount(t)
	toAccount := createTransferTestAccount(t)
	key := "retry-after-failure-" + util.RandomString(12)
	failing := idempotentTransferTestParams(
		fromAccount,
		toAccount,
		fromAccount.Balance+1,
		"will-fail",
	)
	failing.IdempotencyKey = key

	_, err := testStore.IdempotentTransferTx(context.Background(), failing)
	require.ErrorIs(t, err, ErrInsufficientBalance)

	retry := idempotentTransferTestParams(fromAccount, toAccount, 10, "valid-retry")
	retry.IdempotencyKey = key
	result, err := testStore.IdempotentTransferTx(context.Background(), retry)
	require.NoError(t, err)
	require.False(t, result.Replayed)
}

func TestIdempotentTransferLimitFailureDoesNotConsumeKey(t *testing.T) {
	fromAccount, toAccount := createLimitTestAccounts(t, "USD", 2_000_000)
	key := "retry-after-limit-" + util.RandomString(12)
	failing := idempotentTransferTestParams(
		fromAccount,
		toAccount,
		USDPerTransferLimit+1,
		"limit-failure",
	)
	failing.IdempotencyKey = key

	_, err := testStore.IdempotentTransferTx(context.Background(), failing)
	require.ErrorIs(t, err, ErrPerTransferLimitExceeded)

	retry := idempotentTransferTestParams(
		fromAccount,
		toAccount,
		100,
		"valid-after-limit",
	)
	retry.IdempotencyKey = key
	result, err := testStore.IdempotentTransferTx(context.Background(), retry)
	require.NoError(t, err)
	require.False(t, result.Replayed)
}

func TestIdempotentTransferTxExpiredKeyCanBeReused(t *testing.T) {
	fromAccount := createTransferTestAccount(t)
	toAccount := createTransferTestAccount(t)
	firstArg := idempotentTransferTestParams(fromAccount, toAccount, 10, "before-expiry")

	first, err := testStore.IdempotentTransferTx(context.Background(), firstArg)
	require.NoError(t, err)

	_, err = testDB.Exec(
		context.Background(),
		`UPDATE idempotency_keys SET created_at = $1, expires_at = $2
		 WHERE username = $3 AND operation = $4 AND idempotency_key = $5`,
		time.Now().Add(-25*time.Hour),
		time.Now().Add(-time.Hour),
		firstArg.Username,
		IdempotencyOperationInternalTransfer,
		firstArg.IdempotencyKey,
	)
	require.NoError(t, err)

	secondArg := firstArg
	secondArg.Amount = 12
	secondArg.RequestHash = testRequestHash("after-expiry")
	second, err := testStore.IdempotentTransferTx(context.Background(), secondArg)
	require.NoError(t, err)
	require.False(t, second.Replayed)
	require.NotEqual(t, first.Transaction.ID, second.Transaction.ID)
}

func idempotentTransferTestParams(
	fromAccount Account,
	toAccount Account,
	amount int64,
	hashSeed string,
) IdempotentTransferTxParams {
	return IdempotentTransferTxParams{
		TransferTxParams: TransferTxParams{
			FromAccountPublicID: fromAccount.PublicID,
			ToAccountNumber:     toAccount.AccountNumber,
			Amount:              amount,
			Currency:            fromAccount.Currency,
			Username:            fromAccount.Owner,
			Narration:           "Idempotency integration test",
		},
		IdempotencyKey: fmt.Sprintf(
			"transfer-%s-%s",
			hashSeed,
			util.RandomString(12),
		),
		RequestHash: testRequestHash(hashSeed),
	}
}

func testRequestHash(value string) []byte {
	hash := sha256.Sum256([]byte(value))
	return hash[:]
}
