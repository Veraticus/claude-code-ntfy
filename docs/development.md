# Development Guide

## Quick Start

```bash
# Clone the repository
git clone https://github.com/Veraticus/claude-code-ntfy
cd claude-code-ntfy

# Install required tools (one-time setup)
make install-tools

# Build and test
make test
make build

# Run the binary
./build/claude-code-ntfy -help
```

## Development Workflow

### 1. Make changes
Edit code as needed. The project structure:
- `cmd/` - Entry point
- `pkg/` - Core packages
- `scripts/` - Helper scripts

### 2. Test your changes
```bash
make test    # Quick - runs tests with race detection
```

### 3. Before committing
```bash
make fix     # Auto-fix formatting and common issues
make verify  # Run all checks (tests, linting, etc.)
```

## Common Tasks

| Task | Command | Description |
|------|---------|-------------|
| Build | `make build` | Build the binary |
| Test | `make test` | Run tests with race detection |
| Format | `make fmt` | Format code |
| Lint | `make lint` | Run linters |
| Coverage | `make cover` | Generate coverage report |
| Clean | `make clean` | Remove build artifacts |

## Writing Tests

- Use table-driven tests for multiple scenarios
- Mock external dependencies
- Run tests with race detection (automatic with `make test`)
- Check coverage with `make cover`

## Code Standards

The project enforces:
- Standard Go formatting (`gofmt`)
- Clean, idiomatic Go code
- Comprehensive error handling
- No race conditions
- Security best practices

These are automatically checked by `make verify`.