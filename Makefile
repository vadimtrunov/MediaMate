# MediaMate Makefile

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Build
.PHONY: build
build: ## Build binary
	@echo "Building mediamate..."
	@mkdir -p bin
	@go build -v -ldflags="-s -w" -o bin/mediamate ./cmd/mediamate

.PHONY: build-all
build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p bin
	@GOOS=linux GOARCH=amd64 go build -v -ldflags="-s -w" -o bin/mediamate-linux-amd64 ./cmd/mediamate
	@GOOS=linux GOARCH=arm64 go build -v -ldflags="-s -w" -o bin/mediamate-linux-arm64 ./cmd/mediamate
	@GOOS=darwin GOARCH=amd64 go build -v -ldflags="-s -w" -o bin/mediamate-darwin-amd64 ./cmd/mediamate
	@GOOS=darwin GOARCH=arm64 go build -v -ldflags="-s -w" -o bin/mediamate-darwin-arm64 ./cmd/mediamate

# Testing
.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...

.PHONY: test-coverage
test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -v -race -tags=integration ./...

.PHONY: bench
bench: ## Run benchmarks
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Linting
.PHONY: lint
lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	@golangci-lint run ./...

.PHONY: lint-fix
lint-fix: ## Run golangci-lint with auto-fix
	@echo "Running golangci-lint with auto-fix..."
	@golangci-lint run --fix ./...

.PHONY: lint-strict
lint-strict: ## Run golangci-lint in strict mode (all linters)
	@echo "Running golangci-lint in strict mode..."
	@golangci-lint run --enable-all ./...

# Formatting
.PHONY: fmt
fmt: ## Format code with gofumpt
	@echo "Formatting code..."
	@gofumpt -l -w .

.PHONY: fmt-check
fmt-check: ## Check code formatting
	@echo "Checking code formatting..."
	@gofumpt -l -d .

.PHONY: imports
imports: ## Organize imports with goimports
	@echo "Organizing imports..."
	@goimports -w -local github.com/vadimtrunov/MediaMate .

# Code quality
.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

.PHONY: staticcheck
staticcheck: ## Run staticcheck
	@echo "Running staticcheck..."
	@staticcheck ./...

.PHONY: gosec
gosec: ## Run gosec security scanner
	@echo "Running gosec..."
	@gosec -fmt=json -out=gosec-results.json ./...

# Dependencies
.PHONY: deps
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

.PHONY: deps-verify
deps-verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	@go mod verify

.PHONY: tidy
tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	@go mod tidy

# Docker
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t mediamate:latest .

.PHONY: docker-build-multiarch
docker-build-multiarch: ## Build multi-arch Docker image
	@echo "Building multi-arch Docker image..."
	@docker buildx build --platform linux/amd64,linux/arm64 -t mediamate:latest .

# Development
.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install mvdan.cc/gofumpt@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@go install github.com/goreleaser/goreleaser@latest

.PHONY: pre-commit
pre-commit: fmt imports lint test ## Run pre-commit checks
	@echo "✅ Pre-commit checks passed!"

.PHONY: ci
ci: deps-verify lint test ## Run CI checks locally
	@echo "✅ CI checks passed!"

# Cleaning
.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -rf dist/
	@rm -f coverage.out coverage.html
	@rm -f gosec-results.json
	@go clean

.PHONY: clean-all
clean-all: clean ## Clean everything including caches
	@echo "Deep cleaning..."
	@go clean -cache -testcache -modcache

# Running
.PHONY: run
run: ## Run the application
	@go run ./cmd/mediamate

.PHONY: run-dev
run-dev: ## Run in development mode with hot reload
	@air

# Git hooks
.PHONY: install-hooks
install-hooks: ## Install git hooks
	@echo "Installing git hooks..."
	@cp -f scripts/pre-commit.sh .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "✅ Git hooks installed!"

# Documentation
.PHONY: docs
docs: ## Generate documentation
	@echo "Generating documentation..."
	@godoc -http=:6060

# Release
.PHONY: release-dry
release-dry: ## Dry run of release process
	@echo "Dry run of release..."
	@goreleaser release --snapshot --clean

.PHONY: release
release: ## Create a release (requires tag)
	@echo "Creating release..."
	@goreleaser release --clean

# All-in-one commands
.PHONY: check
check: fmt imports lint vet test ## Run all checks
	@echo "✅ All checks passed!"

.PHONY: prepare
prepare: fmt imports lint-fix tidy ## Prepare code for commit
	@echo "✅ Code prepared for commit!"

# Default target
.DEFAULT_GOAL := help
