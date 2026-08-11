package db

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/faelic/monierave/db/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createRandomAccount(t *testing.T) Account {
	t.Helper()
	user := createRandomUser(t)
	account, err := testStore.CreateAccountTx(context.Background(), CreateAccountTxParams{
		Owner:    user.Username,
		Currency: util.RandomCurrency(),
	})
	require.NoError(t, err)
	require.Equal(t, user.Username, account.Owner)
	require.Zero(t, account.Balance)
	require.Equal(t, FinancialAccountStatusActive, account.Status)
	require.True(t, account.PublicID.Valid)
	require.Regexp(t, `^[0-9]{10}$`, account.AccountNumber)
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

	byNumber, err := testQueries.GetAccountByAccountNumber(
		context.Background(),
		account.AccountNumber,
	)
	require.NoError(t, err)
	require.Equal(t, account, byNumber)
}

func TestResolveReceivableAccountLifecycle(t *testing.T) {
	account := createRandomAccount(t)
	user, err := testQueries.GetUser(context.Background(), account.Owner)
	require.NoError(t, err)

	resolved, err := testQueries.ResolveReceivableAccount(
		context.Background(),
		account.AccountNumber,
	)
	require.NoError(t, err)
	require.Equal(t, account.AccountNumber, resolved.AccountNumber)
	require.Equal(t, user.FullName, resolved.AccountName)
	require.Equal(t, account.Currency, resolved.Currency)

	frozen, err := testStore.SetAccountStatusTx(context.Background(), AccountStatusTxParams{
		AccountPublicID: account.PublicID,
		TargetStatus:    FinancialAccountStatusFrozen,
		Actor:           "test-operator",
		Reason:          "verify recipient resolution lifecycle",
		CorrelationID: pgtype.UUID{
			Bytes: uuid.New(), Valid: true,
		},
	})
	require.NoError(t, err)
	require.Equal(t, FinancialAccountStatusFrozen, frozen.Account.Status)
	_, err = testQueries.ResolveReceivableAccount(
		context.Background(),
		account.AccountNumber,
	)
	require.NoError(t, err)

	_, err = testStore.CloseAccountTx(context.Background(), CloseAccountTxParams{
		PublicID: account.PublicID,
		Username: account.Owner,
	})
	require.NoError(t, err)
	_, err = testQueries.ResolveReceivableAccount(
		context.Background(),
		account.AccountNumber,
	)
	require.ErrorIs(t, err, pgx.ErrNoRows)
	_, err = testQueries.ResolveReceivableAccount(
		context.Background(),
		"9999999999",
	)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestGenerateAccountNumber(t *testing.T) {
	const samples = 1_000
	numbers := make(map[string]struct{}, samples)
	format := regexp.MustCompile(`^[0-9]{10}$`)

	for range samples {
		accountNumber, err := generateAccountNumber()
		require.NoError(t, err)
		require.True(t, format.MatchString(accountNumber))
		_, duplicate := numbers[accountNumber]
		require.False(t, duplicate)
		numbers[accountNumber] = struct{}{}
	}
}

func TestCreateAccountRetriesAccountNumberCollision(t *testing.T) {
	generated := []string{"1111111111", "2222222222"}
	generateCalls := 0
	createCalls := 0
	expected := Account{AccountNumber: generated[1]}

	account, err := createAccountWithGeneratedNumber(
		func() (string, error) {
			value := generated[generateCalls]
			generateCalls++
			return value, nil
		},
		func(accountNumber string) (Account, error) {
			createCalls++
			if accountNumber == generated[0] {
				return Account{}, &pgconn.PgError{
					ConstraintName: "accounts_account_number_key",
				}
			}
			return expected, nil
		},
	)

	require.NoError(t, err)
	require.Equal(t, expected, account)
	require.Equal(t, 2, generateCalls)
	require.Equal(t, 2, createCalls)
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
	account1, err := testStore.CreateAccountTx(context.Background(), CreateAccountTxParams{
		Owner: user.Username, Currency: "USD",
	})
	require.NoError(t, err)
	account2, err := testStore.CreateAccountTx(context.Background(), CreateAccountTxParams{
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
	require.Equal(t, account.AccountNumber, closed.AccountNumber)
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
