.PHONY: build run test test-coverage docker-build docker-up docker-down swagger clean

# Build the application
build:
	go build -o bin/main ./cmd/http

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

# Run with docker-compose
docker-up:
	docker-compose up -d

# Stop docker-compose
docker-down:
	docker-compose down

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

