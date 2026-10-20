# API Contracts Guidelines

## Spec and Versioning

- The API spec is a Swagger 2.0 file at `api/api.spec.yaml`. Update it whenever adding or changing endpoints, parameters, or response schemas.
- The API is versioned as `v1` and mounted at `/api/v1/` (production) or `/app/payload-tracker/api/v1/` (when `ENVIRONMENT=DEV`). Route registration lives in `cmd/payload-tracker-api/main.go`.
- This is a read-only (GET-only) REST API. There are no POST/PUT/PATCH/DELETE endpoints.

## Routing

- Use `go-chi/chi/v5` for routing. Register each endpoint on the `sub` router with the `ResponseMetricsMiddleware` middleware, e.g. `sub.With(endpoints.ResponseMetricsMiddleware).Get("/payloads", endpoints.Payloads)`.
- Path parameters use chi's `{param}` syntax, e.g. `{request_id}`. Extract them with `chi.URLParam(r, "request_id")`.
- Rate limiting is applied globally via `httprate.LimitByIP` configured from `RequestConfig.MaxRequestsPerMinute`.

## Response Envelope

- Paginated list endpoints (`/payloads`, `/statuses`) return a JSON object with three required fields: `count` (int64 total), `elapsed` (float64 seconds), and `data` (array). Use `structs.PayloadsData` or `structs.StatusesData`.
- Single-resource endpoints (`/payloads/{request_id}`) return `{"data": [...], "duration": {...}}` via `structs.PayloadRetrievebyID`. Note the `data` field is an array of status records, and `duration` is a map of `"service:source"` to timedelta strings.
- Link endpoints (`archiveLink`, `kibanaLink`) return `{"url": "..."}` via `structs.PayloadArchiveLink` or `structs.PayloadKibanaLink`.
- Role-check endpoints return `{"allowed": true/false}` via `structs.ArchiveLinkRole`.

## Error Responses

- Use the `getErrorBody(message, statusCode)` helper from `internal/endpoints/utils.go`. It produces `structs.ErrorResponse` with three fields: `title` (HTTP status text), `message` (detail string), `status` (int code).
- Write errors through `writeResponse(w, statusCode, body)` which sets `Content-Type: application/json`, writes the status code, then the body.
- Standard HTTP codes used: 200 (success), 400 (bad input/invalid UUID/invalid sort/invalid timestamp), 401 (missing identity header), 403 (missing required LDAP role), 404 (resource not found), 500 (internal error).

## Pagination

- Pagination uses `page` (0-indexed, default 0) and `page_size` (default 10) query parameters. Offset is calculated as `page_size * page` in the query layer.
- Parsed via `initQuery()` in `internal/endpoints/utils.go` into `structs.Query`.

## Sorting

- Sorting uses `sort_by` and `sort_dir` query parameters. Defaults differ by endpoint.
- `/payloads`: default `sort_by=created_at`, `sort_dir=desc`. Valid `sort_by` values: `account`, `org_id`, `inventory_id`, `system_id`, `created_at` (defined in `validAllSortBy`).
- `/payloads/{request_id}`: default `sort_by=date`, `sort_dir=asc`. Valid `sort_by`: `service`, `source`, `status_msg`, `date`, `created_at` (defined in `validIDSortBy`).
- `/statuses`: default `sort_by=date`, `sort_dir=desc`. Valid `sort_by`: `service`, `source`, `request_id`, `status`, `status_msg`, `date`, `created_at` (defined in `validStatusesSortBy`).
- Validate `sort_by` and `sort_dir` using `stringInSlice()` before querying. Return 400 with a message listing valid options on failure.

## Timestamp Filtering

- Timestamp filters use the suffix convention: `_lt`, `_lte`, `_gt`, `_gte` appended to the field name (e.g. `created_at_lt`, `date_gte`).
- `/payloads` supports `created_at_*` filters only. `/statuses` supports both `created_at_*` and `date_*` filters.
- Timestamps are validated against `time.RFC3339` format using `validTimestamps()` in `internal/endpoints/utils.go`. Invalid formats return 400.

## Request ID Validation

- Path parameter `request_id` is validated as a UUID using `uuid.Parse()` from `github.com/google/uuid` via the `isValidUUID()` helper. Invalid UUIDs return 400 and increment the `payload_tracker_api_invalid_request_IDs` Prometheus counter.

## Authentication and Authorization

- Protected endpoints (e.g. `archiveLink`) read the `x-rh-identity` HTTP header (base64-encoded JSON). Missing header returns 401.
- Role checks use `checkForRole()` in `internal/endpoints/utils.go`, which decodes the identity header and checks `identity.associate.Role` for the required role string. Missing role returns 403.

## Verbosity Control

- `/payloads/{request_id}` accepts a `verbosity` query parameter (0, 1, 2). This controls which database columns are selected, defined in `defineVerbosity()` in `internal/queries/queries_api.go`. Verbosity 2 returns the fewest fields; verbosity 0 (default) returns all fields.

## Response Struct Conventions

- Response structs live in `internal/structs/api_structs.go`. Use `json:"field_name"` tags with `snake_case` naming. Use `omitempty` on optional fields in `SinglePayloadData` and `StatusRetrieve`.
- Database models live in `internal/models/models.go` (used by GORM) and `internal/models/db/models.go` (alternative definitions). JSON tags on model structs also use `snake_case`.

## Kafka Message Schema

- Incoming Kafka messages use `message.PayloadStatusMessage` from `internal/models/message/payload-status.go`. Required fields: `service`, `status`, `date`. Optional fields use `omitempty`.
- The `date` field uses a custom `FormatedTime` type that parses RFC3339 with fallback to appending "Z" for timezone-unaware timestamps.

## Adding a New Endpoint

- Add the handler function in the appropriate file under `internal/endpoints/`.
- Register the route in `cmd/payload-tracker-api/main.go` on the `sub` router with `ResponseMetricsMiddleware`.
- Add the endpoint to `api/api.spec.yaml` with parameter definitions, response schemas, and error references.
- Add corresponding response structs to `internal/structs/api_structs.go`.
- Write Ginkgo/Gomega tests in `internal/endpoints/` using `httptest.NewRecorder` and `test.MakeTestRequest()` from `internal/utils/test/helpers.go`. Mock database calls by reassigning the package-level function variables (e.g. `endpoints.RetrievePayloads = mockedFn`).

## Verification

```bash
# Run all unit tests (includes endpoint handler tests)
go test -p 1 -v ./...

# Check that api.spec.yaml is valid YAML
python3 -c "import yaml; yaml.safe_load(open('api/api.spec.yaml'))" 2>&1

# Verify sort_by allowlists in utils.go match the spec enums
grep -n 'validAllSortBy\|validIDSortBy\|validStatusesSortBy\|validSortDir' internal/endpoints/utils.go

# Verify all endpoint routes are registered
grep -n 'sub.With\|sub.Get' cmd/payload-tracker-api/main.go

# Lint Go source
gofmt -l .
```
