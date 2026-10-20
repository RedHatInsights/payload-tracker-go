# Performance Guidelines

## LRU Caching for Payload Fields

- Use the `PayloadFieldsRepository` interface in `internal/queries/queries_consumer.go` for all consumer-side lookups of statuses, services, and sources. Prefer `db_with_cache` (the default) over `db` for the `CONSUMER_PAYLOAD_FIELDS_REPO_IMPL` env var.
- The `PayloadFieldsRepositoryFromCache` decorator wraps `PayloadFieldsRepositoryFromDB` using `hashicorp/golang-lru/v2/expirable`. Cache entries expire after 12 hours. Cache size is unbounded (size `0`).
- When adding a new lookup field to the cache layer, follow the existing pattern: check `cache.Get()` first, fall back to the DB method, then `cache.Add()` only when the DB returns a non-zero-value struct.
- Do not cache zero-value (empty) structs -- the existing code guards against this with `if dbEntry != (models.Statuses{})` checks before calling `Add`.
- Test cache behavior with the `mockPayloadFieldsRepository` pattern in `internal/queries/queries_test.go`: assert the mock is called on a cache miss, then assert it is not called on a subsequent cache hit.

## HTTP Rate Limiting

- The API server applies `httprate.LimitByIP` at the router level in `cmd/payload-tracker-api/main.go`. The limit is configured via `MAX_REQUESTS_PER_MINUTE` (default: 3000 req/min).
- Rate limiting is applied to the top-level `chi.NewRouter()`, so it covers all routes including sub-routers mounted under `/api/v1/`.
- When adding new routes, mount them on the existing `sub` router or the main `r` router to inherit the rate limit. Do not create a separate router that bypasses it.

## Database Connection and Query Patterns

- GORM connects via `internal/db/db.go` using the `gorm.io/driver/postgres` driver. The connection uses GORM's default pool settings (no explicit `SetMaxOpenConns`, `SetMaxIdleConns`, or `SetConnMaxLifetime` calls). Rely on GORM/pgx defaults unless production metrics indicate exhaustion.
- The `payload_statuses` table uses PostgreSQL range partitioning by `date`. Each partition gets btree indexes on `id`, `payload_id`, `service_id`, `source_id`, `status_id`, `date`, and `created_at` (created via `create_partition()` in `migrations/000001_create_payload_tracker_tables.up.sql`).
- Composite indexes exist on `payload_statuses` for common query patterns: `(service_id, created_at)`, `(service_id, date)`, `(service_id, status_id)`, `(status_id, created_at)`, `(status_id, date)`.
- The `payloads` table has single-column btree indexes on `account`, `created_at`, `inventory_id`, `system_id`, and a unique index on `request_id`.
- When adding new filter columns to API queries, add a corresponding database index in a new migration file; follow the naming pattern `payloads_<column>_idx` or `payload_statuses_<col1>_<col2>_idx`.

## Upsert Strategy

- Use `clause.OnConflict` with `DoUpdates: clause.AssignmentColumns(...)` for payload upserts (see `UpsertPayloadByRequestId` in `internal/queries/queries_consumer.go`). Only include columns in the update list when they have non-empty values to avoid overwriting existing data with blank strings.

## Database Insert Retries

- Kafka message handler retries `InsertPayloadStatus` up to `DB_RETRIES` times (default: 3). The retry loop in `internal/kafka/handler.go` breaks on success and logs an error when all retries are exhausted. Increment `IncMessageProcessErrors()` only after all retries fail.
- When adding new DB write operations in the consumer path, follow this retry pattern using `cfg.DatabaseConfig.DBRetries`.

## Prometheus Metrics for Performance Monitoring

- Register all new metrics using `promauto` (`pa`) in `internal/endpoints/metrics.go`. Use `CounterVec` for event counts and `HistogramVec` for latency observations.
- Existing performance metrics to preserve:
  - `payload_tracker_db_seconds` -- histogram of DB response latency, observed via `observeDBTime()`
  - `payload_tracker_message_process_seconds` -- histogram of message processing time, observed via `ObserveMessageProcessTime()`
  - `payload_tracker_requests` -- counter of API requests, incremented via `incRequests()`
  - `payload_tracker_responses` -- counter with `code` label, tracked via `ResponseMetricsMiddleware`
- When adding a new API endpoint, wrap it with `endpoints.ResponseMetricsMiddleware` and call `incRequests()` at handler entry. When adding new consumer processing, call `ObserveMessageProcessTime()` and `IncMessagesProcessed()`.
- The Grafana dashboard at `dashboards/grafana-dashboard-insights-payload-tracker-general.configmap.yaml` references these metric names. Renaming a metric requires updating the dashboard expressions.

## Response Time Tracking

- API endpoints (`Payloads`, `Statuses`) capture `time.Now()` at handler entry and compute `time.Since(start).Seconds()` to include elapsed time in the JSON response as the `elapsed` field. Maintain this pattern for new list endpoints.
- The consumer's `onMessage` handler tracks processing time from message receipt to DB insert, reporting it to `payload_tracker_message_process_seconds`.

## External Service Timeouts

- Storage broker requests use a configurable timeout via `STORAGEBROKERSREQUESTTIMEOUT` (default: 35000ms). The timeout is applied per-request using `http.Client{Timeout: ...}` in `RequestArchiveLink()` in `internal/endpoints/utils.go`.
- Kafka timeout defaults to 10000ms. Consumer poll interval is hardcoded at 100ms in the event loop in `internal/kafka/kafka.go`.

## Verification

```bash
# Confirm LRU cache tests pass (cache hit/miss behavior)
go test -v -run "Checks if we got a cached" ./internal/queries/...

# Run all tests including DB integration tests
make test

# Confirm rate limit config is applied at the router level
grep -n "httprate.LimitByIP" cmd/payload-tracker-api/main.go

# Confirm new endpoints use ResponseMetricsMiddleware
grep -n "ResponseMetricsMiddleware" cmd/payload-tracker-api/main.go

# Confirm all metrics are registered in metrics.go
grep -c "pa.New" internal/endpoints/metrics.go

# Verify no metric renames break the Grafana dashboard
grep -oE 'payload_tracker_[a-z_]+' internal/endpoints/metrics.go | sort -u > /tmp/code_metrics.txt
grep -oE 'payload_tracker_[a-z_]+' dashboards/grafana-dashboard-insights-payload-tracker-general.configmap.yaml | sort -u > /tmp/dash_metrics.txt
comm -23 /tmp/dash_metrics.txt /tmp/code_metrics.txt  # should be empty

# Lint for formatting
make lint
```
