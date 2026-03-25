# tf-migrate Makefile

BINARY_NAME := bin/tf-migrate
E2E_BINARY := bin/e2e-runner
GO := go
MAIN_PACKAGE := ./cmd/tf-migrate
E2E_PACKAGE := ./cmd/e2e-runner

.PHONY: all build build-e2e build-all test test-unit test-integration lint-testdata clean release-snapshot sync-exemptions test-state-upgrader

# Default target: build all binaries
all: build-all

# Build the main tf-migrate binary
build: sync-exemptions
	@mkdir -p bin
	$(GO) build -v -o $(BINARY_NAME) $(MAIN_PACKAGE)

# Build the e2e test runner binary
build-e2e: sync-exemptions
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

# Test GoReleaser build locally (no publish)
release-snapshot:
	goreleaser build --snapshot --clean

# Sync embedded exemption YAML files from e2e source of truth
sync-exemptions:
	@echo "Syncing exemption YAMLs from e2e/ into internal/verifydrift/exemptions/..."
	cp e2e/global-drift-exemptions.yaml internal/verifydrift/exemptions/
	cp -r e2e/drift-exemptions/. internal/verifydrift/exemptions/drift-exemptions/
	@echo "Sync complete"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f tf-migrate e2e-runner
	@echo "Clean complete"
