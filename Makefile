.PHONY: build run test clean docker-build docker-run deps

# Build the agent and mcp server
build:
	go build -o helix-agent ./cmd/agent
	go build -o helix-mcp ./cmd/mcp

# Run with hot reload (requires air)
run:
	air

# Run tests
test:
	go test ./... -race -cover

# Run tests with verbose output
test-verbose:
	go test ./... -v

# Run a specific test
test-one:
	go test -v ./... -run TestName

# Clean build artifacts
clean:
	rm -f helix-agent
	rm -f helix-mcp
	rm -f coverage.out

# Download dependencies
deps:
	go mod download
	go mod tidy

# Verify dependencies
verify:
	go mod verify

# Lint the code
lint:
	golangci-lint run

# Build Docker image
docker-build:
	docker build -t helixops:latest .

# Run with Docker Compose
docker-run:
	docker-compose up -d

# Stop Docker Compose
docker-stop:
	docker-compose down

# Build and run for development
dev: deps build run

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build the agent binary"
	@echo "  run            - Run with hot reload (requires air)"
	@echo "  test           - Run all tests with race detection"
	@echo "  test-verbose   - Run tests with verbose output"
	@echo "  test-one       - Run a specific test"
	@echo "  clean          - Remove build artifacts"
	@echo "  deps           - Download dependencies"
	@echo "  verify         - Verify dependencies"
	@echo "  lint           - Run golangci-lint"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run with Docker Compose"
	@echo "  docker-stop    - Stop Docker Compose"
	@echo "  dev            - Install deps, build, and run"
	@echo "  help           - Show this help"
