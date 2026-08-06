package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestCreateAccountTxCreatesCustomerLedgerAccount(t *testing.T) {
	account := createRandomAccount(t)
	ledger, err := customerLedgerAccount(context.Background(), testQueries, account.ID)
	require.NoError(t, err)
	require.Equal(t, "customer", ledger.Kind)
	require.Equal(t, account.Currency, ledger.Currency)
	require.Equal(t, account.ID, ledger.CustomerAccountID.Int64)
}

func TestDepositAndWithdrawalUseBalancedPostings(t *testing.T) {
	account := createRandomAccount(t)

	deposit, err := testStore.DepositTx(context.Background(), DepositTxParams{
		AccountPublicID: account.PublicID,
		Amount:          1_000,
		Narration:       "Initial test funding",
	})
	require.NoError(t, err)
	require.Equal(t, BankingTransactionTypeDeposit, deposit.Transaction.TransactionType)
	require.Equal(t, BankingTransactionStatusPosted, deposit.Transaction.Status)
	require.Equal(t, int64(1_000), deposit.Account.Balance)
	requireBalancedPostings(t, deposit.Transaction, deposit.Postings)

	withdrawal, err := testStore.WithdrawTx(context.Background(), WithdrawTxParams{
		AccountPublicID: account.PublicID,
		Amount:          250,
		Narration:       "ATM simulation",
	})
	require.NoError(t, err)
	require.Equal(t, BankingTransactionTypeWithdrawal, withdrawal.Transaction.TransactionType)
	require.Equal(t, int64(750), withdrawal.Account.Balance)
	requireBalancedPostings(t, withdrawal.Transaction, withdrawal.Postings)

	ledger, err := customerLedgerAccount(context.Background(), testQueries, account.ID)
	require.NoError(t, err)
	ledgerBalance, err := testQueries.GetLedgerAccountBalance(context.Background(), ledger.ID)
	require.NoError(t, err)
	require.Equal(t, withdrawal.Account.Balance, ledgerBalance)
}

func TestWithdrawalRollbackOnInsufficientBalance(t *testing.T) {
	account := createRandomAccount(t)
	_, err := testStore.WithdrawTx(context.Background(), WithdrawTxParams{
		AccountPublicID: account.PublicID,
		Amount:          1,
		Narration:       "Should roll back",
	})
	require.ErrorIs(t, err, ErrInsufficientBalance)

	stored, err := testQueries.GetAccount(context.Background(), account.ID)
	require.NoError(t, err)
	require.Zero(t, stored.Balance)
	ledger, err := customerLedgerAccount(context.Background(), testQueries, account.ID)
	require.NoError(t, err)
	ledgerBalance, err := testQueries.GetLedgerAccountBalance(context.Background(), ledger.ID)
	require.NoError(t, err)
	require.Zero(t, ledgerBalance)
}

func TestFrozenAccountCanDepositButCannotWithdraw(t *testing.T) {
	account := createRandomAccount(t)
	account, err := testQueries.SetAccountStatus(context.Background(), SetAccountStatusParams{
		ID: account.ID, Status: FinancialAccountStatusFrozen,
	})
	require.NoError(t, err)

	_, err = testStore.DepositTx(context.Background(), DepositTxParams{
		AccountPublicID: account.PublicID, Amount: 100,
	})
	require.NoError(t, err)
	_, err = testStore.WithdrawTx(context.Background(), WithdrawTxParams{
		AccountPublicID: account.PublicID, Amount: 10,
	})
	require.ErrorIs(t, err, ErrAccountFrozen)
}

func TestLedgerRecordsAreImmutable(t *testing.T) {
	account := createRandomAccount(t)
	result, err := testStore.DepositTx(context.Background(), DepositTxParams{
		AccountPublicID: account.PublicID, Amount: 100,
	})
	require.NoError(t, err)

	_, err = testDB.Exec(
		context.Background(),
		"UPDATE ledger_postings SET amount = amount + 1 WHERE id = $1",
		result.Postings[0].ID,
	)
	require.Error(t, err)

	_, err = testDB.Exec(
		context.Background(),
		"DELETE FROM banking_transactions WHERE id = $1",
		result.Transaction.ID,
	)
	require.Error(t, err)

	_, err = testDB.Exec(
		context.Background(),
		"UPDATE banking_transactions SET amount = amount + 1 WHERE id = $1",
		result.Transaction.ID,
	)
	require.Error(t, err)
}

func TestDatabaseRejectsUnbalancedPostedTransaction(t *testing.T) {
	account := createRandomAccount(t)
	ledger, err := customerLedgerAccount(context.Background(), testQueries, account.ID)
	require.NoError(t, err)

	tx, err := testDB.Begin(context.Background())
	require.NoError(t, err)
	q := New(tx)
	id, reference := newTransactionIdentity()
	transaction, err := q.CreateBankingTransaction(
		context.Background(),
		CreateBankingTransactionParams{
			ID:              id,
			Reference:       reference,
			TransactionType: BankingTransactionTypeDeposit,
			Currency:        account.Currency,
			Amount:          100,
			Narration:       "Invalid one-sided transaction",
			InitiatedBy:     "test",
			ReversalOf:      pgtype.UUID{},
		},
	)
	require.NoError(t, err)
	_, err = q.CreateLedgerPosting(context.Background(), CreateLedgerPostingParams{
		TransactionID: transaction.ID, LedgerAccountID: ledger.ID, Amount: 100,
	})
	require.NoError(t, err)
	_, err = q.MarkBankingTransactionPosted(context.Background(), transaction.ID)
	require.NoError(t, err)
	require.Error(t, tx.Commit(context.Background()))
}

func requireBalancedPostings(
	t *testing.T,
	transaction BankingTransaction,
	postings []LedgerPosting,
) {
	t.Helper()
	require.Len(t, postings, 2)
	var total int64
	for _, posting := range postings {
		require.Equal(t, transaction.ID, posting.TransactionID)
		require.NotZero(t, posting.Amount)
		total += posting.Amount
	}
	require.Zero(t, total)
	storedTotal, err := testQueries.GetLedgerPostingTotal(
		context.Background(),
		transaction.ID,
	)
	require.NoError(t, err)
	require.Zero(t, storedTotal)
}
