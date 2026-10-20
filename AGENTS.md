# AGENTS.md

## Project Overview

Payload Tracker is a Go service that tracks payloads through the Red Hat Insights platform. It consists of two binaries built from a single codebase: a REST API server (`pt-api`) and a Kafka consumer (`pt-consumer`). Both share a PostgreSQL database with daily-partitioned status tables. The service runs on OpenShift via Clowder (ClowdApp) and uses Konflux/Tekton for CI/CD.

## Critical Architectural Constraints

- **Two-binary, one-image**: A single container image contains both `pt-api` and `pt-consumer`. The `build/Containerfile` has no `CMD` -- the ClowdApp manifest selects which binary to run. Never add a `CMD` or `ENTRYPOINT`.
- **Read-only API**: All REST endpoints use HTTP GET. There are no POST/PUT/PATCH/DELETE routes. Do not add write endpoints.
- **Dual model packages**: `internal/models/models.go` is used by the API read path; `internal/models/db/models.go` is used by the consumer write path. When changing schema-related fields, update both packages to stay in sync.
- **No circular imports**: `internal/config` and `internal/models` are leaf packages. `internal/kafka` calls metric functions from `internal/endpoints` intentionally -- this is the only allowed cross-cutting call.
- **Clowder-aware config**: `internal/config/config.go` has two branches -- one for Clowder-provisioned infrastructure, one for local dev defaults. New config values that Clowder provides must go in both branches.

## Build and Run

See [README.md](README.md) for complete build and local development setup. Key points for agents:
- Two binaries (`pt-api` and `pt-consumer`) built from `make build-all`
- Requires PostgreSQL with `make run-migration` and `make run-seed`
- Local API dev requires `REQUESTOR_IMPL=mock ./pt-api`

## Running Tests

```bash
make test                         # All tests: go test -p 1 -v ./...
go test -v ./internal/endpoints/  # Unit tests only (no database)
```

Tests use Ginkgo v1 and Gomega. The `-p 1` flag serializes test packages to prevent concurrent DB access conflicts.

## Cross-Cutting Conventions

### Naming and Style
- Use `snake_case` for JSON tags and database columns. Use Go standard `CamelCase` for exported identifiers.
- Alias `internal/logging` as `l` in imports: `l "...internal/logging"`. Alias `internal/models/db` as `models`.
- Group imports in three blocks separated by blank lines: (1) standard library, (2) third-party, (3) internal project packages.
- Format all code with `gofmt`. No additional linters are configured.

### Dependency Injection Pattern
- Endpoint handlers depend on query functions through reassignable package-level `var` declarations, not interfaces. Tests replace these in `BeforeEach` to inject mocks. Prefer this pattern for new endpoint-to-query dependencies.
- The Kafka handler uses a struct (`handler`) holding `*gorm.DB` and a `PayloadFieldsRepository` interface. New handler dependencies go on this struct -- do not use package-level globals in the handler.

### Metrics Discipline
- Every Prometheus metric name is prefixed with `payload_tracker_`. All metrics are defined in `internal/endpoints/metrics.go` using `promauto`.
- Every API route on the `sub` router must be wrapped with `endpoints.ResponseMetricsMiddleware`. Every consumer error path must increment the appropriate counter.
- Renaming a metric requires updating `dashboards/grafana-dashboard-insights-payload-tracker-general.configmap.yaml`.

### Error Handling Pattern
- API errors use `getErrorBody(message, statusCode)` + `writeResponse(w, statusCode, body)` from `internal/endpoints/utils.go`. Do not write error responses without this pattern.
- Use `l.Log.Fatal` only for startup failures that should kill the process. Use `panic` only for infrastructure failures during config loading and HTTP server startup.
- Log at Error then return early on consumer DB failures -- do not propagate panics in message processing.

### PR Expectations
- PRs follow the template in `.github/pull_request_template.md`: What/Why/How/Testing sections plus a Secure Coding Checklist.
- When adding a new API endpoint: add the handler, register the route with `ResponseMetricsMiddleware`, update `api/api.spec.yaml`, add response structs, and write Ginkgo tests.
- When adding a new Kafka message field: update `PayloadStatusMessage`, add to `models/db`, handle in `handler.go`, and apply `strings.ToLower()` if it is a categorical lookup key.

## Common Pitfalls

- **Forgetting the second model package**: Changes to GORM model fields in `internal/models/models.go` that are not mirrored in `internal/models/db/models.go` (or vice versa) cause silent data mismatches between the API and consumer.
- **Missing partition**: The `payload_statuses` table is range-partitioned by date. Inserts fail if no partition exists for the target date. The vacuum CronJob creates tomorrow's partition daily -- do not rely on application code for partition creation.
- **Sort field injection**: Sort fields from query parameters are interpolated into GORM `.Order()` clauses. Always validate against the allowlists in `internal/endpoints/utils.go` before querying. Adding a new sortable field without updating the allowlist silently passes invalid input to the database.
- **Caching empty results**: The LRU cache in `queries_consumer.go` guards against caching zero-value structs. New cache lookups must include the same `if dbEntry != (ModelType{})` check before calling `cache.Add`.

## Docs Index

Detailed guidelines for specific domains are in `docs/`. Read the relevant file before making changes in that area.

| File | Description |
|------|-------------|
| [api-contracts-guidelines.md](docs/api-contracts-guidelines.md) | REST API routing, response envelopes, pagination, sorting, authentication, and how to add new endpoints |
| [async-and-messaging-guidelines.md](docs/async-and-messaging-guidelines.md) | Kafka consumer architecture, event loop pattern, message handling, retry logic, and payload field caching |
| [code-organization-guidelines.md](docs/code-organization-guidelines.md) | Package boundaries under `internal/`, import conventions, dependency flow, and test organization |
| [configuration-guidelines.md](docs/configuration-guidelines.md) | Viper key naming, Clowder vs local defaults, environment variables, and secrets handling |
| [data-validation-guidelines.md](docs/data-validation-guidelines.md) | Request ID validation, message sanitization, timestamp parsing, sort parameter validation, and schema enforcement |
| [database-guidelines.md](docs/database-guidelines.md) | GORM models, migrations, partitioning, indexing, connection setup, query patterns, and data retention |
| [dependency-management-guidelines.md](docs/dependency-management-guidelines.md) | Go module version, automated updates (MintMaker), direct dependencies, pinned versions, and Tekton pipeline references |
| [deployment-guidelines.md](docs/deployment-guidelines.md) | Container image build, ClowdApp template, health probes, migration init containers, vacuum CronJob, and Tekton pipelines |
| [error-handling-guidelines.md](docs/error-handling-guidelines.md) | API error responses, validation ordering, DB retry logic, Kafka consumer error handling, and Prometheus error metrics |
| [go-web-frameworks-guidelines.md](docs/go-web-frameworks-guidelines.md) | Chi router architecture, route registration, rate limiting, middleware, handler patterns, and response writing |
| [integration-guidelines.md](docs/integration-guidelines.md) | Storage-broker integration, RBAC role checking, Kibana link generation, mock implementations, and Clowder config |
| [logging-and-observability-guidelines.md](docs/logging-and-observability-guidelines.md) | Logger setup, log format, log levels, structured fields, Prometheus metrics definitions, and Grafana dashboard |
| [performance-guidelines.md](docs/performance-guidelines.md) | LRU caching, HTTP rate limiting, database indexing, upsert strategy, insert retries, and response time tracking |
| [security-guidelines.md](docs/security-guidelines.md) | Identity header authentication, RBAC, Kafka SASL/SCRAM, database SSL/TLS, request validation, and container security |
| [testing-guidelines.md](docs/testing-guidelines.md) | Ginkgo/Gomega framework, suite bootstrap, two test tiers (unit/integration), mocking patterns, and CI configuration |
