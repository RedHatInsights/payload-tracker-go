# Code Organization Guidelines

Rules for how code is structured and organized in the payload-tracker-go repository. This is a dual-binary Go service: an HTTP API (`payload-tracker-api`) and a Kafka consumer (`payload-tracker-consumer`), both sharing internal packages.

## Project Layout

- All application code lives under `internal/` -- there is no `pkg/` directory. Nothing is exported for external consumption.
- The two entry points are `cmd/payload-tracker-api/main.go` and `cmd/payload-tracker-consumer/main.go`.
- The `internal/migration/main.go` file is a standalone CLI binary for database migrations, built separately via the Makefile.
- Tooling scripts (e.g., database seeders) live under `tools/`, not `cmd/`.
- SQL migration files live in `migrations/` at the project root (not under `internal/`).

## Package Responsibilities

Each `internal/` package has a single responsibility. Do not merge concerns across these boundaries:

| Package | Responsibility |
|---|---|
| `internal/config` | Viper-based configuration loading. Single `Get()` function returns `*TrackerConfig`. |
| `internal/db` | Database connection setup. Exposes a package-level `db.DB` variable (`*gorm.DB`). |
| `internal/endpoints` | HTTP handlers, middleware, request validation, and Prometheus metrics. |
| `internal/kafka` | Kafka consumer creation and message-handling event loop. |
| `internal/queries` | All GORM database queries -- both read (API) and write (consumer) operations. |
| `internal/models` | Top-level GORM model structs used by the API for reads. |
| `internal/models/db` | GORM model structs used by the consumer for writes (includes association fields like `Service`, `Source`, `Status`). |
| `internal/models/message` | Kafka message deserialization types (e.g., `PayloadStatusMessage`). |
| `internal/structs` | API request/response structs (query params, JSON response shapes). |
| `internal/logging` | Logrus logger initialization and CloudWatch formatter. |
| `internal/utils/test` | Shared test helpers and database fixture setup. |

## Dual Model Pattern

This repo has two distinct `models` packages with similar but not identical struct definitions:

- `internal/models/models.go` -- Used by API read queries. Structs include JSON tags for direct serialization in API responses.
- `internal/models/db/models.go` -- Used by the Kafka consumer for writes. Structs include association fields (`Service Services`, `Source Sources`, etc.) for GORM relationship handling.

When importing the consumer models, use a named import: `models "github.com/redhatinsights/payload-tracker-go/internal/models/db"`. The API models are imported directly as `models`.

## Global State Pattern

The repo uses package-level variables for shared state rather than dependency injection structs:

- `db.DB` -- Global `*gorm.DB` instance, set by `db.DbConnect()`.
- `logging.Log` -- Global `*logrus.Logger` instance, set by `logging.InitLogger()`.
- `config.Get()` -- Returns a new `*TrackerConfig` each time (no caching).

Both `main.go` entry points follow the same initialization sequence: `logging.InitLogger()` then `config.Get()` then `db.DbConnect(cfg)`.

## Import Aliases

The codebase uses consistent short aliases for frequently imported packages:

- `l` for `internal/logging` -- used across most packages: `l "github.com/redhatinsights/payload-tracker-go/internal/logging"`
- `models` for `internal/models/db` -- used in kafka and tools packages: `models "github.com/redhatinsights/payload-tracker-go/internal/models/db"`
- `p` for `prometheus/client_golang/prometheus` and `pa` for `prometheus/promauto` in `internal/endpoints/metrics.go`
- `k` for `confluent-kafka-go/v2/kafka` in test files

## Import Group Ordering

Imports are organized in three groups separated by blank lines:
1. Standard library
2. Third-party packages
3. Internal packages (`github.com/redhatinsights/payload-tracker-go/internal/...`)

## Endpoint Handler Conventions

- Handlers are plain functions with signature `func(w http.ResponseWriter, r *http.Request)`, registered directly on `chi` routes.
- Handlers that need configuration use the closure pattern -- a factory function returns `http.HandlerFunc`:
  ```go
  func HealthCheckHandler(db *gorm.DB, cfg config.TrackerConfig) http.HandlerFunc {
      return func(w http.ResponseWriter, r *http.Request) { ... }
  }
  ```
- All HTTP responses go through `writeResponse(w, statusCode, jsonString)` in `internal/endpoints/utils.go`.
- Error responses use `getErrorBody(message, statusCode)` which returns a JSON string from `structs.ErrorResponse`.
- Each handler file in `endpoints/` is named after the resource it handles: `payloads.go`, `statuses.go`, `roles.go`, `health.go`.

## Query Function Pattern

- API query functions in `internal/queries/queries_api.go` are declared as **package-level `var` functions**, not regular functions:
  ```go
  var RetrievePayloads = func(dbQuery *gorm.DB, page int, ...) (int64, []models.Payloads) { ... }
  ```
- This allows tests to replace them with mocks by reassigning the variable: `endpoints.RetrievePayloads = mockedRetrievePayloads`.
- Consumer (write) query functions in `internal/queries/queries_consumer.go` are regular exported functions.
- The `endpoints` package re-exports query function vars for mock swapping:
  ```go
  var RetrievePayloads = queries.RetrievePayloads
  ```

## Kafka Consumer Architecture

- The `kafka` package defines a private `handler` struct that holds `*gorm.DB` and a `PayloadFieldsRepository` interface.
- `PayloadFieldsRepository` is the only interface-based abstraction in the codebase, with two implementations: `PayloadFieldsRepositoryFromDB` (direct DB lookups) and `PayloadFieldsRepositoryFromCache` (decorator with LRU caching). Implementation is selected via `config.ConsumerConfig.ConsumerPayloadFieldsRepoImpl` (`"db"` or `"db_with_cache"`).
- Message processing happens in `handler.onMessage()` (unexported method). The event loop in `kafka.go` calls it for each `*kafka.Message`.

## Metrics

- All Prometheus metrics are defined in `internal/endpoints/metrics.go` using `promauto` -- they are shared across both the API and consumer binaries.
- Metric helper functions (e.g., `IncConsumedMessages()`, `ObserveMessageProcessTime()`) are called from the `kafka` package, creating a deliberate cross-package dependency from `kafka` to `endpoints`.
- The consumer binary serves metrics on a separate HTTP server at `cfg.MetricsPort`. The API binary uses separate routers for the public API and metrics.

## Test Conventions

- Tests use Ginkgo/Gomega (`onsi/ginkgo` v1 and `onsi/gomega`).
- Test suite bootstrap files are named `*_suite_test.go` and call `l.InitLogger()` before `RunSpecs()`.
- Unit tests (no DB) live alongside source files: `internal/endpoints/payloads_test.go`.
- Integration tests (require DB) live in a separate subdirectory: `internal/endpoints/endpoints_db_test/`.
- The `internal/utils/test` package provides `WithDatabase()` which returns a `func() *gorm.DB` closure for use in `BeforeEach` blocks.
- Endpoint unit tests mock DB queries by reassigning package-level function vars (e.g., `endpoints.RetrievePayloads = mockedRetrievePayloads`), and mock the DB accessor by reassigning `endpoints.Db`.
- Test requests are created via `test.MakeTestRequest(uri, queryParams)`.

## File Naming

- Source files use lowercase with underscores: `queries_api.go`, `queries_consumer.go`, `payload-status.go`.
- The `models/message/payload-status.go` file is the only file using hyphens in its name.
- Test files follow standard Go convention: `*_test.go`.
- Each endpoint resource gets its own file, plus `utils.go` for shared helpers and `metrics.go` for Prometheus instrumentation.

## Configuration

- Configuration uses `spf13/viper` with dot-separated keys (e.g., `kafka.bootstrap.servers`) mapped to env vars via `_` replacement (e.g., `KAFKA_BOOTSTRAP_SERVERS`).
- Clowder integration (`redhatinsights/app-common-go`) conditionally overrides defaults when `clowder.IsClowderEnabled()` is true.
- Config is a single flat-ish struct (`TrackerConfig`) with nested sub-config structs (`KafkaCfg`, `DatabaseCfg`, etc.) -- not a config interface.
