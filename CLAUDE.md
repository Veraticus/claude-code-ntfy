# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Claude Code Ntfy is a transparent wrapper for Claude Code that monitors output and sends notifications via ntfy.sh based on configurable patterns and user activity. This is a Go project that implements a PTY-based process wrapper with intelligent notification capabilities.

## Critical Workflow - ALWAYS FOLLOW THIS!

### Research → Plan → Implement
**NEVER JUMP STRAIGHT TO CODING!** Always follow this sequence:
1. **Research**: Explore the codebase, understand existing patterns
2. **Plan**: Create a detailed implementation plan and verify it with me  
3. **Implement**: Execute the plan with validation checkpoints

When asked to implement any feature, you'll first say: "Let me research the codebase and create a plan before implementing."

### Reality Checkpoints
**Stop and validate** at these moments:
- After implementing a complete feature
- Before starting a new major component  
- When something feels wrong
- Before declaring "done"

Run: `make test && make lint`

## Architecture

The project follows a modular architecture with these key components:

- **Process Manager**: Manages Claude Code subprocess lifecycle using PTY for transparent terminal emulation
- **Output Monitor**: Monitors Claude output for configurable regex patterns and tracks visible content
- **Idle Detector**: Platform-specific user activity detection (Linux/macOS)
- **Notification Manager**: Orchestrates notifications with rate limiting and batching
- **Config Loader**: Loads configuration from environment variables and/or YAML config file

Key design principles:
- Zero-impact transparent wrapping of Claude Code
- Cross-platform support (Linux/macOS) 
- 100% test coverage without integration tests
- No modification of Claude Code behavior

## Go-Specific Standards

### FORBIDDEN - NEVER DO THESE:
- **NO interface{}** or **any{}** - use concrete types!
- **NO** keeping old and new code together
- **NO** migration functions or compatibility layers
- **NO** versioned function names (processV2, handleNew)
- **NO** custom error struct hierarchies
- **NO** TODOs in final code

### Required Standards:
- **Delete** old code when replacing it
- **Meaningful names**: `userID` not `id`
- **Early returns** to reduce nesting
- **Concrete types** from constructors: `func NewServer() *Server`
- **Simple errors**: `return fmt.Errorf("context: %w", err)`
- **Table-driven tests** for complex logic

### Our code is complete when:
- ✓ All linters pass with zero warnings
- ✓ All tests pass  
- ✓ Feature works end-to-end
- ✓ Old code is deleted
- ✓ Godoc on all exported symbols

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
- Complex business logic → Write tests first
- Simple CRUD → Write tests after
- Hot paths → Add benchmarks

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

Note: We use `pkg/` for all application code and `internal/` only if we need truly private code.

### Configuration
The wrapper supports configuration via:
1. YAML config file at `~/.config/claude-code-ntfy/config.yaml`
2. Environment variables (prefix: `CLAUDE_NOTIFY_`)
3. Command-line flags

Environment variables override config file settings.

## Performance & Security

### **Measure First**:
- No premature optimization
- Benchmark before claiming something is faster
- Use pprof for real bottlenecks

### **Security Always**:
- Validate all inputs
- Use crypto/rand for randomness
- Never log sensitive information (notification topics, servers)
- No secrets in code or commits

## Key Implementation Details

### Backstop Timer Behavior
The backstop timer only resets when:
- Claude Code produces **visible output** (not just ANSI escape sequences)
- A notification is sent through the system
- A new prompt/session starts (screen clear detected)

The timer is NOT reset by:
- Keyboard input alone (e.g., tmux escape sequences like ctrl-b)
- Terminal control sequences
- Focus events

This ensures notifications only fire when Claude is truly idle, not when you're just switching windows.

### Pattern Matching
- Uses Go's regexp package for pattern matching
- Patterns are compiled once at startup for performance
- Matches are batched to avoid notification spam
- Rate limiting prevents overwhelming the notification service

## Problem-Solving Together

When you're stuck or confused:
1. **Stop** - Don't spiral into complex solutions
2. **Step back** - Re-read the requirements
3. **Simplify** - The simple solution is usually correct
4. **Ask** - "I see two approaches: [A] vs [B]. Which do you prefer?"

## Communication Protocol

### Progress Updates:
```
✓ Implemented backstop timer logic (all tests passing)
✓ Added output monitoring  
✗ Found issue with ANSI sequence detection - investigating
```

### Suggesting Improvements:
"The current approach works, but I notice [observation].
Would you like me to [specific improvement]?"

## Working Together

- This is always a feature branch - no backwards compatibility needed
- When in doubt, we choose clarity over cleverness
- Avoid complex abstractions or "clever" code
- The simple, obvious solution is probably better

**REMINDER**: If working on a complex feature, create a TODO.md to track progress!