# Makefile for OpenTether Enterprise AI Agent

# Variables
BINARY_NAME=opentether
OUTPUT_DIR=output
BINARY_DARWIN=$(OUTPUT_DIR)/$(BINARY_NAME)-darwin-amd64
BINARY_LINUX=$(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64
BINARY_WINDOWS=$(OUTPUT_DIR)/$(BINARY_NAME)-windows-amd64.exe
BINARY_CURRENT=$(OUTPUT_DIR)/$(BINARY_NAME)-$(shell go env GOOS)-$(shell go env GOARCH)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
VERSION?=1.0.0
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Default target
.PHONY: all
all: build-all

# Build for current platform
.PHONY: build
build:
	mkdir -p $(OUTPUT_DIR)
	go build $(LDFLAGS) -o $(BINARY_CURRENT) .

# Build for all platforms (with embedded web UI)
.PHONY: build-all
build-all: build-darwin build-linux build-windows

# Build for Linux (production, static binary - requires MySQL/PostgreSQL)
.PHONY: build-linux
build-linux:
	mkdir -p $(OUTPUT_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY_LINUX) .

# Build for Linux with SQLite support (requires cgo)
.PHONY: build-linux-sqlite
build-linux-sqlite:
	mkdir -p $(OUTPUT_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_LINUX) .

# Build for Windows
.PHONY: build-windows
build-windows:
	mkdir -p $(OUTPUT_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_WINDOWS) .

# Build for macOS
.PHONY: build-darwin
build-darwin:
	mkdir -p $(OUTPUT_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_DARWIN) .

# Run the application
.PHONY: run
run:
	go run main.go

# Run tests
.PHONY: test
test:
	go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(OUTPUT_DIR)/$(BINARY_NAME)-*
	rm -f coverage.out coverage.html

# Install dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

# Lint the code
.PHONY: lint
lint:
	golangci-lint run

# Format code
.PHONY: fmt
fmt:
	go fmt ./...
	gofmt -s -w .

# Create data directory
.PHONY: init
init:
	mkdir -p data logs models/embedding
	@echo "Directory structure created"

# Run with specific config
.PHONY: run-dev
run-dev:
	CONFIG_YAML=config.yaml go run main.go

# Docker build
.PHONY: docker-build
docker-build:
	docker build -t opentether:latest .

# Docker run
.PHONY: docker-run
docker-run:
	docker run -p 8080:8080 -v $(PWD)/data:/app/data opentether:latest

# Help
.PHONY: help
help:
	@echo "OpenTether Makefile Commands"
	@echo "=============================="
	@echo "make all              - Build all platforms (default)"
	@echo "make build            - Build for current platform"
	@echo "make build-all        - Build for all platforms"
	@echo "make build-linux      - Build for Linux (static binary)"
	@echo "make build-linux-sqlite - Build for Linux with SQLite"
	@echo "make build-windows    - Build for Windows"
	@echo "make build-darwin     - Build for macOS"
	@echo "make run              - Run the application"
	@echo "make test             - Run tests"
	@echo "make test-coverage    - Run tests with coverage"
	@echo "make clean            - Clean build artifacts"
	@echo "make deps             - Install dependencies"
	@echo "make fmt              - Format code"
	@echo "make init             - Initialize directory structure"
	@echo "make docker-build     - Build Docker image"
	@echo "make docker-run       - Run Docker container"
