# Error Handling Guidelines

## API Error Responses

- Return errors using `getErrorBody(message, statusCode)` from `internal/endpoints/utils.go`, which produces a `structs.ErrorResponse` with `title`, `message`, and `status` fields.
- Pair every `getErrorBody` call with `writeResponse(w, statusCode, body)` to set `Content-Type: application/json` and write the status code.
- Use `http.StatusBadRequest` (400) for invalid query parameters (`sort_by`, `sort_dir`, `page`, timestamps), invalid UUIDs, and malformed input.
- Use `http.StatusNotFound` (404) when a database lookup returns nil or an empty slice (see `RequestIdPayloads` in `internal/endpoints/payloads.go`).
- Use `http.StatusUnauthorized` (401) for missing `x-rh-identity` headers and `http.StatusForbidden` (403) for insufficient roles.
- Use `http.StatusInternalServerError` (500) with a generic user-facing message like `"Internal Server Issue"` -- log the real error via `l.Log.Error(err)` before responding.

## Validation Before Database Calls

- Validate `sort_by` and `sort_dir` against the endpoint-specific allow-lists (`validAllSortBy`, `validIDSortBy`, `validStatusesSortBy`, `validSortDir`) defined in `internal/endpoints/utils.go` using `stringInSlice`.
- Validate timestamp query parameters with `validTimestamps(q, bool)` in `internal/endpoints/utils.go`, which parses against `time.RFC3339`.
- Validate UUID path parameters with `isValidUUID` from `internal/endpoints/utils.go` (uses `github.com/google/uuid`). Call `IncInvalidAPIRequestIDs()` when validation fails.
- Return 400 with a descriptive message immediately on validation failure -- do not proceed to database queries.

## Database Retry Logic

- The consumer retries `InsertPayloadStatus` using a counter loop governed by `cfg.DatabaseConfig.DBRetries` (default: 3, set via `db.retries` env var).
- Log each failed attempt at Debug level with the attempt count as a structured field: `l.Log.WithFields(logrus.Fields{"attempts": attempts}).Debug(...)`.
- After all retries are exhausted, log at Error level and increment `endpoints.IncMessageProcessErrors()`.
- The retry loop has no backoff delay between attempts. Do not add sleep-based backoff without also updating the corresponding test in `internal/kafka/handler_test.go` ("Should not insert with negative retries").
- Upsert and table-creation operations in the consumer (`UpsertPayloadByRequestId`, `CreateStatusTableEntry`, etc.) are not retried -- only the final `InsertPayloadStatus` call is.

## Kafka Consumer Error Handling

- On `json.Unmarshal` failure for incoming Kafka messages, log at Error level and call `endpoints.IncInvalidConsumerRequestIDs()`, then return early (do not retry).
- When `cfg.DebugConfig.LogStatusJson` is true, include the raw message bytes in the unmarshal error log. When false, omit the raw message to avoid leaking data.
- On request ID validation failure, call `endpoints.IncInvalidConsumerRequestIDs()` and return silently (no error log).
- On any DB write failure during message processing (upsert, table creation), log at Error and call `endpoints.IncMessageProcessErrors()`, then return early.
- In the Kafka event loop (`internal/kafka/kafka.go`), handle `kafka.Error` events by calling `endpoints.IncConsumeErrors()` and logging with `l.Log.Errorf`.

## Error Metrics

Increment error counters at each error site (see `logging-and-observability-guidelines.md` for complete metrics reference):
- `IncInvalidAPIRequestIDs()` - Invalid UUID in API request
- `IncInvalidConsumerRequestIDs()` - Invalid request ID or unmarshal failure in consumer
- `IncMessageProcessErrors()` - DB write failures during message processing
- `IncConsumeErrors()` - Kafka consumer-level errors

## Structured Logging

See `logging-and-observability-guidelines.md` for logger import and usage patterns. Error-specific patterns:
- Use `l.Log.Error(...)` for recoverable operation failures
- Use `l.Log.Fatal(...)` only for startup failures that should terminate the process
- Use `l.Log.WithFields(logrus.Fields{...})` to attach structured context (e.g., retry attempts, request IDs)

## Date/Time Unmarshaling

- The custom `FormatedTime.UnmarshalJSON` in `internal/models/message/payload-status.go` attempts `time.RFC3339` first, then appends `"Z"` and retries for non-timezone-aware timestamps.
- Both parse failures are logged at Error level. The error propagates up to `json.Unmarshal` in the Kafka handler.

## Fatal vs Panic

- Use `l.Log.Fatal(err)` for initialization failures in `internal/db/db.go` and `internal/kafka/kafka.go`.
- Use `panic(...)` for infrastructure-level failures during config loading (`internal/config/config.go`: Kafka CA write failure, RDS CA write failure) and HTTP server startup failures (`cmd/*/main.go`).

## Health Check

- `HealthCheckHandler` in `internal/endpoints/health.go` returns 500 with no body on DB connection errors (`db.DB()` or `Ping()` failure) -- it does not use `getErrorBody`.

## Verification

```bash
# Confirm all API error responses use getErrorBody
grep -rn "writeResponse.*http\.Status" internal/endpoints/ --include="*.go" | grep -v "getErrorBody\|StatusOK"

# Confirm error metrics are incremented at error sites in the consumer
grep -n "IncMessageProcessErrors\|IncInvalidConsumerRequestIDs\|IncConsumeErrors" internal/kafka/ --include="*.go"

# Run tests to verify error response codes
make test

# Check for unhandled errors (err declared but not checked)
grep -n "err :=" internal/ -r --include="*.go" | grep -v "if err\|Expect(err"

# Lint formatting
make lint
```
