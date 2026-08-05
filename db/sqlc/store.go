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
	CreateUserTx(ctx context.Context, arg CreateUserParams) (CreateUserTxResult, error)
	CreateSessionTx(ctx context.Context, arg CreateSessionParams) (Session, error)
	ProcessEmailDeliveryEventTx(ctx context.Context, arg ProcessEmailDeliveryEventParams) (ProcessEmailDeliveryEventResult, error)
	RequestEmailVerificationTx(ctx context.Context, username string) (RequestEmailVerificationTxResult, error)
	ReplayEmailJobTx(ctx context.Context, jobID pgtype.UUID) (ReplayEmailJobTxResult, error)
	RevokeAllUserSessionsTx(ctx context.Context, username, reason, eventType string) error
	RevokeSessionTx(ctx context.Context, id pgtype.UUID, reason, actor string) error
	RotateRefreshTokenTx(ctx context.Context, arg RotateRefreshTokenTxParams) (Session, error)
	TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error)
	UpdateUserTx(ctx context.Context, arg UpdateUserTxParams) (UpdateUserTxResult, error)
	ValidateSession(ctx context.Context, id pgtype.UUID, username string) error
	VerifyUserEmailTx(ctx context.Context, arg VerifyUserEmailTxParams) (User, error)
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
