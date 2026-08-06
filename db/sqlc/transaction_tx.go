package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type TransferTxParams struct {
	FromAccountPublicID pgtype.UUID `json:"from_account_id"`
	ToAccountPublicID   pgtype.UUID `json:"to_account_id"`
	Amount              int64       `json:"amount"`
	Currency            string      `json:"currency"`
	Username            string      `json:"username"`
}

type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount   Account  `json:"to_account"`
	FromEntry   Entry    `json:"from_entry"`
	ToEntry     Entry    `json:"to_entry"`
}

var ErrInsufficientBalance = errors.New("insufficient balance")

// TransferTx performs authorization, lifecycle validation, entries, and balance
// changes while both account rows are locked in a consistent order.
func (store *SQLStore) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		fromAccount, err := q.GetAccountByPublicID(ctx, arg.FromAccountPublicID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAccountNotFound
		}
		if err != nil {
			return err
		}
		toAccount, err := q.GetAccountByPublicID(ctx, arg.ToAccountPublicID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAccountNotFound
		}
		if err != nil {
			return err
		}
		if fromAccount.ID == toAccount.ID {
			return ErrSameAccount
		}

		fromAccount, toAccount, err = lockTransferAccounts(
			ctx,
			q,
			fromAccount.ID,
			toAccount.ID,
		)
		if err != nil {
			return err
		}

		if fromAccount.Owner != arg.Username {
			return ErrAccountNotOwned
		}
		if fromAccount.Currency != arg.Currency || toAccount.Currency != arg.Currency {
			return ErrCurrencyMismatch
		}
		if fromAccount.Status == FinancialAccountStatusFrozen {
			return ErrAccountFrozen
		}
		if fromAccount.Status == FinancialAccountStatusClosed ||
			toAccount.Status == FinancialAccountStatusClosed {
			return ErrAccountClosed
		}

		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: fromAccount.ID,
			ToAccountID:   toAccount.ID,
			Amount:        arg.Amount,
		})
		if err != nil {
			return err
		}

		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: fromAccount.ID,
			Amount:    -arg.Amount,
		})
		if err != nil {
			return err
		}

		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: toAccount.ID,
			Amount:    arg.Amount,
		})
		if err != nil {
			return err
		}

		result.FromAccount, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
			ID:     fromAccount.ID,
			Amount: -arg.Amount,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInsufficientBalance
		}
		if err != nil {
			return err
		}

		result.ToAccount, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
			ID:     toAccount.ID,
			Amount: arg.Amount,
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return TransferTxResult{}, err
	}
	return result, nil
}

func lockTransferAccounts(
	ctx context.Context,
	q *Queries,
	fromAccountID int64,
	toAccountID int64,
) (Account, Account, error) {
	firstID, secondID := fromAccountID, toAccountID
	if firstID > secondID {
		firstID, secondID = secondID, firstID
	}

	first, err := q.GetAccountForUpdate(ctx, firstID)
	if err != nil {
		return Account{}, Account{}, err
	}
	second, err := q.GetAccountForUpdate(ctx, secondID)
	if err != nil {
		return Account{}, Account{}, err
	}

	if first.ID == fromAccountID {
		return first, second, nil
	}
	return second, first, nil
}
