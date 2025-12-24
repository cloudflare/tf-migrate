#!/usr/bin/env bash

# run-e2e-tests
# Wrapper script that builds and runs the e2e test runner

set -euo pipefail

# Get script directory and repository root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

# Change to repository root
cd "$REPO_ROOT"

# Always build binaries to ensure we're using the latest code
echo "Building binaries..."
make build-all
echo ""

# Run the e2e binary with all arguments forwarded
exec ./bin/e2e-runner run "$@"
