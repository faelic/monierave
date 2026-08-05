package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/faelic/monierave/api"
	db "github.com/faelic/monierave/db/sqlc"
	"github.com/faelic/monierave/db/util"
	"github.com/faelic/monierave/mailer"
	"github.com/faelic/monierave/token"
	"github.com/faelic/monierave/worker"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("could not load config: ", err)
	}

	role := "api"
	if len(os.Args) > 1 {
		role = os.Args[1]
	}

	switch role {
	case "api":
		err = runAPI(config)
	case "relay":
		err = runRelay(config)
	case "worker":
		err = runWorker(config)
	case "jobs":
		err = runJobs(config, os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q; expected api, relay, worker, or jobs", role)
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
}

func openStore(ctx context.Context, dbSource string) (*pgxpool.Pool, db.Store, error) {
	pool, err := pgxpool.New(ctx, dbSource)
	if err != nil {
		return nil, nil, fmt.Errorf("create database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("connect to database: %w", err)
	}
	return pool, db.NewStore(pool), nil
}

func runAPI(config util.Config) error {
	if err := util.ValidateAPIConfig(config); err != nil {
		return fmt.Errorf("invalid API config: %w", err)
	}
	if port := os.Getenv("PORT"); port != "" {
		if _, err := strconv.Atoi(port); err != nil {
			return fmt.Errorf("invalid PORT: %w", err)
		}
		if gin.Mode() == gin.DebugMode {
			gin.SetMode(gin.ReleaseMode)
		}
		config.ServerAddress = "0.0.0.0:" + port
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, store, err := openStore(ctx, config.DBSource)
	if err != nil {
		return err
	}
	defer pool.Close()

	server, err := api.NewServer(config, store)
	if err != nil {
		return fmt.Errorf("create API server: %w", err)
	}

	httpServer := &http.Server{
		Addr:              config.ServerAddress,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	}
}

func runRelay(config util.Config) error {
	if err := util.ValidateRelayConfig(config); err != nil {
		return fmt.Errorf("invalid relay config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, store, err := openStore(ctx, config.DBSource)
	if err != nil {
		return err
	}
	defer pool.Close()

	distributor := worker.NewRedisTaskDistributor(asynq.RedisClientOpt{Addr: config.RedisAddress})
	defer distributor.Close()

	relay := worker.NewOutboxRelay(store, distributor, worker.RelayConfig{
		InstanceID:   hostname(),
		BatchSize:    config.RelayBatchSize,
		PollInterval: config.RelayPollInterval,
		ClaimLease:   config.RelayClaimLease,
	})
	return relay.Run(ctx)
}

func runWorker(config util.Config) error {
	if err := util.ValidateWorkerConfig(config); err != nil {
		return fmt.Errorf("invalid worker config: %w", err)
	}

	pool, store, err := openStore(context.Background(), config.DBSource)
	if err != nil {
		return err
	}
	defer pool.Close()

	emailMailer, err := buildMailer(config)
	if err != nil {
		return err
	}
	emailVerificationMaker, err := token.NewEmailVerificationMaker(config.SecretKey)
	if err != nil {
		return fmt.Errorf("create email verification token maker: %w", err)
	}

	processor := worker.NewRedisTaskProcessor(
		asynq.RedisClientOpt{Addr: config.RedisAddress},
		store,
		emailMailer,
		config.WorkerConcurrency,
		emailVerificationMaker,
		config.PublicAPIURL,
		config.EmailVerificationDuration,
	)
	return processor.Run()
}

func buildMailer(config util.Config) (mailer.Mailer, error) {
	switch config.MailerProvider {
	case "log":
		return mailer.NewLogMailer(), nil
	case "resend":
		return mailer.NewResendMailer(config.ResendAPIKey, config.EmailFrom)
	default:
		return nil, fmt.Errorf("unsupported MAILER_PROVIDER %q", config.MailerProvider)
	}
}

func runJobs(config util.Config, args []string) error {
	if config.DBSource == "" {
		return errors.New("DB_SOURCE is required")
	}
	if len(args) == 0 {
		return errors.New("jobs command requires list, show, replay, or audit")
	}

	ctx := context.Background()
	pool, store, err := openStore(ctx, config.DBSource)
	if err != nil {
		return err
	}
	defer pool.Close()

	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("jobs list", flag.ContinueOnError)
		limit := flags.Int("limit", 50, "maximum dead-letter jobs to return")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if *limit <= 0 {
			return errors.New("limit must be greater than zero")
		}
		jobs, err := store.ListDeadLetterEmailJobs(ctx, int32(*limit))
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(jobs)
	case "show":
		jobID, err := parseCLIJobID(args[1:])
		if err != nil {
			return err
		}
		job, err := store.GetEmailJob(ctx, jobID)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(job)
	case "replay":
		jobID, err := parseCLIJobID(args[1:])
		if err != nil {
			return err
		}
		result, err := store.ReplayEmailJobTx(ctx, jobID)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(result)
	case "audit":
		return runJobsAudit(ctx, store, args[1:])
	default:
		return fmt.Errorf("unknown jobs command %q", args[0])
	}
}

type auditLogOutput struct {
	ID            int64              `json:"id"`
	EntityType    string             `json:"entity_type"`
	EntityID      pgtype.UUID        `json:"entity_id"`
	CorrelationID pgtype.UUID        `json:"correlation_id"`
	EventType     string             `json:"event_type"`
	Actor         string             `json:"actor"`
	FromState     pgtype.Text        `json:"from_state"`
	ToState       pgtype.Text        `json:"to_state"`
	Message       pgtype.Text        `json:"message"`
	Metadata      json.RawMessage    `json:"metadata"`
	CreatedAt     pgtype.Timestamptz `json:"created_at"`
}

func runJobsAudit(ctx context.Context, store db.Store, args []string) error {
	flags := flag.NewFlagSet("jobs audit", flag.ContinueOnError)
	jobIDValue := flags.String("id", "", "optional email job UUID")
	limit := flags.Int("limit", 100, "maximum audit records to return")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *limit <= 0 {
		return errors.New("limit must be greater than zero")
	}

	var (
		logs []db.AuditLog
		err  error
	)
	if *jobIDValue == "" {
		logs, err = store.ListRecentAuditLogs(ctx, int32(*limit))
	} else {
		id, parseErr := uuid.Parse(*jobIDValue)
		if parseErr != nil {
			return fmt.Errorf("invalid job ID: %w", parseErr)
		}
		logs, err = store.ListAuditLogsByJob(ctx, db.ListAuditLogsByJobParams{
			EntityID: pgtype.UUID{Bytes: id, Valid: true},
			Limit:    int32(*limit),
		})
	}
	if err != nil {
		return err
	}

	output := make([]auditLogOutput, 0, len(logs))
	for _, entry := range logs {
		output = append(output, auditLogOutput{
			ID:            entry.ID,
			EntityType:    entry.EntityType,
			EntityID:      entry.EntityID,
			CorrelationID: entry.CorrelationID,
			EventType:     entry.EventType,
			Actor:         entry.Actor,
			FromState:     entry.FromState,
			ToState:       entry.ToState,
			Message:       entry.Message,
			Metadata:      json.RawMessage(entry.Metadata),
			CreatedAt:     entry.CreatedAt,
		})
	}
	return json.NewEncoder(os.Stdout).Encode(output)
}

func parseCLIJobID(args []string) (pgtype.UUID, error) {
	flags := flag.NewFlagSet("jobs", flag.ContinueOnError)
	value := flags.String("id", "", "email job UUID")
	if err := flags.Parse(args); err != nil {
		return pgtype.UUID{}, err
	}
	id, err := uuid.Parse(*value)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid job ID: %w", err)
	}
	return pgtype.UUID{Bytes: id, Valid: true}, nil
}

func hostname() string {
	value, err := os.Hostname()
	if err != nil || value == "" {
		return uuid.NewString()
	}
	return value
}
