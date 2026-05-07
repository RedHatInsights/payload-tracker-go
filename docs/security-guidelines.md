# Security Guidelines

Rules for maintaining security patterns in the payload-tracker-go service. This covers the API server (`cmd/payload-tracker-api`), the Kafka consumer (`cmd/payload-tracker-consumer`), and shared internal packages.

## Authentication and Authorization

### Identity Header Verification

Role-protected endpoints use the `checkForRole()` function in `internal/endpoints/utils.go`. It reads the `x-rh-identity` header, base64-decodes it, and checks for a required role in the `identity.associate.Role` array.

- Use `checkForRole(r, roleName)` for any endpoint that requires RBAC. It returns an HTTP status code and error -- write the error response using `writeResponse()` and return early on failure.
- The required role for archive link access is configured via `StorageBrokerURLRole` in `internal/config/config.go` (default: `"platform-archive-download"`).
- Endpoints like `/payloads` and `/statuses` do not perform identity checks. Only `/payloads/{request_id}/archiveLink` and `/roles/archiveLink` enforce role verification.
- When adding a new role-protected endpoint, follow the pattern in `PayloadArchiveLink()` in `internal/endpoints/payloads.go`: call `checkForRole()` before any business logic, return the status code and error body on failure.

### Testing Role Checks

Tests in `internal/endpoints/roles_test.go` verify three cases for every role-protected endpoint: missing identity header (401), identity without required role (403), identity with required role (200). Follow this pattern when adding new role-protected endpoints. Test identity headers are base64-encoded JSON constants defined at the top of the test file.

## Input Validation

### Request ID Validation

Two distinct validation mechanisms exist:

1. **API side** (`internal/endpoints/utils.go`): `isValidUUID()` uses `github.com/google/uuid` to parse UUIDs. Used by `PayloadArchiveLink` and `PayloadKibanaLink` to reject non-UUID request IDs with 400.
2. **Consumer side** (`internal/kafka/handler.go`): `validateRequestID()` checks that the request ID has exactly the configured length (`ValidateRequestIDLength`, default 32). This is a length check, not a UUID parse.

When adding endpoints that accept a `request_id` path parameter, validate it with `isValidUUID()` before processing and call `IncInvalidAPIRequestIDs()` on failure to track the metric.

### Query Parameter Validation

The `initQuery()` function in `internal/endpoints/utils.go` populates a `structs.Query` from URL query parameters. Each endpoint then validates:

- `sort_by` against endpoint-specific allowlists (`validAllSortBy`, `validIDSortBy`, `validStatusesSortBy` in `internal/endpoints/utils.go`)
- `sort_dir` against `validSortDir` (`asc`, `desc`)
- Timestamp parameters via `validTimestamps()` which requires RFC3339 format

Prefer using `stringInSlice()` to validate against these allowlists rather than open-ended string matching.

### Kafka Message Sanitization

The `sanitizePayload()` function in `internal/kafka/handler.go` lowercases `Service`, `Status`, and `Source` fields from incoming Kafka messages before database insertion. Apply the same normalization to any new string fields consumed from Kafka.

## Database Security

### Parameterized Queries

All database queries in `internal/queries/queries_api.go` and `internal/queries/queries_consumer.go` use GORM's parameterized `Where("column = ?", value)` syntax. Do not use string interpolation or `fmt.Sprintf` to build WHERE clauses with user-supplied values. The `ORDER BY` clause is constructed with `fmt.Sprintf` but uses values from validated allowlists (`validSortBy`, `validSortDir`), which is safe only because those values are pre-validated.

### SSL/TLS for Database

In `internal/db/db.go`, the connection DSN sets `sslmode=disable` by default but switches to `sslmode=require` when `DatabaseConfig.RDSCa` is set. The RDS CA path is written by `clowder.LoadedConfig.RdsCa()` when running under Clowder. Do not hardcode `sslmode=disable` in production-targeted code.

### Upsert Pattern

`UpsertPayloadByRequestId()` in `internal/queries/queries_consumer.go` uses GORM's `clause.OnConflict` with an explicit `DoUpdates` column list. When adding upsert logic, specify columns to update explicitly rather than using blanket updates to prevent overwriting protected fields.

## Kafka Security

### SASL/SSL Configuration

The `NewConsumer()` function in `internal/kafka/kafka.go` conditionally configures SASL authentication. When `SASLMechanism` is set (populated from Clowder broker config), it applies `security.protocol`, `sasl.mechanism`, `ssl.ca.location`, `sasl.username`, and `sasl.password`. When adding new Kafka producers or consumers, follow this conditional pattern rather than assuming plaintext.

### Sensitive Kafka Config Logging

The consumer logs `"Connected to Kafka"` but does not log the `ConfigMap` contents. Maintain this practice -- do not log Kafka configuration maps since they contain credentials.

## Secret Management

### Clowder-Based Configuration

All secrets (database credentials, Kafka SASL credentials, CloudWatch keys, RDS CA, Kafka CA) are sourced from Clowder's `LoadedConfig` in `internal/config/config.go` when `clowder.IsClowderEnabled()` returns true. Non-Clowder defaults use environment variables or hardcoded dev values. Do not add new secret-loading paths outside of the Clowder integration block.

### Kubernetes Secrets

The vacuum job in `deployments/clowdapp.yml` reads database credentials from the `payload-tracker-db-creds` Kubernetes secret via `valueFrom.secretKeyRef`. New jobs or init containers that need database access should reference this same secret rather than duplicating credentials.

## Rate Limiting

The API server applies `httprate.LimitByIP()` middleware at the router level in `cmd/payload-tracker-api/main.go`. The limit is configured via `MaxRequestsPerMinute` (default: 3000). This applies to all routes on the main router. The metrics server on the separate port does not have rate limiting applied.

## Error Handling

### Error Response Format

Use `getErrorBody(message, statusCode)` from `internal/endpoints/utils.go` to build JSON error responses. This produces a consistent `structs.ErrorResponse` with `title`, `message`, and `status` fields. Do not return raw error strings or stack traces to clients -- the `PayloadArchiveLink` handler wraps internal errors with `fmt.Sprintf("%v", err)` which can leak implementation details; prefer generic messages for 500 errors (e.g., `"Internal Server Issue"`).

### Debug Logging Control

The `DebugConfig.LogStatusJson` flag (env: `DEBUG_LOG_STATUS_JSON`, default `false`) controls whether raw Kafka message content is included in error logs. This is checked in `internal/kafka/handler.go` before logging raw message values that could contain sensitive payload data. Keep this disabled in production.

## Container Security

The `Dockerfile` uses a multi-stage build with `ubi9/go-toolset` for building and `ubi9/ubi-minimal` for runtime. The final image runs as `USER 1001` (non-root). Do not change the final `USER` directive to root.

## Metrics Isolation

The API server runs two separate HTTP servers: the public API on `PublicPort` (default 8080) and metrics/Prometheus on `MetricsPort` (default 8081). The consumer exposes only the metrics port. Keep Prometheus metrics (`/metrics`) on the dedicated metrics port and do not mount `promhttp.Handler()` on the public router.
