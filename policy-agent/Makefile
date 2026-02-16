.PHONY: build test clean install lint

# Build variables
BINARY_NAME=policy-agent
WEBHOOK_BINARY=policy-webhook
VERSION?=0.1.0
BUILD_DIR=./bin

# Go build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/policy-agent
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(WEBHOOK_BINARY) ./cmd/webhook-server
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	@echo "Tests complete"

test-coverage:
	@echo "Generating coverage report..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint:
	@echo "Running linters..."
	golangci-lint run ./...

install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) ./cmd/policy-agent
	go install $(LDFLAGS) ./cmd/webhook-server

clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	go clean

deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

run:
	go run ./cmd/policy-agent/main.go

run-webhook:
	go run ./cmd/webhook-server/main.go

docker-build:
	docker build -t policy-agent:$(VERSION) -f deploy/docker/Dockerfile.cli .
	docker build -t policy-webhook:$(VERSION) -f deploy/docker/Dockerfile.webhook .

help:
	@echo "Available targets:"
	@echo "  build         - Build binaries"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Generate coverage report"
	@echo "  lint          - Run linters"
	@echo "  install       - Install binaries"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Download dependencies"
	@echo "  run           - Run CLI locally"
	@echo "  run-webhook   - Run webhook server locally"
	@echo "  docker-build  - Build Docker images"
