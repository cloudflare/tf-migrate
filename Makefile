# tf-migrate Makefile

BINARY_NAME := tf-migrate
GO := go
MAIN_PACKAGE := ./cmd/tf-migrate

.PHONY: build test lint-testdata

# Build the binary
build:
	$(GO) build -v -o $(BINARY_NAME) $(MAIN_PACKAGE)

# Run unit tests
test:
	$(GO) test -v -race ./...

# Lint testdata to ensure all resources have cftftest prefix
lint-testdata:
	@echo "Linting integration testdata for naming conventions..."
	@$(GO) run scripts/lint_testdata_names.go