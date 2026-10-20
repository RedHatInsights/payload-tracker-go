# Dependency Management Guidelines

## Go Module Version

- Pin the Go version in `go.mod` to a specific patch release (e.g., `go 1.26.3`).
- When bumping Go, update all three locations: `go.mod`, `build/Containerfile` (`GOTOOLCHAIN` ARG), and `.github/workflows/pr.yml` (`go-version`).
- The `build/Containerfile` uses `registry.access.redhat.com/ubi9/go-toolset:latest` as the builder image. Because UBI9 go-toolset may ship an older Go, set `ARG GOTOOLCHAIN=go<version>+auto` in the Containerfile to force the correct toolchain.

## Automated Dependency Updates

- MintMaker (red-hat-konflux bot) opens PRs for Go module updates on branches named `konflux/mintmaker/master/<module-path>-<major>.x`. These PRs are auto-merged after CI passes.
- MintMaker also updates Konflux pipeline references in `.tekton/*.yaml` files. Those PRs follow the commit message format: `chore(deps): update dependency github.com/redhatinsights/konflux-pipelines to v<version>`.
- Dependabot previously handled security-sensitive transitive dependency bumps (e.g., `golang.org/x/crypto`, `golang.org/x/net`). Dependabot PRs use the commit format `Bump <module> from <old> to <new>`.
- Do not create a `renovate.json` or `.renovaterc` -- this repo relies on MintMaker and does not use Renovate.

## Commit Message Conventions for Dependency Changes

- Automated indirect/transitive updates: `chore(deps): update module <path> to <version>`
- Automated direct dependency fixes: `fix(deps): update module <path> to <version>`
- Manual Go version bumps: `chore: bump Go version from <old> to <new>`
- Manual bulk updates: descriptive message referencing the Jira ticket, e.g., `[RHCLOUD-45708] Bump Go version + update quay image path`

## No Vendor Directory

- This repo uses Go module proxy mode (no `vendor/` directory). Do not run `go mod vendor`.

## Direct Dependencies and Their Roles

| Module | Role |
|---|---|
| `confluentinc/confluent-kafka-go/v2` | Kafka consumer (CGo-based, requires librdkafka) |
| `gorm.io/gorm` + `gorm.io/driver/postgres` | ORM and PostgreSQL driver for application queries |
| `lib/pq` | PostgreSQL driver for `golang-migrate` (migration tool only) |
| `golang-migrate/migrate/v4` | Database schema migrations (`internal/migration/`) |
| `prometheus/client_golang` | Prometheus metrics via `promauto` (`internal/endpoints/metrics.go`) |
| `redhatinsights/app-common-go` | Clowder configuration integration |
| `redhatinsights/platform-go-middlewares/v2` | CloudWatch logging middleware (pinned at `v2.0.0-beta.2`) |
| `go-chi/chi/v5` + `go-chi/httprate` | HTTP router and rate limiting |
| `onsi/ginkgo` + `onsi/gomega` | BDD test framework (used across all test suites) |
| `spf13/viper` + `spf13/cobra` | Configuration management and CLI |
| `aws/aws-sdk-go` | AWS SDK for S3/CloudWatch operations |

## Pinned and Pre-Release Dependencies

- `platform-go-middlewares/v2` is pinned at `v2.0.0-beta.2`. Do not upgrade this without verifying the CloudWatch logging integration in `internal/logging/logging.go`.
- `onsi/ginkgo` is on v1 (`v1.16.5`), not v2. Test suites import `github.com/onsi/ginkgo`, not `github.com/onsi/ginkgo/v2`. Upgrading to ginkgo v2 requires rewriting test suite bootstrap files.

## Kafka Client Constraints

- The Kafka client (`confluent-kafka-go/v2`) uses CGo and links against librdkafka. The UBI9 builder image provides the required C toolchain.
- When upgrading the Kafka client, verify that the `kafka.ConfigMap` keys in `internal/kafka/kafka.go` remain compatible with the new librdkafka version.
- Local development uses `confluentinc/cp-kafka:7.9.2` in `compose.yml`. Keep this image version aligned with the Kafka client compatibility matrix.

## PostgreSQL Version

- The ClowdApp spec in `deployments/clowdapp.yml` declares `database.version: 16`. Dependency changes involving the Postgres driver (`pgx/v5`, `lib/pq`) should remain compatible with PostgreSQL 16.

## Security Scanning

- The `RedHatInsights/platform-security-gh-workflow` runs Grype vulnerability scanning and Syft SBOM generation on every PR and push to `master` and `security-compliance` branches.
- Prioritize dependency bumps that address CVEs flagged by this scanner. Commit messages for security fixes should reference the advisory (e.g., `fix GHSA-xxxx-xxxx-xxxx`).

## Adding a New Dependency

- Add direct dependencies to the `require` block in `go.mod` (not the `// indirect` block).
- Run `go mod tidy` after any dependency change to clean up `go.sum`.
- Prefer well-maintained modules already in the Red Hat ecosystem (`redhatinsights/*`) when platform alternatives exist.

## Removing a Dependency

- Remove the import from all `.go` files first, then run `go mod tidy` to prune `go.mod` and `go.sum`.

## Tekton Pipeline References

- Tekton pipeline definitions in `.tekton/*.yaml` reference a pinned version of `konflux-pipelines` (e.g., `v1.67.1`). MintMaker updates these automatically. When manually updating, change all four files: `payload-tracker-go-pull-request.yaml`, `payload-tracker-go-push.yaml`, `payload-tracker-go-sc-pull-request.yaml`, `payload-tracker-go-sc-push.yaml`.

## Verification

```bash
# Verify go.mod is tidy (should produce no output)
go mod tidy && git diff --exit-code go.mod go.sum

# Verify all dependencies resolve
go mod verify

# Build all binaries to catch import or linking errors
make build-all

# Run the full test suite
go test -p 1 -v ./...

# Check for known vulnerabilities (if govulncheck is installed)
govulncheck ./...
```
