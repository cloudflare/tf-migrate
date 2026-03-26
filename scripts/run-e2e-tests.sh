#!/usr/bin/env bash

# run-e2e-tests
# Wrapper script that builds and runs the e2e test runner
#
# Usage:
#   # v4→v5 migration tests (default)
#   ./scripts/run-e2e-tests.sh [OPTIONS]
#
#   # v5→v5 upgrade tests
#   ./scripts/run-e2e-tests.sh v5-upgrade [OPTIONS]
#
# Examples:
#   # Run v4→v5 migration tests
#   ./scripts/run-e2e-tests.sh --apply-exemptions --uses-provider-state-upgrader
#
#   # Run v5→v5 upgrade tests
#   ./scripts/run-e2e-tests.sh v5-upgrade --from-version 5.18.0 --provider ../provider --apply-exemptions
#   ./scripts/run-e2e-tests.sh v5-upgrade --from-version 5.18.0 --provider ../provider --clean

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

# Check if first argument is "v5-upgrade" for v5→v5 tests
if [[ "${1:-}" == "v5-upgrade" ]]; then
    # Run v5-upgrade command with remaining arguments
    exec ./bin/e2e-runner "$@"
else
    # Default: run v4→v5 migration tests
    exec ./bin/e2e-runner run "$@"
fi
