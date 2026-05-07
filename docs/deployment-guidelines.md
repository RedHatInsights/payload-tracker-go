# Deployment Guidelines

Rules for deploying and modifying deployment configurations in payload-tracker-go. This service runs on the Red Hat Consoledot platform using Clowder.

## Architecture: Single Image, Multiple Binaries

This repo builds one container image containing four binaries: `pt-api`, `pt-consumer`, `pt-migration`, and `pt-seeder`. The deployment manifests select which binary to run via `command`. The `build/Containerfile` intentionally has no `CMD` directive.

- `pt-api` -- HTTP API server (cmd/payload-tracker-api/main.go)
- `pt-consumer` -- Kafka consumer (cmd/payload-tracker-consumer/main.go)
- `pt-migration` -- Database schema migration (internal/migration/main.go)
- `pt-seeder` -- Test data seeder (tools/db-seeder/main.go)

When adding a new binary, add it to the `Containerfile` build stage, the `COPY --from=builder` stage, and the `Makefile` `build-all` target.

## Container Image Build

The container image is defined in `build/Containerfile`. A symlink `Dockerfile -> build/Containerfile` exists at the repo root because Tekton pipelines reference `Dockerfile`.

- Base builder image: `registry.access.redhat.com/ubi9/go-toolset`
- Runtime image: `registry.access.redhat.com/ubi9/ubi-minimal:latest`
- The runtime stage runs as `USER 1001` (non-root). Preserve this.
- The `migrations/` directory is copied into the runtime image at `/migrations/` for use by `pt-migration`.

## ClowdApp Manifest (deployments/clowdapp.yml)

This is the primary deployment manifest. It is an OpenShift Template wrapping a `ClowdApp` (apiVersion: `cloud.redhat.com/v1alpha1`).

### Deployment Structure

The ClowdApp defines two deployments and one cron job:

1. **api** -- runs `./pt-api`, has `webServices.public.enabled: True`, uses an initContainer to run `./pt-migration upgrade`
2. **consumer** -- runs `./pt-consumer`, no web services, no initContainer
3. **vacuum** (CronJob) -- runs a shell script from ConfigMap `payload-tracker-go-db-cleaner-config`, uses `registry.redhat.io/rhel9/postgresql-16:9.7` image (not the app image)

### Init Container Pattern

The API deployment runs database migrations via an initContainer before the main container starts. The initContainer uses the same app image (`${IMAGE}:${IMAGE_TAG}`) with `inheritEnv: true` and runs `./pt-migration upgrade`.

When modifying migrations, keep them in `migrations/` as sequential numbered files (e.g., `000001_*.up.sql`, `000001_*.down.sql`). The migration tool is `golang-migrate/migrate/v4`.

### Health Probes

The API and consumer use different health probe configurations:

| Component | Liveness Path | Readiness Path | Port |
|-----------|--------------|----------------|------|
| api       | `/health`    | `/health`      | 8000 |
| consumer  | `/live`      | `/ready`       | 9000 |

The API health check (`internal/endpoints/health.go`) verifies the database connection via `db.Ping()`. The consumer maps both `/live` and `/ready` to the same `HealthCheckHandler`.

Port 8000 and 9000 are the Clowder-assigned ports (not the app-level ports 8080/8081 used in local dev). Clowder remaps these automatically.

### Resource Limits

All three components (api, consumer, vacuum) follow the same parameterized pattern with separate template parameters per component. Default values for all:

- Requests: 200m CPU, 256Mi memory
- Limits: 500m CPU, 512Mi memory

When adding a new deployment, create component-specific parameters (e.g., `NEWCOMPONENT_CPU_LIMIT`) rather than reusing existing ones.

### Replica Counts

- API and consumer both default to 3 replicas (`${{API_REPLICAS}}`, `${{CONSUMER_REPLICAS}}`)
- Use `${{ }}` (double-brace) syntax for integer parameters in OpenShift Templates, `${ }` (single-brace) for strings

### Kafka Configuration

The ClowdApp declares one Kafka topic: `platform.payload-status` with 3 replicas and 20 partitions. The consumer reads from this topic using `confluent-kafka-go/v2`.

### Database

- ClowdApp database name: `payloadtracker`, version: `16` (PostgreSQL 16)
- DB credentials come from the Clowder-managed secret `payload-tracker-db-creds`
- The vacuum job uses configurable secret key names via parameters (`DB_SECRET_HOSTNAME_KEY`, etc.) with defaults like `db.name`, `db.host`

### Environment Variables

The ClowdApp env vars use `${PARAM_NAME}` referencing template parameters. Sensitive values in the vacuum job use `valueFrom.secretKeyRef`. The API deployment passes Kibana and storage-broker config; the consumer passes only `LOG_LEVEL` and `DEBUG_LOG_STATUS_JSON`.

## Legacy Manifest (deployments/deploy.yml)

`deployments/deploy.yml` is a legacy Kubernetes Deployment manifest (not ClowdApp). It references the old image path `quay.io/cloudservices/payload-tracker-go` and uses port 8080 with path `/v1/health`. Do not use this as a reference for new work; use `deployments/clowdapp.yml`.

## Tekton CI/CD Pipelines (.tekton/)

Four PipelineRun definitions exist, all using `docker-build-oci-ta` from `konflux-pipelines`:

| File | Trigger | Branch | Image Tag |
|------|---------|--------|-----------|
| `payload-tracker-go-push.yaml` | push | master | `{{revision}}` |
| `payload-tracker-go-pull-request.yaml` | pull_request | master | `on-pr-{{revision}}` |
| `payload-tracker-go-sc-push.yaml` | push | security-compliance | `{{revision}}` |
| `payload-tracker-go-sc-pull-request.yaml` | pull_request | security-compliance | `on-pr-{{revision}}` |

Conventions:
- Pipeline version is pinned to a specific tag (currently `v1.64.0`) in all four files. Update all four files together when bumping.
- PR images include `on-pr-` prefix in the tag and set `image-expires-after: 5d`
- Push images to `quay.io/redhat-user-workloads/hcc-integrations-tenant/payload-tracker/payload-tracker-go`
- SC (security-compliance) images go to a separate path: `.../payload-tracker-go-sc/payload-tracker-sc`
- All pipelines reference `dockerfile: Dockerfile` (the root symlink)
- `max-keep-runs: "3"` is set on all pipelines
- Namespace: `hcc-integrations-tenant`

## GitHub Actions (.github/workflows/)

- `pr.yml` -- PR checks: runs on `ubuntu-22.04`, sets up PostgreSQL service container (user/pass/db: `crc`), runs `make run-migration`, `make build-all`, then `go test ./...` with Go 1.24
- `security-workflow-template.yml` -- Runs the RedHatInsights platform security scan (Grype + Syft) on pushes/PRs to master and security-compliance branches

## Clowder Configuration Awareness

The app uses `redhatinsights/app-common-go` to detect Clowder. When `clowder.IsClowderEnabled()` is true (deployed environment), config values like ports, DB credentials, Kafka brokers, and CloudWatch settings are pulled from Clowder's injected config. When false (local dev), defaults point to localhost services. See `internal/config/config.go`.

Key ports:
- `publicPort` -- 8080 locally, Clowder-assigned in deployment
- `metricsPort` -- 8081 locally, Clowder-assigned in deployment

## Local Development (compose.yml)

`compose.yml` runs the API and consumer as separate containers from the same image, differentiated by `command` (`/pt-api` vs `/pt-consumer`). PostgreSQL uses credentials `crc/crc/crc`. Kafka runs via `confluentinc/cp-kafka:7.9.2` on port 29092.

## Grafana Dashboard

`dashboards/grafana-dashboard-insights-payload-tracker-general.configmap.yaml` contains the Grafana dashboard as a ConfigMap. Update this file when adding new Prometheus metrics exposed via the `/metrics` endpoint (served by `promhttp.Handler()`).
