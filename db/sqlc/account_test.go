package db

import (
	"context"
	"testing"
	"time"

	"github.com/faelic/monierave/db/util"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func createRandomAccount(t *testing.T) Account {
	t.Helper()
	user := createRandomUser(t)
	account, err := testStore.CreateAccountTx(context.Background(), CreateAccountParams{
		Owner:    user.Username,
		Currency: util.RandomCurrency(),
	})
	require.NoError(t, err)
	require.Equal(t, user.Username, account.Owner)
	require.Zero(t, account.Balance)
	require.Equal(t, FinancialAccountStatusActive, account.Status)
	require.True(t, account.PublicID.Valid)
	require.True(t, account.CreatedAt.Valid)
	require.True(t, account.UpdatedAt.Valid)
	require.False(t, account.ClosedAt.Valid)
	return account
}

func fundAccount(t *testing.T, account Account, amount int64) Account {
	t.Helper()
	funded, err := testStore.DepositTx(context.Background(), DepositTxParams{
		AccountPublicID: account.PublicID,
		Amount:          amount,
		Narration:       "Test funding",
	})
	require.NoError(t, err)
	return funded.Account
}

func TestCreateAndGetAccountByPublicID(t *testing.T) {
	account := createRandomAccount(t)

	got, err := testQueries.GetAccountByPublicID(context.Background(), account.PublicID)
	require.NoError(t, err)
	require.Equal(t, account, got)

	owned, err := testQueries.GetOwnedAccountByPublicID(
		context.Background(),
		GetOwnedAccountByPublicIDParams{
			PublicID: account.PublicID,
			Owner:    account.Owner,
		},
	)
	require.NoError(t, err)
	require.Equal(t, account, owned)
}

func TestOwnedAccountLookupHidesForeignAccount(t *testing.T) {
	account := createRandomAccount(t)
	_, err := testQueries.GetOwnedAccountByPublicID(
		context.Background(),
		GetOwnedAccountByPublicIDParams{
			PublicID: account.PublicID,
			Owner:    util.RandomOwner(),
		},
	)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestListAccount(t *testing.T) {
	user := createRandomUser(t)
	account1, err := testStore.CreateAccountTx(context.Background(), CreateAccountParams{
		Owner: user.Username, Currency: "USD",
	})
	require.NoError(t, err)
	account2, err := testStore.CreateAccountTx(context.Background(), CreateAccountParams{
		Owner: user.Username, Currency: "EUR",
	})
	require.NoError(t, err)

	accounts, err := testQueries.ListAccount(context.Background(), ListAccountParams{
		Owner: user.Username, Limit: 5, Offset: 0,
	})
	require.NoError(t, err)
	require.Equal(t, []Account{account1, account2}, accounts)
}

func TestAddAccountBalance(t *testing.T) {
	account := createRandomAccount(t)
	updated := fundAccount(t, account, 50)
	require.Equal(t, int64(50), updated.Balance)
	require.True(t, updated.UpdatedAt.Time.After(account.UpdatedAt.Time) ||
		updated.UpdatedAt.Time.Equal(account.UpdatedAt.Time))

	updated, err := testQueries.AddAccountBalanceInternal(context.Background(), AddAccountBalanceInternalParams{
		ID: updated.ID, Amount: -20,
	})
	require.NoError(t, err)
	require.Equal(t, int64(30), updated.Balance)
	_, err = testQueries.AddAccountBalanceInternal(context.Background(), AddAccountBalanceInternalParams{
		ID: updated.ID, Amount: 20,
	})
	require.NoError(t, err)
}

func TestAddAccountBalanceRejectsOverdraft(t *testing.T) {
	account := fundAccount(t, createRandomAccount(t), 25)
	_, err := testQueries.AddAccountBalanceInternal(context.Background(), AddAccountBalanceInternalParams{
		ID: account.ID, Amount: -26,
	})
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestCloseAccountTx(t *testing.T) {
	account := createRandomAccount(t)
	closed, err := testStore.CloseAccountTx(context.Background(), CloseAccountTxParams{
		PublicID: account.PublicID,
		Username: account.Owner,
	})
	require.NoError(t, err)
	require.Equal(t, FinancialAccountStatusClosed, closed.Status)
	require.True(t, closed.ClosedAt.Valid)
	require.WithinDuration(t, time.Now(), closed.ClosedAt.Time, time.Second)

	_, err = testStore.CloseAccountTx(context.Background(), CloseAccountTxParams{
		PublicID: account.PublicID,
		Username: account.Owner,
	})
	require.ErrorIs(t, err, ErrAccountClosed)
}

func TestCloseAccountTxRejectsNonZeroBalanceAndForeignOwner(t *testing.T) {
	account := fundAccount(t, createRandomAccount(t), 100)

	_, err := testStore.CloseAccountTx(context.Background(), CloseAccountTxParams{
		PublicID: account.PublicID,
		Username: account.Owner,
	})
	require.ErrorIs(t, err, ErrAccountBalanceNotZero)

	_, err = testStore.CloseAccountTx(context.Background(), CloseAccountTxParams{
		PublicID: account.PublicID,
		Username: util.RandomOwner(),
	})
	require.ErrorIs(t, err, ErrAccountNotOwned)
}
