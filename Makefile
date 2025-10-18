# Makefile for Web Scraper

.PHONY: help build build-api build-cli test test-verbose test-coverage clean run run-api install lint fmt vet check all

# Default target
help:
	@echo "Available targets:"
	@echo "  build          - Build CLI application"
	@echo "  build-api      - Build API server"
	@echo "  build-all      - Build both CLI and API server"
	@echo "  install        - Build and install to GOPATH/bin"
	@echo "  test           - Run all tests"
	@echo "  test-verbose   - Run tests with verbose output"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  coverage-html  - Generate HTML coverage report"
	@echo "  run            - Run CLI application (requires URL variable)"
	@echo "  run-api        - Run API server (optional: PORT, DB variables)"
	@echo "  clean          - Remove build artifacts"
	@echo "  fmt            - Format Go code"
	@echo "  vet            - Run go vet"
	@echo "  lint           - Run golangci-lint (if installed)"
	@echo "  check          - Run fmt, vet, and test"
	@echo "  all            - Run check and build"
	@echo ""
	@echo "Examples:"
	@echo "  make build-api"
	@echo "  make run URL=https://example.com"
	@echo "  make run-api PORT=3000 DB=./data/scraper.db"
	@echo "  make test-coverage"

# Build the CLI application (default to API for integration tests)
build: build-api

build-cli:
	@echo "Building CLI scraper..."
	@go build -o scraper-bin -ldflags="-s -w"
	@chmod +x scraper-bin
	@echo "Build complete: scraper-bin"

# Build the API server
build-api:
	@echo "Building API server..."
	@go build -o scraper-api -ldflags="-s -w" ./cmd/api
	@chmod +x scraper-api
	@echo "Build complete: scraper-api"

# Build both
build-both: build-cli build-api
	@echo "Both builds complete"

# Install to GOPATH/bin
install:
	@echo "Installing scraper..."
	@go install
	@echo "Install complete"

# Run all tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -cover ./...

# Generate HTML coverage report
coverage-html:
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@echo "Opening in browser..."
	@open coverage.html 2>/dev/null || xdg-open coverage.html 2>/dev/null || echo "Please open coverage.html manually"

# Run the CLI application (requires URL variable)
run: build-cli
	@if [ -z "$(URL)" ]; then \
		echo "Error: URL variable is required"; \
		echo "Usage: make run URL=https://example.com"; \
		exit 1; \
	fi
	@echo "Scraping $(URL)..."
	@./scraper-bin -url "$(URL)" -pretty

# Run the API server
run-api: build-api
	@echo "Starting API server..."
	@./scraper-api $(if $(PORT),-addr :$(PORT)) $(if $(DB),-db $(DB))

# Run with custom options
run-custom: build
	@if [ -z "$(URL)" ]; then \
		echo "Error: URL variable is required"; \
		echo "Usage: make run-custom URL=https://example.com OLLAMA_URL=http://localhost:11434"; \
		exit 1; \
	fi
	@./scraper-bin -url "$(URL)" $(if $(PRETTY),-pretty) $(if $(OLLAMA_URL),-ollama-url $(OLLAMA_URL)) $(if $(OLLAMA_MODEL),-ollama-model $(OLLAMA_MODEL)) $(if $(TIMEOUT),-timeout $(TIMEOUT))

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f scraper-bin scraper-api
	@rm -f coverage.out coverage.html
	@rm -f scraper.db scraper.db-journal
	@echo "Clean complete"

# Format Go code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Run golangci-lint (if installed)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Skipping..."; \
		echo "Install with: brew install golangci-lint"; \
	fi

# Check code quality
check: fmt vet test
	@echo "All checks passed!"

# Run everything
all: check build
	@echo "Build and checks complete!"

# Development workflow - watch and test (requires entr)
watch:
	@if command -v entr >/dev/null 2>&1; then \
		echo "Watching for changes..."; \
		ls **/*.go | entr -c make test; \
	else \
		echo "entr not installed. Install with: brew install entr"; \
		exit 1; \
	fi

# Benchmark tests
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies updated"

# Verify dependencies
verify:
	@echo "Verifying dependencies..."
	@go mod verify

# Build for multiple platforms (cross-compile)
build-cross:
	@echo "Building for multiple platforms..."
	@mkdir -p dist
	@echo "Building for Linux (amd64)..."
	@GOOS=linux GOARCH=amd64 go build -o dist/scraper-linux-amd64 -ldflags="-s -w"
	@echo "Building for Linux (arm64)..."
	@GOOS=linux GOARCH=arm64 go build -o dist/scraper-linux-arm64 -ldflags="-s -w"
	@echo "Building for macOS (amd64)..."
	@GOOS=darwin GOARCH=amd64 go build -o dist/scraper-darwin-amd64 -ldflags="-s -w"
	@echo "Building for macOS (arm64)..."
	@GOOS=darwin GOARCH=arm64 go build -o dist/scraper-darwin-arm64 -ldflags="-s -w"
	@echo "Building for Windows (amd64)..."
	@GOOS=windows GOARCH=amd64 go build -o dist/scraper-windows-amd64.exe -ldflags="-s -w"
	@echo "All builds complete. Check dist/ directory"
	@ls -lh dist/

# Quick test - only fast tests
test-quick:
	@echo "Running quick tests..."
	@go test -short ./...

# Show test coverage by package
coverage-by-package:
	@echo "Coverage by package:"
	@go test -coverprofile=coverage.out ./... > /dev/null 2>&1
	@go tool cover -func=coverage.out | grep total

# Update all dependencies to latest
update-deps:
	@echo "Updating dependencies to latest versions..."
	@go get -u ./...
	@go mod tidy
	@echo "Dependencies updated"

# Generate documentation
docs:
	@echo "Generating documentation..."
	@go doc -all > GODOC.txt
	@echo "Documentation generated: GODOC.txt"

# Docker build (if you want to add Docker support later)
docker-build:
	@echo "Building Docker image..."
	@docker build -t scraper:latest .

# Show project statistics
stats:
	@echo "Project Statistics:"
	@echo "==================="
	@echo "Go files:"
	@ls -1 **/*.go 2>/dev/null | wc -l | xargs echo "  "
	@echo "Total lines of Go code:"
	@cat **/*.go 2>/dev/null | wc -l | xargs echo "  "
	@echo "Test files:"
	@ls -1 **/*_test.go 2>/dev/null | wc -l | xargs echo "  "
	@echo "Packages:"
	@ls -d */ 2>/dev/null | wc -l | xargs echo "  "
