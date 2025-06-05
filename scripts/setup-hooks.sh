#!/usr/bin/env bash
# setup-hooks.sh - Install git pre-commit hooks

set -euo pipefail

# Check if we're in a git repository
if [ ! -d .git ]; then
    echo "Error: Not in a git repository"
    exit 1
fi

# Create hooks directory if it doesn't exist
mkdir -p .git/hooks

# Create pre-commit hook
cat > .git/hooks/pre-commit << 'EOF'
#!/usr/bin/env bash
# Pre-commit hook for claude-code-ntfy

echo "Running pre-commit checks..."

# Run verification
if ! make verify; then
    echo ""
    echo "Pre-commit checks failed!"
    echo "Run 'make fix' to auto-fix issues, then try again."
    exit 1
fi

echo "Pre-commit checks passed!"
EOF

# Make hook executable
chmod +x .git/hooks/pre-commit

echo "✅ Git pre-commit hook installed successfully!"
echo "The hook will run 'make verify' before each commit."