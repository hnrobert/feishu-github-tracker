.PHONY: build run test clean docker-build docker-up docker-down docker-logs

# Build the application
build:
	go build -o bin/feishu-github-tracker ./cmd/feishu-github-tracker

# Run the application locally
run:
	go run ./cmd/feishu-github-tracker

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Build Docker image
docker-build:
	docker-compose build

# Start Docker containers
docker-up:
	docker-compose up -d

# Stop Docker containers
docker-down:
	docker-compose down

# View Docker logs
docker-logs:
	docker-compose logs -f

# Restart Docker containers
docker-restart:
	docker-compose restart

# Lint code
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...
