# Crypto Portfolio Tracker


## Features

## Architecture

This project follows **Hexagonal Architecture (Ports & Adapters)** principles:

- **Domain Layer** (`/internal/domain`): Core business logic and entities
- **Application Layer** (`/internal/application`): Use cases and orchestration
- **Adapters Layer** (`/internal/adapters`): Infrastructure implementations
- **Ports** (`/internal/ports`): Interface definitions

## Getting Started

### Prerequisites

- Go 1.25 or later
- Docker and Docker Compose (optional)

### Installation

1. Clone the repository
2. Install dependencies:
```bash
make deps
# or
go mod download
```

### Running the Application

#### Local Development

```bash
# Run directly
make run
# or
go run ./cmd/http

# With development mode (colored logs)
go run ./cmd/http -dev
```

#### Docker

```bash
# Build and run with docker-compose
make docker-up

# Or build manually
make docker-build
docker run -p 8080:8080 crypto-portfolio-tracker:latest
```

### Configuration

## API Documentation

### Swagger UI

Once the server is running, access Swagger documentation at:
- http://localhost:8080/swagger/index.html

### Generate Swagger Docs

```bash
# Install swag CLI first
go install github.com/swaggo/swag/cmd/swag@latest

# Generate docs
make swagger
```

### Endpoints

## Testing

### Run Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage
```

### Test Coverage

The test suite covers:

- **Interface-based design**: Portfolio service tests with mock repositories
- **Cache-aside pattern**: Price service tests for cache hit/miss scenarios
- **Fallback patterns**: Price service tests for primary/fallback provider failures
- **Method signature detection**: Transaction domain tests for signature extraction
- **Transaction categorization**: Transaction domain tests for type classification

## Docker

### Build

```bash
make docker-build
```

### Run with Docker Compose

```bash
make docker-up
```

### Stop

```bash
make docker-down
```

## Development

### Project Structure

```
.
├── cmd/
│   └── http/              # Application entry point
├── config/                # Configuration
├── internal/
│   ├── adapters/         # Infrastructure adapters
│   │   ├── http/         # HTTP server/client
│   │   ├── portfolio/    # Portfolio repository
│   │   ├── price/        # Price providers
│   │   └── logger/        # Structured logging
│   ├── application/       # Application services
│   │   ├── portfolio/     # Portfolio service
│   │   ├── price/         # Price service (cache-aside)
│   │   └── transaction/   # Transaction service
│   ├── domain/            # Domain entities
│   │   ├── portfolio/     # Portfolio domain
│   │   ├── price/         # Price domain
│   │   └── transaction/   # Transaction domain
│   └── ports/             # Interface definitions
│       └── http/           # HTTP ports
└── static/                # Static files (coins.json)
```

## Key Features Implementation

### Structured Logging

Uses `zap` for structured logging with JSON output in production and colored output in development mode.

### Cache-Aside Pattern

Price service implements cache-aside pattern:
1. Check cache first
2. On cache miss, fetch from primary provider
3. If primary fails, fallback to mock provider
4. Cache successful results

### Graceful Shutdown

Server handles SIGTERM and SIGINT signals, allowing in-flight requests to complete before shutdown (10s timeout).

### Health Checks

Health endpoint returns service status, timestamp, and version information.

## License

Apache 2.0

