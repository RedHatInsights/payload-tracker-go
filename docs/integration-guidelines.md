# Integration Guidelines

Rules for external service integrations, HTTP clients, Kafka consumption, and inter-service communication in payload-tracker-go.

## Storage-Broker HTTP Client

### Requestor Implementation Selection

The `RequestorImpl` config field (`requestor.impl` env var) controls which archive link implementation is used. `CreatePayloadArchiveLinkHandler` in `internal/endpoints/payloads.go` switches on this value:

- `"storage-broker"` -- calls the real storage-broker service via `RequestArchiveLink`
- `"mock"` -- returns a synthetic URL pointing back to the local API via `MockArchiveLink`

When adding a new requestor implementation, add a new case in `CreatePayloadArchiveLinkHandler` and return an `http.HandlerFunc`. Log an error and return `nil` for unsupported values.

### HTTP Client Timeout Convention

The storage-broker HTTP client in `RequestArchiveLink` (`internal/endpoints/utils.go`) creates a new `http.Client` per closure invocation with `Timeout` set from config in **milliseconds**:

```go
client := http.Client{
    Timeout: time.Duration(timeout) * time.Millisecond,
}
```

The default is 35000ms (35 seconds), configured via `storageBrokerRequestTimeout`. When adding new HTTP clients for external services, follow this pattern: accept timeout as `int` (milliseconds) from config, convert with `time.Duration(timeout) * time.Millisecond`.

### Storage-Broker URL Construction

The storage-broker request appends `?request_id=<uuid>` to the base URL. The base URL default is `http://storage-broker-processor:8000/archive/url`. Do not include query parameters in the base URL config.

### Archive Link Role Authorization

The `PayloadArchiveLink` and `RolesArchiveLink` endpoints require the `x-rh-identity` header with a specific role. The required role is configured via `storageBrokerURLRole` (default: `"platform-archive-download"`). Use `checkForRole` from `internal/endpoints/utils.go` for role validation. This function:

1. Returns `401` if the identity header is missing
2. Returns `401` if it cannot be base64-decoded or unmarshalled
3. Returns `403` if the required role is not in the associate's Role list
4. Returns `200` on success

## Kibana Link Generation

`PayloadKibanaLink` in `internal/endpoints/payloads.go` constructs a Kibana URL from three config values in `KibanaConfig`:

- `DashboardURL` -- base Kibana discover URL
- `Index` -- the Kibana index pattern ID
- `ServiceField` -- field name used for service filtering (default: `"app"`)

The query string is built as `request_id:<uuid>`, with an optional ` AND <serviceField>:<service>` clause when the `service` query param is provided. UUID validation via `isValidUUID` is required before generating the link; return 400 for invalid UUIDs and increment `IncInvalidAPIRequestIDs()`.

## Kafka Consumer Integration

### Consumer Configuration

Kafka consumer setup in `internal/kafka/kafka.go` uses two distinct `ConfigMap` branches:

- **With SASL**: when `SASLMechanism` is non-empty, includes `security.protocol`, `sasl.mechanism`, `ssl.ca.location`, and credentials
- **Without SASL**: uses `auto.offset.reset` and `auto.commit.interval.ms` instead

Both branches set `go.logs.channel.enable: true` and `allow.auto.create.topics: true`.

### Message Handler Pattern

The `handler` struct in `internal/kafka/handler.go` holds a `*gorm.DB` and a `PayloadFieldsRepository`. The `onMessage` method:

1. Unmarshals JSON into `message.PayloadStatusMessage`
2. Validates request ID length against config
3. Sanitizes fields to lowercase via `sanitizePayload`
4. Upserts the payload record
5. Looks up or creates service/status/source records
6. Inserts the payload status with a retry loop

### DB Insert Retry Logic

The consumer retries `InsertPayloadStatus` up to `DBRetries` times (default: 3, configured via `db.retries`). There is no backoff between attempts -- retries happen immediately in a tight loop:

```go
retries, attempts := cfg.DatabaseConfig.DBRetries, 0
for retries > attempts {
    err := queries.InsertPayloadStatus(h.db, sanitizedPayloadStatus).Error
    if err == nil {
        break
    }
    attempts += 1
}
```

When adding new retry logic in the consumer, follow this same pattern using `DBRetries` from config.

### Date Parsing in Kafka Messages

`FormatedTime` in `internal/models/message/payload-status.go` has a custom `UnmarshalJSON` that:

1. Replaces spaces with `T` in the timestamp string
2. Tries parsing with `time.RFC3339`
3. On failure, appends `"Z"` and retries

This handles both timezone-aware and timezone-naive timestamps from upstream services.

## PayloadFieldsRepository Pattern

### Interface and Implementations

`PayloadFieldsRepository` in `internal/queries/queries_consumer.go` defines three methods: `GetStatus`, `GetService`, `GetSource`. Two implementations exist:

- `PayloadFieldsRepositoryFromDB` -- queries the database directly
- `PayloadFieldsRepositoryFromCache` -- wraps another `PayloadFieldsRepository` with LRU caches (12-hour TTL, unbounded size)

Selection is controlled by `consumer.payload.fields.repo.impl`:
- `"db"` -- direct DB lookups
- `"db_with_cache"` -- cached DB lookups (default)

### Cache Decorator Pattern

`PayloadFieldsRepositoryFromCache` wraps any `PayloadFieldsRepository` (stored as `PayloadFields`). On cache miss, it delegates to the inner repository and caches non-empty results. When adding new cached lookups, follow this pattern: check cache first, delegate on miss, only cache non-zero-value results.

## Integration Testing Conventions

### Two Test Tiers

- **Unit tests** (e.g., `internal/endpoints/payloads_test.go`): use package-level `var` function references (`RetrievePayloads`, `RetrieveStatuses`, `RetrieveRequestIdPayloads`, `Db`) that are reassigned to mock functions in `BeforeEach`. No database required.
- **DB integration tests** (e.g., `internal/endpoints/endpoints_db_test/`): use `test.WithDatabase()` from `internal/utils/test/fixtures.go` to get a real Postgres connection. These tests create actual records in the database.

### Mocking External Services in Tests

For storage-broker tests, use `httptest.NewServer` to create a mock HTTP server and pass its URL to `RequestArchiveLink`:

```go
mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"url": "www.example.com"}`))
}))
handler = endpoints.PayloadArchiveLink(endpoints.RequestArchiveLink(mockServer.URL, 10))
```

### Mocking Database Dependencies

For unit tests, reassign the package-level function vars in `BeforeEach`:

```go
endpoints.RetrievePayloads = mockedRetrievePayloads
endpoints.Db = db  // for DB integration tests
```

For repository mocking, implement `PayloadFieldsRepository` interface with a struct tracking call state (see `mockPayloadFieldsRepository` in `internal/queries/queries_test.go`).

### Test Framework

All tests use Ginkgo/Gomega. Every test suite file calls `l.InitLogger()` before `RunSpecs`. Use `test.MakeTestRequest` from `internal/utils/test/helpers.go` to create test HTTP requests with query parameters. For URL params (e.g., `request_id`), inject via `chi.NewRouteContext()`:

```go
rctx := chi.NewRouteContext()
rctx.URLParams.Add("request_id", requestId)
req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
```

## Metrics for Integration Points

Increment the appropriate Prometheus counter from `internal/endpoints/metrics.go` at each integration boundary:

- `IncConsumedMessages()` -- when a Kafka message is received
- `IncConsumeErrors()` -- on Kafka consumer errors
- `IncMessagesProcessed()` -- after successful message processing
- `IncInvalidAPIRequestIDs()` -- on invalid UUID in API archive/kibana link endpoints
- `IncInvalidConsumerRequestIDs()` -- on invalid request ID length in consumer
- `ObserveMessageProcessTime()` -- duration from message receipt to DB insert

## Error Response Format

All API error responses use `structs.ErrorResponse` via `getErrorBody()` in `internal/endpoints/utils.go`, producing JSON with `title`, `message`, and `status` fields. Use `writeResponse(w, statusCode, getErrorBody(message, statusCode))` for consistency.
