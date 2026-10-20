@AGENTS.md

# Claude Code Quick Reference

## Build Commands

```bash
make build-all                    # Build pt-api, pt-consumer, pt-migration, pt-seeder
make test                         # All tests (serialized: go test -p 1 -v ./...)
make lint                         # Format with gofmt
go test -v ./internal/endpoints/  # Fast unit tests only (no DB needed)
```

## Local Dev Setup

Requires a running PostgreSQL (default: crc/crc@localhost:5432/crc).

```bash
make run-migration                # Apply schema + partitions
make run-seed                     # Populate test data
REQUESTOR_IMPL=mock ./pt-api     # Run API with mock storage-broker
./pt-consumer                     # Run Kafka consumer
```

## CI Checks (GitHub Actions)

PR checks run `make build-all` then `go test ./...` against a Postgres service container. Ensure both pass locally before pushing.

## Workflow Notes

- No pre-commit hooks or linters beyond `gofmt` -- run `make lint` before committing.
- Tests are serialized (`-p 1`) to avoid DB conflicts; do not add `-race` or parallel flags.
- The two-binary architecture means a single container image contains both `pt-api` and `pt-consumer` -- the ClowdApp manifest selects which binary to run.
