package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

type transactionHistoryFixture struct {
	account      Account
	counterparty Account
	deposit      MoneyMovementTxResult
	transfer     TransferTxResult
	withdrawal   MoneyMovementTxResult
}

func TestTransactionHistoryOrderingRunningBalanceAndFilters(t *testing.T) {
	fixture := createTransactionHistoryFixture(t)
	ctx := context.Background()

	rows, err := testQueries.ListOwnedAccountTransactions(
		ctx,
		historyParams(fixture.account, 10),
	)
	require.NoError(t, err)
	require.Len(t, rows, 3)

	require.Equal(t, fixture.withdrawal.Transaction.Reference, rows[0].Reference)
	require.Equal(t, "outgoing", rows[0].Direction)
	require.Equal(t, int64(700), rows[0].BalanceAfter)
	require.Equal(t, "Monierave", rows[0].Counterparty)

	require.Equal(t, fixture.transfer.Transaction.Reference, rows[1].Reference)
	require.Equal(t, "outgoing", rows[1].Direction)
	require.Equal(t, int64(800), rows[1].BalanceAfter)
	require.Equal(t, fixture.counterparty.AccountNumber, rows[1].Counterparty)

	require.Equal(t, fixture.deposit.Transaction.Reference, rows[2].Reference)
	require.Equal(t, "incoming", rows[2].Direction)
	require.Equal(t, int64(1_000), rows[2].BalanceAfter)

	depositOnly := historyParams(fixture.account, 10)
	depositOnly.TransactionType = pgtype.Text{
		String: BankingTransactionTypeDeposit,
		Valid:  true,
	}
	rows, err = testQueries.ListOwnedAccountTransactions(ctx, depositOnly)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(1_000), rows[0].BalanceAfter)

	outgoingOnly := historyParams(fixture.account, 10)
	outgoingOnly.Direction = pgtype.Text{String: "outgoing", Valid: true}
	rows, err = testQueries.ListOwnedAccountTransactions(ctx, outgoingOnly)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, int64(700), rows[0].BalanceAfter)
	require.Equal(t, int64(800), rows[1].BalanceAfter)
}

func TestTransactionHistoryCursorPagination(t *testing.T) {
	fixture := createTransactionHistoryFixture(t)
	ctx := context.Background()

	firstPage, err := testQueries.ListOwnedAccountTransactions(
		ctx,
		historyParams(fixture.account, 2),
	)
	require.NoError(t, err)
	require.Len(t, firstPage, 2)

	cursor := firstPage[len(firstPage)-1]
	secondParams := historyParams(fixture.account, 2)
	secondParams.CursorCreatedAt = cursor.CreatedAt
	secondParams.CursorID = cursor.ID
	secondPage, err := testQueries.ListOwnedAccountTransactions(ctx, secondParams)
	require.NoError(t, err)
	require.Len(t, secondPage, 1)
	require.Equal(t, fixture.deposit.Transaction.ID, secondPage[0].ID)
}

func TestTransactionDetailIsOwnershipScoped(t *testing.T) {
	fixture := createTransactionHistoryFixture(t)
	ctx := context.Background()

	senderView, err := testQueries.GetOwnedTransactionByReference(
		ctx,
		GetOwnedTransactionByReferenceParams{
			Username:  fixture.account.Owner,
			Reference: fixture.transfer.Transaction.Reference,
		},
	)
	require.NoError(t, err)
	require.Equal(t, "outgoing", senderView.Direction)
	require.Equal(t, fixture.counterparty.AccountNumber, senderView.Counterparty)

	recipientView, err := testQueries.GetOwnedTransactionByReference(
		ctx,
		GetOwnedTransactionByReferenceParams{
			Username:  fixture.counterparty.Owner,
			Reference: fixture.transfer.Transaction.Reference,
		},
	)
	require.NoError(t, err)
	require.Equal(t, "incoming", recipientView.Direction)
	require.Equal(t, fixture.account.AccountNumber, recipientView.Counterparty)

	foreignUser := createRandomUser(t)
	_, err = testQueries.GetOwnedTransactionByReference(
		ctx,
		GetOwnedTransactionByReferenceParams{
			Username:  foreignUser.Username,
			Reference: fixture.transfer.Transaction.Reference,
		},
	)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestStatementOpeningAndClosingBalances(t *testing.T) {
	fixture := createTransactionHistoryFixture(t)
	ctx := context.Background()

	allTime, err := testQueries.GetOwnedAccountStatementBalances(
		ctx,
		GetOwnedAccountStatementBalancesParams{
			AccountPublicID: fixture.account.PublicID,
			Username:        fixture.account.Owner,
		},
	)
	require.NoError(t, err)
	require.Zero(t, allTime.OpeningBalance)
	require.Equal(t, int64(700), allTime.ClosingBalance)

	throughTransfer, err := testQueries.GetOwnedAccountStatementBalances(
		ctx,
		GetOwnedAccountStatementBalancesParams{
			FromTime:        fixture.transfer.Transaction.CreatedAt,
			ToTime:          fixture.withdrawal.Transaction.CreatedAt,
			AccountPublicID: fixture.account.PublicID,
			Username:        fixture.account.Owner,
		},
	)
	require.NoError(t, err)
	require.Equal(t, int64(1_000), throughTransfer.OpeningBalance)
	require.Equal(t, int64(800), throughTransfer.ClosingBalance)
}

func createTransactionHistoryFixture(t *testing.T) transactionHistoryFixture {
	t.Helper()
	ctx := context.Background()
	user := createRandomUser(t)
	account, err := testStore.CreateAccountTx(ctx, CreateAccountTxParams{
		Owner: user.Username, Currency: "USD",
	})
	require.NoError(t, err)
	otherUser := createRandomUser(t)
	counterparty, err := testStore.CreateAccountTx(ctx, CreateAccountTxParams{
		Owner: otherUser.Username, Currency: "USD",
	})
	require.NoError(t, err)

	deposit, err := testStore.DepositTx(ctx, DepositTxParams{
		AccountPublicID: account.PublicID,
		Amount:          1_000,
		Narration:       "Statement opening deposit",
	})
	require.NoError(t, err)
	time.Sleep(2 * time.Millisecond)

	transfer, err := testStore.TransferTx(ctx, TransferTxParams{
		FromAccountPublicID: account.PublicID,
		ToAccountNumber:     counterparty.AccountNumber,
		Amount:              200,
		Currency:            account.Currency,
		Username:            account.Owner,
		Narration:           "Statement transfer",
	})
	require.NoError(t, err)
	time.Sleep(2 * time.Millisecond)

	withdrawal, err := testStore.WithdrawTx(ctx, WithdrawTxParams{
		AccountPublicID: account.PublicID,
		Amount:          100,
		Narration:       "Statement withdrawal",
	})
	require.NoError(t, err)

	return transactionHistoryFixture{
		account:      account,
		counterparty: counterparty,
		deposit:      deposit,
		transfer:     transfer,
		withdrawal:   withdrawal,
	}
}

func historyParams(
	account Account,
	limit int32,
) ListOwnedAccountTransactionsParams {
	return ListOwnedAccountTransactionsParams{
		PageLimit:       limit,
		AccountPublicID: account.PublicID,
		Username:        account.Owner,
	}
}
