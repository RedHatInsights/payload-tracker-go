# Code Organization Guidelines

## Two-Binary Architecture

This project produces two separate binaries from `cmd/`: the REST API server (`cmd/payload-tracker-api/`) and the Kafka consumer (`cmd/payload-tracker-consumer/`). Place each binary's entry point in its own `cmd/<binary-name>/main.go`. A third entry point for database migrations lives at `internal/migration/main.go` (built via `make pt-migration`).

- Prefer placing new business logic in `internal/` packages, not in `cmd/` `main.go` files. The `main.go` files wire dependencies (config, DB, router, consumer) and start servers.
- Place developer tooling (seeders, generators) under `tools/<tool-name>/main.go`.

## Package Boundaries Under `internal/`

Organize code into these single-responsibility packages:

- `internal/config` -- configuration loading via Viper with Clowder integration. Expose one `Get()` function returning `*TrackerConfig`.
- `internal/db` -- database connection setup. Exposes the `DB` package-level `*gorm.DB` variable and connection functions (`DbConnect`, `DbSqlConnect`).
- `internal/models` -- GORM model structs used by the API read path (joins across tables). The root `models` package defines the query-facing models with `json` and `gorm` tags.
- `internal/models/db` -- GORM model structs used by the consumer write path (direct table inserts/upserts). Import with alias: `models "...internal/models/db"`.
- `internal/models/message` -- Kafka message deserialization structs (e.g., `PayloadStatusMessage`). Used exclusively by `internal/kafka`.
- `internal/structs` -- API request/response structs (`Query`, `PayloadsData`, `ErrorResponse`). These are not GORM models; they represent HTTP contract types.
- `internal/queries` -- all database query logic split into two files: `queries_api.go` (read queries for endpoints) and `queries_consumer.go` (write queries for the consumer plus the `PayloadFieldsRepository` interface and its DB/cache implementations).
- `internal/endpoints` -- HTTP handler functions, middleware, Prometheus metrics, and request utilities. This package is imported by both `cmd/` binaries.
- `internal/kafka` -- Kafka consumer creation (`kafka.go`) and message handling (`handler.go`).
- `internal/logging` -- global logger initialization. Exposes `Log *logrus.Logger` at package level.
- `internal/utils/test` -- test fixtures and helpers (e.g., `WithDatabase`, `MakeTestRequest`).

## Import Alias Conventions

- Alias `internal/logging` as `l` when using it alongside other imports: `l "...internal/logging"`. Some files import it unaliased as `logging` -- either form is acceptable but `l` is the dominant pattern (used in 11 of 13 importing files).
- Alias `internal/models/db` as `models` to distinguish it from the root `internal/models` package: `models "...internal/models/db"`.
- Alias `prometheus/client_golang/prometheus` as `p` and `prometheus/promauto` as `pa` in `internal/endpoints/metrics.go`.
- Do not alias `internal/config` -- import it directly. The one instance of `config "..."` in `kafka.go` is an exception, not a convention to follow.

## Import Grouping

Group imports in this order separated by blank lines:
1. Standard library
2. Third-party packages
3. Internal project packages (`github.com/redhatinsights/payload-tracker-go/internal/...`)

## Dependency Flow

Respect this dependency direction (arrows mean "may import"):

```
cmd/* --> internal/config, internal/db, internal/endpoints, internal/kafka, internal/logging
internal/endpoints --> internal/config, internal/db, internal/logging, internal/queries, internal/structs
internal/kafka --> internal/config, internal/endpoints (metrics only), internal/logging, internal/models/db, internal/models/message, internal/queries
internal/queries --> internal/config, internal/models, internal/models/db, internal/structs
internal/structs --> internal/models
internal/db --> internal/config, internal/logging
internal/logging --> internal/config
```

- `internal/endpoints` calls `internal/queries` functions but does not import `internal/models/db` or `internal/models/message`.
- `internal/kafka` calls metric increment functions from `internal/endpoints` (e.g., `endpoints.IncConsumedMessages()`). This cross-package call is intentional -- metrics are centralized in `endpoints/metrics.go`.
- Avoid circular imports. `internal/config` and `internal/models` are leaf packages with no internal dependencies (except `config` uses `app-common-go`).

## Testability via Package-Level Function Variables

Endpoint handler functions depend on query functions through reassignable package-level `var` declarations, not interfaces:

```go
var RetrievePayloads = queries.RetrievePayloads
var Db               = getDb
```

Tests replace these variables in `BeforeEach` to inject mocks without changing production code. Prefer this pattern for new endpoint-to-query dependencies rather than introducing constructor injection.

## Test Organization

- Place unit tests adjacent to the code they test (`payloads_test.go` next to `payloads.go`).
- Use Ginkgo/Gomega. Each package with tests has a `*_suite_test.go` that calls `RegisterFailHandler(Fail)`, initializes the logger via `l.InitLogger()`, and invokes `RunSpecs`.
- Place integration tests requiring a real database in a separate sub-directory: `internal/endpoints/endpoints_db_test/`.
- Place reusable test utilities in `internal/utils/test/` (package name `test`).
- Endpoint tests use external test packages (`package endpoints_test`), while kafka handler tests use the internal package (`package kafka`).

## SQL Migrations

Place migration files in `migrations/` at the repo root using `golang-migrate` naming: `NNNNNN_description.up.sql` and `NNNNNN_description.down.sql`. The migration runner at `internal/migration/main.go` reads from `file://migrations`.

## Verification

```bash
# Confirm no circular imports
go build ./...

# Confirm package structure matches conventions
find internal -maxdepth 1 -type d | sort

# Verify import grouping and formatting
gofmt -l .

# Run all tests
go test -p 1 -v ./...

# Run linting
make lint
```
