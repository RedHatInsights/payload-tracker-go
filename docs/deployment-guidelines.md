# Deployment Guidelines

## Dual-Deployment Architecture

This repo produces a single container image that runs as two separate ClowdApp deployments in `deployments/clowdapp.yml`: `api` (runs `./pt-api`) and `consumer` (runs `./pt-consumer`). The `command` field in the ClowdApp manifest selects which binary to launch -- the `build/Containerfile` intentionally has no `CMD` directive.

- When adding a new deployment, follow this pattern: build the binary in the multi-stage `build/Containerfile`, `COPY --from=builder` it to the runtime image, and set the `command` in `deployments/clowdapp.yml`.
- Prefer parameterized resource limits per deployment (`CPU_LIMIT`/`MEMORY_LIMIT` for API, `CONSUMER_CPU_LIMIT`/`CONSUMER_MEMORY_LIMIT` for consumer) rather than sharing a single set.

## Container Image Build (`build/Containerfile`)

- Builder stage: `registry.access.redhat.com/ubi9/go-toolset:latest`. Runtime stage: `registry.access.redhat.com/ubi9/ubi-minimal:latest`.
- Build all four binaries in a single `RUN` layer: `pt-api`, `pt-consumer`, `pt-migration`, `pt-seeder`.
- Pin the Go toolchain via `ARG GOTOOLCHAIN=go1.26.3+auto` -- keep this in sync with the version in `go.mod`.
- The runtime image runs as `USER 1001` (non-root). Avoid adding steps that require root after the final `USER 1001` line.
- The `Dockerfile` symlink at the repo root points to `build/Containerfile` -- Tekton pipelines reference `Dockerfile`.

## Health Probes

The API and consumer use different health-check paths and ports:

| Deployment | Liveness | Readiness | Port |
|------------|----------|-----------|------|
| api | `/health` | `/health` | 8000 (Clowder `publicPort`) |
| consumer | `/live` | `/ready` | 9000 (Clowder `metricsPort`) |

- The API health endpoint is defined in `internal/endpoints/health.go` and pings the database via `db.Ping()`.
- The consumer reuses the same `HealthCheckHandler` but mounts it on `/live` and `/ready` on its metrics HTTP server (`cmd/payload-tracker-consumer/main.go`).
- When Clowder is not enabled, `publicPort` defaults to `8080` and `metricsPort` to `8081` (see `internal/config/config.go`).

## Database Migrations

- Migrations live in `migrations/` using `golang-migrate` numbered files (`000001_create_payload_tracker_tables.{up,down}.sql`).
- The API deployment runs `./pt-migration upgrade` as an `initContainer` with `inheritEnv: true` before the main container starts.
- Add new migration files with the next sequential number prefix. The `down` file should cleanly reverse the `up` file.
- The migration binary is built from `internal/migration/main.go` and uses `cobra` subcommands (`upgrade` / `downgrade`).

## Cron Job: Vacuum / Partition Management

The `vacuum` job in `deployments/clowdapp.yml` handles daily DB maintenance:

- Runs on schedule `${CLEANER_SCHEDULE}` (default: `00 17 * * *`), suspended by default (`CLEANER_SUSPEND: 'true'`).
- Uses a dedicated PostgreSQL image (`registry.redhat.io/rhel9/postgresql-16:9.7`), not the app image.
- Executes a ConfigMap-mounted script (`payload-tracker-go-db-cleaner-config` / `clean.sh`) that creates tomorrow's partition, drops the partition older than `RETENTION_DAYS` (default 7), deletes old payloads, and runs `VACUUM ANALYZE`.
- DB credentials come from the `payload-tracker-db-creds` secret with configurable key names (`DB_SECRET_HOSTNAME_KEY`, etc.).
- When modifying partition logic, update both the `clean.sh` in the ConfigMap section of `deployments/clowdapp.yml` and the `create_partition`/`drop_partition` functions in `migrations/000001_create_payload_tracker_tables.up.sql`.

## Tekton / Konflux Pipelines (`.tekton/`)

Four PipelineRun manifests cover two branches and two event types:

| File | Branch | Trigger |
|------|--------|---------|
| `payload-tracker-go-pull-request.yaml` | `master` | `pull_request` |
| `payload-tracker-go-push.yaml` | `master` | `push` |
| `payload-tracker-go-sc-pull-request.yaml` | `security-compliance` | `pull_request` |
| `payload-tracker-go-sc-push.yaml` | `security-compliance` | `push` |

- Pipeline reference: `docker-build-oci-ta` from `RedHatInsights/konflux-pipelines` (pinned to `v1.67.1`).
- PR images get `on-pr-{{revision}}` tag with 5-day expiry. Push images get `{{revision}}` tag with no expiry.
- The `master` branch builds to the `payload-tracker` Konflux application; `security-compliance` builds to `payload-tracker-sc`.
- Service accounts are branch-specific: `build-pipeline-payload-tracker-go` vs `build-pipeline-payload-tracker-sc`.
- `max-keep-runs` is set to `3` for all pipelines.
- When updating the pipeline version, update the `pipelinesascode.tekton.dev/pipeline` annotation URL in all four files.

## ClowdApp Template Parameters

- All resource limits and replica counts are parameterized -- avoid hardcoding values in the spec.
- Default replicas: 3 for both `api` and `consumer` (`API_REPLICAS`, `CONSUMER_REPLICAS`).
- Default resource requests/limits: `200m`/`500m` CPU, `256Mi`/`512Mi` memory (applies to API, consumer, and vacuum job independently).
- The `envName` parameter defaults to `payload-tracker-api`.
- Kafka topic `platform.payload-status` is declared with 3 replicas and 20 partitions.
- `optionalDependencies`: `storage-broker`, `ingress`, `rbac`.

## CI Scripts

- `pr_check.sh`: Bonfire-based ephemeral environment deployment with IQE smoke tests (`payload-tracker` plugin, `--single-replicas`).
- `build_deploy.sh`: Docker-based image build and push to `quay.io/cloudservices/payload-tracker-go`. Tags `qa` for non-security-compliance branches; tags `sc-YYYYMMDD-<sha>` for the `security-compliance` branch.
- `.github/workflows/pr.yml`: Runs `make run-migration`, `make build-all`, and `go test ./...` against a PostgreSQL service container.

## Verification

```bash
# Confirm Containerfile builds all four binaries in a single RUN
grep -c '^RUN.*go build' build/Containerfile      # expect: 1

# Confirm no CMD in Containerfile (command set via ClowdApp)
grep -c '^CMD\|^ENTRYPOINT' build/Containerfile    # expect: 0

# Confirm runtime user is non-root
grep 'USER 1001' build/Containerfile               # expect: match

# Confirm health probe paths match code
grep '/health' deployments/clowdapp.yml            # API probes
grep '/live\|/ready' deployments/clowdapp.yml      # consumer probes

# Confirm initContainer runs migration
grep 'pt-migration' deployments/clowdapp.yml       # expect: ./pt-migration upgrade

# Confirm Tekton pipeline version is consistent across all files
grep 'konflux-pipelines/raw/' .tekton/*.yaml       # all should reference same version

# Confirm all template parameters have defaults
grep -c 'name:.*value:' deployments/clowdapp.yml   # cross-check parameter coverage

# Build all binaries locally
make build-all

# Run tests
go test ./...
```
