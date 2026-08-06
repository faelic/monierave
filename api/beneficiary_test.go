package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/faelic/monierave/db/mock"
	db "github.com/faelic/monierave/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateBeneficiaryAPI(t *testing.T) {
	owner := randomAccount().Owner
	destination := randomAccount()
	now := time.Now().UTC()
	beneficiary := db.Beneficiary{
		ID:                   pgtype.UUID{Bytes: uuid.New(), Valid: true},
		Owner:                owner,
		DestinationAccountID: destination.ID,
		Nickname:             "Landlord",
		CreatedAt:            timestamp(now),
		UpdatedAt:            timestamp(now),
	}
	result := db.CreateBeneficiaryTxResult{
		Beneficiary:        beneficiary,
		DestinationAccount: destination,
	}

	testCases := []struct {
		name       string
		storeError error
		statusCode int
		errorCode  string
	}{
		{name: "OK", statusCode: http.StatusCreated},
		{name: "MissingAccount", storeError: db.ErrAccountNotFound, statusCode: http.StatusNotFound},
		{name: "ClosedAccount", storeError: db.ErrAccountClosed, statusCode: http.StatusConflict, errorCode: "account_closed"},
		{name: "Duplicate", storeError: db.ErrBeneficiaryAlreadyExists, statusCode: http.StatusConflict, errorCode: "beneficiary_already_exists"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			store.EXPECT().CreateBeneficiaryTx(
				gomock.Any(),
				db.CreateBeneficiaryTxParams{
					Owner:                      owner,
					DestinationAccountPublicID: destination.PublicID,
					Nickname:                   "Landlord",
				},
			).Return(result, tc.storeError)

			server := newTestServer(t, store)
			body, err := json.Marshal(createBeneficiaryRequest{
				DestinationAccountID: publicUUID(destination.PublicID),
				Nickname:             "  Landlord  ",
			})
			require.NoError(t, err)
			request := httptest.NewRequest(
				http.MethodPost,
				"/beneficiaries",
				bytes.NewReader(body),
			)
			addAuthorization(
				t,
				request,
				server.tokenMaker,
				authorizationTypeBearer,
				owner,
				time.Minute,
			)
			recorder := httptest.NewRecorder()
			server.router.ServeHTTP(recorder, request)

			require.Equal(t, tc.statusCode, recorder.Code)
			if tc.storeError == nil {
				var response beneficiaryResponse
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
				require.Equal(t, publicUUID(beneficiary.ID), response.ID)
				require.Equal(t, "Landlord", response.Nickname)
				require.Equal(
					t,
					publicUUID(destination.PublicID),
					response.DestinationAccountID,
				)
				require.NotContains(t, recorder.Body.String(), `"owner"`)
			}
			if tc.errorCode != "" {
				var response map[string]any
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
				require.Equal(t, tc.errorCode, response["code"])
			}
		})
	}
}

func TestBeneficiaryAPIRejectsBlankNickname(t *testing.T) {
	destination := randomAccount()
	owner := randomAccount().Owner
	server := newTestServer(
		t,
		mockdb.NewMockStore(gomock.NewController(t)),
	)
	body, err := json.Marshal(createBeneficiaryRequest{
		DestinationAccountID: publicUUID(destination.PublicID),
		Nickname:             "   ",
	})
	require.NoError(t, err)
	request := httptest.NewRequest(
		http.MethodPost,
		"/beneficiaries",
		bytes.NewReader(body),
	)
	addAuthorization(
		t,
		request,
		server.tokenMaker,
		authorizationTypeBearer,
		owner,
		time.Minute,
	)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestListBeneficiariesAPIUsesOwnerScope(t *testing.T) {
	owner := randomAccount().Owner
	now := time.Now().UTC()
	rows := []db.ListOwnedBeneficiariesRow{
		{
			ID:                         pgtype.UUID{Bytes: uuid.New(), Valid: true},
			Nickname:                   "Savings",
			DestinationAccountPublicID: pgtype.UUID{Bytes: uuid.New(), Valid: true},
			Currency:                   "USD",
			DestinationAccountStatus:   db.FinancialAccountStatusActive,
			CreatedAt:                  timestamp(now),
			UpdatedAt:                  timestamp(now),
		},
	}

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().ListOwnedBeneficiaries(
		gomock.Any(),
		db.ListOwnedBeneficiariesParams{
			Owner: owner, Limit: 20, Offset: 0,
		},
	).Return(rows, nil)

	server := newTestServer(t, store)
	request := httptest.NewRequest(http.MethodGet, "/beneficiaries", nil)
	addAuthorization(
		t,
		request,
		server.tokenMaker,
		authorizationTypeBearer,
		owner,
		time.Minute,
	)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response []beneficiaryResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response, 1)
	require.Equal(t, "Savings", response[0].Nickname)
}

func TestUpdateAndDeleteBeneficiaryAPIHideForeignRecords(t *testing.T) {
	owner := randomAccount().Owner
	beneficiaryID := pgtype.UUID{Bytes: uuid.New(), Valid: true}

	t.Run("Update", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		store.EXPECT().UpdateOwnedBeneficiaryNickname(
			gomock.Any(),
			db.UpdateOwnedBeneficiaryNicknameParams{
				Nickname:      "Updated",
				BeneficiaryID: beneficiaryID,
				Owner:         owner,
			},
		).Return(db.UpdateOwnedBeneficiaryNicknameRow{}, pgx.ErrNoRows)

		server := newTestServer(t, store)
		body := bytes.NewBufferString(`{"nickname":"Updated"}`)
		request := httptest.NewRequest(
			http.MethodPatch,
			"/beneficiaries/"+publicUUID(beneficiaryID),
			body,
		)
		addAuthorization(
			t,
			request,
			server.tokenMaker,
			authorizationTypeBearer,
			owner,
			time.Minute,
		)
		recorder := httptest.NewRecorder()
		server.router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusNotFound, recorder.Code)
	})

	t.Run("Delete", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		store.EXPECT().DeleteOwnedBeneficiary(
			gomock.Any(),
			db.DeleteOwnedBeneficiaryParams{
				BeneficiaryID: beneficiaryID,
				Owner:         owner,
			},
		).Return(int64(0), nil)

		server := newTestServer(t, store)
		request := httptest.NewRequest(
			http.MethodDelete,
			"/beneficiaries/"+publicUUID(beneficiaryID),
			nil,
		)
		addAuthorization(
			t,
			request,
			server.tokenMaker,
			authorizationTypeBearer,
			owner,
			time.Minute,
		)
		recorder := httptest.NewRecorder()
		server.router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusNotFound, recorder.Code)
	})
}
