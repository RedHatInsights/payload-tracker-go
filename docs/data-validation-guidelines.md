# Data Validation Guidelines

## Request ID Validation

- Validate `request_id` length in the Kafka consumer using `validateRequestID()` in `internal/kafka/handler.go`. The configurable length is set via `validate.request.id.length` (default: 32, matching a UUID without dashes). When the configured length is 0, length validation is skipped.
- Validate `request_id` as a standard UUID (with dashes) using `isValidUUID()` in `internal/endpoints/utils.go` for the `archiveLink` and `kibanaLink` API endpoints. This function uses `github.com/google/uuid` `uuid.Parse()`.
- The `/payloads/{request_id}` endpoint does not perform UUID format validation on the path parameter; it relies on database lookup and returns 404 when no match is found.
- Increment `payload_tracker_consumer_invalid_request_IDs` via `IncInvalidConsumerRequestIDs()` when a consumer request ID fails validation. Increment `payload_tracker_api_invalid_request_IDs` via `IncInvalidAPIRequestIDs()` when an API request ID fails UUID validation.

## Message Sanitization (Kafka Consumer)

- Lowercase the `Service`, `Status`, and `Source` fields of every incoming `PayloadStatusMessage` via `sanitizePayload()` in `internal/kafka/handler.go` using `strings.ToLower()`.
- Only lowercase `Source` when it is non-empty; skip the call when the field is blank.
- Sanitization happens after JSON unmarshaling and request ID validation, but before any database lookups or inserts.

## Timestamp Validation

- Parse all timestamp query parameters (`created_at_lt`, `created_at_lte`, `created_at_gt`, `created_at_gte`, `date_lt`, `date_lte`, `date_gt`, `date_gte`) against `time.RFC3339` in the `validTimestamps()` function in `internal/endpoints/utils.go`.
- The `/payloads` endpoint validates only `created_at_*` timestamps (calls `validTimestamps(q, false)`). The `/statuses` endpoint validates both `created_at_*` and `date_*` timestamps (calls `validTimestamps(q, true)`).
- The Kafka consumer's `FormatedTime.UnmarshalJSON()` in `internal/models/message/payload-status.go` parses incoming dates with `time.RFC3339`. It replaces spaces with `T` to handle `"2021-08-04 07:45:26"` style inputs, and appends `"Z"` as a fallback when the initial parse fails (for timestamps missing timezone info).

## Sort Parameter Validation

- Validate `sort_by` and `sort_dir` against allow-lists defined in `internal/endpoints/utils.go` using `stringInSlice()`:
  - `/payloads`: `validAllSortBy` = `[account, org_id, inventory_id, system_id, created_at]`, defaults to `created_at`
  - `/payloads/{request_id}`: `validIDSortBy` = `[service, source, status_msg, date, created_at]`, defaults to `date`
  - `/statuses`: `validStatusesSortBy` = `[service, source, request_id, status, status_msg, date, created_at]`, defaults to `date`
  - `sort_dir`: `validSortDir` = `[asc, desc]`, defaults to `desc`
- Return HTTP 400 with a message listing valid options when validation fails.

## Pagination Defaults

- `initQuery()` in `internal/endpoints/utils.go` sets `Page` to 0 and `PageSize` to 10 by default. Both are parsed from query params via `strconv.Atoi()`, and the parse error is propagated to the caller as a 400 response.

## Kafka Message Schema Enforcement

- The `PayloadStatusMessage` struct in `internal/models/message/payload-status.go` defines the expected Kafka message shape. Required JSON fields: `service`, `status`, `date`. Optional fields: `source`, `account`, `org_id`, `request_id`, `inventory_id`, `system_id`, `status_msg`.
- When JSON unmarshaling fails, the handler logs the error, increments the invalid consumer request ID counter, and returns early without inserting into the database.
- `Source` is treated as optional throughout: the consumer checks `payloadStatus.Source != ""` before looking up or creating a source entry; `InsertPayloadStatus()` in `internal/queries/queries_consumer.go` omits `source_id` from the insert when no source is set.

## Database Upsert Validation

- `UpsertPayloadByRequestId()` in `internal/queries/queries_consumer.go` conditionally adds columns to the upsert's `ON CONFLICT` update list only when the field is non-empty (`Account`, `OrgId`, `InventoryId`, `SystemId`). The `request_id` column is included unconditionally.
- The `payloads.request_id` column has a `UNIQUE` constraint in the database schema (`migrations/000001_create_payload_tracker_tables.up.sql`), enforcing uniqueness at the DB level.

## Identity Header Validation (API)

- The `checkForRole()` function in `internal/endpoints/utils.go` validates the `x-rh-identity` header: returns 401 if the header is missing, attempts base64 decoding and JSON unmarshaling into an `IdentityHeader` struct, then checks for a required role string in the `identity.associate.Role` array. Returns 403 if the role is absent.
- Only the `archiveLink` and `roles/archiveLink` endpoints require identity header validation.

## Error Response Format

- Return validation errors as JSON using the `ErrorResponse` struct from `internal/structs/api_structs.go` with fields `title` (from `http.StatusText()`), `message`, and `status` (integer HTTP code). Use `getErrorBody()` in `internal/endpoints/utils.go` to construct these.

## Adding New Validated Fields

- For new API query parameters: add the field to the `Query` struct in `internal/structs/api_structs.go`, read it from `r.URL.Query().Get()` in `initQuery()`, and if it is a sort field, add it to the appropriate `validSortBy` slice in `internal/endpoints/utils.go`.
- For new Kafka message fields: add the field to `PayloadStatusMessage` in `internal/models/message/payload-status.go` with the appropriate `json` tag, add a corresponding column to the DB model in `internal/models/db/models.go`, and handle the field in `createPayload()` or the status-insert logic in `internal/kafka/handler.go`.
- If the new field is a string that represents a categorical value (like service, status, source), apply `strings.ToLower()` in `sanitizePayload()`.

## Verification

```bash
# Run the full test suite (requires a running PostgreSQL instance)
make test

# Check that sanitizePayload lowercases service, status, and source
grep -n 'strings.ToLower' internal/kafka/handler.go

# Confirm UUID validation uses google/uuid
grep -n 'uuid.Parse' internal/endpoints/utils.go

# Confirm timestamp validation uses time.RFC3339
grep -n 'time.RFC3339' internal/endpoints/utils.go internal/models/message/payload-status.go

# Verify sort_by allow-lists match endpoint usage
grep -n 'validAllSortBy\|validIDSortBy\|validStatusesSortBy\|validSortDir' internal/endpoints/utils.go internal/endpoints/payloads.go internal/endpoints/statuses.go

# Lint formatting
make lint
```
