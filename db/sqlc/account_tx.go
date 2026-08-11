package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

const accountNumberGenerationAttempts = 5

type CreateAccountTxParams struct {
	Owner    string
	Currency string
}

type CloseAccountTxParams struct {
	PublicID pgtype.UUID
	Username string
}

// CreateAccountTx creates the customer-facing account and its corresponding
// ledger account as one atomic unit.
func (store *SQLStore) CreateAccountTx(
	ctx context.Context,
	arg CreateAccountTxParams,
) (Account, error) {
	return createAccountWithGeneratedNumber(
		generateAccountNumber,
		func(accountNumber string) (Account, error) {
			return store.createAccountAttempt(ctx, CreateAccountParams{
				Owner:         arg.Owner,
				Currency:      arg.Currency,
				AccountNumber: accountNumber,
			})
		},
	)
}

func createAccountWithGeneratedNumber(
	generate func() (string, error),
	create func(string) (Account, error),
) (Account, error) {
	for attempt := 0; attempt < accountNumberGenerationAttempts; attempt++ {
		accountNumber, err := generate()
		if err != nil {
			return Account{}, err
		}

		account, err := create(accountNumber)
		if !isAccountNumberCollision(err) {
			return account, err
		}
	}
	return Account{}, fmt.Errorf("generate unique account number after %d attempts", accountNumberGenerationAttempts)
}

func (store *SQLStore) createAccountAttempt(
	ctx context.Context,
	arg CreateAccountParams,
) (Account, error) {
	var account Account
	err := store.execTx(ctx, func(q *Queries) error {
		var err error
		account, err = q.CreateAccount(ctx, arg)
		if err != nil {
			return err
		}
		_, err = q.CreateCustomerLedgerAccount(ctx, CreateCustomerLedgerAccountParams{
			CustomerAccountID: pgtype.Int8{Int64: account.ID, Valid: true},
			Currency:          account.Currency,
		})
		if err != nil {
			return err
		}
		_, err = createFinancialAudit(ctx, q, financialAuditParams{
			EntityType:    "account",
			EntityID:      account.PublicID,
			CorrelationID: account.PublicID,
			EventType:     "account_created",
			Actor:         account.Owner,
			ToState:       account.Status,
			Metadata: map[string]any{
				"currency": account.Currency,
			},
		})
		return err
	})
	return account, err
}

func isAccountNumberCollision(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.ConstraintName == "accounts_account_number_key"
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
		if err != nil {
			return err
		}
		_, err = createFinancialAudit(ctx, q, financialAuditParams{
			EntityType:    "account",
			EntityID:      closed.PublicID,
			CorrelationID: closed.PublicID,
			EventType:     "account_closed",
			Actor:         arg.Username,
			FromState:     account.Status,
			ToState:       closed.Status,
		})
		if err != nil {
			return err
		}
		_, _, err = createFinancialNotification(ctx, q, financialNotificationParams{
			Username:      closed.Owner,
			EventType:     DomainEventAccountClosed,
			EntityType:    "account",
			EntityID:      closed.PublicID,
			CorrelationID: closed.PublicID,
			OccurredAt:    closed.UpdatedAt.Time,
			AccountStatus: closed.Status,
		})
		return err
	})
	if err != nil {
		return Account{}, err
	}
	return closed, nil
}
