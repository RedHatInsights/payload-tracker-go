# Configuration Guidelines

## Viper Key Naming and Environment Variable Mapping

- Define all viper keys in `internal/config/config.go` within the `Get()` function using `options.SetDefault()`.
- Use dot-separated lowercase keys for viper defaults (e.g., `kafka.group.id`, `db.host`). Viper's `SetEnvKeyReplacer` converts dots to underscores, so the env var for `kafka.group.id` is `KAFKA_GROUP_ID`.
- Camel-cased keys like `storageBrokerURL` map to env var `STORAGEBROKERURL` (all-uppercase, no separators). Prefer dot-separated keys for new config to get readable env var names.
- `options.AutomaticEnv()` is called after all defaults are set, making every viper key overridable via its corresponding env var without additional code.

## Clowder vs Local Defaults

- The `clowder.IsClowderEnabled()` branch sets defaults for infrastructure that Clowder provisions: Kafka brokers, topic names, ports, database credentials, and CloudWatch config. The `else` branch sets local development defaults.
- When adding a new config value that Clowder provides, add it inside the `if clowder.IsClowderEnabled()` block reading from `clowder.LoadedConfig`, with a local fallback in the `else` block.
- Config values that are the same regardless of Clowder (e.g., `kafka.timeout`, `db.retries`, `requestor.impl`) go before the Clowder branch as unconditional defaults.
- Kafka SASL credentials, Kafka CA, and RDS CA are extracted from Clowder after the `TrackerConfig` struct is populated (lines 230-259 of `config.go`). These fields have no local defaults -- they are empty strings when Clowder is not enabled.

## TrackerConfig Struct Rules

- Every new config value needs: (1) a field on the appropriate sub-struct in `TrackerConfig`, (2) a `SetDefault` call in `Get()`, and (3) an `options.Get*()` assignment when populating the struct.
- Group related config into the existing sub-structs: `KafkaCfg`, `DatabaseCfg`, `CloudwatchCfg`, `RequestCfg`, `KibanaCfg`, `DebugCfg`, `ConsumerCfg`.
- `config.Get()` returns a new `*TrackerConfig` on every call (it creates a fresh `viper.New()`). Callers in `cmd/` call it once at startup and pass the result down. Some endpoint handlers call `config.Get()` inline (e.g., `roles.go`, `payloads.go`); prefer passing the config from the caller when possible.

## Environment Mode (DEV vs PROD)

- The `Environment` field defaults to `"PROD"`. Set env var `ENVIRONMENT=DEV` to enable DEV mode.
- DEV mode changes the API mount path from `/api/v1/` to `/app/payload-tracker/api/v1/` (in `cmd/payload-tracker-api/main.go`). This path matches the frontend proxy prefix.
- DEV mode is activated by comparing `cfg.Environment == "DEV"` as a string -- not a boolean. Keep this exact comparison when adding new environment-conditional behavior.

## Database Configuration

- Local defaults: user=`crc`, password=`crc`, dbname=`crc`, host=`0.0.0.0`, port=`5432`. These match the `compose.yml` postgres service.
- SSL mode is derived at connection time in `internal/db/db.go`: `sslmode=disable` unless `RDSCa` is set (Clowder-provisioned), then `sslmode=require`.
- `db.retries` (default `3`) controls how many times the consumer retries a failed `InsertPayloadStatus` before giving up.

## Kafka Configuration

- When `SASLMechanism` is non-empty (Clowder with auth), the consumer uses a SASL-authenticated config map including `security.protocol`, `sasl.mechanism`, `ssl.ca.location`, `sasl.username`, `sasl.password`.
- When `SASLMechanism` is empty (local dev), the consumer uses a simpler config map with `auto.offset.reset` and `auto.commit.interval.ms`.
- The Kafka topic key `topic.payload.status` maps to env var `TOPIC_PAYLOAD_STATUS`. In Clowder, it reads from `clowder.KafkaTopics["platform.payload-status"].Name`.

## Logging Configuration

- `LogLevel` accepts `"DEBUG"`, `"ERROR"`, or defaults to `INFO` for any other value. Set via env var `LOGLEVEL`.
- During tests (`flag.Lookup("test.v") != nil`), the log level is forced to `FatalLevel` regardless of config.
- CloudWatch logging is enabled only when `CWAccessKey` is non-empty. Locally, this requires setting `CW_AWS_ACCESS_KEY_ID` and `CW_AWS_SECRET_ACCESS_KEY` env vars.

## Rate Limiting

- The API applies `httprate.LimitByIP` using `cfg.RequestConfig.MaxRequestsPerMinute` (default `3000`). Override via env var `MAX_REQUESTS_PER_MINUTE`.

## Requestor Implementation

- `requestor.impl` (env var `REQUESTOR_IMPL`) controls archive link behavior. Values: `"storage-broker"` (default, calls external storage-broker service) or `"mock"` (returns a local URL and registers a mock endpoint).
- When `REQUESTOR_IMPL=mock`, an additional route `GET /archive/{id}` is mounted. For local development, set `REQUESTOR_IMPL=mock`.

## Consumer Repository Implementation

- `consumer.payload.fields.repo.impl` (env var `CONSUMER_PAYLOAD_FIELDS_REPO_IMPL`) controls caching strategy. Values: `"db_with_cache"` (default, LRU cache with 12h TTL) or `"db"` (direct DB lookups).

## ClowdApp Deployment (deployments/clowdapp.yml)

- Environment variables in the ClowdApp template are the canonical list of externally-tunable config: `LOG_LEVEL`, `STORAGEBROKERURL`, `KIBANA_URL`, `KIBANA_INDEX`, `KIBANA_SERVICE_FIELD`, `SSL_CERT_DIR`, `DEBUG_LOG_STATUS_JSON`.
- DB credentials in the vacuum job come from the `payload-tracker-db-creds` secret, not from Clowder config.

## Secrets Handling

- DB passwords, Kafka SASL credentials, CloudWatch keys, and RDS/Kafka CA paths are populated from `clowder.LoadedConfig` in production. They are not hardcoded in `config.go` except for local dev defaults (`crc`/`crc`).
- Avoid adding secrets as viper defaults in the non-Clowder branch. The existing pattern uses `os.Getenv()` for CloudWatch keys in local mode, keeping them out of source.

## Verification

```bash
# Confirm config key has a SetDefault and struct assignment
grep -n 'SetDefault.*your_new_key' internal/config/config.go
grep -n 'options.Get.*your_new_key' internal/config/config.go

# Confirm env var override works (dots become underscores, case-insensitive)
# Example: key "kafka.group.id" -> env var KAFKA_GROUP_ID
grep 'SetEnvKeyReplacer' internal/config/config.go

# Verify no secrets are hardcoded (should only show "crc" local dev defaults)
grep -inE '(password|secret|token)' internal/config/config.go

# Run tests
go test -p 1 -v ./...

# Verify ClowdApp template env vars match config.go keys
grep -E 'name: [A-Z_]+' deployments/clowdapp.yml | sort
```
