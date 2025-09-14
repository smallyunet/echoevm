#!/bin/bash
# Wrapper maintained for backward compatibility with older docs.
# Delegates to ../test.sh (new unified test runner).
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
exec "$PROJECT_ROOT/../test.sh" --binary "$@"
