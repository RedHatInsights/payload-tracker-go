- [Overview](#overview)
- [Architecture](#architecture)
- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [REST API Endpoints](#rest-api-endpoints)
- [Message Formats](#message-formats)
- [Development](#development)
    - [Prerequisites](#prerequisites)
    - [Launching the Service](#launching-the-service)
    - [Local Development with Payload Tracker UI](#local-development-with-payload-tracker-ui)
    - [Running Tests](#running-tests)
- [Documentation](#documentation)
# Payload Tracker

## Overview
The Payload Tracker is a centralized location for tracking payloads through the Platform. Finding the status (current, or past) of a payload is difficult as logs are spread amongst various services and locations. Furthermore, Prometheus is meant purely for an aggregation of metrics and not for individualized transactions or tracking.

The Payload Tracker aims to provide a mechanism to query for a `request_id,` `inventory_id,` or `system_uuid` (physical machine-id) and see the current, last or previous X statuses of this upload through the platform. In the future, hopefully it will allow for more robust filtering based off of `service,` `account,` and `status.`

The ultimate goal of this service is to say that the upload made it through X services and was successful, or that the upload made it through X services was a failure and why it was a failure.

## Architecture
Payload Tracker is a service that lives in `platform-<env>`. This service has its own database representative of the current payload status in the platform. There are REST API endpoints that give access to the payload status. This service listens to messages on the Kafka MQ topic `platform.payload-status.` There is now a front-end UI for this service located in the same `platform-<env>`. It is respectively titled "payload-tracker-frontend."

The service is built as two separate binaries from a shared codebase:
- **`pt-api`** -- the REST API server (Chi router on port 8080) that serves payload status queries
- **`pt-consumer`** -- the Kafka consumer that reads from `platform.payload-status` and writes to PostgreSQL

Both binaries share internal packages for configuration, database access, models, and logging.

## Tech Stack

- **Language**: Go 1.24
- **Database**: PostgreSQL (with table partitioning)
- **Message Broker**: Apache Kafka (via confluent-kafka-go)
- **ORM**: GORM
- **HTTP Router**: Chi v5 (with httprate for rate limiting)
- **Configuration**: Viper + Clowder (via app-common-go)
- **Logging**: Logrus (with CloudWatch support via AWS SDK)
- **Metrics**: Prometheus client
- **Testing**: Ginkgo v1 + Gomega
- **Migrations**: golang-migrate
- **Container**: Built with Dockerfile, deployed via ClowdApp/Tekton

## Project Structure

```
cmd/
  payload-tracker-api/        # pt-api binary entrypoint
  payload-tracker-consumer/   # pt-consumer binary entrypoint
internal/
  config/                     # Viper-based configuration
  db/                         # Database connection and setup
  endpoints/                  # HTTP handlers and Prometheus metrics
  kafka/                      # Kafka consumer logic
  logging/                    # Logrus logging setup
  migration/                  # Database migration runner
  models/                     # API models and DB (consumer) models
  queries/                    # Database query functions
  structs/                    # Shared struct definitions
  utils/                      # Utility functions
api/                          # OpenAPI spec (api.spec.yaml)
migrations/                   # SQL migration files
dashboards/                   # Grafana dashboard definitions
deployments/                  # ClowdApp manifest
```

## REST API Endpoints
Please see the Swagger Spec for API Endpoints. The API Swagger Spec is located in `api/api.spec.yaml`.


## Message Formats
Simply send a message on the 'platform.payload-status' for your given Kafka MQ Broker in the appropriate environment. Currently, the following fields are required:

    org_id
    service
    request_id
    status
    date

```
{ 	
    'service': 'The services name processing the payload',
    'source': 'This is indicative of a third party rule hit analysis. (not Insights Client)',
    'account': 'The RH associated account',
    'org_id': 'The RH associated org id',
    'request_id': 'The ID of the payload (This should be a UUID)',
    'inventory_id': 'The ID of the entity in terms of the inventory (This should be a UUID)',
    'system_id': 'The ID of the entity in terms of the actual system (This should be a UUID)',
    'status': 'received|processing|success|error|etc',
    'status_msg': 'Information relating to the above status, should more verbiage be needed (in the event of an error)',
    'date': 'Timestamp for the message relating to the status above. (This should be in RFC3339 UTC format: "2022-03-17T16:56:10Z")'
}
```
The following statuses are required:
```
'received' 
'success/error' # success OR error
```

## Development
#### Prerequisites
```
docker
docker-compose
Go >= 1.24
```

#### Launching the Service
Launch DB, Zookeeper and Kafka
```
$> docker compose up payload-tracker-db
$> docker compose up zookeeper
$> docker compose up kafka
```
Migrate and seed the DB
```
$> make run-migration
$> make run-seed
```
Compile the source code for API and Consumer into a go binary:
```
$> make build-all
```
Launch the application

The mock requestor implementation allows you to get payload URLS from the local
machine
```
$> REQUESTOR_IMPL=mock ./pt-api
$> ./pt-consumer
```
The API should now be available on TCP port 8080
```
$> curl http://localhost:8080/api/v1/
$> lubdub
```

#### Local Development with Payload Tracker UI
Follow steps to run Payload Tracker UI (Dev Setup)
https://github.com/RedHatInsights/payload-tracker-frontend#dev-setup
Compile the source code for the API and Consumer into go binary:
```
$> make build-all
```
Launch the application in DEV mode
```
$> ENVIRONMENT=DEV REQUESTOR_IMPL=mock ./pt-api
$> ./pt-consumer
```
The API should now be available on port 8080
```
$> curl http://localhost:8080/app/payload-tracker/api/v1/
$> lubdub
```

## Running Tests
Use `go tests` to test the application
```
$> go test ./...
```

The tests also use a PostgreSQL database to run some tests. When testing locally, a PostgreSQL server needs to be up and running. On github, this is handled by a github actions workflow: [here](https://github.com/RedHatInsights/payload-tracker-go/blob/master/.github/workflows/pr.yml).

## Documentation

- [AGENTS.md](AGENTS.md) -- AI agent guidance, coding conventions, and architectural patterns
- [docs/](docs/) -- Detailed guideline files for each domain:
  - [API Contracts](docs/api-contracts-guidelines.md) -- REST API routes, response formats, pagination
  - [Async and Messaging](docs/async-and-messaging-guidelines.md) -- Kafka consumer, message schema, offset management
  - [Code Organization](docs/code-organization-guidelines.md) -- Project layout, package responsibilities
  - [Configuration](docs/configuration-guidelines.md) -- Viper config, Clowder integration, environment variables
  - [Database](docs/database-guidelines.md) -- PostgreSQL schema, GORM usage, migrations
  - [Data Validation](docs/data-validation-guidelines.md) -- Input validation and sanitization
  - [Dependency Management](docs/dependency-management-guidelines.md) -- Go modules, automated updates
  - [Deployment](docs/deployment-guidelines.md) -- Container builds, ClowdApp, Tekton pipelines
  - [Error Handling](docs/error-handling-guidelines.md) -- Error patterns and logging
  - [Integration](docs/integration-guidelines.md) -- External service clients and mocking
  - [Logging and Observability](docs/logging-and-observability-guidelines.md) -- Logrus, CloudWatch, Prometheus
  - [Performance](docs/performance-guidelines.md) -- Rate limiting, caching, query optimization
  - [Security](docs/security-guidelines.md) -- Authentication, secrets, container security
  - [Testing](docs/testing-guidelines.md) -- Ginkgo framework, test patterns, DB setup
