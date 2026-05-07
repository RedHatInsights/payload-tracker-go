# Configuration Guidelines

Configuration in this repo is centralized in `internal/config/config.go` via a single `Get()` function that returns a `*TrackerConfig`. It uses Viper with `AutomaticEnv()` and Clowder integration for cloud-native deployment.

## Viper Configuration Pattern

- All configuration is defined in `config.Get()` using `viper.SetDefault()` followed by `options.AutomaticEnv()`. Never read environment variables directly with `os.Getenv()` for application config -- let Viper handle env-to-config mapping.
- The env key replacer is `strings.NewReplacer(".", "_")`, which means a Viper key like `kafka.bootstrap.servers` maps to the environment variable `KAFKA_BOOTSTRAP_SERVERS`. Viper handles case-insensitive matching.
- Config values are read once at startup via `config.Get()` in each binary's `main()`. The returned `*TrackerConfig` is then passed as a parameter to subsystems (DB, Kafka, logging). Do not call `config.Get()` in hot paths -- store the reference.
- Some endpoints (e.g., `PayloadKibanaLink`, `RolesArchiveLink`) call `config.Get()` directly at request time. For new code, prefer receiving config as a parameter via handler closures (as done with `CreatePayloadArchiveLinkHandler` and `HealthCheckHandler`).

## TrackerConfig Struct Organization

All config lives in `TrackerConfig` with nested sub-structs for each domain:

| Struct | Purpose |
|--------|---------|
| `KafkaCfg` | Bootstrap servers, auth, topic, consumer group |
| `DatabaseCfg` | Postgres connection params, retry count, RDS CA |
| `CloudwatchCfg` | AWS logging credentials and region |
| `RequestCfg` | Rate limiting, request ID validation, requestor implementation |
| `KibanaCfg` | Dashboard URL, index, service field |
| `DebugCfg` | Debug feature flags (e.g., `LogStatusJson`) |
| `ConsumerCfg` | Consumer-specific settings (payload fields repo impl) |

When adding a new config option: add a field to the appropriate sub-struct (or create a new one), set a default with `options.SetDefault()`, and map it in the struct literal at the bottom of `Get()`.

## Clowder Integration

The `Get()` function branches on `clowder.IsClowderEnabled()`:

- **Clowder enabled (production)**: Database, Kafka, ports, and CloudWatch config come from `clowder.LoadedConfig`. Kafka SASL auth and CA certs are extracted from the Clowder broker config after the main struct is built.
- **Clowder disabled (local dev)**: Hardcoded local defaults are used (e.g., `localhost:29092` for Kafka, `crc/crc/crc` for DB on `0.0.0.0:5432`, ports `8080`/`8081`).

When adding config that differs between local and production, add it inside the `if clowder.IsClowderEnabled()` / `else` block using `options.SetDefault()` in both branches.

## Environment Variable Naming

Viper key-to-env mapping uses dot-to-underscore replacement. Conventions observed in the codebase and `deployments/clowdapp.yml`:

- **Flat keys** use camelCase in Viper: `logLevel` -> `LOGLEVEL`, `publicPort` -> `PUBLICPORT`, `storageBrokerURL` -> `STORAGEBROKERURL`
- **Dotted keys** use lowercase dot-separated: `kafka.bootstrap.servers` -> `KAFKA_BOOTSTRAP_SERVERS`, `db.host` -> `DB_HOST`, `debug.log.status.json` -> `DEBUG_LOG_STATUS_JSON`
- Prefer dotted keys for new config options -- they produce more readable env var names. Flat camelCase keys result in all-uppercase env vars with no separators (e.g., `STORAGEBROKERREQUESTTIMEOUT`), which are harder to read.

## Dual Binary Architecture

The repo produces three binaries from `cmd/`:
- `payload-tracker-api` (pt-api): HTTP API server using `PublicPort` and `MetricsPort`
- `payload-tracker-consumer` (pt-consumer): Kafka consumer using only `MetricsPort`
- `internal/migration/main.go` (pt-migration): DB migration tool

All three call `config.Get()` and receive the full `TrackerConfig`. The API and consumer share config but use different subsets. The ClowdApp deploys them as separate pods with separate env var sets -- the API gets `STORAGEBROKERURL` and Kibana vars; the consumer gets `DEBUG_LOG_STATUS_JSON`.

## Feature Flags and Implementation Switching

Two config values act as implementation selectors:

1. **`RequestorImpl`** (env: `REQUESTOR_IMPL`, default: `"storage-broker"`): Controls the archive link handler. Values: `"storage-broker"` (production) or `"mock"` (dev/test). When `"mock"`, an additional test endpoint is mounted at `/archive/{id}`.

2. **`ConsumerPayloadFieldsRepoImpl`** (env: `CONSUMER_PAYLOAD_FIELDS_REPO_IMPL`, default: `"db_with_cache"`): Controls the consumer's payload fields repository. Values: `"db"` (direct DB lookups) or `"db_with_cache"` (LRU cache with 12-hour TTL wrapping DB).

3. **`Environment`** (env: `ENVIRONMENT`, default: `"PROD"`): When set to `"DEV"`, the API mounts routes under `/app/payload-tracker/api/v1/` instead of `/api/v1/`.

## Secrets and Sensitive Config

- Database credentials and Kafka SASL credentials come from Clowder in production -- they are never set via plain environment variables in deployed pods.
- CloudWatch AWS keys use `os.Getenv()` directly in the non-Clowder branch (`CW_AWS_ACCESS_KEY_ID`, `CW_AWS_SECRET_ACCESS_KEY`). This is the only place `os.Getenv()` is used for config outside of `os.Hostname()`.
- Kafka CA and RDS CA paths are written to disk by the `clowder.LoadedConfig.KafkaCa()` and `clowder.LoadedConfig.RdsCa()` helpers. Failures to write CA files result in a `panic`.
- SSL mode for Postgres is automatically set to `"require"` when `RDSCa` is non-empty (in `internal/db/db.go`).

## Config in Tests

- Tests call `config.Get()` directly, which returns local-dev defaults since Clowder is not enabled in test environments.
- Test database setup in `internal/utils/test/fixtures.go` reads DB config from `config.Get()` and always uses `sslmode=disable`.
- To override config in tests, modify the returned struct directly (e.g., `cfg.ConsumerConfig.ConsumerPayloadFieldsRepoImpl = "db"` as done in `internal/kafka/handler_test.go`). Do not set environment variables in tests.

## Adding New Configuration

1. Add a field to the appropriate sub-struct in `internal/config/config.go` (or create a new sub-struct for a new domain)
2. Add `options.SetDefault("dotted.key.name", defaultValue)` -- use dotted lowercase keys
3. If the value differs between local and production, add it inside both branches of the `clowder.IsClowderEnabled()` check
4. Map the value in the `TrackerConfig` struct literal using the matching `options.GetString/GetInt/GetBool` call
5. Add the environment variable to the appropriate deployment in `deployments/clowdapp.yml` (api deployment, consumer deployment, or both)
6. Pass the config value through the `*TrackerConfig` parameter -- avoid calling `config.Get()` from within handler or processing code
