.PHONY: build test test-verbose test-coverage lint clean install dev

# Version information
VERSION ?= $$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $$(git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILDTIME ?= $$(date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X 'github.com/craigderington/prox/internal/version.Version=$(VERSION)' \
           -X 'github.com/craigderington/prox/internal/version.Commit=$(COMMIT)' \
           -X 'github.com/craigderington/prox/internal/version.BuildTime=$(BUILDTIME)'

# Build the binary
build:
	go build -ldflags "$(LDFLAGS)" -o prox .

# Run all tests
test:
	go test ./... -v

# Run tests with verbose output
test-verbose:
	go test ./... -v -race

# Run tests with coverage
test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linting
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -f prox
	rm -f coverage.out coverage.html

# Install binary globally
install: build
	sudo mv prox /usr/local/bin/

# Development mode with hot reload (if you have air installed)
dev:
	air

# Run tests and lint
check: test lint

# Full CI pipeline
ci: clean test-coverage lint build