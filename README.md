# Payload Tracker

A Go service that tracks Red Hat Insights payloads through the Platform, providing centralized status tracking via REST API and Kafka message consumption.

**Built with**: Go 1.26.3, PostgreSQL, Kafka, Chi router, GORM, deployed on OpenShift via Clowder

## Overview

The Payload Tracker provides a centralized location for tracking payloads through the Platform. Finding the status (current or past) of a payload is difficult as logs are spread amongst various services and locations. This service allows querying by `request_id`, `inventory_id`, or `system_uuid` to see current, last, or previous statuses of an upload through the platform.

### Architecture

Payload Tracker consists of two binaries built from a single codebase:
- **`pt-api`** — REST API server providing payload status queries
- **`pt-consumer`** — Kafka consumer listening to `platform.payload-status` topic

Both share a PostgreSQL database with daily-partitioned status tables. The service runs on OpenShift via Clowder (ClowdApp) and uses Konflux/Tekton for CI/CD.

### Project Structure

- `cmd/payload-tracker-api/` and `cmd/payload-tracker-consumer/` - Main entry points for the two binaries
- `internal/` - Application code (endpoints, kafka consumer, models, config, queries)
- `api/` - OpenAPI specification (`api.spec.yaml`)
- `build/` - Container build files
- `migrations/` - Database schema migrations
- `deployments/` - Clowder deployment manifests
- `.tekton/` - Konflux CI/CD pipeline definitions
- `dashboards/` - Grafana dashboard definitions

## Documentation

- [AGENTS.md](AGENTS.md) - Architectural constraints, build/test instructions, cross-cutting conventions
- [docs/](docs/) - Domain-specific guidelines:
  - API contracts, async/messaging, code organization
  - Configuration, data validation, database patterns
  - Dependency management, deployment, error handling
  - Go web frameworks (Chi), integration patterns
  - Logging/observability, performance, security, testing

## REST API Endpoints

See the OpenAPI specification in `api/api.spec.yaml` for complete API documentation.

Key endpoints:
- `GET /api/v1/payloads` - List all payloads with filtering and pagination
- `GET /api/v1/payloads/{request_id}` - Get payload status by request ID
- `GET /api/v1/statuses` - List all statuses with filtering
- `GET /api/v1/payloads/{request_id}/archiveLink` - Get archive download link (requires RBAC role)

## Message Formats

Send messages to the `platform.payload-status` Kafka topic. Required fields:

```json
{
    "service": "The service name processing the payload",
    "org_id": "The RH associated org id",
    "request_id": "The ID of the payload (UUID format)",
    "status": "received|processing|success|error",
    "date": "Timestamp in RFC3339 UTC format (e.g., '2022-03-17T16:56:10Z')"
}
```

Optional fields: `source`, `account`, `inventory_id`, `system_id`, `status_msg`

Required statuses: `received`, `processing`, `success`

## Development

### Prerequisites

- Go >= 1.26.3
- Docker and Docker Compose
- PostgreSQL (via Docker or local)
- Kafka (via Docker or local)

### Launching the Service

```bash
# Start local infrastructure (Postgres, Kafka, Zookeeper)
docker compose up payload-tracker-db zookeeper kafka -d

# Run database migrations and seed data
make run-migration
make run-seed

# Build all binaries
make build-all

# Run the API and consumer (local dev mode)
REQUESTOR_IMPL=mock ./pt-api
./pt-consumer
```

### Local Development with Payload Tracker UI

The frontend UI is available at https://github.com/RedHatInsights/payload-tracker-frontend

Follow the frontend repository's README for setup instructions.

### Running Tests

```bash
# All tests (requires running PostgreSQL with migrations applied)
make test                         # runs: go test -p 1 -v ./...

# Unit tests only (no database required)
go test -v ./internal/endpoints/

# Lint
make lint                         # runs: gofmt -l . && gofmt -s -w .
```

Tests use Ginkgo v1 and Gomega. The `-p 1` flag serializes test packages to prevent concurrent DB access conflicts. CI runs against a PostgreSQL service container with credentials `crc`/`crc`/`crc`.

## Contributing

See [AGENTS.md](AGENTS.md) for architectural constraints and development conventions.

## License

This project is available as open source under the terms of the Apache License 2.0.
