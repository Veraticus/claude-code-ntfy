#!/usr/bin/env bash
# install-tools.sh - Install all required development tools

set -euo pipefail

echo "Installing required Go development tools..."

# Function to install a tool
install_tool() {
    local name=$1
    local package=$2
    
    echo -n "Installing $name... "
    if go install "$package" 2>/dev/null; then
        echo "✓"
    else
        echo "✗"
        echo "  Failed to install $name"
        return 1
    fi
}

# Install all tools
install_tool "golangci-lint" "github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
install_tool "staticcheck" "honnef.co/go/tools/cmd/staticcheck@latest"
install_tool "gosec" "github.com/securego/gosec/v2/cmd/gosec@latest"
install_tool "ineffassign" "github.com/gordonklaus/ineffassign@latest"
install_tool "misspell" "github.com/client9/misspell/cmd/misspell@latest"
install_tool "errcheck" "github.com/kisielk/errcheck@latest"
install_tool "goimports" "golang.org/x/tools/cmd/goimports@latest"

echo ""
echo "✅ All tools installed successfully!"
echo ""
echo "Make sure $HOME/go/bin is in your PATH:"
echo "  export PATH=\$PATH:\$HOME/go/bin"