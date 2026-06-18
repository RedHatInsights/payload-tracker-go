# Go Web Frameworks Guidelines

## Router Architecture

- Use `github.com/go-chi/chi/v5` as the HTTP router. Do not introduce alternative routers.
- Create separate `chi.NewRouter()` instances for public API traffic and metrics/health traffic, bound to different ports (`PublicPort` and `MetricsPort` from config). See `cmd/payload-tracker-api/main.go` lines 40-42 for the three-router pattern (`r`, `mr`, `sub`).
- Mount API routes on a sub-router and attach it to the main router via `r.Mount("/api/v1/", sub)`. In `DEV` environment, use the path `/app/payload-tracker/api/v1/` instead.
- Expose Prometheus metrics via `promhttp.Handler()` on the metrics router, not the public API router.
- Register the root liveness probe (`lubdub`) on both the main and metrics routers at `/`.

## Route Registration

- Register all API sub-routes with `sub.With(endpoints.ResponseMetricsMiddleware).Get(...)` so every API response code is tracked by Prometheus. Do not register API routes without this middleware.
- Use `chi.URLParam(r, "param_name")` to extract path parameters inside handlers. In tests, inject path params via `chi.NewRouteContext()` and `rctx.URLParams.Add(...)` set on the request context with key `chi.RouteCtxKey`.
- This API is read-only: all routes use `.Get(...)`. Follow this pattern unless a new endpoint explicitly requires a different HTTP method.

## Rate Limiting

- Apply `httprate.LimitByIP` from `github.com/go-chi/httprate` as global middleware on the main router (`r.Use(...)`). The limit value comes from `cfg.RequestConfig.MaxRequestsPerMinute` with a 1-minute window.
- Do not apply rate limiting to the metrics or health routers.

## Middleware: Response Metrics

- `ResponseMetricsMiddleware` in `internal/endpoints/metrics.go` wraps `http.ResponseWriter` with `metricTrackingResponseWriter` to increment the `payload_tracker_responses` counter (labeled by `code`) on each `WriteHeader` call.
- When adding new middleware that wraps `http.ResponseWriter`, implement all three interface methods: `Header()`, `WriteHeader(int)`, `Write([]byte)`.
- Apply `ResponseMetricsMiddleware` per-route using chi's `.With(...)` inline syntax, not as global middleware.

## Handler Patterns

- Prefer package-level functions with signature `func(w http.ResponseWriter, r *http.Request)` for simple handlers (e.g., `Payloads`, `Statuses`, `RolesArchiveLink`).
- Use closure-returning factory functions (`func(...) http.HandlerFunc`) when a handler needs injected dependencies or configuration. Examples: `HealthCheckHandler` (injects `*gorm.DB` and config), `PayloadArchiveLink` (injects an archive-link fetcher function), `CreatePayloadArchiveLinkHandler` (selects implementation by config).
- Extract query parameters via `r.URL.Query().Get("param")`. Initialize a `structs.Query` struct with defaults via the `initQuery(r)` helper in `internal/endpoints/utils.go`.

## Response Writing

- Use the `writeResponse(w, statusCode, body)` helper from `internal/endpoints/utils.go` for all JSON responses. It sets `Content-Type: application/json`, writes the status code, then writes the body.
- Use `getErrorBody(message, statusCode)` to produce JSON error responses matching the `structs.ErrorResponse` shape (`title`, `message`, `status` fields).
- For non-JSON responses (health checks, liveness probes), set `Content-Type` explicitly and call `w.WriteHeader` / `w.Write` directly.

## Testing Chi URL Parameters

- For handlers that read chi URL params, inject a `chi.RouteContext` into the request context in tests:
  ```go
  rctx := chi.NewRouteContext()
  rctx.URLParams.Add("param_name", "value")
  req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
  ```

## Dual-Server Startup

- The API binary (`cmd/payload-tracker-api/main.go`) starts two `http.Server` instances: the metrics server in a goroutine, the public API server on the main goroutine. Both call `panic(err)` on `ListenAndServe` failure.
- The consumer binary (`cmd/payload-tracker-consumer/main.go`) follows the same pattern for its metrics-only server.

## Verification

```bash
# Confirm all routes use ResponseMetricsMiddleware
grep -n '\.Get\|\.Post\|\.Put\|\.Delete\|\.Patch' cmd/payload-tracker-api/main.go | grep 'sub\.' | grep -v 'ResponseMetricsMiddleware'
# Expected: empty (no sub-routes without middleware), except conditional mock routes

# Confirm no alternative routers introduced
grep -rn 'gorilla/mux\|gin-gonic\|echo\|fiber\|httprouter' --include="*.go" .
# Expected: empty

# Confirm httprate is only on the main router
grep -n 'httprate' cmd/payload-tracker-api/main.go

# Run unit tests
make test

# Lint formatting
make lint
```
