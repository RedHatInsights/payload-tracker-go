@AGENTS.md

# Claude Code Configuration

## Build and Test Commands

Before suggesting changes or creating PRs, run:

```bash
make lint        # Runs gofmt
make build-all   # Compiles all binaries
make test        # Runs full test suite (requires PostgreSQL)
```

## Local Development Setup

Tests require a running PostgreSQL database:

```bash
docker compose up payload-tracker-db  # Start PostgreSQL
make run-migration                     # Initialize schema
make test                              # Run tests
```

## Pre-commit Hooks

This repository does NOT use pre-commit hooks. The `make lint` target is not enforced in CI.

## Workflow Preferences

- **Always run `make lint` and `make test`** before suggesting a pull request
- Consult AGENTS.md and the relevant `docs/*-guidelines.md` files before making domain-specific changes
