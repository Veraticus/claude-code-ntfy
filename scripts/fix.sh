#!/usr/bin/env bash
# fix.sh - Auto-fix common issues in claude-code-ntfy
#
# This script automatically fixes:
# - Code formatting (gofmt -s)
# - Import organization (goimports)
# - Common misspellings
# - Module dependencies (go mod tidy)

set -euo pipefail

echo "🔧 Auto-fixing issues..."

echo "  → Formatting code..."
gofmt -w -s .

echo "  → Organizing imports..."
if ! command -v goimports &> /dev/null; then
    echo "    Installing goimports..."
    go install golang.org/x/tools/cmd/goimports@latest
fi
goimports -w .

echo "  → Fixing misspellings..."
if ! command -v misspell &> /dev/null; then
    echo "    Installing misspell..."
    go install github.com/client9/misspell/cmd/misspell@latest
fi
misspell -w .

echo "  → Tidying modules..."
go mod tidy

echo "✅ Auto-fixes applied!"