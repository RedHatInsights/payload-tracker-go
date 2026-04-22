# Data Validation Guidelines

This document describes the data validation conventions used in the payload-tracker-go repository. Follow these patterns when adding new endpoints, Kafka message handlers, or query parameters.

## 1. UUID Validation

UUID validation uses `github.com/google/uuid` via the `isValidUUID` helper in `internal/endpoints/utils.go`.

**When to validate:** UUID validation is only applied on endpoints that pass the request_id to external services (`/archiveLink`, `/kibanaLink`). The `/payloads/{request_id}` lookup endpoint does NOT validate UUID format -- it passes the raw value to the DB query and returns 404 if not found.

**Pattern:**
```go
if !isValidUUID(reqID) {
    IncInvalidAPIRequestIDs()
    writeResponse(w, http.StatusBadRequest, getErrorBody(fmt.Sprintf("%s is not a valid UUID", reqID), http.StatusBadRequest))
    return
}
```

**Rules:**
- Call `IncInvalidAPIRequestIDs()` before returning the 400 response to track invalid IDs in Prometheus metrics
- Return HTTP 400 with the invalid value in the error message

## 2. Request ID Length Validation (Kafka Consumer)

The consumer validates request_id length via `validateRequestID()` in `internal/kafka/handler.go`. The expected length is configured by `validate.request.id.length` (default: 32, i.e., UUID without dashes).

**Rules:**
- When `ValidateRequestIDLength` is 0, length validation is skipped entirely
- A request ID of exactly the configured length passes; anything else (including empty string) is rejected
- On failure, call `IncInvalidConsumerRequestIDs()` and silently drop the message (no error response -- this is a consumer, not an API)
- The default length of 32 means UUIDs with dashes (36 chars) will be rejected

## 3. Timestamp Validation (RFC3339)

All timestamp query parameters are validated using `time.Parse(time.RFC3339, ts)` in the `validTimestamps()` function in `internal/endpoints/utils.go`.

**Timestamp parameters:** `created_at_lt`, `created_at_lte`, `created_at_gt`, `created_at_gte`, `date_lt`, `date_lte`, `date_gt`, `date_gte`.

**Rules:**
- The `/payloads` endpoint validates only `created_at_*` timestamps (calls `validTimestamps(q, false)`)
- The `/statuses` endpoint validates all eight timestamp fields (calls `validTimestamps(q, true)`)
- Empty timestamp values are allowed and skipped
- Non-empty timestamps that fail `time.RFC3339` parsing return HTTP 400 with message `"invalid timestamp format provided"`

## 4. Kafka Message Date Deserialization

The `FormatedTime` type in `internal/models/message/payload-status.go` implements custom `UnmarshalJSON` for incoming Kafka message dates.

**Fallback behavior:**
1. Parse as RFC3339
2. If whitespace exists between date and time parts, replace with `T`
3. If parsing fails, append `Z` (assume UTC) and retry
4. If still fails, return error and the entire message is dropped

This is the only custom JSON deserializer in the codebase. All other fields use standard `json.Unmarshal`.

## 5. Input Sanitization (Lowercase Normalization)

The `sanitizePayload()` function in `internal/kafka/handler.go` lowercases Kafka message fields before DB storage.

**Fields normalized to lowercase:** `service`, `status`, and `source` (only if non-empty).

**Fields NOT normalized:** `request_id`, `account`, `org_id`, `inventory_id`, `system_id`, `status_msg`.

Do not lowercase additional fields without verifying downstream consumers are not case-sensitive.

## 6. Query Parameter Validation (Allowed Values)

Sort parameters are validated using `stringInSlice()` against predefined slices in `internal/endpoints/utils.go`.

**Allowed sort_dir values:** `asc`, `desc` (applies to all endpoints).

**Allowed sort_by values differ by endpoint:**

| Endpoint | Variable | Allowed Values |
|---|---|---|
| `/payloads` | `validAllSortBy` | `account`, `org_id`, `inventory_id`, `system_id`, `created_at` |
| `/payloads/{request_id}` | `validIDSortBy` | `service`, `source`, `status_msg`, `date`, `created_at` |
| `/statuses` | `validStatusesSortBy` | `service`, `source`, `request_id`, `status`, `status_msg`, `date`, `created_at` |

**Rules:**
- Return HTTP 400 with message listing all valid options: `"sort_by must be one of ..."` joined by commas
- The `/payloads` endpoint defaults `sort_by` to `created_at`; all other endpoints default to `date`
- The default `sort_dir` is `desc` for all endpoints

## 7. Pagination Parameters

Pagination is handled in `initQuery()` in `internal/endpoints/utils.go`.

**Rules:**
- `page` and `page_size` are parsed with `strconv.Atoi()`; non-integer values return HTTP 400
- Defaults: `page=0`, `page_size=10`
- There is no upper-bound validation on `page_size` -- any integer is accepted

## 8. Identity Header Validation

The `checkForRole()` function in `internal/endpoints/utils.go` validates the `x-rh-identity` header.

**Validation sequence:**
1. Missing header returns HTTP 401 `"Missing Identity Header"`
2. Header is base64-decoded, failure returns HTTP 401
3. JSON is unmarshalled into a struct with `identity.associate.Role` path
4. If the required role is not found in the roles array, returns HTTP 403

**Rules:**
- Only the `/archiveLink` and `/roles/archiveLink` endpoints enforce role checks
- The required role is configured via `StorageBrokerURLRole` (default: `"platform-archive-download"`)

## 9. Error Response Format

All validation errors use the `ErrorResponse` struct from `internal/structs/api_structs.go`.

```go
type ErrorResponse struct {
    Title   string `json:"title"`   // http.StatusText(status) e.g. "Bad Request"
    Message string `json:"message"` // Specific error detail
    Status  int    `json:"status"`  // HTTP status code as integer
}
```

Construct errors via `getErrorBody(message, statusCode)` and send with `writeResponse(w, statusCode, body)`. The `Content-Type` header is always set to `application/json`.

## 10. Database Query Safety

Query parameters used in GORM `Where` clauses use parameterized queries throughout `internal/queries/queries_api.go`.

```go
dbQuery.Where("account = ?", apiQuery.Account)
```

Time-range conditions in `chainTimeConditions()` also use parameterized queries. The `sort_by` and `sort_dir` values are interpolated into `ORDER BY` clauses as raw strings, but they are pre-validated against allowlists before reaching the query layer.

## 11. Upsert Behavior and Conditional Updates

In `UpsertPayloadByRequestId()` (`internal/queries/queries_consumer.go`), only non-empty optional fields are included in the update clause. The `request_id` is always included.

**Fields conditionally updated:** `account`, `org_id`, `inventory_id`, `system_id` -- each is added to the upsert only if non-empty in the incoming message. This prevents overwriting existing data with blank values.

## 12. Consumer Retry and Error Handling

When inserting a payload status fails, the consumer retries up to `db.retries` times (default: 3). Failures are logged at debug level with the attempt count. There is no backoff between retries.

If the initial JSON unmarshal or request_id validation fails, the message is silently dropped with no retries.
