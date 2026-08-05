package db

import (
	"context"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/faelic/monierave/db/util"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testQueries *Queries
var testStore Store
var testDB *pgxpool.Pool

func TestMain(m *testing.M) {
	config, err := util.LoadConfig("../../")
	if err != nil {
		log.Printf("skipping db/sqlc tests: cannot load config: %v", err)
		os.Exit(0)
	}
	ctx := context.Background()

	testDBSource := config.DBTestSource
	if testDBSource == "" {
		testDBSource = config.DBSource
	}
	poolConfig, err := pgxpool.ParseConfig(testDBSource)
	if err != nil {
		log.Fatalf("invalid test database configuration: %v", err)
	}
	if !strings.HasSuffix(poolConfig.ConnConfig.Database, "_test") {
		log.Fatalf(
			"refusing to run database tests against %q; DB_SOURCE must use a dedicated *_test database",
			poolConfig.ConnConfig.Database,
		)
	}

	connPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Printf("skipping db/sqlc tests: could not create database pool: %v", err)
		os.Exit(0)
	}

	if err := connPool.Ping(ctx); err != nil {
		log.Printf("skipping db/sqlc tests: database is unreachable: %v", err)
		os.Exit(0)
	}

	defer connPool.Close()

	testDB = connPool
	testQueries = New(testDB)
	testStore = NewStore(testDB)

	code := m.Run()
	testDB.Close()

	os.Exit(code)
}
