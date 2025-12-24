#!/usr/bin/env bash

# clean-state-modules
# Wrapper script that builds and runs the e2e clean command

set -euo pipefail

# Get script directory and repository root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

# Change to repository root
cd "$REPO_ROOT"

# Always build the e2e binary to ensure we're using the latest code
echo "Building e2e test runner..."
make build-e2e
echo ""

# Run the e2e clean command with modules as comma-separated list
exec ./bin/e2e-runner clean --modules "$@"
