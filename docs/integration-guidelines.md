# Integration Guidelines

## Storage-Broker Integration

- Select the archive-link handler implementation via `RequestConfig.RequestorImpl` in `internal/config/config.go`. The two supported values are `"storage-broker"` (production) and `"mock"` (local development). See `configuration-guidelines.md` for config details.
- Build HTTP clients for storage-broker using the closure pattern in `RequestArchiveLink` (`internal/endpoints/utils.go`): construct an `http.Client` with `time.Duration(timeout) * time.Millisecond` and call `client.Get(baseUrl + "?request_id=" + reqID)`. Unmarshal the response body into `structs.PayloadArchiveLink`.
- Validate `request_id` as a UUID with `isValidUUID` before making outbound storage-broker calls. Return HTTP 400 for invalid UUIDs and increment `IncInvalidAPIRequestIDs()`.

## RBAC / Role Checking

- Gate archive-download endpoints behind role checks using `checkForRole` (see `security-guidelines.md` for implementation details).
- Both `/payloads/{request_id}/archiveLink` and `/roles/archiveLink` use the same configured role. Keep this parity when adding new role-gated endpoints.

## Kibana Link Generation

- Generate Kibana URLs in `PayloadKibanaLink` (`internal/endpoints/payloads.go`) using three config values from `KibanaConfig`: `DashboardURL`, `Index`, and `ServiceField`.
- Override these via env vars `KIBANA_URL`, `KIBANA_INDEX`, and `KIBANA_SERVICE_FIELD`. Defaults are set in `internal/config/config.go`.
- Build the Kibana query string as `request_id:<uuid>`, appending ` AND <serviceField>:<service>` when the optional `service` query parameter is present. Embed the query into the Kibana URL with `_g` and `_a` parameters matching the existing Lucene query format.

## Kafka Consumer Integration

- Create consumers via `kafka.NewConsumer` in `internal/kafka/kafka.go`. See `async-and-messaging-guidelines.md` for configuration details, event loop patterns, and retry logic.

## Mock Implementations for Local Development

- Set `REQUESTOR_IMPL=mock` to use `MockArchiveLink` instead of the real storage-broker client. This handler returns a self-referencing URL pointing at the local API: `http://<hostname>:<port>/app/payload-tracker/api/v1/archive/<reqID>`.
- When `REQUESTOR_IMPL=mock`, the API router registers an additional `GET /archive/{id}` route handled by `ArchiveHandler` in `internal/endpoints/utils.go`, which returns a plain-text stub response.

## Clowder Optional Dependencies

- The ClowdApp manifest (`deployments/clowdapp.yml`) declares `storage-broker`, `ingress`, and `rbac` as `optionalDependencies`. Treat these as optional — the service should start and handle requests even when these dependencies are unavailable.

## Test Patterns for Integration Code

- Mock storage-broker in tests using `httptest.NewServer` that returns a fixed JSON response, then pass its URL into `RequestArchiveLink`. See `PayloadArchiveLink` tests in `internal/endpoints/payloads_test.go`.
- Mock the `PayloadFieldsRepository` interface with a struct implementing `GetStatus`, `GetService`, `GetSource`. See `mockPayloadFieldsRepository` in `internal/queries/queries_test.go`.
- Use base64-encoded identity headers in tests: `validIdentityHeader` (contains `platform-archive-download` role) and `invalidIdentityHeader` (lacks it), defined in `internal/endpoints/roles_test.go`.

## Verification

```bash
# Confirm storage-broker config defaults are present
grep -n "storageBrokerURL" internal/config/config.go

# Confirm mock implementation switch exists
grep -n "RequestorImpl" internal/endpoints/payloads.go internal/config/config.go

# Confirm RBAC role check is applied to archive endpoints
grep -n "checkForRole" internal/endpoints/payloads.go internal/endpoints/roles.go

# Confirm Kibana config fields are used
grep -n "KibanaConfig" internal/endpoints/payloads.go internal/config/config.go

# Confirm timeout is applied in milliseconds
grep -n "time.Duration(timeout) \* time.Millisecond" internal/endpoints/utils.go

# Run tests (requires local Postgres)
make test
```
