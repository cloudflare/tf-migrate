# tf-migrate Makefile

BINARY_NAME := bin/tf-migrate
E2E_BINARY := bin/e2e-runner
GO := go
MAIN_PACKAGE := ./cmd/tf-migrate
E2E_PACKAGE := ./cmd/e2e-runner

.PHONY: all build build-e2e build-all test test-unit test-integration lint-testdata clean

# Default target: build all binaries
all: build-all

# Build the main tf-migrate binary
build:
	@mkdir -p bin
	$(GO) build -v -o $(BINARY_NAME) $(MAIN_PACKAGE)

# Build the e2e test runner binary
build-e2e:
	@mkdir -p bin
	$(GO) build -v -o $(E2E_BINARY) $(E2E_PACKAGE)

# Build both binaries
build-all: build build-e2e

# Run all tests (unit + e2e + integration)
test: test-unit test-integration

# Run unit tests
test-unit:
	$(GO) test -v -race ./internal/...

# Run integration tests
test-integration:
	$(GO) test -v -race ./integration/...

# Lint testdata to ensure all resources have cftftest prefix
lint-testdata:
	@echo "Linting integration testdata for naming conventions..."
	@$(GO) run scripts/lint_testdata_names.go

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f tf-migrate e2e-runner
	@echo "Clean complete"
