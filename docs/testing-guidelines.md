# Testing Guidelines

This document covers repo-specific testing conventions for payload-tracker-go. All tests use **Ginkgo v1** (`github.com/onsi/ginkgo`) with **Gomega** matchers.

## Running Tests

Run all tests sequentially (required due to shared database state):

```
make test
```

This executes `go test -p 1 -v ./...`. The `-p 1` flag is required -- tests share a PostgreSQL database and must not run in parallel.

## Test Framework: Ginkgo v1

This repo uses Ginkgo v1, not v2. Import paths must use the v1 package:

```go
import (
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)
```

Both Ginkgo and Gomega are dot-imported in every test file. Use `Describe`, `Context`, `It`, `BeforeEach`, and `AfterEach` directly without a package prefix.

## Test Suite Bootstrap

Each tested package has a `*_suite_test.go` file that bootstraps Ginkgo and initializes the logger. Follow this exact pattern:

```go
package <package_name>_test  // or package <package_name> for white-box tests

import (
    "testing"
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
    l "github.com/redhatinsights/payload-tracker-go/internal/logging"
)

func TestEndpoints(t *testing.T) {
    RegisterFailHandler(Fail)
    l.InitLogger()
    RunSpecs(t, "<Suite Name>")
}
```

`l.InitLogger()` must be called in every suite bootstrap function before `RunSpecs`.

## Two Test Categories

### 1. Unit Tests (Mocked DB)

Located in the same `_test` package as the code under test (e.g., `internal/endpoints/payloads_test.go` in package `endpoints_test`).

These tests mock database calls by reassigning package-level function variables. The endpoint package exposes replaceable function vars:

```go
// In endpoints/payloads.go
var (
    RetrievePayloads          = queries.RetrievePayloads
    RetrieveRequestIdPayloads = queries.RetrieveRequestIdPayloads
    Db                        = getDb
)

// In endpoints/statuses.go
var RetrieveStatuses = queries.RetrieveStatuses
```

In tests, replace these in `BeforeEach`:

```go
BeforeEach(func() {
    endpoints.RetrievePayloads = mockedRetrievePayloads
})
```

Mock functions must match the exact signature of the original. Use blank identifiers for unused parameters:

```go
func mockedRetrievePayloads(_ *gorm.DB, _ int, _ int, _ structs.Query) (int64, []models.Payloads) {
    return payloadReturnCount, payloadReturnData
}
```

Control mock return values via package-level variables set inside each `It` block before calling the handler.

### 2. Integration Tests (Real DB)

Located in a separate test package with `_db_test` suffix (e.g., `internal/endpoints/endpoints_db_test/`). The `internal/kafka/handler_test.go` tests also use a real DB but are in the `kafka` package (white-box testing to access unexported `handler` struct).

Use `test.WithDatabase()` to get a database connection:

```go
db := test.WithDatabase()

BeforeEach(func() {
    endpoints.Db = db
})
```

`test.WithDatabase()` returns a `func() *gorm.DB` closure. Call `db()` to get the actual connection. It opens a new connection in `BeforeEach` and closes it in `AfterEach` automatically.

Integration tests require a running PostgreSQL instance. Run `make run-migration` before tests. Start the database with `docker compose up payload-tracker-db`.

## Database Setup for Integration Tests

- Insert test data directly via GORM: `db().Create(&payloadData)`
- Check insert errors with: `Expect(db().Create(&data).Error).ToNot(HaveOccurred())`
- Use `uuid.New().String()` from `github.com/google/uuid` to generate unique IDs for test data
- For `payload_statuses` records, create all FK references (status, source, service, payload) first
- Use unique names with timestamps for lookup-table entries to avoid unique constraint violations: `fmt.Sprintf("test-status-%d", time.Now().Unix())`

## HTTP Handler Testing Pattern

Use `net/http/httptest` for all endpoint tests:

```go
rr = httptest.NewRecorder()
handler = http.HandlerFunc(endpoints.Payloads)

req, err := test.MakeTestRequest("/api/v1/payloads", query)
Expect(err).To(BeNil())
handler.ServeHTTP(rr, req)
Expect(rr.Code).To(Equal(200))
```

- Build requests with `test.MakeTestRequest(uri, queryParams)` from `internal/utils/test/helpers.go`. It accepts a `map[string]interface{}` for query parameters.
- For endpoints needing URL params (e.g., `{request_id}`), inject Chi route context:

```go
rctx := chi.NewRouteContext()
rctx.URLParams.Add("request_id", requestId)
req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
```

- Parse response bodies by reading `rr.Body` with `ioutil.ReadAll` then `json.Unmarshal` into the appropriate structs type.

## Mocking External HTTP Services

For endpoints that call external services (e.g., storage-broker), create an `httptest.Server` in `BeforeEach`:

```go
mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"url": "www.example.com"}`))
}))
handler = http.HandlerFunc(endpoints.PayloadArchiveLink(
    endpoints.RequestArchiveLink(mockServer.URL, 10),
))
```

## Interface-Based Mocking (Queries Package)

The `queries` package uses the `PayloadFieldsRepository` interface for status/service/source lookups. Mock it with a struct implementing all three methods:

```go
type mockPayloadFieldsRepository struct {
    getStatusCalled  bool
    getServiceCalled bool
    getSourceCalled  bool
}

func (m *mockPayloadFieldsRepository) GetStatus(statusName string) models.Statuses {
    m.getStatusCalled = true
    return models.Statuses{Id: 1234, Name: statusName}
}
```

Use boolean flags to verify cache hit/miss behavior.

## Kafka Message Testing

Build Kafka messages for handler tests using `confluent-kafka-go/v2/kafka.Message`:

```go
func newKafkaMessage(value message.PayloadStatusMessage) *k.Message {
    msgValue, err := json.Marshal(value)
    Expect(err).ToNot(HaveOccurred())
    topic := "topic.payload.status"
    return &k.Message{
        Value: msgValue,
        TopicPartition: k.TopicPartition{
            Topic: &topic, Partition: 0, Offset: k.Offset(0),
        },
    }
}
```

The kafka handler tests are white-box (package `kafka`, not `kafka_test`) to access the unexported `handler` struct and `onMessage` method. After processing a message, verify results by querying the database directly with `queries.RetrieveRequestIdPayloads`.

## Identity Header Testing

For endpoints requiring RBAC (e.g., `archiveLink`), use base64-encoded JSON identity headers:

- Set valid/invalid headers as package-level constants (see `internal/endpoints/roles_test.go`)
- Set via `req.Header.Set("x-rh-identity", validIdentityHeader)`
- Test three cases: missing header (401), wrong role (403), correct role (200)

## Test Structure Conventions

- `Describe` blocks name the function or endpoint under test
- `Context` blocks describe the input scenario (e.g., "With invalid sort_dir parameter")
- `It` blocks describe expected behavior (e.g., "should return HTTP 400")
- Reset state in `BeforeEach`, including creating new `httptest.ResponseRecorder` and re-assigning mocked functions
- Use `Expect(err).To(BeNil())` for error checks on request creation and `Expect(err).ToNot(HaveOccurred())` for GORM operations
