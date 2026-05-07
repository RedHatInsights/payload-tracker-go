# Async and Messaging Guidelines

This repo consumes Kafka messages from the `platform.payload-status` topic using `confluent-kafka-go/v2`. The consumer is a separate binary (`cmd/payload-tracker-consumer/main.go`) that polls messages synchronously in a single-goroutine event loop.

## Architecture

- The system has two separate binaries built from one image: `pt-api` and `pt-consumer`. The Dockerfile sets no default CMD; the deployment determines which binary runs.
- The consumer binary (`cmd/payload-tracker-consumer/main.go`) starts a Kafka consumer event loop and a lightweight HTTP server for health checks (`/live`, `/ready`) and Prometheus metrics (`/metrics`) on the metrics port.
- All Kafka code lives in `internal/kafka/`. The `kafka.go` file handles consumer creation and the event loop; `handler.go` handles message processing.

## Consumer Event Loop

- The event loop in `NewConsumerEventLoop` (`internal/kafka/kafka.go`) uses `consumer.Poll(100)` with a 100ms timeout. Do not change the poll-based pattern to a channel-based one.
- The loop runs synchronously on a single goroutine -- messages are processed one at a time, in order. There is no parallel message processing.
- Graceful shutdown is handled via OS signal interception (`SIGINT`, `SIGTERM`) using a `sigchan` channel checked in the `select` statement alongside poll.
- After the loop exits, `consumer.Close()` is called to clean up the consumer. Always preserve this cleanup.

## Consumer Configuration

- Consumer config is built in `NewConsumer` (`internal/kafka/kafka.go`) with two paths: SASL-authenticated (when `SASLMechanism` is set) and plain (no auth). The SASL path omits `auto.offset.reset` and `auto.commit.interval.ms` from the config map.
- Default consumer group ID: `payload_tracker` (set in `internal/config/config.go`).
- Default auto offset reset: `latest`. Default auto commit interval: `5000ms`.
- Topic name comes from Clowder config key `platform.payload-status` or defaults to `"platform.payload-status"` locally.
- `allow.auto.create.topics` and `go.logs.channel.enable` are always set to `true`.
- All Kafka config defaults live in `internal/config/config.go` using Viper. Override via environment variables (dots replaced with underscores, e.g., `KAFKA_GROUP_ID`).

## Offset Management

- This repo uses Kafka's automatic offset commit (no manual `CommitMessage` calls). The `auto.commit.interval.ms` config (default 5000) controls commit frequency.
- The `OffsetsCommitted` event is handled in the event loop: errors are logged at Error level, successful commits are logged at Trace level. Do not add manual offset management.

## Event Loop Event Handling

The event loop handles exactly three event types from `consumer.Poll`:

| Event Type | Action |
|---|---|
| `*kafka.Message` | Increment `payload_tracker_consumed_messages` counter, call `handler.onMessage()` |
| `kafka.Error` | Increment `payload_tracker_consume_errors` counter, log the error |
| `kafka.OffsetsCommitted` | Log error if commit failed, otherwise trace-log and ignore |

All other event types are logged at Info level and ignored. Preserve this pattern when adding new event handling.

## Message Schema (`PayloadStatusMessage`)

The message schema is defined in `internal/models/message/payload-status.go`:

```go
type PayloadStatusMessage struct {
    Service     string       `json:"service"`
    Source      string       `json:"source,omitempty"`
    Account     string       `json:"account,omitempty"`
    OrgID       string       `json:"org_id,omitempty"`
    RequestID   string       `json:"request_id,omitempty"`
    InventoryID string       `json:"inventory_id,omitempty"`
    SystemID    string       `json:"system_id,omitempty"`
    Status      string       `json:"status"`
    StatusMSG   string       `json:"status_msg,omitempty"`
    PayloadID   uint
    Date        FormatedTime `json:"date"`
}
```

- `Service`, `Status`, and `Date` are the only required fields (no `omitempty`).
- `Date` uses a custom `FormatedTime` type that parses RFC3339, with fallback logic that appends `"Z"` for timezone-naive timestamps and replaces spaces with `"T"`.
- `PayloadID` has no JSON tag -- it is not deserialized from Kafka but set internally after DB upsert.

## Message Processing Pipeline (`handler.onMessage`)

The `onMessage` method in `internal/kafka/handler.go` follows this exact sequence. Do not reorder these steps:

1. **Unmarshal** JSON into `PayloadStatusMessage`. On failure, log error and return (skip message).
2. **Validate** `RequestID` length against `ValidateRequestIDLength` config (default: 32 chars). Invalid messages are silently dropped with a metric increment.
3. **Sanitize** -- lowercase `Service`, `Status`, and `Source` via `sanitizePayload()`.
4. **Upsert** into `Payloads` table by `request_id` (conflict resolution on `request_id` column). Only non-empty fields are included in the update clause.
5. **Resolve foreign keys** -- look up or create `Status`, `Service`, and `Source` entries. Uses `PayloadFieldsRepository` interface (DB or cached).
6. **Insert** into `PayloadStatuses` table with DB retry loop.

- On any DB error during steps 4-5, log the error and return (drop the message). There is no dead-letter queue or requeue mechanism.
- The `Source` field is optional; if empty, it is omitted from the `PayloadStatuses` insert.

## DB Retry Logic

- Only the final `InsertPayloadStatus` call has retry logic. Other DB operations (upsert, create status/service/source) fail immediately with no retry.
- Retries are controlled by `DatabaseConfig.DBRetries` (default: 3). The retry loop has no backoff or delay between attempts.
- The retry loop counts up from 0 to `DBRetries`. Setting `DBRetries` to a negative value skips insertion entirely.

## PayloadFieldsRepository and Caching

- The `PayloadFieldsRepository` interface (`internal/queries/queries_consumer.go`) abstracts lookups for Status, Service, and Source.
- Two implementations: `PayloadFieldsRepositoryFromDB` (direct DB queries) and `PayloadFieldsRepositoryFromCache` (LRU cache with 12-hour TTL, wrapping the DB implementation).
- Default implementation is `db_with_cache` (set via `CONSUMER_PAYLOAD_FIELDS_REPO_IMPL` env var). Use `"db"` for tests or when caching is undesirable.
- Cache uses `hashicorp/golang-lru/v2/expirable` with unbounded size (max size 0) and 12-hour expiration.
- Cache is populated on miss (read-through). New entries created via `CreateStatusTableEntry`/`CreateServiceTableEntry`/`CreateSourceTableEntry` are NOT added to the cache -- they will be cached on next lookup.

## Prometheus Metrics

All consumer metrics are defined in `internal/endpoints/metrics.go`. Increment metrics using these exported functions:

| Function | Metric | When |
|---|---|---|
| `IncConsumedMessages()` | `payload_tracker_consumed_messages` | Every polled `*kafka.Message` |
| `IncConsumeErrors()` | `payload_tracker_consume_errors` | Every `kafka.Error` event |
| `IncMessagesProcessed()` | `payload_tracker_messages_processed` | After successful message processing |
| `IncInvalidConsumerRequestIDs()` | `payload_tracker_consumer_invalid_request_IDs` | When request ID validation fails |
| `ObserveMessageProcessTime()` | `payload_tracker_message_process_seconds` | Duration from start of `onMessage` to DB insert |

- `IncConsumedMessages` is called in the event loop (before processing). `IncMessagesProcessed` and `ObserveMessageProcessTime` are called inside `onMessage` (after processing). These are distinct counts -- consumed != processed.

## Testing Kafka Handlers

- Handler tests use Ginkgo/Gomega and require a running PostgreSQL database (connected via `test.WithDatabase()`).
- Tests construct `*kafka.Message` objects directly using `newKafkaMessage()` -- they do not use a real Kafka broker.
- Set `cfg.ConsumerConfig.ConsumerPayloadFieldsRepoImpl = "db"` in tests (not `"db_with_cache"`).
- Validate outcomes by querying the database with `queries.RetrieveRequestIdPayloads()`, not by inspecting handler return values (the handler returns nothing).

## Local Development

- `compose.yml` provides Kafka (Confluent `cp-kafka:7.9.2` with Zookeeper), PostgreSQL, and both the API and consumer services.
- Local Kafka bootstrap server: `kafka:29092` (inside Docker network) or `localhost:29092` (from host).
- The `platform.payload-status` topic is auto-created (Kafka and consumer both have auto-create enabled).

## Clowder Integration

- In Clowder environments, topic names are resolved via `clowder.KafkaTopics["platform.payload-status"].Name`. The actual topic name may differ from `platform.payload-status` in managed environments.
- Kafka SASL credentials and CA certificates are loaded from Clowder config when `broker.Authtype` is set.
- The topic is configured with 3 replicas and 20 partitions in `deployments/clowdapp.yml`.
