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
	user := createRandomUser(t)

	args := CreateAccountParams{
		Owner:    user.Username,
		Balance:  util.RandomMoney(),
		Currency: util.RandomCurrency(),
	}

	account, err := testQueries.CreateAccount(context.Background(), args)
	require.NoError(t, err)
	require.NotEmpty(t, account)

	require.Equal(t, args.Owner, account.Owner)
	require.Equal(t, args.Balance, account.Balance)
	require.Equal(t, args.Currency, account.Currency)

	require.NotZero(t, account.ID)
	require.NotZero(t, account.CreatedAt)

	return account
}
func TestCreateAccount(t *testing.T) {
	createRandomAccount(t)
}

func TestGetAccount(t *testing.T) {
	account1 := createRandomAccount(t)
	account2, err := testQueries.GetAccount(context.Background(), account1.ID)

	require.NoError(t, err)
	require.NotEmpty(t, account2)

	require.Equal(t, account1.ID, account2.ID)
	require.Equal(t, account1.Owner, account2.Owner)
	require.Equal(t, account1.Balance, account2.Balance)
	require.Equal(t, account1.Currency, account2.Currency)
	require.WithinDuration(t, account1.CreatedAt.Time, account2.CreatedAt.Time, time.Second)
}

func TestUpdateAccount(t *testing.T) {
	account1 := createRandomAccount(t)

	args := UpdateAccountParams{
		ID:      account1.ID,
		Balance: util.RandomMoney(),
	}

	account2, err := testQueries.UpdateAccount(context.Background(), args)
	require.NoError(t, err)
	require.NotEmpty(t, account2)

	require.Equal(t, account1.ID, account2.ID)
	require.Equal(t, account1.Owner, account2.Owner)
	require.Equal(t, args.Balance, account2.Balance)
	require.Equal(t, account1.Currency, account2.Currency)
	require.WithinDuration(t, account1.CreatedAt.Time, account2.CreatedAt.Time, time.Second)
}

func TestDeleteAccount(t *testing.T) {
	account1 := createRandomAccount(t)
	err := testQueries.DeleteAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	account2, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.Error(t, err)
	require.Empty(t, account2)
}

func TestListAccount(t *testing.T) {
	user := createRandomUser(t)

	account1, err := testQueries.CreateAccount(context.Background(), CreateAccountParams{
		Owner:    user.Username,
		Balance:  util.RandomMoney(),
		Currency: "USD",
	})
	require.NoError(t, err)

	account2, err := testQueries.CreateAccount(context.Background(), CreateAccountParams{
		Owner:    user.Username,
		Balance:  util.RandomMoney(),
		Currency: "EUR",
	})
	require.NoError(t, err)

	args := ListAccountParams{
		Owner:  user.Username,
		Limit:  5,
		Offset: 0,
	}

	accounts, err := testQueries.ListAccount(context.Background(), args)
	require.NoError(t, err)
	require.Len(t, accounts, 2)

	require.Contains(t, accounts, account1)
	require.Contains(t, accounts, account2)

	for _, account := range accounts {
		require.NotEmpty(t, account)
		require.Equal(t, user.Username, account.Owner)
	}
}

func TestAddAccountBalance(t *testing.T) {
	account1 := createRandomAccount(t)

	addedAmount := int64(50)
	account2, err := testQueries.AddAccountBalance(context.Background(), AddAccountBalanceParams{
		ID:     account1.ID,
		Amount: addedAmount,
	})
	require.NoError(t, err)
	require.Equal(t, account1.Balance+addedAmount, account2.Balance)

	subtractedAmount := int64(-20)
	account3, err := testQueries.AddAccountBalance(context.Background(), AddAccountBalanceParams{
		ID:     account1.ID,
		Amount: subtractedAmount,
	})
	require.NoError(t, err)
	require.Equal(t, account2.Balance+subtractedAmount, account3.Balance)
}

func TestAddAccountBalanceInsufficientBalance(t *testing.T) {
	account := createRandomAccount(t)

	_, err := testQueries.AddAccountBalance(context.Background(), AddAccountBalanceParams{
		ID:     account.ID,
		Amount: -(account.Balance + 1),
	})
	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	updatedAccount, getErr := testQueries.GetAccount(context.Background(), account.ID)
	require.NoError(t, getErr)
	require.Equal(t, account.Balance, updatedAccount.Balance)
}

func TestGetAccountForUpdate(t *testing.T) {
	account1 := createRandomAccount(t)
	account2, err := testQueries.GetAccountForUpdate(context.Background(), account1.ID)

	require.NoError(t, err)
	require.NotEmpty(t, account2)
	require.Equal(t, account1.ID, account2.ID)
	require.Equal(t, account1.Owner, account2.Owner)
	require.Equal(t, account1.Balance, account2.Balance)
	require.Equal(t, account1.Currency, account2.Currency)
	require.WithinDuration(t, account1.CreatedAt.Time, account2.CreatedAt.Time, time.Second)
}
