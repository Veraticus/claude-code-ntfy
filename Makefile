.PHONY: all build test fmt lint clean install help

# Default target
all: fmt test build

# Build binary
build:
	go build -o build/claude-code-ntfy ./cmd/claude-code-ntfy

# Run tests
test:
	go test -race -cover ./...

# Format code
fmt:
	gofmt -w -s .

# Run linters
lint:
	golangci-lint run
	staticcheck ./...

# Clean build artifacts
clean:
	rm -rf build/ coverage.*

# Install binary
install:
	go install ./cmd/claude-code-ntfy

# Development helpers
.PHONY: cover fix verify quick setup-hooks install-tools

# Generate coverage report
cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Auto-fix issues
fix:
	@bash scripts/fix.sh

# Comprehensive verification
verify:
	@bash scripts/verify.sh

# Quick check (format and test)
quick: fmt test

# Setup git hooks
setup-hooks:
	@bash scripts/setup-hooks.sh

# Install required development tools
install-tools:
	@bash scripts/install-tools.sh

update-nix:
	@echo "Updating all Nix hashes to current HEAD..."
	@./scripts/update-nix-hashes.sh $(ARGS)

# Help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build    Build the binary"
	@echo "  test     Run tests with race detection"
	@echo "  fmt      Format code"
	@echo "  lint     Run linters"
	@echo "  clean    Remove build artifacts"
	@echo "  install  Install binary"
	@echo "  cover    Generate coverage report"
	@echo "  fix      Auto-fix formatting and other issues"
	@echo "  verify   Run all checks (for CI/pre-commit)"
	@echo "  quick    Format and test (for development)"
