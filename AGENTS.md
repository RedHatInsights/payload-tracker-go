# Payload Tracker Go

Payload Tracker is a dual-binary Go service that tracks payload statuses across the Red Hat Consoledot platform. It consumes Kafka messages from `platform.payload-status` and exposes them through a read-only REST API. For full project context, see [README.md](README.md).

## Docs Index

Detailed guidelines for each domain are maintained in `docs/`. Consult the relevant file before making changes in that area:

- [API Contracts](docs/api-contracts-guidelines.md) - REST API routes, response envelopes, query parameters, pagination, sorting, and OpenAPI spec maintenance
- [Async and Messaging](docs/async-and-messaging-guidelines.md) - Kafka consumer event loop, message schema, processing pipeline, offset management, and caching
- [Code Organization](docs/code-organization-guidelines.md) - Project layout, package responsibilities, dual model pattern, query function patterns, and import conventions
- [Configuration](docs/configuration-guidelines.md) - Viper-based config, Clowder integration, environment variables, and feature flags
- [Database](docs/database-guidelines.md) - PostgreSQL schema, GORM usage, table partitioning, migrations, upserts, and indexing
- [Data Validation](docs/data-validation-guidelines.md) - UUID validation, request ID length checks, timestamp parsing, input sanitization, and query parameter validation
- [Dependency Management](docs/dependency-management-guidelines.md) - Go modules, MintMaker/Renovate automation, Konflux pipeline references, and container base images
- [Deployment](docs/deployment-guidelines.md) - Container builds, ClowdApp manifest, Tekton pipelines, GitHub Actions, health probes, and local development
- [Error Handling](docs/error-handling-guidelines.md) - API error responses, Kafka consumer error handling, DB error patterns, and logging conventions
- [Integration](docs/integration-guidelines.md) - Storage-broker HTTP client, Kibana link generation, PayloadFieldsRepository interface, and external service mocking
- [Logging and Observability](docs/logging-and-observability-guidelines.md) - Logrus setup, log levels, CloudWatch integration, Prometheus metrics, and Grafana dashboard
- [Performance](docs/performance-guidelines.md) - Rate limiting, LRU caching, partitioning, connection management, and query optimization
- [Security](docs/security-guidelines.md) - Authentication, identity headers, parameterized queries, SASL/SSL, secret management, and container security
- [Testing](docs/testing-guidelines.md) - Ginkgo v1 framework, unit vs integration tests, DB setup, HTTP handler testing, and mock patterns

## Cross-Cutting Conventions

### Naming Conventions

**Go identifiers** follow standard Go conventions with these repo-specific patterns:
- Exported types use plural nouns matching DB table names: `Payloads`, `PayloadStatuses`, `Services`, `Sources`, `Statuses`. Treat these as proper names -- do not rename to singular form.
- Config sub-structs use the `Cfg` suffix: `KafkaCfg`, `DatabaseCfg`, `CloudwatchCfg`, `RequestCfg`, `KibanaCfg`, `DebugCfg`, `ConsumerCfg`.
- Prometheus metric helper functions follow the pattern `Inc<MetricName>()` for counters and `Observe<MetricName>()` for histograms. Keep exported helpers capitalized for cross-package use, lowercase for package-internal use.
- Prometheus metric names are prefixed with `payload_tracker_` and use snake_case.

**JSON fields** use `snake_case` in all API responses and Kafka messages. GORM struct tags use `snake_case` column names.

**Files** use lowercase with underscores (`queries_api.go`, `queries_consumer.go`). The one exception is `models/message/payload-status.go` which uses a hyphen. Prefer underscores for new files.

**Viper config keys** use dot-separated lowercase (`kafka.bootstrap.servers`, `db.host`). Flat camelCase keys like `storageBrokerURL` exist for legacy reasons but produce unreadable env vars (`STORAGEBROKERURL`). Use dotted keys for anything new.

### Code Style

**Formatting**: The project uses `gofmt` via `make lint` (which runs `gofmt -l .` and `gofmt -s -w .`). No additional linters (golangci-lint, staticcheck) are configured.

**Import ordering**: Three groups separated by blank lines: (1) standard library, (2) third-party packages, (3) internal packages (`github.com/redhatinsights/payload-tracker-go/internal/...`).

**Import aliases**: The logging package is aliased as `l` in most files. The consumer DB models package is aliased as `models`. Prometheus client is `p` and promauto is `pa` in `metrics.go`. Confluent Kafka is `k` in test files. Follow whatever alias is used in the file you are editing.

**Error string style**: The codebase is inconsistent -- some error messages start with uppercase and some with lowercase, some use `"ERROR: "` prefix and some do not. When modifying existing code, match the style of the surrounding code. For new code, prefer lowercase error messages without the `"ERROR"` prefix (the log level already conveys severity).

### Architectural Patterns

**Global state over dependency injection**: The repo uses package-level variables (`db.DB`, `logging.Log`) rather than dependency injection containers. The only interface-based abstraction is `PayloadFieldsRepository`. Do not introduce DI frameworks or widespread interface patterns -- they would conflict with the existing architecture.

**Replaceable function variables for testability**: API query functions are declared as `var` instead of `func` so tests can swap them with mocks. Endpoint files re-export these vars (e.g., `var RetrievePayloads = queries.RetrievePayloads`). When adding new query functions that API handlers call, follow this pattern -- declare as `var` in queries, re-export in the endpoint file.

**Handler closure pattern**: Handlers that need config or dependencies use a factory function returning `http.HandlerFunc` (e.g., `HealthCheckHandler(db, cfg)`, `CreatePayloadArchiveLinkHandler(cfg)`). Plain handlers that only need the global DB use bare function signatures. Use the closure pattern for new handlers that need injected dependencies.

**Single-goroutine consumer**: Kafka message processing is deliberately single-threaded. Do not add goroutine pools or parallel message processing without addressing the shared `handler` struct state and the sequential processing guarantees.

**Dual binary, shared internals**: Both binaries import from the same `internal/` packages. Changes to shared packages (config, db, endpoints/metrics, logging, models, queries) affect both binaries. Always consider both code paths when modifying shared code.

### Common Pitfalls

- **Two models packages with similar structs**: `internal/models/models.go` (API reads, includes JSON tags) and `internal/models/db/models.go` (consumer writes, includes association fields). When adding a schema field, update both files. Forgetting one causes silent data loss or missing API fields.

- **`sort_by` passes through to SQL ORDER BY**: The `sort_by` value from query parameters ends up in `fmt.Sprintf("%s %s", apiQuery.SortBy, apiQuery.SortDir)` which is interpolated directly into the query. This is only safe because the value is validated against an allowlist first. If you add a new `sort_by` option, add it to the correct `validXxxSortBy` slice -- not the generic `validSortBy` which is unused for validation.

- **Count queries are commented out**: Both `RetrievePayloads` and `RetrieveStatuses` have commented-out `dbQuery.Model(&payloads).Count(&count)` lines. The `count` field in API responses is always `0`. Do not uncomment these without understanding the performance implications on the partitioned `payload_statuses` table.

- **`config.Get()` creates a new config each call**: It re-reads Viper defaults and environment variables every time. Some endpoints call `config.Get()` in the request path (e.g., `PayloadKibanaLink`, `PayloadArchiveLink`). This works but is wasteful. For new code, accept config as a parameter via the handler closure pattern.

- **Integration tests share a database**: Tests run with `go test -p 1` (sequential) because they share a PostgreSQL instance. Test data is not cleaned between test suites. Use unique identifiers (e.g., `fmt.Sprintf("test-status-%d", time.Now().Unix())`) for lookup table entries to avoid unique constraint violations.

- **`lubdub` health endpoint is duplicated**: Both `main.go` files define their own `lubdub` function. This is a simple root-path handler returning `"lubdub"` as `text/plain`. It is not a shared function.

- **`PayloadArchiveLink` leaks internal errors**: The handler wraps internal errors with `fmt.Sprintf("%v", err)` and returns them as 500 responses. The Kibana link handler does the same. Be aware this can expose implementation details to clients.

- **Tekton pipeline version must be updated in all 4 files**: The `.tekton/` directory has four PipelineRun files that all reference the same pipeline version URL. When bumping the version, update all four or the builds will use inconsistent pipelines.

### Build and CI

**Local build**: Run `make build-all` to compile all three binaries (`pt-api`, `pt-consumer`, `pt-migration`). The `pt-seeder` is a separate target (`make pt-seeder`). All binaries are output to the repo root.

**Local tests**: Run `make test` (which runs `go test -p 1 -v ./...`). A PostgreSQL database must be running first -- start it with `docker compose up payload-tracker-db` and initialize with `make run-migration`.

**PR checks**: The GitHub Actions PR workflow (`pr.yml`) runs on `ubuntu-22.04` with a PostgreSQL service container, executes `make run-migration`, `make build-all`, then `go test ./...`. All three steps must pass. The Tekton pipelines run a container image build in Konflux.

**No pre-commit hooks or linter CI**: There are no pre-commit hooks configured. The `make lint` target runs `gofmt` but is not enforced in CI. The PR workflow only checks build and test.

### Commit Message Conventions

The repo uses a mix of commit message styles (no enforced standard):
- Jira-linked changes: `[RHCLOUD-XXXXX] Description (#PR)` (e.g., `[RHCLOUD-45708] Bump Go version + update quay image path (#469)`)
- Feature/fix changes: short imperative description with PR number (e.g., `Don't cache the query misses (#378)`)
- Automated dependency updates: `Update module <path> to <version>` or `update dependency <path> to <version>` (generated by MintMaker)
- No conventional commits prefix is required, though MintMaker occasionally uses `chore(deps):` or `fix(deps):`

### PR Expectations

- PRs must pass the GitHub Actions PR check (build + test) and the Tekton pipeline build
- Dependency update PRs from MintMaker (red-hat-konflux[bot]) should be reviewed for CI passing before merge
- When modifying API endpoints, update `api/api.spec.yaml` to match
- When adding Prometheus metrics, add a panel to the Grafana dashboard in `dashboards/`
- When adding migrations, use the next sequential number and include both `.up.sql` and `.down.sql` files
