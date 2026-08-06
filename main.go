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
	osuser "os/user"
	"strconv"
	"strings"
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
	"github.com/jackc/pgx/v5"
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
	case "banking":
		err = runBanking(config, os.Args[2:])
	default:
		err = fmt.Errorf(
			"unknown command %q; expected api, relay, worker, jobs, or banking",
			role,
		)
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

type bankingCommandOutput struct {
	TransactionID string `json:"transaction_id"`
	Reference     string `json:"reference"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	AccountID     string `json:"account_id"`
	Balance       int64  `json:"balance"`
	Currency      string `json:"currency"`
	Actor         string `json:"actor"`
	CorrelationID string `json:"correlation_id"`
	AuditLogID    int64  `json:"audit_log_id"`
}

func runBanking(config util.Config, args []string) error {
	if config.DBSource == "" {
		return errors.New("DB_SOURCE is required")
	}
	if len(args) == 0 {
		return errors.New("banking command requires deposit, withdraw, freeze, unfreeze, reverse, reconcile, or audit")
	}
	switch args[0] {
	case "deposit", "withdraw", "freeze", "unfreeze", "reverse", "reconcile", "audit":
	default:
		return fmt.Errorf(
			"unknown banking command %q; expected deposit, withdraw, freeze, unfreeze, reverse, reconcile, or audit",
			args[0],
		)
	}

	ctx := context.Background()
	pool, store, err := openStore(ctx, config.DBSource)
	if err != nil {
		return err
	}
	defer pool.Close()

	switch args[0] {
	case "deposit", "withdraw":
		return runBankingMovement(ctx, store, args[0], args[1:])
	case "freeze", "unfreeze":
		return runBankingAccountControl(ctx, store, args[0], args[1:])
	case "reverse":
		return runBankingReversal(ctx, store, args[1:])
	case "reconcile":
		return runBankingReconcile(ctx, store, args[1:])
	case "audit":
		return runBankingAudit(ctx, store, args[1:])
	}
	return nil
}

func runBankingMovement(
	ctx context.Context,
	store db.Store,
	command string,
	args []string,
) error {
	flags := flag.NewFlagSet("banking "+command, flag.ContinueOnError)
	accountValue := flags.String("account", "", "public account UUID")
	amount := flags.Int64("amount", 0, "positive amount in minor currency units")
	narration := flags.String("narration", "", "operator transaction narration")
	actor := flags.String("actor", defaultOperatorActor(), "operator identity")
	if err := flags.Parse(args); err != nil {
		return err
	}
	accountID, err := uuid.Parse(*accountValue)
	if err != nil {
		return fmt.Errorf("invalid account ID: %w", err)
	}
	if *amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if strings.TrimSpace(*actor) == "" {
		return errors.New("actor is required")
	}

	publicID := pgtype.UUID{Bytes: accountID, Valid: true}
	correlationID := uuid.New()
	var result db.MoneyMovementTxResult
	switch command {
	case "deposit":
		result, err = store.DepositTx(ctx, db.DepositTxParams{
			AccountPublicID: publicID,
			Amount:          *amount,
			Narration:       *narration,
			Actor:           strings.TrimSpace(*actor),
			CorrelationID:   pgtype.UUID{Bytes: correlationID, Valid: true},
		})
	case "withdraw":
		result, err = store.WithdrawTx(ctx, db.WithdrawTxParams{
			AccountPublicID: publicID,
			Amount:          *amount,
			Narration:       *narration,
			Actor:           strings.TrimSpace(*actor),
			CorrelationID:   pgtype.UUID{Bytes: correlationID, Valid: true},
		})
	}
	if err != nil {
		return err
	}

	return json.NewEncoder(os.Stdout).Encode(bankingCommandOutput{
		TransactionID: uuid.UUID(result.Transaction.ID.Bytes).String(),
		Reference:     result.Transaction.Reference,
		Type:          result.Transaction.TransactionType,
		Status:        result.Transaction.Status,
		AccountID:     uuid.UUID(result.Account.PublicID.Bytes).String(),
		Balance:       result.Account.Balance,
		Currency:      result.Account.Currency,
		Actor:         strings.TrimSpace(*actor),
		CorrelationID: correlationID.String(),
		AuditLogID:    result.AuditLog.ID,
	})
}

type accountControlOutput struct {
	AccountID     string `json:"account_id"`
	Status        string `json:"status"`
	Actor         string `json:"actor"`
	Reason        string `json:"reason"`
	CorrelationID string `json:"correlation_id"`
	AuditLogID    int64  `json:"audit_log_id"`
}

func runBankingAccountControl(
	ctx context.Context,
	store db.Store,
	command string,
	args []string,
) error {
	flags := flag.NewFlagSet("banking "+command, flag.ContinueOnError)
	accountValue := flags.String("account", "", "public account UUID")
	defaultReason := ""
	if command == "unfreeze" {
		defaultReason = "Operator unfreeze"
	}
	reason := flags.String("reason", defaultReason, "required operator reason")
	actor := flags.String("actor", defaultOperatorActor(), "operator identity")
	if err := flags.Parse(args); err != nil {
		return err
	}
	accountID, err := uuid.Parse(*accountValue)
	if err != nil {
		return fmt.Errorf("invalid account ID: %w", err)
	}
	if strings.TrimSpace(*reason) == "" {
		return errors.New("reason is required")
	}
	if strings.TrimSpace(*actor) == "" {
		return errors.New("actor is required")
	}

	targetStatus := db.FinancialAccountStatusFrozen
	if command == "unfreeze" {
		targetStatus = db.FinancialAccountStatusActive
	}
	correlationID := uuid.New()
	result, err := store.SetAccountStatusTx(ctx, db.AccountStatusTxParams{
		AccountPublicID: pgtype.UUID{Bytes: accountID, Valid: true},
		TargetStatus:    targetStatus,
		Actor:           strings.TrimSpace(*actor),
		Reason:          strings.TrimSpace(*reason),
		CorrelationID: pgtype.UUID{
			Bytes: correlationID, Valid: true,
		},
	})
	if err != nil {
		return err
	}

	return json.NewEncoder(os.Stdout).Encode(accountControlOutput{
		AccountID:     uuid.UUID(result.Account.PublicID.Bytes).String(),
		Status:        result.Account.Status,
		Actor:         strings.TrimSpace(*actor),
		Reason:        strings.TrimSpace(*reason),
		CorrelationID: correlationID.String(),
		AuditLogID:    result.AuditLog.ID,
	})
}

type reversalAccountOutput struct {
	AccountID string `json:"account_id"`
	Balance   int64  `json:"balance"`
	Currency  string `json:"currency"`
	Status    string `json:"status"`
}

type reversalOutput struct {
	OriginalTransactionID string                  `json:"original_transaction_id"`
	OriginalReference     string                  `json:"original_reference"`
	OriginalStatus        string                  `json:"original_status"`
	ReversalTransactionID string                  `json:"reversal_transaction_id"`
	ReversalReference     string                  `json:"reversal_reference"`
	Actor                 string                  `json:"actor"`
	Reason                string                  `json:"reason"`
	CorrelationID         string                  `json:"correlation_id"`
	AuditLogID            int64                   `json:"audit_log_id"`
	Accounts              []reversalAccountOutput `json:"accounts"`
}

func runBankingReversal(
	ctx context.Context,
	store db.Store,
	args []string,
) error {
	flags := flag.NewFlagSet("banking reverse", flag.ContinueOnError)
	transactionValue := flags.String("transaction", "", "original transaction UUID")
	reason := flags.String("reason", "", "required reversal reason")
	actor := flags.String("actor", defaultOperatorActor(), "operator identity")
	if err := flags.Parse(args); err != nil {
		return err
	}
	transactionID, err := uuid.Parse(*transactionValue)
	if err != nil {
		return fmt.Errorf("invalid transaction ID: %w", err)
	}
	if strings.TrimSpace(*reason) == "" {
		return errors.New("reason is required")
	}
	if strings.TrimSpace(*actor) == "" {
		return errors.New("actor is required")
	}

	correlationID := uuid.New()
	result, err := store.ReverseTransactionTx(
		ctx,
		db.ReverseTransactionTxParams{
			TransactionID: pgtype.UUID{
				Bytes: transactionID, Valid: true,
			},
			Actor:  strings.TrimSpace(*actor),
			Reason: strings.TrimSpace(*reason),
			CorrelationID: pgtype.UUID{
				Bytes: correlationID, Valid: true,
			},
		},
	)
	if err != nil {
		return err
	}
	accounts := make([]reversalAccountOutput, 0, len(result.Accounts))
	for _, account := range result.Accounts {
		accounts = append(accounts, reversalAccountOutput{
			AccountID: uuid.UUID(account.PublicID.Bytes).String(),
			Balance:   account.Balance,
			Currency:  account.Currency,
			Status:    account.Status,
		})
	}
	return json.NewEncoder(os.Stdout).Encode(reversalOutput{
		OriginalTransactionID: uuid.UUID(result.Original.ID.Bytes).String(),
		OriginalReference:     result.Original.Reference,
		OriginalStatus:        result.Original.Status,
		ReversalTransactionID: uuid.UUID(result.Reversal.ID.Bytes).String(),
		ReversalReference:     result.Reversal.Reference,
		Actor:                 strings.TrimSpace(*actor),
		Reason:                strings.TrimSpace(*reason),
		CorrelationID:         correlationID.String(),
		AuditLogID:            result.AuditLog.ID,
		Accounts:              accounts,
	})
}

func defaultOperatorActor() string {
	if value := strings.TrimSpace(os.Getenv("MONIERAVE_OPERATOR")); value != "" {
		return value
	}
	current, err := osuser.Current()
	if err == nil && strings.TrimSpace(current.Username) != "" {
		return current.Username
	}
	return "operator_cli"
}

var errReconciliationDrift = errors.New("reconciliation detected financial drift")

type reconciliationOutput struct {
	RunID           string                   `json:"run_id"`
	AccountPublicID string                   `json:"account_id,omitempty"`
	CheckedAt       time.Time                `json:"checked_at"`
	Consistent      bool                     `json:"consistent"`
	IssueCount      int                      `json:"issue_count"`
	Issues          []db.ReconciliationIssue `json:"issues"`
	AuditLogID      int64                    `json:"audit_log_id"`
}

func runBankingReconcile(
	ctx context.Context,
	store db.Store,
	args []string,
) error {
	flags := flag.NewFlagSet("banking reconcile", flag.ContinueOnError)
	accountValue := flags.String("account", "", "optional public account UUID")
	actor := flags.String("actor", defaultOperatorActor(), "operator identity")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*actor) == "" {
		return errors.New("actor is required")
	}

	var accountID pgtype.UUID
	if strings.TrimSpace(*accountValue) != "" {
		parsed, err := uuid.Parse(*accountValue)
		if err != nil {
			return fmt.Errorf("invalid account ID: %w", err)
		}
		accountID = pgtype.UUID{Bytes: parsed, Valid: true}
	}

	report, err := store.Reconcile(ctx, db.ReconciliationParams{
		AccountPublicID: accountID,
		Actor:           strings.TrimSpace(*actor),
	})
	if err != nil {
		return err
	}
	output := reconciliationOutput{
		RunID:           uuid.UUID(report.RunID.Bytes).String(),
		AccountPublicID: uuidString(report.AccountPublicID),
		CheckedAt:       report.CheckedAt,
		Consistent:      report.Consistent,
		IssueCount:      len(report.Issues),
		Issues:          report.Issues,
		AuditLogID:      report.AuditLog.ID,
	}
	if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
		return err
	}
	if !report.Consistent {
		return errReconciliationDrift
	}
	return nil
}

type financialAuditPostingOutput struct {
	Amount    int64     `json:"amount"`
	Ledger    string    `json:"ledger"`
	AccountID string    `json:"account_id,omitempty"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
}

type transactionAuditOutput struct {
	TransactionID string                        `json:"transaction_id"`
	Reference     string                        `json:"reference"`
	Type          string                        `json:"type"`
	Status        string                        `json:"status"`
	Currency      string                        `json:"currency"`
	Amount        int64                         `json:"amount"`
	Narration     string                        `json:"narration"`
	ReversalOf    string                        `json:"reversal_of,omitempty"`
	Postings      []financialAuditPostingOutput `json:"postings"`
	AuditLogs     []auditLogOutput              `json:"audit_logs"`
}

type accountAuditOutput struct {
	AccountID     string           `json:"account_id"`
	Owner         string           `json:"owner"`
	Currency      string           `json:"currency"`
	Status        string           `json:"status"`
	CachedBalance int64            `json:"cached_balance"`
	LedgerExists  bool             `json:"ledger_exists"`
	LedgerBalance int64            `json:"ledger_balance"`
	AuditLogs     []auditLogOutput `json:"audit_logs"`
}

func runBankingAudit(
	ctx context.Context,
	store db.Store,
	args []string,
) error {
	flags := flag.NewFlagSet("banking audit", flag.ContinueOnError)
	transactionValue := flags.String("transaction", "", "transaction UUID")
	accountValue := flags.String("account", "", "public account UUID")
	if err := flags.Parse(args); err != nil {
		return err
	}
	hasTransaction := strings.TrimSpace(*transactionValue) != ""
	hasAccount := strings.TrimSpace(*accountValue) != ""
	if hasTransaction == hasAccount {
		return errors.New("provide exactly one of --transaction or --account")
	}
	if hasTransaction {
		return runTransactionAudit(ctx, store, *transactionValue)
	}
	return runAccountAudit(ctx, store, *accountValue)
}

func runTransactionAudit(
	ctx context.Context,
	store db.Store,
	value string,
) error {
	id, err := uuid.Parse(value)
	if err != nil {
		return fmt.Errorf("invalid transaction ID: %w", err)
	}
	transactionID := pgtype.UUID{Bytes: id, Valid: true}
	transaction, err := store.GetBankingTransaction(ctx, transactionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.ErrTransactionNotFound
	}
	if err != nil {
		return err
	}
	postings, err := store.ListFinancialAuditPostings(ctx, transactionID)
	if err != nil {
		return err
	}
	logs, err := store.ListAuditLogsByEntity(ctx, db.ListAuditLogsByEntityParams{
		EntityType: "banking_transaction",
		EntityID:   transactionID,
	})
	if err != nil {
		return err
	}

	postingOutput := make([]financialAuditPostingOutput, 0, len(postings))
	for _, posting := range postings {
		ledgerName := posting.Kind
		if posting.Code.Valid {
			ledgerName = posting.Code.String
		}
		postingOutput = append(postingOutput, financialAuditPostingOutput{
			Amount:    posting.Amount,
			Ledger:    ledgerName,
			AccountID: uuidString(posting.AccountPublicID),
			Currency:  posting.Currency,
			CreatedAt: posting.CreatedAt.Time,
		})
	}
	return json.NewEncoder(os.Stdout).Encode(transactionAuditOutput{
		TransactionID: uuid.UUID(transaction.ID.Bytes).String(),
		Reference:     transaction.Reference,
		Type:          transaction.TransactionType,
		Status:        transaction.Status,
		Currency:      transaction.Currency,
		Amount:        transaction.Amount,
		Narration:     transaction.Narration,
		ReversalOf:    uuidString(transaction.ReversalOf),
		Postings:      postingOutput,
		AuditLogs:     newAuditLogOutputs(logs),
	})
}

func runAccountAudit(
	ctx context.Context,
	store db.Store,
	value string,
) error {
	id, err := uuid.Parse(value)
	if err != nil {
		return fmt.Errorf("invalid account ID: %w", err)
	}
	accountID := pgtype.UUID{Bytes: id, Valid: true}
	account, err := store.GetAccountByPublicID(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.ErrAccountNotFound
	}
	if err != nil {
		return err
	}

	var ledgerBalance int64
	ledgerExists := true
	ledger, err := store.GetCustomerLedgerAccount(
		ctx,
		pgtype.Int8{Int64: account.ID, Valid: true},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		ledgerExists = false
	} else if err != nil {
		return err
	} else {
		ledgerBalance, err = store.GetLedgerAccountBalance(ctx, ledger.ID)
		if err != nil {
			return err
		}
	}
	logs, err := store.ListAccountFinancialAuditLogs(ctx, accountID)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(accountAuditOutput{
		AccountID:     uuid.UUID(account.PublicID.Bytes).String(),
		Owner:         account.Owner,
		Currency:      account.Currency,
		Status:        account.Status,
		CachedBalance: account.Balance,
		LedgerExists:  ledgerExists,
		LedgerBalance: ledgerBalance,
		AuditLogs:     newAuditLogOutputs(logs),
	})
}

func uuidString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return uuid.UUID(value.Bytes).String()
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

	return json.NewEncoder(os.Stdout).Encode(newAuditLogOutputs(logs))
}

func newAuditLogOutputs(logs []db.AuditLog) []auditLogOutput {
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
	return output
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
