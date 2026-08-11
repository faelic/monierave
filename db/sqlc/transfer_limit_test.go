package db

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransferTxPerTransferLimit(t *testing.T) {
	fromAccount, toAccount := createLimitTestAccounts(t, "USD", 2_000_000)

	_, err := testStore.TransferTx(context.Background(), TransferTxParams{
		FromAccountPublicID: fromAccount.PublicID,
		ToAccountNumber:     toAccount.AccountNumber,
		Amount:              USDPerTransferLimit + 1,
		Currency:            "USD",
		Username:            fromAccount.Owner,
	})
	require.ErrorIs(t, err, ErrPerTransferLimitExceeded)

	stored, err := testQueries.GetAccount(context.Background(), fromAccount.ID)
	require.NoError(t, err)
	require.Equal(t, fromAccount.Balance, stored.Balance)
}

func TestTransferTxDailyLimitBoundary(t *testing.T) {
	fromAccount, toAccount := createLimitTestAccounts(t, "USD", 4_000_000)
	for _, amount := range []int64{1_000_000, 1_000_000, 500_000} {
		_, err := testStore.TransferTx(context.Background(), TransferTxParams{
			FromAccountPublicID: fromAccount.PublicID,
			ToAccountNumber:     toAccount.AccountNumber,
			Amount:              amount,
			Currency:            "USD",
			Username:            fromAccount.Owner,
		})
		require.NoError(t, err)
	}

	fromLedger, err := customerLedgerAccount(
		context.Background(),
		testQueries,
		fromAccount.ID,
	)
	require.NoError(t, err)
	dailyTotal, err := testQueries.GetDailyOutgoingTransferTotal(
		context.Background(),
		fromLedger.ID,
	)
	require.NoError(t, err)
	require.Equal(t, USDDailyLimit, dailyTotal)

	_, err = testStore.TransferTx(context.Background(), TransferTxParams{
		FromAccountPublicID: fromAccount.PublicID,
		ToAccountNumber:     toAccount.AccountNumber,
		Amount:              1,
		Currency:            "USD",
		Username:            fromAccount.Owner,
	})
	require.ErrorIs(t, err, ErrDailyTransferLimitExceeded)
}

func TestConcurrentTransfersCannotExceedDailyLimit(t *testing.T) {
	fromAccount, toAccount := createLimitTestAccounts(t, "USD", 5_000_000)

	const workers = 3
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := testStore.TransferTx(context.Background(), TransferTxParams{
				FromAccountPublicID: fromAccount.PublicID,
				ToAccountNumber:     toAccount.AccountNumber,
				Amount:              1_000_000,
				Currency:            "USD",
				Username:            fromAccount.Owner,
			})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	succeeded := 0
	rejected := 0
	for err := range errs {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrDailyTransferLimitExceeded):
			rejected++
		default:
			require.NoError(t, err)
		}
	}
	require.Equal(t, 2, succeeded)
	require.Equal(t, 1, rejected)

	storedFrom, err := testQueries.GetAccount(context.Background(), fromAccount.ID)
	require.NoError(t, err)
	require.Equal(t, int64(3_000_000), storedFrom.Balance)
	storedTo, err := testQueries.GetAccount(context.Background(), toAccount.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2_000_000), storedTo.Balance)
}

func TestTransferLimitsSupportEUR(t *testing.T) {
	fromAccount, toAccount := createLimitTestAccounts(t, "EUR", 1_500_000)
	_, err := testStore.TransferTx(context.Background(), TransferTxParams{
		FromAccountPublicID: fromAccount.PublicID,
		ToAccountNumber:     toAccount.AccountNumber,
		Amount:              EURPerTransferLimit,
		Currency:            "EUR",
		Username:            fromAccount.Owner,
	})
	require.NoError(t, err)
}

func createLimitTestAccounts(
	t *testing.T,
	currency string,
	funding int64,
) (Account, Account) {
	t.Helper()
	fromUser := createRandomUser(t)
	fromAccount, err := testStore.CreateAccountTx(
		context.Background(),
		CreateAccountTxParams{Owner: fromUser.Username, Currency: currency},
	)
	require.NoError(t, err)
	deposit, err := testStore.DepositTx(context.Background(), DepositTxParams{
		AccountPublicID: fromAccount.PublicID,
		Amount:          funding,
		Narration:       "Transfer limit test funding",
	})
	require.NoError(t, err)

	toUser := createRandomUser(t)
	toAccount, err := testStore.CreateAccountTx(
		context.Background(),
		CreateAccountTxParams{Owner: toUser.Username, Currency: currency},
	)
	require.NoError(t, err)
	return deposit.Account, toAccount
}
