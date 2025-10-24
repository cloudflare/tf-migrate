# tf-migrate Makefile

BINARY_NAME := tf-migrate
GO := go
MAIN_PACKAGE := ./cmd/tf-migrate

.PHONY: build test

# Build the binary
build:
	$(GO) build -v -o $(BINARY_NAME) $(MAIN_PACKAGE)

# Run unit tests
test:
	$(GO) test -v -race ./...