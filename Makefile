.PHONY: build run test test-coverage docker-build docker-up docker-down swagger clean migrate-up migrate-down migrate-reset migrate-status migrate-up-docker migrate-down-docker

# Build the application (CGO_ENABLED=1 required for sqlite3)
build:
	CGO_ENABLED=1 go build -o bin/main ./cmd/http

# Run the application
run:
	go run ./cmd/http

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Generate Swagger documentation
swagger:
	swag init -g cmd/http/main.go -o docs

# Build Docker image
docker-build:
	docker build -t crypto-portfolio-tracker:latest .

# Build Docker image for debugging
docker-build-debug:
	docker build -f Dockerfile.debug -t crypto-portfolio-tracker:debug .

# Run with docker-compose
docker-up:
	docker-compose up -d

# Run debug service with docker-compose
docker-up-debug:
	docker-compose up -d app-debug

# Stop docker-compose
docker-down:
	docker-compose down

# Stop debug service
docker-down-debug:
	docker-compose stop app-debug
	docker-compose rm -f app-debug

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf docs/
	rm -f coverage.out coverage.html

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run linter
lint:
	golangci-lint run

# Database migration variables
DB_PATH ?= ./data/portfolio.db
MIGRATIONS_DIR = migrations
MIGRATE_SCRIPT = scripts/migrate.sh

# Run all up migrations
migrate-up:
	@DB_PATH=$(DB_PATH) MIGRATIONS_DIR=$(MIGRATIONS_DIR) $(MIGRATE_SCRIPT) up

# Run all down migrations (rollback)
migrate-down:
	@DB_PATH=$(DB_PATH) MIGRATIONS_DIR=$(MIGRATIONS_DIR) $(MIGRATE_SCRIPT) down

# Reset database (drop all and reapply migrations)
migrate-reset:
	@DB_PATH=$(DB_PATH) MIGRATIONS_DIR=$(MIGRATIONS_DIR) $(MIGRATE_SCRIPT) reset

# Show migration status
migrate-status:
	@DB_PATH=$(DB_PATH) MIGRATIONS_DIR=$(MIGRATIONS_DIR) $(MIGRATE_SCRIPT) status

