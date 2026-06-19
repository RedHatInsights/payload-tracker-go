# Logging and Observability Guidelines

## Logging

### Logger Setup
- Import the global logger via the alias `l "github.com/redhatinsights/payload-tracker-go/internal/logging"` and reference it as `l.Log`.
- When you need both the alias and the full package path (as in `internal/endpoints/payloads.go`), import the `logging` package directly only for the secondary reference; prefer the `l` alias for log calls.
- Call `logging.InitLogger()` at the start of `main()` before any other initialization. The logger is not safe to use before this call.

### Log Format
- All log output uses the `CustomCloudwatch` JSON formatter defined in `internal/logging/logging.go`. Each entry contains `@timestamp`, `@version`, `message`, `levelname`, `source_host`, `app` (hardcoded to `"payload-tracker"`), and `caller`.
- Do not introduce a second formatter or switch to a text formatter; the JSON structure is consumed by CloudWatch.

### Log Levels
- The `LOG_LEVEL` environment variable controls the level; valid values are `DEBUG`, `ERROR`, and `INFO` (default).
- During tests (`flag.Lookup("test.v") != nil`), the level is forced to `FatalLevel` to suppress output. Do not add log-level overrides that bypass this.
- Use `l.Log.Debug` for per-message processing details (e.g., raw Kafka message bodies).
- Use `l.Log.Info` for lifecycle events (DB connected, Kafka connected, signal caught).
- Use `l.Log.Error` for operation failures (unmarshal errors, DB insert failures, identity header issues).
- Use `l.Log.Fatal` only for unrecoverable startup failures (DB connection, Kafka consumer creation, cache initialization).

### Structured Fields
- Use `l.Log.WithFields(logrus.Fields{...})` when attaching contextual key-value pairs (see `internal/endpoints/utils.go:153` and `internal/kafka/handler.go:136`).
- Import `logrus` directly from `"github.com/sirupsen/logrus"` when you need `logrus.Fields`.
- Avoid passing structured data as positional arguments to `l.Log.Error()`; prefer `WithFields` for anything beyond simple error messages.

### Debug Guarding
- Guard verbose message content behind the `cfg.DebugConfig.LogStatusJson` flag (env var `DEBUG_LOG_STATUS_JSON`). See `internal/kafka/handler.go:37-41` for the pattern: log raw message bodies only when this flag is true.

### CloudWatch Integration
- CloudWatch is configured via Clowder-provided credentials (`CW_AWS_ACCESS_KEY_ID`, `CW_AWS_SECRET_ACCESS_KEY`, `CW_REGION`, `LOG_GROUP`). The hook is added only when `CWAccessKey` is non-empty.
- Uses `platform-go-middlewares/v2/logging/cloudwatch` with a 10-second batch writer duration.

## Prometheus Metrics

### Metric Definitions
All metrics are defined in `internal/endpoints/metrics.go` using `promauto` (auto-registered). Every metric name is prefixed with `payload_tracker_`.

| Metric Name | Type | Purpose |
|---|---|---|
| `payload_tracker_requests` | Counter | Total API requests |
| `payload_tracker_responses` | Counter | Response counts, labeled by `code` |
| `payload_tracker_db_seconds` | Histogram | DB query latency (seconds) |
| `payload_tracker_consumed_messages` | Counter | Kafka messages consumed |
| `payload_tracker_consume_errors` | Counter | Kafka consumption errors |
| `payload_tracker_messages_processed` | Counter | Messages successfully processed |
| `payload_tracker_message_process_seconds` | Histogram | Message processing latency |
| `payload_tracker_message_process_errors` | Counter | Message processing failures |
| `payload_tracker_api_invalid_request_IDs` | Counter | Invalid UUIDs on API endpoints |
| `payload_tracker_consumer_invalid_request_IDs` | Counter | Invalid request IDs from Kafka |

### Incrementing Metrics
- Use the exported `Inc*` and `Observe*` functions in `internal/endpoints/metrics.go` (e.g., `IncConsumedMessages()`, `ObserveMessageProcessTime(elapsed)`). Do not call `.With(p.Labels{}).Inc()` directly from other packages.
- When adding a new counter, follow the existing pattern: define a `pa.NewCounterVec` at package level, then create an exported `Inc<Name>()` wrapper.
- Histograms accept `time.Duration` and convert via `.Seconds()` internally.

### Response Code Tracking
- The `ResponseMetricsMiddleware` wraps `http.ResponseWriter` to capture status codes. Apply it via `sub.With(endpoints.ResponseMetricsMiddleware)` on API routes (see `cmd/payload-tracker-api/main.go:64-70`).
- The middleware records the `code` label as a string (e.g., `"200"`, `"404"`).

### Pairing Metrics with Logging
- When logging an error in the consumer, also increment the corresponding error counter. See `internal/kafka/handler.go` where each `l.Log.Error` call is followed by `endpoints.IncMessageProcessErrors()` or `endpoints.IncInvalidConsumerRequestIDs()`.

## Metrics Endpoint Architecture

### Dedicated Metrics Port
- Both `pt-api` and `pt-consumer` serve Prometheus metrics on a **separate port** (`MetricsPort`, default `8081` locally, Clowder-assigned in production) from the application API (`PublicPort`, default `8080`).
- The metrics server is started in a goroutine; the main server (API) or event loop (consumer) runs on the main goroutine.
- The metrics router mounts `promhttp.Handler()` at `/metrics` and a heartbeat (`lubdub`) at `/`.
- The consumer's metrics router also exposes `/live` and `/ready` health endpoints used by Kubernetes probes (see `deployments/clowdapp.yml:93-107`).

## Grafana Dashboard

The dashboard definition lives at `dashboards/grafana-dashboard-insights-payload-tracker-general.configmap.yaml` as a Kubernetes ConfigMap. Key panels and their PromQL queries:

- **Messages Processed**: `payload_tracker_messages_processed{container="payload-tracker-consumer"}`
- **API Responses (rate)**: `sum(increase(payload_tracker_responses[1m])) by (code)`
- **API Response Counts**: `sum(sum_over_time(payload_tracker_responses[$__interval])) by (code)`
- **Consumption Errors (SLO 5%)**: `sum(increase(payload_tracker_consume_errors[$__range])) / sum(increase(payload_tracker_consumed_messages[$__range]))`
- **Invalid Consumer Request Count**: `sum(increase(payload_tracker_consumer_invalid_request_IDs[$__range]))`
- **Non-5xx Percentage (SLO 95%)**: `sum(sum_over_time(payload_tracker_responses{code!~"5.*"}[$__interval])) / sum(sum_over_time(payload_tracker_responses[$__interval]))`
- **Database Query Times**: `sum(payload_tracker_db_seconds_bucket) by (le)`
- **Message Processing Times**: `sum(payload_tracker_message_process_seconds_bucket) by (le)`

When adding a new metric, add a corresponding panel to this ConfigMap so it appears on the Grafana dashboard.

## Verification

```bash
# Run all tests (log output suppressed at FatalLevel during tests)
make test

# Build both binaries to confirm metrics/logging code compiles
make build-all

# Verify all metric names follow the payload_tracker_ prefix convention
grep -n 'Name:' internal/endpoints/metrics.go | grep -v 'payload_tracker_'

# Verify every Inc*/Observe* wrapper has a corresponding metric var
grep -n '^func Inc\|^func Observe\|^func inc' internal/endpoints/metrics.go

# Check that new log calls use the l.Log alias pattern
grep -rn 'logrus\.New\|logrus\.StandardLogger\|log\.New' internal/ --include="*.go"

# Confirm metrics endpoint is mounted on the metrics router (mr), not the API router (r)
grep -n 'promhttp' cmd/payload-tracker-api/main.go cmd/payload-tracker-consumer/main.go

# Lint formatting
make lint
```
