# Async and Messaging Guidelines

## Kafka Consumer Architecture

- The consumer binary lives at `cmd/payload-tracker-consumer/main.go` and runs as a separate process from the API (`cmd/payload-tracker-api/main.go`). Keep these entrypoints independent -- the consumer binary starts its own HTTP server solely for `/metrics`, `/live`, and `/ready` endpoints.
- Prefer `confluent-kafka-go/v2` (`github.com/confluentinc/confluent-kafka-go/v2/kafka`) for all Kafka operations — it's the established client library and changing it would require infrastructure coordination.
- The consumer subscribes to a single topic configured via `KafkaConfig.KafkaTopic` (env: `TOPIC_PAYLOAD_STATUS`, default `platform.payload-status`). When adding new topics, follow the same `NewConsumer` + `NewConsumerEventLoop` pattern in `internal/kafka/kafka.go`.

## Consumer Configuration Conventions

- Kafka config diverges based on SASL: when `SASLMechanism` is set, configure `security.protocol`, `sasl.*`, and `ssl.ca.location`; otherwise configure `auto.offset.reset` and `auto.commit.interval.ms`. Preserve this conditional in `NewConsumer`.
- Default offset reset is `latest` (env: `KAFKA_AUTO_OFFSET_RESET`). Default auto-commit interval is 5000ms (env: `KAFKA_AUTO_COMMIT_INTERVAL_MS`). Coordinate with the deployment team before changing these defaults.
- Consumer group ID defaults to `payload_tracker` (env: `KAFKA_GROUP_ID`). `allow.auto.create.topics` is enabled; `go.logs.channel.enable` is enabled.
- All config is loaded through Viper with `_` as env key separator in `internal/config/config.go`. New Kafka settings go in `KafkaCfg` struct and the `Get()` function.

## Event Loop Pattern

- `NewConsumerEventLoop` in `internal/kafka/kafka.go` uses a synchronous `for/select` loop with `consumer.Poll(100)` (100ms timeout). Replacing this with a channel-based or goroutine-per-message approach would require rewriting the shutdown logic.
- Handle three event types in the type switch: `*kafka.Message` (process), `kafka.Error` (log + increment error metric), `kafka.OffsetsCommitted` (log errors at Error level, otherwise trace). Log and ignore all other event types.
- Graceful shutdown: the loop listens on a `sigchan` for `SIGINT`/`SIGTERM`, sets `run = false`, then calls `consumer.Close()` after exiting the loop.

## Message Handler (`internal/kafka/handler.go`)

- The `handler` struct holds `*gorm.DB` and a `PayloadFieldsRepository`. New handler dependencies go on this struct — avoid package-level globals for DB or repository access in the handler.
- Processing order in `onMessage` is: (1) unmarshal JSON, (2) validate request ID, (3) sanitize payload fields, (4) upsert into Payloads table, (5) resolve/create Status, Service, Source entries, (6) insert PayloadStatus with retry loop. Preserve this order.
- On unmarshal failure or invalid request ID, return early without inserting — avoid attempting partial processing.

## Message Format and Validation

- Inbound messages conform to `message.PayloadStatusMessage` in `internal/models/message/payload-status.go`. The `Date` field uses a custom `FormatedTime` type that handles RFC3339, whitespace-separated datetime (replaced with `T`), and timezone-unaware timestamps (appends `Z`).
- Request ID validation uses configurable length (default 32, env: `VALIDATE_REQUEST_ID_LENGTH`). Set to 0 to disable length validation. Invalid request IDs increment `payload_tracker_consumer_invalid_request_IDs` and return early.
- Sanitization lowercases `Service`, `Status`, and `Source` fields via `sanitizePayload`. Apply the same lowercasing to any new string fields that serve as lookup keys.

## DB Insert Retry Logic

- The final `InsertPayloadStatus` call uses a counted retry loop controlled by `DatabaseConfig.DBRetries` (default 3, env: `DB_RETRIES`). There is no backoff delay between retries.
- When all retries are exhausted, log at Error level and increment `payload_tracker_message_process_errors`. The message is not requeued -- it is dropped after retry exhaustion.
- Negative `DBRetries` values skip the insert entirely (the loop condition `retries > attempts` is never true). Test coverage for this case exists in `internal/kafka/handler_test.go`.

## Payload Field Caching

- `PayloadFieldsRepository` in `internal/queries/queries_consumer.go` has two implementations: `db` (direct DB lookups) and `db_with_cache` (LRU cache wrapping DB, default). Selection is via `ConsumerPayloadFieldsRepoImpl` (env: `CONSUMER_PAYLOAD_FIELDS_REPO_IMPL`).
- LRU caches use `hashicorp/golang-lru/v2/expirable` with 12-hour TTL and unlimited size (0). When adding new cached field types, follow the same pattern: check cache, fall back to DB, populate cache on DB hit.

## Prometheus Metrics

- Increment the correct counter at each processing stage. The counters are defined in `internal/endpoints/metrics.go`:
  - `IncConsumedMessages()` -- on every `*kafka.Message` event (before processing)
  - `IncInvalidConsumerRequestIDs()` -- on unmarshal failure or invalid request ID
  - `IncMessagesProcessed()` -- after successful processing (before DB insert retry)
  - `IncMessageProcessErrors()` -- on any DB operation failure
  - `IncConsumeErrors()` -- on `kafka.Error` events
  - `ObserveMessageProcessTime()` -- histogram of processing duration
- The consumer exposes metrics on a separate port (env: `METRICSPORT`, default `8081`) via `promhttp.Handler()` at `/metrics`.

## Testing Conventions

- Kafka handler tests use Ginkgo/Gomega with a real PostgreSQL database (`test.WithDatabase()` from `internal/utils/test/fixtures.go`). The test suite is in `internal/kafka/handler_test.go`.
- Build test messages with `newKafkaMessage()` which marshals a `PayloadStatusMessage` to JSON and wraps it in a `*kafka.Message` with topic and partition metadata.
- Test the `handler.onMessage` method directly — prefer mocking the Kafka consumer in unit tests rather than spinning up a real consumer.
- Set `ConsumerPayloadFieldsRepoImpl` to `"db"` in tests (`BeforeEach`) to bypass the LRU cache layer.

## Verification

```bash
# Build both binaries to catch compile errors in consumer or handler changes
make pt-consumer && make pt-api

# Run all tests (includes kafka handler tests against a local PostgreSQL)
make test

# Check formatting
gofmt -l .

# Verify new Prometheus metric functions are exported and called
grep -rn "func Inc\|func Observe" internal/endpoints/metrics.go
grep -rn "endpoints\.Inc\|endpoints\.Observe" internal/kafka/

# Confirm no package-level DB globals in the kafka package
grep -rn "var.*gorm\|var.*DB" internal/kafka/
```
