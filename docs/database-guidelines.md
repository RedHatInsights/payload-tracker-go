# Database Guidelines

Rules for working with PostgreSQL via GORM in the payload-tracker-go service. This service tracks payload statuses consumed from Kafka and exposes them through a REST API.

## Schema and Models

- **Two model packages exist**: `internal/models/db/models.go` (consumer-side, used for DB writes) and `internal/models/models.go` (API-side, used for reads). Keep them in sync when modifying schema fields.
- The `PayloadStatuses` table has a **composite primary key** of `(id, date)` required by PostgreSQL range partitioning. Both fields carry the `gorm:"primaryKey"` tag.
- Lookup tables (`Services`, `Sources`, `Statuses`) use `int32` IDs with unique `name` constraints. These are referenced as foreign keys from `payload_statuses`.
- `Payloads.RequestId` has a unique constraint and is the natural lookup key across the application.
- `source_id` on `PayloadStatuses` is **nullable** -- source is not always present in Kafka messages. When inserting without a source, use `db.Omit("source_id")` as done in `InsertPayloadStatus()` in `internal/queries/queries_consumer.go`.

## Table Partitioning

- `payload_statuses` is **range-partitioned by the `date` column**. Partitions are created and dropped via PL/pgSQL functions defined in the migration: `create_partition()` and `drop_partition()`.
- Partition naming convention: `partition_YYYYMMDD_YYYYMMDD` (derived from `get_date_string()` function).
- Each partition gets its own btree indexes on: `id` (unique), `payload_id`, `service_id`, `source_id`, `status_id`, `date`, and `created_at`.
- **New partitions are created one day ahead** and **old partitions are dropped** by the scheduled vacuum CronJob (see `deployments/clowdapp.yml`). The retention period defaults to 7 days (`RETENTION_DAYS`).
- When writing queries against `payload_statuses`, prefer filtering by `date` to enable partition pruning.

## Migrations

- Migrations use `golang-migrate/migrate/v4` with SQL files in the `migrations/` directory.
- Migration files follow the pattern: `NNNNNN_description.up.sql` / `NNNNNN_description.down.sql`.
- The migration tool runs as an **init container** (`pt-migration upgrade`) before the API pod starts, configured in `deployments/clowdapp.yml`.
- The down migration uses `m.Steps(-1)` to revert exactly one step at a time.
- Add new migration files with the next sequence number. Do not modify existing migration files.

## Upsert Pattern

- The `payloads` table uses GORM's `clause.OnConflict` for upserts, conflicting on `request_id`. See `UpsertPayloadByRequestId()` in `internal/queries/queries_consumer.go`.
- The upsert **only updates non-empty fields** to avoid overwriting existing data with blank values. Each field (`account`, `org_id`, `inventory_id`, `system_id`) is conditionally added to `columnsToUpdate`:
```go
if payload.Account != "" {
    columnsToUpdate = append(columnsToUpdate, "account")
}
```
- `request_id` is always in the update list to satisfy the ON CONFLICT clause.

## Query Patterns

- API queries use **GORM query chaining**: conditions are appended to `*gorm.DB` only when the corresponding filter parameter is non-empty. See `RetrievePayloads` and `RetrieveStatuses` in `internal/queries/queries_api.go`.
- Time-range filters (`created_at_lt`, `created_at_gte`, `date_gt`, etc.) are chained via `chainTimeConditions()` using parameterized `WHERE` clauses. Use `?` placeholders for all user-supplied values.
- Multi-table status queries use explicit SQL joins (not GORM preloading): `JOIN payloads`, `JOIN services`, `FULL OUTER JOIN sources`, `JOIN statuses`. Sources use `FULL OUTER JOIN` because `source_id` is nullable.
- Pagination uses `Limit(pageSize).Offset(pageSize * page)` with page starting at 0.
- Sort direction is passed directly as a string (`asc`/`desc`) validated against `validSortDir` in `internal/endpoints/utils.go`.

## Indexing Strategy

- `payloads` table indexes: unique on `id`, unique on `request_id`, plus btree on `account`, `created_at`, `inventory_id`, `system_id`.
- `payload_statuses` has **composite indexes** for common query patterns: `(service_id, created_at)`, `(service_id, date)`, `(service_id, status_id)`, `(status_id, created_at)`, `(status_id, date)`.
- Lookup tables (`services`, `sources`, `statuses`) have unique indexes on both `id` and `name`.
- Per-partition indexes are created automatically by `create_partition()`.

## Caching Layer

- Lookup table results (`Services`, `Sources`, `Statuses`) are cached using `hashicorp/golang-lru/v2/expirable` with a **12-hour TTL** and unbounded size (size parameter `0`).
- The caching layer is configured via `CONSUMER_PAYLOAD_FIELDS_REPO_IMPL` env var, defaulting to `db_with_cache`. The `db` option bypasses the cache.
- Cache implementation wraps the DB repository via the `PayloadFieldsRepository` interface in `internal/queries/queries_consumer.go`.

## Scheduled Vacuum Job

- A CronJob defined in `deployments/clowdapp.yml` runs the `clean.sh` script on schedule (default: `00 17 * * *`, suspended by default).
- The job performs four operations in order:
  1. Creates tomorrow's partition via `create_partition()`
  2. Drops the partition older than `RETENTION_DAYS` (default 7) via `drop_partition()`
  3. Deletes orphaned rows from `payloads` older than `RETENTION_DAYS`
  4. Runs `VACUUM ANALYZE payloads`
- Partition creation retries up to `MAX_NUMBER_OF_RETRIES` (default 3) with `SLEEP_TIME` (default 10s) between attempts.

## Database Connection

- The global GORM connection is stored in `db.DB` (package `internal/db`). It is initialized once at startup via `DbConnect()` and accessed throughout the application.
- SSL mode is `disable` by default; it switches to `require` when `RDSCa` is configured (production via Clowder).
- A separate `database/sql` connection (`DbSqlConnect()`) is used only by the migration tool.
- Insert retries are configured via `DB_RETRIES` (default 3) in the Kafka consumer handler (`internal/kafka/handler.go`).

## Testing

- Database tests use Ginkgo/Gomega and connect to a real PostgreSQL instance (not mocked). Test setup is in `internal/utils/test/fixtures.go`.
- Each test gets a fresh GORM connection via `test.WithDatabase()`, which opens/closes connections in `BeforeEach`/`AfterEach`.
- Query functions (`RetrievePayloads`, `RetrieveStatuses`, `RetrieveRequestIdPayloads`) are declared as `var` (not `func`) in `internal/queries/queries_api.go` so they can be replaced with test doubles in endpoint tests.

## Foreign Keys

- `payload_statuses.payload_id` references `payloads.id` with `ON DELETE CASCADE` -- dropping a payload automatically removes its statuses.
- `payload_statuses.service_id`, `source_id`, and `status_id` reference their respective lookup tables without cascade -- lookup entries must not be deleted while referenced.
