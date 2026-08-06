package db

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestCreateBeneficiaryTxAndOwnedOperations(t *testing.T) {
	owner := createRandomUser(t)
	destinationOwner := createRandomUser(t)
	destination, err := testStore.CreateAccountTx(
		context.Background(),
		CreateAccountParams{
			Owner: destinationOwner.Username, Currency: "USD",
		},
	)
	require.NoError(t, err)

	created, err := testStore.CreateBeneficiaryTx(
		context.Background(),
		CreateBeneficiaryTxParams{
			Owner:                      owner.Username,
			DestinationAccountPublicID: destination.PublicID,
			Nickname:                   "Rent account",
		},
	)
	require.NoError(t, err)
	require.True(t, created.Beneficiary.ID.Valid)
	require.Equal(t, destination.ID, created.Beneficiary.DestinationAccountID)
	require.Equal(t, destination.PublicID, created.DestinationAccount.PublicID)

	rows, err := testQueries.ListOwnedBeneficiaries(
		context.Background(),
		ListOwnedBeneficiariesParams{
			Owner: owner.Username, Limit: 20,
		},
	)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, destination.PublicID, rows[0].DestinationAccountPublicID)
	require.Equal(t, "Rent account", rows[0].Nickname)

	foreignOwner := createRandomUser(t)
	_, err = testQueries.UpdateOwnedBeneficiaryNickname(
		context.Background(),
		UpdateOwnedBeneficiaryNicknameParams{
			Nickname:      "Stolen",
			BeneficiaryID: created.Beneficiary.ID,
			Owner:         foreignOwner.Username,
		},
	)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	updated, err := testQueries.UpdateOwnedBeneficiaryNickname(
		context.Background(),
		UpdateOwnedBeneficiaryNicknameParams{
			Nickname:      "Monthly rent",
			BeneficiaryID: created.Beneficiary.ID,
			Owner:         owner.Username,
		},
	)
	require.NoError(t, err)
	require.Equal(t, "Monthly rent", updated.Nickname)

	deleted, err := testQueries.DeleteOwnedBeneficiary(
		context.Background(),
		DeleteOwnedBeneficiaryParams{
			BeneficiaryID: created.Beneficiary.ID,
			Owner:         foreignOwner.Username,
		},
	)
	require.NoError(t, err)
	require.Zero(t, deleted)

	deleted, err = testQueries.DeleteOwnedBeneficiary(
		context.Background(),
		DeleteOwnedBeneficiaryParams{
			BeneficiaryID: created.Beneficiary.ID,
			Owner:         owner.Username,
		},
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted)
}

func TestCreateBeneficiaryTxRejectsDuplicateAndClosedDestination(t *testing.T) {
	owner := createRandomUser(t)
	destinationOwner := createRandomUser(t)
	destination, err := testStore.CreateAccountTx(
		context.Background(),
		CreateAccountParams{
			Owner: destinationOwner.Username, Currency: "USD",
		},
	)
	require.NoError(t, err)

	arg := CreateBeneficiaryTxParams{
		Owner:                      owner.Username,
		DestinationAccountPublicID: destination.PublicID,
		Nickname:                   "Primary",
	}
	_, err = testStore.CreateBeneficiaryTx(context.Background(), arg)
	require.NoError(t, err)
	arg.Nickname = "Duplicate nickname"
	_, err = testStore.CreateBeneficiaryTx(context.Background(), arg)
	require.ErrorIs(t, err, ErrBeneficiaryAlreadyExists)

	closedOwner := createRandomUser(t)
	closedDestination, err := testStore.CreateAccountTx(
		context.Background(),
		CreateAccountParams{
			Owner: closedOwner.Username, Currency: "USD",
		},
	)
	require.NoError(t, err)
	_, err = testStore.CloseAccountTx(
		context.Background(),
		CloseAccountTxParams{
			PublicID: closedDestination.PublicID,
			Username: closedDestination.Owner,
		},
	)
	require.NoError(t, err)

	_, err = testStore.CreateBeneficiaryTx(
		context.Background(),
		CreateBeneficiaryTxParams{
			Owner:                      owner.Username,
			DestinationAccountPublicID: closedDestination.PublicID,
			Nickname:                   "Closed",
		},
	)
	require.ErrorIs(t, err, ErrAccountClosed)

	_, err = testStore.CreateBeneficiaryTx(
		context.Background(),
		CreateBeneficiaryTxParams{
			Owner: owner.Username,
			DestinationAccountPublicID: pgtype.UUID{
				Bytes: uuid.New(), Valid: true,
			},
			Nickname: "Missing",
		},
	)
	require.ErrorIs(t, err, ErrAccountNotFound)
}
