# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Claude Code Ntfy is a transparent wrapper for Claude Code that monitors output and sends notifications via ntfy.sh based on configurable patterns and user activity. This is a Go project that implements a PTY-based process wrapper with intelligent notification capabilities.

## Architecture

The project follows a modular architecture with these key components:

- **Process Manager**: Manages Claude Code subprocess lifecycle using PTY for transparent terminal emulation
- **Output Monitor**: Monitors Claude output for configurable regex patterns
- **Idle Detector**: Platform-specific user activity detection (Linux/macOS)
- **Notification Manager**: Orchestrates notifications with rate limiting and batching
- **Config Loader**: Loads configuration from environment variables and/or YAML config file

Key design principles:
- Zero-impact transparent wrapping of Claude Code
- Cross-platform support (Linux/macOS) 
- 100% test coverage without integration tests
- No modification of Claude Code behavior

## Common Development Tasks

### Build Commands
```bash
# Core commands
make build    # Build the binary
make test     # Run tests with race detection
make fmt      # Format code
make lint     # Run linters (golangci-lint + staticcheck)

# Development helpers
make quick    # Format and test (for development)
make fix      # Auto-fix formatting and other issues
make verify   # Run all checks (for CI/pre-commit)
make cover    # Generate test coverage report
```

### Development Workflow

**Prerequisites**: Run `make install-tools` once to install all required development tools.

**Simple and effective workflow:**

1. **During development**: `make test` - Fast feedback on functionality
2. **Before committing**:
   ```bash
   make fix      # Auto-fix issues
   make verify   # Ensure everything passes
   ```

The project uses standard Go tools:
- `gofmt` for formatting
- `go test -race` for testing
- `golangci-lint` with default settings for comprehensive linting
- `staticcheck` for advanced static analysis

### Testing Strategy
- All tests use mocks - no real Claude Code execution
- Platform-specific code uses build tags
- Mock implementations for all interfaces
- Table-driven tests for multiple scenarios
- No network calls or file I/O in tests (except config tests)

### Project Structure
```
pkg/
├── config/          # Configuration loading and validation
├── process/         # Process management and PTY handling
├── monitor/         # Output monitoring and pattern matching
├── notification/    # Notification management, batching, rate limiting
├── idle/            # Platform-specific idle detection
├── interfaces/      # Core interface definitions
└── testutil/        # Testing utilities

cmd/
└── claude-code-ntfy/  # Main entry point
```

### Testing Strategy
- All tests use mocks - no real Claude Code execution
- Platform-specific code uses build tags
- Mock implementations for all interfaces
- Table-driven tests for multiple scenarios
- No network calls or file I/O in tests (except config tests)

### Configuration
The wrapper supports configuration via:
1. YAML config file at `~/.config/claude-code-ntfy/config.yaml`
2. Environment variables (prefix: `CLAUDE_NOTIFY_`)
3. Command-line flags

Environment variables override config file settings.