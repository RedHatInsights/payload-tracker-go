# Testing Guidelines

## Framework and Imports

- Use Ginkgo v1 (`github.com/onsi/ginkgo`) and Gomega (`github.com/onsi/gomega`) for all tests.
- Dot-import both Ginkgo and Gomega in every test file:
  ```go
  . "github.com/onsi/ginkgo"
  . "github.com/onsi/gomega"
  ```
- Import the shared test helpers as `"github.com/redhatinsights/payload-tracker-go/internal/utils/test"` and call them via the `test` package prefix.

## Suite Bootstrap

- Each test package has a `*_suite_test.go` file containing a single `Test*` function that registers the Ginkgo fail handler, initializes the logger, and runs specs:
  ```go
  func TestEndpoints(t *testing.T) {
      RegisterFailHandler(Fail)
      l.InitLogger()
      RunSpecs(t, "Suite Name")
  }
  ```
- Import the logger as `l "github.com/redhatinsights/payload-tracker-go/internal/logging"` and call `l.InitLogger()` before `RunSpecs`.

## Two Test Tiers

1. **Unit tests (mocked DB)** -- `internal/endpoints/` package (`endpoints_test`). These use package-level function variables (`endpoints.RetrievePayloads`, `endpoints.RetrieveStatuses`, `endpoints.RetrieveRequestIdPayloads`) reassigned in `BeforeEach` to mock functions that return canned data. No database is required.
2. **Integration tests (live DB)** -- `internal/endpoints/endpoints_db_test/` package (`endpoints_db_test`), `internal/kafka/` package (white-box, `package kafka`), and `internal/queries/` package (white-box, `package queries`). These call `test.WithDatabase()` to get a real PostgreSQL connection.

## Mocking Pattern for Endpoint Tests

- The production code exposes query functions as reassignable package-level variables (e.g., `endpoints.RetrievePayloads`). Override these in `BeforeEach`:
  ```go
  BeforeEach(func() {
      endpoints.RetrievePayloads = mockedRetrievePayloads
  })
  ```
- Define mock functions at file scope matching the original signature but returning package-level test vars (`payloadReturnCount`, `payloadReturnData`). Set those vars inside each `It` block to control returned data per scenario.
- For DB-backed endpoint tests, set `endpoints.Db` to the connection returned by `test.WithDatabase()` in `BeforeEach`.

## Database Test Helper (`internal/utils/test/fixtures.go`)

- `test.WithDatabase()` returns a `func() *gorm.DB` closure. Call the closure inside test bodies to get the connection.
- It opens a PostgreSQL connection in `BeforeEach` using config defaults (`crc`/`crc`/`crc` on `0.0.0.0:5432`) and closes it in `AfterEach`.
- DB tests require a running PostgreSQL instance with migrations applied. Use `make run-migration` before running tests.

## HTTP Request Helper (`internal/utils/test/helpers.go`)

- Use `test.MakeTestRequest(uri, queryParams)` to build `*http.Request` objects. It accepts a `map[string]interface{}` for query parameters and formats them into the URL.
- For URL path parameters (e.g., `request_id`), use `chi.NewRouteContext()` and inject it via `context.WithValue(req.Context(), chi.RouteCtxKey, rctx)`.

## HTTP Testing Pattern

- Use `httptest.NewRecorder()` for the response writer, create a new recorder in `BeforeEach`.
- Wrap endpoint functions with `http.HandlerFunc(endpoints.FunctionName)`.
- Call `handler.ServeHTTP(rr, req)` and assert on `rr.Code` and `rr.Body`.
- Deserialize response bodies with `ioutil.ReadAll` + `json.Unmarshal` into the appropriate `structs.*` type, then assert individual fields.

## BDD Structure Convention

- Use `Describe` for the top-level subject (endpoint name or component).
- Use `Context` for preconditions ("With a valid request", "With invalid sort_dir parameter").
- Use `It` for the specific assertion ("should return HTTP 200").
- Use `BeforeEach` for per-test setup (recorder, handler, mocks). Avoid `BeforeSuite`/`AfterSuite`; the existing codebase does not use them.

## Kafka Handler Tests (White-Box)

- Kafka handler tests live in `internal/kafka/handler_test.go` under `package kafka` (white-box) to access unexported types like `handler` and `validateRequestID`.
- Build Kafka messages using a helper that marshals a `message.PayloadStatusMessage` into a `*kafka.Message` with a topic partition.
- After calling `msgHandler.onMessage(...)`, verify side effects by querying the database with `queries.RetrieveRequestIdPayloads`.

## Query Tests (White-Box)

- Query tests in `internal/queries/queries_test.go` use `package queries` (white-box) to test unexported functions like `newPayloadFieldsRepositoryFromCache`.
- Mock the `PayloadFieldsRepository` interface with a struct that tracks which methods were called via boolean fields, to verify caching behavior (cache miss vs. cache hit).

## Running Tests

- **All tests (with DB):** Start PostgreSQL (`docker compose up payload-tracker-db -d`), run `make run-migration`, then run `make test` (which executes `go test -p 1 -v ./...`).
- **Unit tests only (no DB):** Run `go test -v ./internal/endpoints/` -- this package uses mocked queries and does not need PostgreSQL.
- The `-p 1` flag in the Makefile serializes test packages to prevent concurrent DB access conflicts.

## CI Configuration (`.github/workflows/pr.yml`)

- CI uses a PostgreSQL service container (`postgres` image, `crc`/`crc`/`crc` credentials, port 5432).
- CI runs: `make run-migration`, `make build-all`, then `go test ./...`.
- Ensure new test packages that require a database work with these defaults.

## Verification

```bash
# Build succeeds
make build-all

# Unit tests pass (no DB required)
go test -v ./internal/endpoints/

# All tests pass (requires PostgreSQL running with migrations applied)
make test

# Verify Ginkgo dot-imports are used in new test files
find internal -name "*_test.go" -type f -exec grep -L 'onsi/ginkgo' {} \;

# Verify suite bootstrap exists for any new test package
find internal -name "*_suite_test.go" -printf "%h\n" | sort -u
```
