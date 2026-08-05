package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	db "github.com/faelic/monierave/db/sqlc"
	"github.com/faelic/monierave/db/util"
	"github.com/faelic/monierave/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, store db.Store) *Server {
	config := util.Config{
		SecretKey:             util.RandomString(32),
		AccessTokenDuration:   time.Minute,
		RefreshTokenDuration:  time.Minute,
		RefreshCookieName:     "monierave_refresh",
		RefreshCookieSameSite: "lax",
		AllowedOrigins:        "http://localhost:3000",
	}

	server, err := NewServer(config, testSessionStore{Store: store})
	require.NoError(t, err)

	return server
}

type testSessionStore struct {
	db.Store
}

func (testSessionStore) ValidateSession(
	context.Context,
	pgtype.UUID,
	string,
) error {
	return nil
}

func (store testSessionStore) CreateSessionTx(
	ctx context.Context,
	arg db.CreateSessionParams,
) (db.Session, error) {
	return store.Store.CreateSession(ctx, arg)
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func addAuthorization(
	t *testing.T,
	request *http.Request,
	tokenMaker token.Maker,
	authorizationType string,
	username string,
	duration time.Duration,
) {
	value, payload, err := tokenMaker.CreateAccessToken(username, uuid.New(), duration)
	require.NoError(t, err)
	require.NotEmpty(t, payload)

	authorizationHeader := fmt.Sprintf("%s %s", authorizationType, value)
	request.Header.Set(authorizationHeaderKey, authorizationHeader)
}
