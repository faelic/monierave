package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	FinancialAccountStatusActive = "active"
	FinancialAccountStatusFrozen = "frozen"
	FinancialAccountStatusClosed = "closed"
)

var (
	ErrAccountNotFound       = errors.New("account not found")
	ErrAccountNotOwned       = errors.New("account does not belong to user")
	ErrAccountBalanceNotZero = errors.New("account balance must be zero before closure")
	ErrAccountFrozen         = errors.New("account is frozen")
	ErrAccountClosed         = errors.New("account is closed")
	ErrCurrencyMismatch      = errors.New("account currency mismatch")
	ErrSameAccount           = errors.New("source and destination accounts must differ")
)

type CloseAccountTxParams struct {
	PublicID pgtype.UUID
	Username string
}

// CloseAccountTx serializes closure with transfers and enforces lifecycle rules
// while the account row is locked.
func (store *SQLStore) CloseAccountTx(
	ctx context.Context,
	arg CloseAccountTxParams,
) (Account, error) {
	var closed Account

	err := store.execTx(ctx, func(q *Queries) error {
		account, err := q.GetAccountByPublicIDForUpdate(ctx, arg.PublicID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAccountNotFound
		}
		if err != nil {
			return err
		}
		if account.Owner != arg.Username {
			return ErrAccountNotOwned
		}
		if account.Status == FinancialAccountStatusClosed {
			return ErrAccountClosed
		}
		if account.Balance != 0 {
			return ErrAccountBalanceNotZero
		}

		closed, err = q.CloseAccount(ctx, account.ID)
		return err
	})
	if err != nil {
		return Account{}, err
	}
	return closed, nil
}
