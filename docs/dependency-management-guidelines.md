# Dependency Management Guidelines

Rules for managing Go module dependencies and Konflux pipeline references in payload-tracker-go.

## Automated Dependency Updates

- Dependencies are managed by **MintMaker** (Konflux's built-in Renovate-based bot), which runs as `red-hat-konflux[bot]`. It replaced Dependabot, which was used historically.
- MintMaker creates branches named `konflux/mintmaker/master/<package-name>-<major>.x` (e.g., `konflux/mintmaker/master/github.com-spf13-pflag-1.x`).
- MintMaker uses two commit message styles depending on the update type:
  - Go modules: `Update module <path> to <version>` or conventional commits like `chore(deps): update module <path> to <version>` / `fix(deps): update module <path> to <version>`.
  - Konflux pipeline refs: `update dependency github.com/redhatinsights/konflux-pipelines to <version>` or `Update dependency github.com/RedHatInsights/konflux-pipelines to <version>` (case varies historically).
  - Digest-only updates (no semver): `Update golang.org/x/exp digest to <short-hash>`.
- MintMaker PRs should be reviewed for CI passing (PR Check workflow + Tekton pipeline) before merging.

## Go Module Version Policy

- The `go.mod` file uses Go 1.24 (currently `go 1.24.13`). Patch version bumps within the same minor are done via RHCLOUD Jira tickets (e.g., `[RHCLOUD-45708] Bump Go version`).
- When bumping the Go directive in `go.mod`, also verify the `go-version` field in `.github/workflows/pr.yml` uses a compatible version (currently `'1.24'`, which is the minor-only form).
- The builder image in `build/Containerfile` uses `registry.access.redhat.com/ubi9/go-toolset` without a version tag. It tracks the latest Go available in UBI9 and does not need manual version pinning.

## Vendoring

- This repo does **not** vendor dependencies. The `vendor/` directory is commented out in `.gitignore`. Dependencies are fetched at build time via `go get -d ./...` in the Containerfile.
- Do not introduce vendoring without team discussion.

## Konflux Pipeline References

- All four Tekton PipelineRun files in `.tekton/` reference the same pipeline version via a raw GitHub URL:
  ```
  pipelinesascode.tekton.dev/pipeline: https://github.com/RedHatInsights/konflux-pipelines/raw/v1.64.0/pipelines/docker-build-oci-ta.yaml
  ```
- When updating the Konflux pipeline version, update the version tag in **all four** `.tekton/*.yaml` files simultaneously:
  - `payload-tracker-go-push.yaml`
  - `payload-tracker-go-pull-request.yaml`
  - `payload-tracker-go-sc-push.yaml`
  - `payload-tracker-go-sc-pull-request.yaml`
- The version tag format is `v<major>.<minor>.0` (e.g., `v1.64.0`). Only the version number in the URL changes; the pipeline name (`docker-build-oci-ta.yaml`) stays the same.

## Direct vs Indirect Dependencies

- Direct dependencies are listed in the first `require` block in `go.mod` (without `// indirect` comment). These are the project's actual imports.
- Indirect dependencies are in the second `require` block, all marked `// indirect`. These are transitive.
- When updating a direct dependency, run `go mod tidy` afterward to ensure `go.sum` stays consistent. The commit history shows a dedicated fix for this (`Run go mod tidy as well`, commit `cbfdcf2`).

## Key Direct Dependencies and Constraints

| Dependency | Purpose | Notes |
|---|---|---|
| `confluentinc/confluent-kafka-go/v2` | Kafka consumer | Uses v2; requires C library (librdkafka) at build time |
| `go-chi/chi/v5` | HTTP router | v5 API |
| `golang-migrate/migrate/v4` | DB migrations | Drives `pt-migration` binary; migrations live in `migrations/` |
| `gorm.io/gorm` + `gorm.io/driver/postgres` | ORM / DB | PostgreSQL via pgx driver |
| `redhatinsights/app-common-go` | Clowder config | Reads ClowdApp environment config |
| `redhatinsights/platform-go-middlewares/v2` | Platform middleware | Uses `v2.0.0-beta.2` (pre-release pin) |
| `onsi/ginkgo` + `onsi/gomega` | Test framework | Used for BDD-style tests |

- `platform-go-middlewares/v2` is pinned to a beta pre-release (`v2.0.0-beta.2`). Do not blindly update this; verify API compatibility if a new version is available.

## Bulk Dependency Updates

- The team occasionally performs manual bulk updates in a single commit (e.g., commit `00ad55c` titled `Update deps 1`). These update `go.mod`, `go.sum`, and sometimes `.github/workflows/pr.yml` together.
- When performing a bulk update:
  1. Update `go.mod` with new versions.
  2. Run `go mod tidy` to clean up `go.sum`.
  3. Run `make build-all` to verify compilation.
  4. Run `go test ./...` to verify tests pass.

## Security Scanning

- The repo uses the ConsoleDot Platform Security Scan workflow (`.github/workflows/security-workflow-template.yml`) which runs Anchore Grype (vulnerability scan) and Syft (SBOM generation) on pushes and PRs to `main`, `master`, and `security-compliance` branches.
- Security-critical dependency updates (e.g., `golang.org/x/crypto`, `golang.org/x/net`) have historically been fast-tracked, sometimes via Dependabot security advisories.

## Container Base Images

- Builder stage: `registry.access.redhat.com/ubi9/go-toolset` (untagged, latest).
- Runtime stage: `registry.access.redhat.com/ubi9/ubi-minimal:latest`.
- DB cleaner job in `deployments/clowdapp.yml`: `registry.redhat.io/rhel9/postgresql-16:9.7` (pinned to specific tag).
- Kafka image in `compose.yml`: `confluentinc/cp-kafka:7.9.2` (pinned for local dev).
- Prefer pinned tags for production images. The builder image being untagged is an existing pattern but not ideal for reproducibility.

## Commit Message Conventions for Dependency Updates

- Konflux pipeline updates: `update dependency github.com/redhatinsights/konflux-pipelines to v<version>` or `Update dependency github.com/RedHatInsights/konflux-pipelines to v<version>` (case varies)
- Go module updates (MintMaker): `Update module <path> to <version>`
- Go version bumps: `[RHCLOUD-<ticket>] Bump Go version` (linked to Jira)
- Manual bulk updates: descriptive title like `Update deps 1`

## Files to Modify Per Update Type

| Update type | Files changed |
|---|---|
| Go module dependency | `go.mod`, `go.sum` |
| Konflux pipeline version | All 4 `.tekton/*.yaml` files |
| Go language version | `go.mod`, possibly `.github/workflows/pr.yml` |
| Container base image | `build/Containerfile` or `deployments/clowdapp.yml` |
