# Performance Guidelines

Rules for maintaining and extending performance-sensitive patterns in the payload-tracker-go service. This service has two binaries -- an API server (`cmd/payload-tracker-api`) and a Kafka consumer (`cmd/payload-tracker-consumer`) -- both backed by a partitioned PostgreSQL database.

## HTTP Rate Limiting

- The API server uses `httprate.LimitByIP` applied at the router level in `cmd/payload-tracker-api/main.go`. The limit is configured via `max.requests.per.minute` (default: 3000). Do not add additional rate limiting middleware on subroutes -- all `/api/v1/` routes inherit the router-level limiter.
- When changing the rate limit default in `internal/config/config.go`, coordinate with the `RequestCfg.MaxRequestsPerMinute` field. The env var override is `MAX_REQUESTS_PER_MINUTE`.

## LRU Caching for Payload Fields

- The consumer uses an LRU cache layer (`PayloadFieldsRepositoryFromCache` in `internal/queries/queries_consumer.go`) wrapping the database repository. This caches `Statuses`, `Services`, and `Sources` lookups by name.
- The cache implementation uses `github.com/hashicorp/golang-lru/v2/expirable` (imported as `expirable_lru`) with size `0` (unbounded) and a 12-hour TTL. The three caches (status, service, source) are independent instances.
- The cache follows a read-through pattern: on miss, it queries the DB via the wrapped `PayloadFieldsRepositoryFromDB`, then stores the result. Only non-empty structs are cached -- zero-value results (new/unknown names) are intentionally not cached so the next message triggers a DB create.
- The implementation is selected by `consumer.payload.fields.repo.impl` config (default: `db_with_cache`). Setting it to `db` disables caching entirely. Do not add other values without updating `NewPayloadFieldsRepository`.
- When adding new cacheable lookup tables (similar to services/sources/statuses), follow the same pattern: add a typed `expirable_lru.LRU` field to `PayloadFieldsRepositoryFromCache`, implement the `Get*` method with cache-miss fallthrough, and initialize the cache in `newPayloadFieldsRepositoryFromCache` with matching TTL.

## Database Schema and Partitioning

- The `payload_statuses` table is **range-partitioned by the `date` column**. Each partition is created via the `create_partition()` PL/pgSQL function in `migrations/000001_create_payload_tracker_tables.up.sql`. Partitions are named `partition_YYYYMMDD_YYYYMMDD`.
- Each partition automatically gets per-partition indexes on: `id` (unique), `payload_id`, `service_id`, `source_id`, `status_id`, `date`, and `created_at`. These are created inside `create_partition()`.
- The composite primary key on `payload_statuses` is `(id, date)` -- the partition key must be part of the PK. Do not change this without understanding PostgreSQL partitioning constraints.
- The `payloads` table has a unique constraint on `request_id`, which enables the upsert pattern via `clause.OnConflict{Columns: []clause.Column{{Name: "request_id"}}}` in `UpsertPayloadByRequestId`.

## Database Indexes and Query Patterns

- The migration creates composite indexes on `payload_statuses` for common query patterns: `(service_id, created_at)`, `(service_id, date)`, `(service_id, status_id)`, `(status_id, created_at)`, and `(status_id, date)`. When adding new filter combinations to `RetrieveStatuses` or `RetrievePayloads`, check whether a supporting composite index exists.
- The `payloads` table has individual btree indexes on `account`, `created_at`, `inventory_id`, `system_id`, and a unique index on `request_id`. The `RetrievePayloads` function in `internal/queries/queries_api.go` applies WHERE clauses for these columns conditionally -- these indexes support that pattern.
- All query filtering in `queries_api.go` uses GORM's parameterized `Where("column = ?", value)` to prevent SQL injection. Maintain this pattern; do not use string interpolation for user-provided filter values.
- Time-range filtering uses `chainTimeConditions` which generates `<`, `<=`, `>`, `>=` comparisons. The `lt/lte/gt/gte` suffix convention must be preserved for any new timestamp filters.

## Database Connection Management

- The API server uses a single global `*gorm.DB` instance (`db.DB` in `internal/db/db.go`) initialized at startup via `DbConnect`. GORM uses pgx/v5 under the hood, which provides built-in connection pooling via `jackc/puddle/v2`. The current code does not configure pool size -- it uses pgx defaults.
- The `internal/db/db.go` module does not configure `SetMaxOpenConns`, `SetMaxIdleConns`, or `SetConnMaxLifetime` on the underlying `*sql.DB`. If adding pool tuning, extract the `*sql.DB` via `db.DB.DB()` and set these after `gorm.Open`.
- The test helper (`internal/utils/test/fixtures.go`) creates a new connection per test and closes it in `AfterEach`. This is intentional for test isolation but does not reflect production connection management.

## Consumer Processing and Retries

- Kafka message processing is single-threaded in `NewConsumerEventLoop` (`internal/kafka/kafka.go`): one goroutine polls and processes messages sequentially via `consumer.Poll(100)`. Do not add concurrent message processing without addressing shared state in the `handler` struct.
- The `handler.onMessage` method performs multiple sequential DB operations per message: upsert payload, lookup/create status, service, source, then insert payload_status. The DB insert has a retry loop controlled by `cfg.DatabaseConfig.DBRetries` (default: 3). Retries loop immediately with no backoff.
- Each message processes 3-4 cache lookups (status + service + optionally source) before the final insert. The cache significantly reduces DB round-trips since the set of distinct service/status/source names is small relative to message volume.

## Prometheus Metrics

- All Prometheus metrics are registered via `promauto` in `internal/endpoints/metrics.go` at package init time. Use `promauto.NewCounterVec` or `promauto.NewHistogramVec` for new metrics -- do not use `prometheus.MustRegister` directly.
- The `payload_tracker_db_seconds` histogram tracks DB query latency and is observed in the `Payloads` endpoint handler. The `payload_tracker_message_process_seconds` histogram tracks consumer message processing time. When adding new endpoints or processing paths, observe these metrics at the appropriate scope.
- Response code tracking uses `metricTrackingResponseWriter` which wraps `http.ResponseWriter` and increments `payload_tracker_responses` by status code label. This is applied via `ResponseMetricsMiddleware` on all `/api/v1/` subroutes. The metrics server runs on a separate port (`cfg.MetricsPort`, default 8081).

## Query Pagination

- The `initQuery` function in `internal/endpoints/utils.go` defaults to `Page: 0, PageSize: 10`. Pagination uses `LIMIT`/`OFFSET` via GORM's `.Limit(pageSize).Offset(pageSize * page)`. There is no maximum page size enforced -- consider the impact of large `page_size` values on DB performance.
- The `/payloads/{request_id}` endpoint (`RetrieveRequestIdPayloads`) does **not** use pagination -- it returns all statuses for a given request_id. This is acceptable because a single payload typically has a small number of status entries.

## External Service Calls

- The archive link endpoint calls the storage-broker with a per-request `http.Client` created in the closure returned by `RequestArchiveLink` (`internal/endpoints/utils.go`). The timeout is configured via `storageBrokerRequestTimeout` (default: 35000ms). A new `http.Client` is created per call -- this means transport/connection reuse depends on Go's default transport pooling.

## Verbosity-Based Field Selection

- The `defineVerbosity` function in `internal/queries/queries_api.go` reduces the SQL SELECT field list based on verbosity level (`0`, `1`, `2`). Higher verbosity means fewer fields returned. This reduces data transfer from the database. Maintain this pattern when adding new fields -- add them to the appropriate verbosity level rather than always selecting all columns.
