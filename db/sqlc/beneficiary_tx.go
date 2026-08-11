package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrBeneficiaryAlreadyExists = errors.New("beneficiary already exists")

type CreateBeneficiaryTxParams struct {
	Owner                    string
	DestinationAccountNumber string
	Nickname                 string
}

type CreateBeneficiaryTxResult struct {
	Beneficiary        Beneficiary
	DestinationAccount Account
}

// CreateBeneficiaryTx serializes validation with account closure so creation
// observes one consistent lifecycle state.
func (store *SQLStore) CreateBeneficiaryTx(
	ctx context.Context,
	arg CreateBeneficiaryTxParams,
) (CreateBeneficiaryTxResult, error) {
	var result CreateBeneficiaryTxResult
	err := store.execTx(ctx, func(q *Queries) error {
		account, err := q.GetAccountByAccountNumber(ctx, arg.DestinationAccountNumber)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAccountNotFound
		}
		if err != nil {
			return err
		}
		account, err = q.GetAccountForUpdate(ctx, account.ID)
		if err != nil {
			return err
		}
		if account.Status == FinancialAccountStatusClosed {
			return ErrAccountClosed
		}

		beneficiary, err := q.CreateBeneficiary(ctx, CreateBeneficiaryParams{
			Owner:                arg.Owner,
			DestinationAccountID: account.ID,
			Nickname:             arg.Nickname,
		})
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) &&
			pgErr.ConstraintName == "beneficiaries_owner_destination_key" {
			return ErrBeneficiaryAlreadyExists
		}
		if err != nil {
			return err
		}

		result.Beneficiary = beneficiary
		result.DestinationAccount = account
		return nil
	})
	if err != nil {
		return CreateBeneficiaryTxResult{}, err
	}
	return result, nil
}
