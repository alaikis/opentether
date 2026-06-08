# Makefile for OpenTether Enterprise AI Agent

# Variables
BINARY_NAME=wisehoof
BINARY_DARWIN=$(BINARY_NAME)-darwin-amd64
BINARY_LINUX=$(BINARY_NAME)-linux-amd64
BINARY_WINDOWS=$(BINARY_NAME).exe
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Default target
.PHONY: all
all: build

# Build for current platform
.PHONY: build
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

# Build for all platforms
.PHONY: build-all
build-all:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_DARWIN) .
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_LINUX) .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_WINDOWS) .

# Build for Linux (production)
.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_LINUX) .

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
	rm -f $(BINARY_NAME) $(BINARY_DARWIN) $(BINARY_LINUX) $(BINARY_WINDOWS)
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
	docker build -t wisehoof:latest .

# Docker run
.PHONY: docker-run
docker-run:
	docker run -p 8080:8080 -v $(PWD)/data:/app/data wisehoof:latest

# Help
.PHONY: help
help:
	@echo "OpenTether Makefile Commands"
	@echo "=========================="
	@echo "make all         - Build the application"
	@echo "make build       - Build for current platform"
	@echo "make build-all   - Build for all platforms"
	@echo "make build-linux - Build for Linux"
	@echo "make run         - Run the application"
	@echo "make test        - Run tests"
	@echo "make test-coverage - Run tests with coverage"
	@echo "make clean       - Clean build artifacts"
	@echo "make deps        - Install dependencies"
	@echo "make fmt         - Format code"
	@echo "make init        - Initialize directory structure"
	@echo "make docker-build - Build Docker image"
	@echo "make docker-run  - Run Docker container"
