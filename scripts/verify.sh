#!/usr/bin/env bash
# verify.sh - Run comprehensive verification checks for claude-code-ntfy
# 
# This script runs all quality checks including:
# - Code formatting
# - Go vet
# - Tests with race detection  
# - Linting (golangci-lint, staticcheck)
# - Security scanning
# - Code quality checks
#
# Exit codes:
#   0 - All checks passed
#   1 - One or more checks failed

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "\n🔍 Running comprehensive verification...\n"

# Track if any checks fail
FAILED=0

# Check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Run each check
run_check() {
    local name=$1
    local command=$2
    
    echo -e "==> ${name}..."
    if eval "${command}"; then
        echo -e "${GREEN}✅ ${name} passed${NC}\n"
    else
        echo -e "${RED}❌ ${name} failed${NC}\n"
        FAILED=1
    fi
}

# Run all checks
run_check "Formatting" "gofmt -l . | (! grep .)"
run_check "Go vet" "go vet ./..."
run_check "Tests" "go test -race -cover ./..."

# Check required tools
check_required_tool() {
    local tool=$1
    local install_cmd=$2
    
    if ! command_exists "$tool"; then
        echo -e "${RED}❌ Required tool '$tool' is not installed${NC}"
        echo -e "   Install with: $install_cmd"
        FAILED=1
        return 1
    fi
    return 0
}

# Check all required tools first
echo -e "==> Checking required tools..."
TOOLS_OK=1
check_required_tool "golangci-lint" "go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" || TOOLS_OK=0
check_required_tool "staticcheck" "go install honnef.co/go/tools/cmd/staticcheck@latest" || TOOLS_OK=0
check_required_tool "gosec" "go install github.com/securego/gosec/v2/cmd/gosec@latest" || TOOLS_OK=0
check_required_tool "ineffassign" "go install github.com/gordonklaus/ineffassign@latest" || TOOLS_OK=0
check_required_tool "misspell" "go install github.com/client9/misspell/cmd/misspell@latest" || TOOLS_OK=0
check_required_tool "errcheck" "go install github.com/kisielk/errcheck@latest" || TOOLS_OK=0

if [ $TOOLS_OK -eq 0 ]; then
    echo -e "${RED}❌ Missing required tools. Please install them before running verification.${NC}\n"
    exit 1
fi
echo -e "${GREEN}✅ All required tools installed${NC}\n"

# Run all checks
run_check "Golangci-lint" "golangci-lint run"
run_check "Staticcheck" "staticcheck ./..."
run_check "Security" "gosec -quiet ./..."
run_check "Ineffassign" "ineffassign ./..."
run_check "Misspell" "misspell -error ."
run_check "Error check" "errcheck ./..."

# Note: interface{} usage is checked by golangci-lint's gocritic linter
# Note: TODO/FIXME comments are checked by golangci-lint's godox linter

echo ""

# Final result
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All verification checks passed!${NC}\n"
    exit 0
else
    echo -e "${RED}❌ Some checks failed. Run 'make fix' to auto-fix issues.${NC}\n"
    exit 1
fi