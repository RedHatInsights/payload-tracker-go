# Error Handling Guidelines

Rules for error handling in the payload-tracker-go codebase. This covers API handlers, Kafka consumer message processing, database operations, and logging conventions.

## API Handler Error Responses

### Use `getErrorBody()` and `writeResponse()` for all HTTP error responses

Every API handler in `internal/endpoints/` returns errors through the same two utility functions defined in `internal/endpoints/utils.go`. Do not write error JSON manually.

```go
writeResponse(w, http.StatusBadRequest, getErrorBody("descriptive message", http.StatusBadRequest))
return
```

The `getErrorBody()` function produces a `structs.ErrorResponse` with three fields: `title` (from `http.StatusText`), `message`, and `status` (integer code).

### Return immediately after writing an error response

Every error branch in a handler calls `writeResponse` then `return`. Do not continue processing after an error response.

### HTTP status code conventions

Use these status codes consistently with their existing meanings:

- **400 Bad Request** -- invalid query parameters (`sort_by`, `sort_dir`, `page`, `page_size`), invalid timestamps, invalid UUID format
- **401 Unauthorized** -- missing `x-rh-identity` header
- **403 Forbidden** -- identity header present but required role is missing
- **404 Not Found** -- payload lookup by `request_id` returns nil or empty slice; archive link URL is empty
- **500 Internal Server Error** -- `json.Marshal` failures, upstream service errors (e.g., storage-broker). Use the generic message `"Internal Server Issue"` for marshal failures

### Validation order in handlers

Handlers validate input in this order: (1) parse query parameters via `initQuery`, (2) validate `sort_by` against endpoint-specific allowed lists, (3) validate `sort_dir`, (4) validate timestamps. Each endpoint has its own `validXxxSortBy` slice -- use the correct one:

- `/payloads` uses `validAllSortBy`
- `/payloads/{request_id}` uses `validIDSortBy`
- `/statuses` uses `validStatusesSortBy`

### UUID validation for path parameters

Validate `request_id` path parameters with `isValidUUID()` (uses `github.com/google/uuid`) before making external requests. Increment `IncInvalidAPIRequestIDs()` when validation fails.

## Kafka Consumer Error Handling

### Silently drop invalid messages -- do not crash

In `internal/kafka/handler.go`, the `onMessage` function logs errors and returns early for bad messages. It does not return errors or panic. Each error in the message processing pipeline (unmarshal, upsert, table entry creation) is logged then the message is abandoned via `return`.

### Kafka event loop error types

In `internal/kafka/kafka.go`, the consumer event loop in `NewConsumerEventLoop` handles three Kafka event types:

- `*kafka.Message` -- processed via `handler.onMessage`
- `kafka.Error` -- logged with `l.Log.Errorf` and counted via `endpoints.IncConsumeErrors()`
- `kafka.OffsetsCommitted` -- only logged as error if `e.Error != nil`

### Conditional raw message logging

Only include the raw Kafka message value in unmarshal error logs when `cfg.DebugConfig.LogStatusJson` is `true`. This prevents leaking payload data in production logs.

```go
if cfg.DebugConfig.LogStatusJson {
    l.Log.Error("ERROR: Unmarshaling Payload Status Event: ", err, " Raw Message: ", string(msg.Value))
} else {
    l.Log.Error("ERROR: Unmarshaling Payload Status Event: ", err)
}
```

### Request ID validation in consumer

Use `validateRequestID()` which checks length against `cfg.RequestConfig.ValidateRequestIDLength` (default: 32). Invalid IDs increment `endpoints.IncInvalidConsumerRequestIDs()` and cause the message to be silently dropped.

## Database Error Handling

### Retry loop for payload status inserts

The final `InsertPayloadStatus` call in `onMessage` uses a retry loop controlled by `cfg.DatabaseConfig.DBRetries` (default: 3). The loop breaks on success. Failed attempts are logged at Debug level with the attempt count.

```go
retries, attempts := cfg.DatabaseConfig.DBRetries, 0
for retries > attempts {
    err := queries.InsertPayloadStatus(h.db, sanitizedPayloadStatus).Error
    if err == nil {
        break
    }
    l.Log.WithFields(logrus.Fields{"attempts": attempts}).Debug("Failed to insert sanitized PayloadStatus with ERROR: ", err)
    attempts += 1
}
```

No other database operations use retries. Upserts and table entry creates (status, service, source) fail immediately and abandon the message.

### GORM result pattern

Database write functions in `internal/queries/queries_consumer.go` return `*gorm.DB` (the result object) as the first return value. Check `.Error` on the result to detect failures.

### Fatal on DB connection failure

`db.DbConnect()` in `internal/db/db.go` calls `l.Log.Fatal(err)` if the initial GORM connection fails. This is the only place where a database error is fatal.

### Query functions do not return errors

`RetrievePayloads`, `RetrieveRequestIdPayloads`, and `RetrieveStatuses` in `internal/queries/queries_api.go` do not return error values. They execute GORM queries and return results directly. Handlers check for nil/empty results rather than errors.

## Startup and Configuration Errors

### Use `panic` for server startup failures

Both `cmd/payload-tracker-api/main.go` and `cmd/payload-tracker-consumer/main.go` call `panic(err)` when `http.Server.ListenAndServe` fails.

### Use `l.Log.Fatal` for unrecoverable initialization errors

Fatal is used for: database connection failure (`db.DbConnect`), Kafka consumer creation failure, and `PayloadFieldsRepository` initialization failure. These terminate the process.

### Use `panic` for Clowder CA failures

In `internal/config/config.go`, Kafka CA and RDS CA write failures use `panic("...")` with a descriptive string (not the error object).

## Logging Conventions

### Use the global `l.Log` logger

All files import `l "github.com/redhatinsights/payload-tracker-go/internal/logging"` and use `l.Log`. This is a `*logrus.Logger` instance initialized in `internal/logging/logging.go`.

### Log level conventions

- `l.Log.Error` / `l.Log.Errorf` -- for operation failures (DB inserts, unmarshal errors, external service errors)
- `l.Log.Debug` -- for retry attempt details and message processing traces
- `l.Log.Info` / `l.Log.Infof` -- for successful operations and lifecycle events
- `l.Log.Fatal` -- only for unrecoverable startup failures

### Prefix consumer error log messages with "ERROR"

Consumer-side error logs in `internal/kafka/handler.go` prefix messages with `"ERROR"` or `"ERROR:"` as a convention (e.g., `"ERROR Payload table upsert failed: "`, `"Error Creating Statuses Table Entry ERROR: "`).

### Use `logrus.Fields` for structured context

When logging with structured context, use `l.Log.WithFields(logrus.Fields{...})` as seen in role checking (`internal/endpoints/utils.go`) and retry logging (`internal/kafka/handler.go`).

## Prometheus Metrics for Errors

Increment the appropriate Prometheus counter when an error occurs. All counters are defined in `internal/endpoints/metrics.go`:

- `IncConsumeErrors()` -- Kafka consumer-level errors
- `IncMessageProcessErrors()` -- message processing errors (defined but not currently called in handler code)
- `IncInvalidConsumerRequestIDs()` -- invalid request IDs from Kafka messages
- `IncInvalidAPIRequestIDs()` -- invalid request IDs from API requests
- `responseCodes` counter (via `ResponseMetricsMiddleware`) -- tracks all HTTP response status codes automatically

## Date/Time Parsing Error Handling

`FormatedTime.UnmarshalJSON` in `internal/models/message/payload-status.go` attempts RFC3339 parsing first. If that fails, it appends `"Z"` and retries (to handle timezone-unaware timestamps). Both parse failures are logged, but only the final error is returned to the caller.
