#!/bin/bash
# Placeholder advanced test script retained for compatibility.
# For now it invokes the unified runner with all suites in verbose mode.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
exec "$PROJECT_ROOT/../test.sh" --verbose "$@"
