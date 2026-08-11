package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store interface {
	Querier
	CreateAccountTx(ctx context.Context, arg CreateAccountTxParams) (Account, error)
	CreateUserTx(ctx context.Context, arg CreateUserParams) (CreateUserTxResult, error)
	CloseAccountTx(ctx context.Context, arg CloseAccountTxParams) (Account, error)
	CreateBeneficiaryTx(ctx context.Context, arg CreateBeneficiaryTxParams) (CreateBeneficiaryTxResult, error)
	CreateExclusiveSessionTx(ctx context.Context, arg CreateSessionParams) (Session, error)
	DepositTx(ctx context.Context, arg DepositTxParams) (MoneyMovementTxResult, error)
	IdempotentTransferTx(ctx context.Context, arg IdempotentTransferTxParams) (TransferTxResult, error)
	ProcessEmailDeliveryEventTx(ctx context.Context, arg ProcessEmailDeliveryEventParams) (ProcessEmailDeliveryEventResult, error)
	Reconcile(ctx context.Context, arg ReconciliationParams) (ReconciliationReport, error)
	RecordLoginFailure(ctx context.Context, arg LoginFailureAuditParams) error
	RequestEmailVerificationTx(ctx context.Context, username string) (RequestEmailVerificationTxResult, error)
	ReplayEmailJobTx(ctx context.Context, jobID pgtype.UUID) (ReplayEmailJobTxResult, error)
	ReverseTransactionTx(ctx context.Context, arg ReverseTransactionTxParams) (ReverseTransactionTxResult, error)
	RevokeAllUserSessionsTx(ctx context.Context, username, reason, eventType string) error
	RevokeSessionTx(
		ctx context.Context,
		id pgtype.UUID,
		deviceTokenHash []byte,
		reason string,
		actor string,
	) error
	RotateRefreshTokenTx(ctx context.Context, arg RotateRefreshTokenTxParams) (Session, error)
	SetAccountStatusTx(ctx context.Context, arg AccountStatusTxParams) (AccountStatusTxResult, error)
	TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error)
	UpdateUserTx(ctx context.Context, arg UpdateUserTxParams) (UpdateUserTxResult, error)
	ValidateSession(
		ctx context.Context,
		id pgtype.UUID,
		username string,
		deviceTokenHash []byte,
	) error
	VerifyUserEmailTx(ctx context.Context, arg VerifyUserEmailTxParams) (User, error)
	WithdrawTx(ctx context.Context, arg WithdrawTxParams) (MoneyMovementTxResult, error)
}

// store provides function to execute db queries and transactions
type SQLStore struct {
	*Queries
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) Store {
	return &SQLStore{
		db:      db,
		Queries: New(db),
	}
}

// executes a function within a database transaction
func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx error: %v, rb err: %v", err, rbErr)
		}
		return err
	}
	return tx.Commit(ctx)
}
