# Security Guidelines

## Identity Header Authentication (x-rh-identity)

- Read the `x-rh-identity` header via `r.Header.Get("x-rh-identity")` -- this is the sole authentication mechanism for protected API endpoints.
- Decode the header value with `base64.StdEncoding.DecodeString`, then `json.Unmarshal` into the nested struct `Identity > Associate > Roles` (JSON field name `"Role"`, not `"Roles"`).
- Return `http.StatusUnauthorized` (401) when the header is missing or cannot be decoded/unmarshalled. Return `http.StatusForbidden` (403) when the required role is absent.
- Implement role checks through the `checkForRole(r *http.Request, role string) (int, error)` function in `internal/endpoints/utils.go`. Do not duplicate this logic; call `checkForRole` from every handler that requires authorization.

## Role-Based Access Control for Archive Downloads

- The configurable role name is stored in `config.Get().StorageBrokerURLRole` (env: `STORAGEBROKERURLROLE`, default: `"platform-archive-download"`).
- Two endpoints enforce role checks: `/payloads/{request_id}/archiveLink` (in `PayloadArchiveLink`) and `/roles/archiveLink` (in `RolesArchiveLink`). Both call `checkForRole` with `config.Get().StorageBrokerURLRole`.
- When adding new role-protected endpoints, follow this pattern from `internal/endpoints/roles.go`:

```go
statusCode, err := checkForRole(r, config.Get().StorageBrokerURLRole)
if err != nil {
    writeResponse(w, statusCode, getErrorBody(fmt.Sprintf("%v", err), statusCode))
    return
}
```

## Kafka SASL/SCRAM Authentication

- Kafka authentication is conditional: when `config.KafkaConfig.SASLMechanism` is non-empty, the consumer uses SASL with `security.protocol`, `sasl.mechanism`, `sasl.username`, `sasl.password`, and `ssl.ca.location` from config. When empty, it falls back to plaintext without authentication.
- SASL credentials are sourced from Clowder's broker config (`broker.Sasl.*`) only when `broker.Authtype != nil`. See `internal/config/config.go` lines 231-238.
- The Kafka CA certificate path is written by `clowder.LoadedConfig.KafkaCa(broker)` when `broker.Cacert != nil`, and stored in `KafkaConfig.KafkaCA`. Reference this field for `ssl.ca.location` -- do not hardcode CA paths.
- When modifying `internal/kafka/kafka.go`, preserve the two-branch `ConfigMap` structure: one branch for SASL-enabled (with auth fields) and one for plaintext (without auth fields). Do not merge them into a single map.

## Database SSL/TLS Configuration

- PostgreSQL connections use `sslmode=disable` by default and switch to `sslmode=require` when `config.DatabaseConfig.RDSCa` is non-empty. This logic lives in `internal/db/db.go` in both `DbConnect` and `DbSqlConnect`.
- The RDS CA path is provisioned by `clowder.LoadedConfig.RdsCa()` when `clowder.LoadedConfig.Database.RdsCa != nil`. See `internal/config/config.go` lines 251-258. The app panics if CA writing fails -- this is intentional to prevent unencrypted database connections in production.

## SSL Certificate Directory

- The `SSL_CERT_DIR` environment variable is set in `deployments/clowdapp.yml` to a colon-separated list of certificate directories: `/etc/ssl/certs:/etc/pki/tls/certs:/system/etc/security/cacerts:/cdapp/certs`. Preserve this multi-path format when modifying the deployment.

## Request Validation

- Request IDs from Kafka messages are validated by `validateRequestID` in `internal/kafka/handler.go`, which checks length against `config.RequestConfig.ValidateRequestIDLength` (default: 32). Messages with invalid request IDs are silently dropped after incrementing the `payload_tracker_consumer_invalid_request_IDs` metric.
- Archive link request IDs from the API are validated as UUIDs using `uuid.Parse` in `isValidUUID` (`internal/endpoints/utils.go`). Invalid UUIDs return 400 and increment the `payload_tracker_api_invalid_request_IDs` metric.
- Query parameter sort fields are validated against allowlists (`validAllSortBy`, `validIDSortBy`, `validStatusesSortBy`) before being interpolated into GORM order clauses. When adding new sortable fields, add them to the appropriate allowlist in `internal/endpoints/utils.go`.

## Rate Limiting

- The API applies IP-based rate limiting via `httprate.LimitByIP` in `cmd/payload-tracker-api/main.go`, configured by `config.RequestConfig.MaxRequestsPerMinute` (env: `MAX_REQUESTS_PER_MINUTE`, default: 3000 per minute).

## Logging Security Considerations

- The `PayloadArchiveLink` handler logs the raw `x-rh-identity` header value at Info level (line 174 of `internal/endpoints/payloads.go`). This is the base64-encoded identity -- avoid adding logic that logs the decoded identity content.
- The `checkForRole` function logs the raw identity header and parsed roles at Info level when a role check fails. Avoid logging decoded sensitive fields (account numbers, org IDs) from the identity header in new code.
- Kafka message bodies are logged at Debug level. The `debug.log.status.json` config flag (default: `false`) controls whether raw message JSON is included in error logs for unmarshal failures.

## Container Security

- The Containerfile (`build/Containerfile`) builds as `USER 0` (root) but switches to `USER 1001` (non-root) for the runtime stage. Preserve this non-root runtime user when modifying the Containerfile.

## Verification

```bash
# Run the existing test suite (includes identity header authentication scenarios)
make test

# Verify that role-check test coverage exists for both endpoints
grep -n "x-rh-identity\|checkForRole\|StatusUnauthorized\|StatusForbidden" internal/endpoints/roles_test.go internal/endpoints/payloads_test.go

# Verify the Kafka consumer conditionally applies SASL config
grep -A5 "SASLMechanism" internal/kafka/kafka.go

# Verify database SSL mode toggling based on RDS CA presence
grep -B2 -A1 "sslmode" internal/db/db.go

# Verify the container runs as non-root
grep "USER" build/Containerfile
```
