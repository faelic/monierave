package observability

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type counter struct {
	value atomic.Uint64
}

func (counter *counter) Add(delta uint64) {
	counter.value.Add(delta)
}

func (counter *counter) Value() uint64 {
	return counter.value.Load()
}

type labeledCounter struct {
	mu     sync.RWMutex
	values map[string]*counter
}

func (labeled *labeledCounter) Add(labels string, delta uint64) {
	labeled.mu.RLock()
	value := labeled.values[labels]
	labeled.mu.RUnlock()
	if value == nil {
		labeled.mu.Lock()
		value = labeled.values[labels]
		if value == nil {
			value = &counter{}
			labeled.values[labels] = value
		}
		labeled.mu.Unlock()
	}
	value.Add(delta)
}

func (labeled *labeledCounter) Snapshot() map[string]uint64 {
	labeled.mu.RLock()
	defer labeled.mu.RUnlock()
	values := make(map[string]uint64, len(labeled.values))
	for labels, value := range labeled.values {
		values[labels] = value.Value()
	}
	return values
}

type Registry struct {
	requests       labeledCounter
	requestLatency labeledCounter
	transferResult labeledCounter
	rateLimited    labeledCounter
	databaseErrors counter
	workerRetries  counter

	outboxLagBits       atomic.Uint64
	dlqSize             atomic.Int64
	reconciliationDrift atomic.Int64
}

func NewRegistry() *Registry {
	return &Registry{
		requests:       labeledCounter{values: make(map[string]*counter)},
		requestLatency: labeledCounter{values: make(map[string]*counter)},
		transferResult: labeledCounter{values: make(map[string]*counter)},
		rateLimited:    labeledCounter{values: make(map[string]*counter)},
	}
}

var Default = NewRegistry()

func (registry *Registry) ObserveRequest(method, route string, status int, elapsed time.Duration) {
	labels := prometheusLabels(map[string]string{
		"method": method,
		"route":  route,
		"status": strconv.Itoa(status),
	})
	registry.requests.Add(labels, 1)
	registry.requestLatency.Add(labels, uint64(max(0, elapsed.Microseconds())))
}

func (registry *Registry) RecordTransfer(result string) {
	registry.transferResult.Add(prometheusLabels(map[string]string{"result": result}), 1)
}

func (registry *Registry) RecordRateLimit(endpoint string) {
	registry.rateLimited.Add(prometheusLabels(map[string]string{"endpoint": endpoint}), 1)
}

func (registry *Registry) RecordDatabaseError() {
	registry.databaseErrors.Add(1)
}

func (registry *Registry) RecordWorkerRetry() {
	registry.workerRetries.Add(1)
}

func (registry *Registry) SetWorkerRetries(value uint64) {
	for {
		current := registry.workerRetries.value.Load()
		if value <= current || registry.workerRetries.value.CompareAndSwap(current, value) {
			return
		}
	}
}

func (registry *Registry) SetOperationalGauges(
	outboxLagSeconds float64,
	dlqSize int64,
	reconciliationDrift int64,
	workerRetries int64,
) {
	registry.outboxLagBits.Store(math.Float64bits(max(0, outboxLagSeconds)))
	registry.dlqSize.Store(max(0, dlqSize))
	registry.reconciliationDrift.Store(max(0, reconciliationDrift))
	registry.SetWorkerRetries(uint64(max(0, workerRetries)))
}

func (registry *Registry) SetReconciliationDrift(value int64) {
	registry.reconciliationDrift.Store(max(0, value))
}

func (registry *Registry) Handler(refresh func() error) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		if refresh != nil {
			if err := refresh(); err != nil {
				registry.RecordDatabaseError()
			}
		}
		writer.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		registry.writePrometheus(writer)
	})
}

func (registry *Registry) writePrometheus(writer io.Writer) {
	writeLabeled(writer, "monierave_http_requests_total", "HTTP requests.", registry.requests.Snapshot())
	writeLabeled(
		writer,
		"monierave_http_request_duration_microseconds_total",
		"Cumulative HTTP request duration in microseconds.",
		registry.requestLatency.Snapshot(),
	)
	writeLabeled(
		writer,
		"monierave_transfer_results_total",
		"Transfer results by stable result code.",
		registry.transferResult.Snapshot(),
	)
	writeLabeled(
		writer,
		"monierave_rate_limited_requests_total",
		"Rate-limited requests by endpoint.",
		registry.rateLimited.Snapshot(),
	)
	fmt.Fprintf(
		writer,
		"# HELP monierave_database_errors_total Database operation errors.\n"+
			"# TYPE monierave_database_errors_total counter\n"+
			"monierave_database_errors_total %d\n",
		registry.databaseErrors.Value(),
	)
	fmt.Fprintf(
		writer,
		"# HELP monierave_worker_retries_total Worker retryable failures.\n"+
			"# TYPE monierave_worker_retries_total counter\n"+
			"monierave_worker_retries_total %d\n",
		registry.workerRetries.Value(),
	)
	fmt.Fprintf(
		writer,
		"# HELP monierave_outbox_lag_seconds Age of the oldest pending outbox event.\n"+
			"# TYPE monierave_outbox_lag_seconds gauge\n"+
			"monierave_outbox_lag_seconds %g\n",
		math.Float64frombits(registry.outboxLagBits.Load()),
	)
	fmt.Fprintf(
		writer,
		"# HELP monierave_email_dlq_size Dead-letter email job count.\n"+
			"# TYPE monierave_email_dlq_size gauge\n"+
			"monierave_email_dlq_size %d\n",
		registry.dlqSize.Load(),
	)
	fmt.Fprintf(
		writer,
		"# HELP monierave_reconciliation_drift_total Current reconciliation issue count.\n"+
			"# TYPE monierave_reconciliation_drift_total gauge\n"+
			"monierave_reconciliation_drift_total %d\n",
		registry.reconciliationDrift.Load(),
	)
}

func writeLabeled(writer io.Writer, name, help string, values map[string]uint64) {
	fmt.Fprintf(writer, "# HELP %s %s\n# TYPE %s counter\n", name, help, name)
	labels := make([]string, 0, len(values))
	for label := range values {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	for _, label := range labels {
		fmt.Fprintf(writer, "%s%s %d\n", name, label, values[label])
	}
}

func prometheusLabels(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	labels := make([]string, 0, len(keys))
	for _, key := range keys {
		value := strings.NewReplacer(
			"\\", "\\\\",
			"\"", "\\\"",
			"\n", "\\n",
		).Replace(values[key])
		labels = append(labels, fmt.Sprintf(`%s="%s"`, key, value))
	}
	return "{" + strings.Join(labels, ",") + "}"
}
