# Logging and Observability Guidelines

This repo uses logrus for structured logging with CloudWatch forwarding, and Prometheus for metrics. Both the API (`cmd/payload-tracker-api`) and the consumer (`cmd/payload-tracker-consumer`) follow the same patterns.

## Logger Setup

- Use the singleton logger at `internal/logging/logging.go`. Access it as `logging.Log` or alias the package as `l` and use `l.Log`.
- Call `logging.InitLogger()` once at the start of `main()`, before any other initialization. Both entry points (`cmd/payload-tracker-api/main.go` and `cmd/payload-tracker-consumer/main.go`) do this first.
- Do not create your own `logrus.Logger` instances. Use the shared `logging.Log`.

## Logger Import Convention

The logging package is imported two ways in this codebase. Follow the convention used in the file you are editing:

```go
// Most files use the alias:
l "github.com/redhatinsights/payload-tracker-go/internal/logging"
// Then: l.Log.Info(...)

// Some files import without alias:
"github.com/redhatinsights/payload-tracker-go/internal/logging"
// Then: logging.Log.Info(...)
```

Prefer the `l` alias for new code to match the majority of files.

## Log Levels

The config (`internal/config/config.go`) supports three levels via the `logLevel` config key:

| Level | Use for |
|-------|---------|
| `DEBUG` | Message processing details, DB retry attempts, generated links |
| `INFO` | Startup milestones ("Setting up DB", "Connected to Kafka"), successful operations with audit value (archive link generation) |
| `ERROR` | Failed operations: unmarshal errors, DB upsert/insert failures, identity header decode failures |
| `FATAL` | Unrecoverable startup failures: DB connection failure, Kafka consumer creation failure |

During tests, the log level is automatically set to `FatalLevel` to suppress output. Do not add test-specific log suppression logic.

## Log Format

All logs are JSON-formatted via `CustomCloudwatch` formatter in `internal/logging/logging.go`. Each log entry automatically includes:
- `@timestamp`, `@version`, `message`, `levelname`
- `source_host` (hostname), `app` (hardcoded `"payload-tracker"`), `caller` (function name via `ReportCaller: true`)

Do not manually add these fields to log messages.

## Structured Fields with WithFields

Use `logrus.Fields` with `WithFields` when logging contextual data that should be machine-parseable. This repo uses it sparingly -- only for retry attempts and role-checking:

```go
l.Log.WithFields(logrus.Fields{"attempts": attempts}).Debug("Failed to insert ...")
l.Log.WithFields(logrus.Fields{
    "role":              role,
    "roles_from_header": identityHeaderData.Identity.Associate.Roles,
    "identity_header":   identityHeader,
}).Infof("Unable to find required role")
```

For simple error logging, the codebase uses positional args instead: `l.Log.Error("ERROR: description: ", err)`.

## Error Logging Conventions

- Prefix error messages with the operation context: `"ERROR: Unmarshaling Payload Status Event: "`, `"ERROR Payload table upsert failed: "`, `"Error Creating Statuses Table Entry ERROR: "`.
- Pass the error as the last argument: `l.Log.Error("description: ", err)`.
- Use `Errorf` for formatted strings with request context: `l.Log.Errorf("Error getting archive link from storage-broker for request id: %s, error: %v", reqID, err)`.
- For JSON unmarshal errors, conditionally log the raw message only when `DebugConfig.LogStatusJson` is true (controlled by config key `debug.log.status.json`), to avoid logging sensitive payload data in production.

## CloudWatch Integration

CloudWatch forwarding is configured automatically in `InitLogger()` when `CWAccessKey` is set (populated from Clowder config in production). The hook uses `platform-go-middlewares/v2/logging/cloudwatch` with a 10-second batch write interval. No additional setup is needed -- the hook is added to the global logger's hooks.

## Prometheus Metrics

All metrics are defined in `internal/endpoints/metrics.go` using `promauto` (auto-registered). The metrics server runs on a separate port (`MetricsPort`, default `8081`) serving `/metrics` via `promhttp.Handler()`.

### Existing Metrics

| Metric Name | Type | Labels | Purpose |
|---|---|---|---|
| `payload_tracker_requests` | Counter | none | Total API requests |
| `payload_tracker_responses` | Counter | `code` | Response counts by HTTP status code |
| `payload_tracker_db_seconds` | Histogram | none | DB query latency |
| `payload_tracker_consumed_messages` | Counter | none | Kafka messages consumed |
| `payload_tracker_consume_errors` | Counter | none | Kafka consume errors |
| `payload_tracker_messages_processed` | Counter | none | Successfully processed messages |
| `payload_tracker_message_process_errors` | Counter | none | Message processing errors |
| `payload_tracker_message_process_seconds` | Histogram | none | Message processing latency |
| `payload_tracker_api_invalid_request_IDs` | Counter | none | Invalid request IDs (API) |
| `payload_tracker_consumer_invalid_request_IDs` | Counter | none | Invalid request IDs (consumer) |

### Adding New Metrics

- Define metrics as package-level `var` in `internal/endpoints/metrics.go` using `promauto.NewCounterVec` or `promauto.NewHistogramVec`.
- Prefix all metric names with `payload_tracker_`.
- Create exported `Inc*()` or `Observe*()` helper functions for metrics used outside the `endpoints` package (the consumer calls `endpoints.IncConsumedMessages()`, `endpoints.ObserveMessageProcessTime()`, etc.).
- Keep unexported helpers (lowercase) for metrics used only within the `endpoints` package (e.g., `incRequests()`, `observeDBTime()`).

### Response Metrics Middleware

`ResponseMetricsMiddleware` in `internal/endpoints/metrics.go` wraps `http.ResponseWriter` to track response codes. Apply it to all API routes using Chi's `With()`:

```go
sub.With(endpoints.ResponseMetricsMiddleware).Get("/payloads", endpoints.Payloads)
```

This middleware intercepts `WriteHeader` to increment `payload_tracker_responses` with the status code label.

## Timing Observations

For DB queries in API endpoints, capture timing with:

```go
start := time.Now()
// ... DB operation ...
observeDBTime(time.Since(start))
```

For Kafka message processing, call both the timing and count metrics at the end of processing:

```go
endpoints.ObserveMessageProcessTime(time.Since(start))
endpoints.IncMessagesProcessed()
```

On processing errors, increment the error counter instead: `endpoints.IncMessageProcessErrors()`.

## Grafana Dashboard

The dashboard definition lives in `dashboards/grafana-dashboard-insights-payload-tracker-general.configmap.yaml`. It visualizes: API uptime, consumer uptime, response codes, DB query time heatmap, message processing time heatmap, consumer error rate, invalid request ID counts, Kafka topic lag, and pod restart counts. When adding a new Prometheus metric, add a corresponding panel to this dashboard.

## What Not to Log

- Do not log raw Kafka message payloads at `INFO` or `ERROR` level. Use `DEBUG` level, and respect the `LogStatusJson` config flag for error cases.
- Do not log database credentials or connection strings. The `db.go` module logs only `"DB initialization complete"` on success.
- Do not log the full `x-rh-identity` header contents at error level -- the repo logs it only at `INFO` level during successful archive link generation.
