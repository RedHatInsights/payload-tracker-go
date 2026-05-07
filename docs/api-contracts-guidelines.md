# API Contracts Guidelines

Rules for maintaining and extending the payload-tracker-go REST API. The API is defined in `api/api.spec.yaml` (Swagger 2.0) and implemented via `go-chi` handlers in `internal/endpoints/`.

## Route Structure and Versioning

- All API routes are mounted under `/api/v1/` (production) or `/app/payload-tracker/api/v1/` (DEV environment). This prefix is set in `cmd/payload-tracker-api/main.go` via `r.Mount()`.
- New endpoints must be registered on the `sub` router in `cmd/payload-tracker-api/main.go` with the `endpoints.ResponseMetricsMiddleware` middleware applied: `sub.With(endpoints.ResponseMetricsMiddleware).Get("/path", handler)`.
- This API is read-only. All endpoints use GET. There are no POST/PUT/DELETE routes.
- Path parameters use `{param_name}` chi syntax. Extract them with `chi.URLParam(r, "param_name")`.

## Response Envelope Patterns

Three distinct response shapes exist. Use the correct one for each endpoint type:

**Paginated list responses** (`/payloads`, `/statuses`) -- wrap in `structs.PayloadsData` or `structs.StatusesData`:
```json
{"count": 0, "elapsed": 0.0, "data": []}
```
The structs are defined in `internal/structs/api_structs.go`. Both include `count` (int64), `elapsed` (float64, seconds), and `data` (array).

**Single-resource responses** (`/payloads/{request_id}`) -- wrap in `structs.PayloadRetrievebyID`:
```json
{"data": [], "duration": {}}
```
The `duration` field is a `map[string]string` of service timing durations computed by `queries.CalculateDurations()`.

**URL responses** (`/archiveLink`, `/kibanaLink`) -- return a flat object with a `url` field using `structs.PayloadArchiveLink` or `structs.PayloadKibanaLink`.

## Error Response Format

Use the `getErrorBody()` helper in `internal/endpoints/utils.go` for all error responses. It produces `structs.ErrorResponse` with three fields:
```json
{"title": "Bad Request", "message": "sort_by must be one of ...", "status": 400}
```
The `title` is derived from `http.StatusText(status)`. Write errors via `writeResponse(w, statusCode, getErrorBody(msg, statusCode))`.

## Query Parameter Conventions

### Pagination
- Parameters: `page` (default 0, zero-indexed) and `page_size` (default 10).
- Parsed as integers in `initQuery()` in `internal/endpoints/utils.go`. Invalid integers cause a 400 error.
- Offset is computed as `pageSize * page` in `internal/queries/queries_api.go`.

### Sorting
- Parameters: `sort_by` and `sort_dir` (values: `asc`, `desc`).
- Each endpoint has its own allowed `sort_by` values defined as slices in `internal/endpoints/utils.go`:
  - `/payloads`: `validAllSortBy` = `[account, org_id, inventory_id, system_id, created_at]`, default `created_at`
  - `/payloads/{request_id}`: `validIDSortBy` = `[service, source, status_msg, date, created_at]`, default `date`
  - `/statuses`: `validStatusesSortBy` = `[service, source, request_id, status, status_msg, date, created_at]`, default `date`
- When adding a new endpoint, define a new `validXxxSortBy` slice and validate against it. Return 400 with the allowed values listed in the message.

### Timestamp Filters
- Use the suffix convention: `_lt`, `_lte`, `_gt`, `_gte` appended to the field name (e.g., `created_at_lt`, `date_gte`).
- All timestamps must parse as `time.RFC3339`. Validation is done by `validTimestamps()` in `internal/endpoints/utils.go`.
- `/payloads` only supports `created_at_*` filters. `/statuses` supports both `created_at_*` and `date_*` filters.
- The `validTimestamps()` function takes an `all` boolean: `false` for payloads-only (created_at), `true` for statuses (created_at + date).

### String Filters
- String filters are exact-match equality checks applied via GORM `Where("column = ?", value)` in `internal/queries/queries_api.go`.
- Available filters depend on the endpoint: `/payloads` supports `account`, `org_id`, `inventory_id`, `system_id`. `/statuses` supports `service`, `source`, `status`, `status_msg`.

## Verbosity Control

The `/payloads/{request_id}` endpoint accepts a `verbosity` query parameter (0, 1, 2) that controls which database columns are returned. This is handled by `defineVerbosity()` in `internal/queries/queries_api.go`:
- `0` (default): All fields including IDs, account info, service, source, status, timestamps.
- `1`: Subset -- service, status, inventory_id, date, status_msg.
- `2`: Minimal -- service, status, date only.

## Authentication and Authorization

- Endpoints requiring authorization check the `x-rh-identity` header, which is base64-encoded JSON. Parsing is done in `checkForRole()` in `internal/endpoints/utils.go`.
- Role-gated endpoints (e.g., `/archiveLink`) return 401 for missing identity headers, 403 for insufficient roles.
- The required role is configured via `StorageBrokerURLRole` in config (default: `platform-archive-download`).

## UUID Validation

- Path parameters that are UUIDs (like `request_id`) must be validated using `isValidUUID()` in `internal/endpoints/utils.go`, which uses `github.com/google/uuid`.
- Invalid UUIDs return 400 and increment the `payload_tracker_api_invalid_request_IDs` Prometheus counter via `IncInvalidAPIRequestIDs()`.

## OpenAPI Spec Maintenance

- The spec lives at `api/api.spec.yaml` using Swagger 2.0 format with `basePath: /v1`.
- When adding or modifying endpoints, update the spec to match. Keep the `definitions` section for shared response schemas and `responses` section for reusable error responses (`BadRequest`, `NotFound`, `Unauthorized`, `Forbidden`, `InternalServerError`).
- Enum values in the spec (e.g., `sort_by` allowed values) must match the `validXxxSortBy` slices in `internal/endpoints/utils.go`.

## Content Type

- All API responses set `Content-Type: application/json` via `writeResponse()` in `internal/endpoints/utils.go`. The health endpoint is the exception, returning `text/plain`.

## Response Struct Conventions

- Response structs live in `internal/structs/api_structs.go`. Use `json:"field_name"` tags with `snake_case` naming.
- Use `omitempty` on fields that may be absent (see `SinglePayloadData`).
- Database model structs live separately in `internal/models/models.go` (for JSON-tagged API models) and `internal/models/db/models.go` (for GORM-tagged DB models). Do not mix API response concerns into DB models.

## Testing API Endpoints

- Tests use Ginkgo/Gomega in `internal/endpoints/*_test.go`.
- DB queries are mocked by reassigning package-level function variables (e.g., `endpoints.RetrievePayloads = mockedRetrievePayloads`).
- Use `test.MakeTestRequest()` from `internal/utils/test/helpers.go` to construct test requests with query parameters.
- Test all validation paths: valid requests (200), invalid sort_by (400), invalid sort_dir (400), invalid timestamps (400), missing/invalid identity headers (401/403), and not-found cases (404).

## Rate Limiting

- API requests are rate-limited by IP using `httprate.LimitByIP()` middleware, configured via `RequestConfig.MaxRequestsPerMinute` (default: 3000 per minute). This is applied at the router level in `cmd/payload-tracker-api/main.go`.
