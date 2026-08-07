package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	mockdb "github.com/faelic/monierave/db/mock"
	db "github.com/faelic/monierave/db/sqlc"
	"github.com/faelic/monierave/db/util"
	"github.com/faelic/monierave/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateUserAPI(t *testing.T) {
	password := util.RandomString(8)
	user, err := randomUser(password)
	require.NoError(t, err)

	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUserTx(gomock.Any(), gomock.Any()).
					Times(1).
					DoAndReturn(func(ctx any, arg db.CreateUserParams) (db.CreateUserTxResult, error) {
						err := util.CheckPassword(password, arg.HashedPassword)
						require.NoError(t, err)
						require.Equal(t, user.Username, arg.Username)
						require.Equal(t, user.FullName, arg.FullName)
						require.Equal(t, user.Email, arg.Email)
						return db.CreateUserTxResult{User: user}, nil
					})
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
				requireBodyMatchUser(t, recorder.Body, user)
			},
		},
		{
			name: "OKWithUppercaseInput",
			body: gin.H{
				"username":  strings.ToUpper(user.Username),
				"password":  password,
				"full_name": user.FullName,
				"email":     strings.ToUpper(user.Email),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUserTx(gomock.Any(), gomock.Any()).
					Times(1).
					DoAndReturn(func(ctx any, arg db.CreateUserParams) (db.CreateUserTxResult, error) {
						err := util.CheckPassword(password, arg.HashedPassword)
						require.NoError(t, err)
						require.Equal(t, strings.ToLower(user.Username), arg.Username)
						require.Equal(t, user.FullName, arg.FullName)
						require.Equal(t, strings.ToLower(user.Email), arg.Email)

						normalizedUser := user
						normalizedUser.Username = strings.ToLower(user.Username)
						normalizedUser.Email = strings.ToLower(user.Email)

						return db.CreateUserTxResult{User: normalizedUser}, nil
					})
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
			},
		},
		{
			name: "DuplicateUsername",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUserTx(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.CreateUserTxResult{}, &pgconn.PgError{
						Code:           "23505",
						ConstraintName: "users_pkey",
					})
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "InternalError",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUserTx(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.CreateUserTxResult{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InvalidUsername",
			body: gin.H{
				"username":  "invalid-user",
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUserTx(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidEmail",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     "invalid-email",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUserTx(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			request, err := http.NewRequest(http.MethodPost, "/users", bytes.NewReader(data))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestLoginUserAPI(t *testing.T) {
	password := util.RandomString(8)
	user, err := randomUser(password)
	require.NoError(t, err)

	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder, tokenMaker stringVerifier)
	}{
		{
			name: "OK",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(user, nil)

				store.EXPECT().
					CreateSession(gomock.Any(), gomock.Any()).
					Times(1).
					DoAndReturn(func(ctx any, arg db.CreateSessionParams) (db.Session, error) {
						require.Equal(t, user.Username, arg.Username)
						require.Len(t, arg.RefreshTokenHash, 32)
						require.True(t, arg.RefreshTokenID.Valid)
						require.Len(t, arg.DeviceTokenHash, 32)
						require.NotEmpty(t, arg.UserAgent)
						require.NotEmpty(t, arg.ClientIp)
						require.True(t, arg.ID.Valid)
						require.True(t, arg.ExpiresAt.Valid)

						return db.Session{
							ID:               arg.ID,
							Username:         arg.Username,
							RefreshTokenHash: arg.RefreshTokenHash,
							RefreshTokenID:   arg.RefreshTokenID,
							DeviceTokenHash:  arg.DeviceTokenHash,
							UserAgent:        arg.UserAgent,
							ClientIp:         arg.ClientIp,
							ExpiresAt:        arg.ExpiresAt,
						}, nil
					})
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder, tokenMaker stringVerifier) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchLoginUser(t, recorder.Body, user, tokenMaker)
				require.NotContains(t, recorder.Body.String(), "refresh_token")
				require.NotContains(t, recorder.Body.String(), "device_token")
				cookies := recorder.Result().Cookies()
				require.Len(t, cookies, 2)
				require.True(t, cookies[0].HttpOnly)
				require.True(t, cookies[1].HttpOnly)
				require.NotEqual(t, cookies[0].Name, cookies[1].Name)
			},
		},
		{
			name: "UserNotFound",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(db.User{}, pgx.ErrNoRows)

				store.EXPECT().
					CreateSession(gomock.Any(), gomock.Any()).
					Times(0)

				store.EXPECT().
					RecordLoginFailure(gomock.Any(), db.LoginFailureAuditParams{
						Username:  user.Username,
						ClientIP:  "127.0.0.1",
						UserAgent: "api-test",
						Reason:    "unknown_user",
					}).
					Return(nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder, tokenMaker stringVerifier) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "WrongPassword",
			body: gin.H{
				"username": user.Username,
				"password": util.RandomString(8),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(user, nil)

				store.EXPECT().
					CreateSession(gomock.Any(), gomock.Any()).
					Times(0)

				store.EXPECT().
					RecordLoginFailure(gomock.Any(), db.LoginFailureAuditParams{
						Username:  user.Username,
						ClientIP:  "127.0.0.1",
						UserAgent: "api-test",
						Reason:    "invalid_password",
					}).
					Return(nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder, tokenMaker stringVerifier) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "InternalError",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(db.User{}, sql.ErrConnDone)

				store.EXPECT().
					CreateSession(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder, tokenMaker stringVerifier) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "CreateSessionError",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(user, nil)

				store.EXPECT().
					CreateSession(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.Session{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder, tokenMaker stringVerifier) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InvalidUsername",
			body: gin.H{
				"username": "invalid-user",
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Any()).
					Times(0)

				store.EXPECT().
					CreateSession(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder, tokenMaker stringVerifier) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidPassword",
			body: gin.H{
				"username": user.Username,
				"password": "123",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Any()).
					Times(0)

				store.EXPECT().
					CreateSession(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder, tokenMaker stringVerifier) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			request, err := http.NewRequest(http.MethodPost, "/users/login", bytes.NewReader(data))
			require.NoError(t, err)

			request.Header.Set("User-Agent", "api-test")
			request.RemoteAddr = "127.0.0.1:12345"

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder, server.tokenMaker)
		})
	}
}

func TestUpdateUserEmailUsesTransactionalOutbox(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)
	user, err := randomUser(util.RandomString(8))
	require.NoError(t, err)
	newEmail := "new-address@example.com"

	store.EXPECT().
		UpdateUserTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.UpdateUserTxParams) (db.UpdateUserTxResult, error) {
			require.Equal(t, user.Username, arg.Username)
			require.True(t, arg.Email.Valid)
			require.Equal(t, newEmail, arg.Email.String)

			updated := user
			updated.Email = newEmail
			updated.EmailDeliverabilityStatus = db.EmailDeliverabilityPending
			return db.UpdateUserTxResult{
				User:         updated,
				EmailChanged: true,
			}, nil
		})

	body, err := json.Marshal(gin.H{"email": "NEW-ADDRESS@EXAMPLE.COM"})
	require.NoError(t, err)
	request, err := http.NewRequest(http.MethodPatch, "/users/me", bytes.NewReader(body))
	require.NoError(t, err)
	addAuthorization(
		t,
		request,
		server.tokenMaker,
		authorizationTypeBearer,
		user.Username,
		time.Minute,
	)

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	requireBodyMatchUser(t, recorder.Body, db.User{
		Username:                  user.Username,
		FullName:                  user.FullName,
		Email:                     newEmail,
		EmailDeliverabilityStatus: db.EmailDeliverabilityPending,
	})
}

func TestGetUserEmailStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)
	user, err := randomUser(util.RandomString(8))
	require.NoError(t, err)
	user.EmailDeliverabilityStatus = db.EmailDeliverabilityUndeliverable
	user.EmailBouncedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	job := db.EmailJob{
		ID:             pgtype.UUID{Bytes: uuid.New(), Valid: true},
		Username:       user.Username,
		Recipient:      user.Email,
		Status:         db.EmailJobStatusSent,
		DeliveryStatus: db.EmailDeliveryStatusBounced,
		BounceType:     pgtype.Text{String: "Permanent", Valid: true},
		BounceSubtype:  pgtype.Text{String: "General", Valid: true},
		BounceMessage:  pgtype.Text{String: "mailbox does not exist", Valid: true},
	}

	store.EXPECT().GetUser(gomock.Any(), user.Username).Return(user, nil)
	store.EXPECT().
		GetLatestEmailJobForCurrentAddress(gomock.Any(), user.Username).
		Return(job, nil)

	request, err := http.NewRequest(http.MethodGet, "/users/me/email-status", nil)
	require.NoError(t, err)
	addAuthorization(
		t,
		request,
		server.tokenMaker,
		authorizationTypeBearer,
		user.Username,
		time.Minute,
	)

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	var response emailStatusResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, user.Email, response.Email)
	require.Equal(t, db.EmailDeliverabilityUndeliverable, response.DeliverabilityStatus)
	require.NotNil(t, response.LatestJob)
	require.Equal(t, db.EmailDeliveryStatusBounced, response.LatestJob.DeliveryStatus)
	require.Equal(t, "Permanent", response.LatestJob.BounceType.String)
}

func TestVerifyUserEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)
	user, err := randomUser(util.RandomString(8))
	require.NoError(t, err)
	jobID := uuid.New()
	value, _, err := server.emailVerificationMaker.Create(
		user.Username,
		user.Email,
		jobID.String(),
		time.Hour,
	)
	require.NoError(t, err)

	request, err := http.NewRequest(
		http.MethodGet,
		"/users/verify-email?token="+url.QueryEscape(value),
		nil,
	)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "Confirm your email")
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
	require.Equal(t, "no-referrer", recorder.Header().Get("Referrer-Policy"))
	require.Equal(t, "DENY", recorder.Header().Get("X-Frame-Options"))
	require.NotEmpty(t, recorder.Header().Get("Content-Security-Policy"))

	verifiedUser := user
	verifiedUser.AccountStatus = db.AccountStatusActive
	verifiedUser.EmailVerifiedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	store.EXPECT().
		VerifyUserEmailTx(gomock.Any(), db.VerifyUserEmailTxParams{
			Username: user.Username,
			Email:    user.Email,
			JobID:    pgtype.UUID{Bytes: jobID, Valid: true},
		}).
		Return(verifiedUser, nil)

	request, err = http.NewRequest(
		http.MethodPost,
		"/users/verify-email",
		strings.NewReader(url.Values{"token": []string{value}}.Encode()),
	)
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder = httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "Email verified")
}

func TestConfirmUserEmailJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)
	user, err := randomUser(util.RandomString(8))
	require.NoError(t, err)
	jobID := uuid.New()
	value, _, err := server.emailVerificationMaker.Create(
		user.Username,
		user.Email,
		jobID.String(),
		time.Hour,
	)
	require.NoError(t, err)

	verifiedUser := user
	verifiedUser.AccountStatus = db.AccountStatusActive
	verifiedUser.EmailVerifiedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	store.EXPECT().
		VerifyUserEmailTx(gomock.Any(), db.VerifyUserEmailTxParams{
			Username: user.Username,
			Email:    user.Email,
			JobID:    pgtype.UUID{Bytes: jobID, Valid: true},
		}).
		Return(verifiedUser, nil)

	body, err := json.Marshal(map[string]string{"token": value})
	require.NoError(t, err)
	request := httptest.NewRequest(
		http.MethodPost,
		"/users/verify-email",
		bytes.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"message":"email verified successfully"`)
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
}

func TestConfirmUserEmailFormRendersInvalidTokenError(t *testing.T) {
	ctrl := gomock.NewController(t)
	server := newTestServer(t, mockdb.NewMockStore(ctrl))

	request := httptest.NewRequest(
		http.MethodPost,
		"/users/verify-email",
		strings.NewReader(url.Values{"token": []string{"invalid-token"}}.Encode()),
	)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "text/html")
	require.Contains(t, recorder.Body.String(), "Verification unavailable")
}

func TestResendUserEmailVerification(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)
	username := util.RandomOwner()
	jobID := uuid.New()

	store.EXPECT().
		RequestEmailVerificationTx(gomock.Any(), username).
		Return(db.RequestEmailVerificationTxResult{
			EmailJob: db.EmailJob{ID: pgtype.UUID{Bytes: jobID, Valid: true}},
		}, nil)

	request, err := http.NewRequest(
		http.MethodPost,
		"/users/me/resend-verification",
		nil,
	)
	require.NoError(t, err)
	addAuthorization(
		t,
		request,
		server.tokenMaker,
		authorizationTypeBearer,
		username,
		time.Minute,
	)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusAccepted, recorder.Code)
	require.Contains(t, recorder.Body.String(), "verification email queued")
}

type stringVerifier interface {
	VerifyAccessToken(token string) (*token.Payload, error)
}

func randomUser(password string) (db.User, error) {
	hashedPassword, err := util.HashPassword(password)
	if err != nil {
		return db.User{}, err
	}

	return db.User{
		Username:       util.RandomOwner(),
		HashedPassword: hashedPassword,
		FullName:       util.RandomOwner(),
		Email:          util.RandomEmail(),
	}, nil
}

func requireBodyMatchUser(t *testing.T, body *bytes.Buffer, user db.User) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotUser userResponse
	err = json.Unmarshal(data, &gotUser)
	require.NoError(t, err)

	require.Equal(t, user.Username, gotUser.Username)
	require.Equal(t, user.FullName, gotUser.FullName)
	require.Equal(t, user.Email, gotUser.Email)
}

func requireBodyMatchLoginUser(t *testing.T, body *bytes.Buffer, user db.User, tokenMaker stringVerifier) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotRsp loginUserResponse
	err = json.Unmarshal(data, &gotRsp)
	require.NoError(t, err)

	require.True(t, gotRsp.SessionID.Valid)
	require.NotEmpty(t, gotRsp.AccessToken)
	require.False(t, gotRsp.AccessTokenExpiresAt.IsZero())

	accessPayload, err := tokenMaker.VerifyAccessToken(gotRsp.AccessToken)
	require.NoError(t, err)
	require.Equal(t, user.Username, accessPayload.Username)
	require.Equal(t, uuid.UUID(gotRsp.SessionID.Bytes), accessPayload.SessionID)

	require.Equal(t, user.Username, gotRsp.User.Username)
	require.Equal(t, user.FullName, gotRsp.User.FullName)
	require.Equal(t, user.Email, gotRsp.User.Email)
}
