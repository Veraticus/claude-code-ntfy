# Claude Code Ntfy - Architecture

## Overview

Claude Code Ntfy is a transparent wrapper for Claude Code that sends a notification when Claude needs your attention. The wrapper preserves all Claude Code functionality while adding intelligent inactivity detection that respects user awareness.

### Goals
- Zero-impact transparent wrapping of Claude Code
- Smart inactivity detection with user input awareness
- Single notification per idle period
- Cross-platform support (Linux/macOS)
- Simple, focused functionality

### Non-Goals
- Pattern matching or regex-based notifications
- Complex notification rules or conditions
- Rate limiting or batching multiple notifications
- Status indicators or UI elements
- Modifying Claude Code behavior

## Architecture

### Simplified Architecture

```
┌─────────────────┐
│   User Input    │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────────┐
│           claude-code-ntfy              │
│                                         │
│  ┌─────────────┐    ┌────────────────┐ │
│  │   Config    │    │    Process     │ │
│  │   Loader    │    │    Manager     │ │
│  └─────────────┘    └───────┬────────┘ │
│                             │          │
│  ┌─────────────┐    ┌───────┴────────┐ │
│  │   Output    │◄───┤      PTY       │ │
│  │   Monitor   │    │    Manager     │ │
│  └──────┬──────┘    └───────┬────────┘ │
│         │                   │          │
│         ▼                   ▼          │
│  ┌─────────────┐    ┌────────────────┐ │
│  │  Backstop   │    │ Input Handler  │ │
│  │  Notifier   │◄───┤   (stdin)      │ │
│  └──────┬──────┘    └────────────────┘ │
│         │                              │
│         ▼                              │
│  ┌─────────────┐                      │
│  │ Ntfy Client │                      │
│  └─────────────┘                      │
└─────────────────────────────────────────┘
         │
         ▼
┌─────────────────┐
│  Claude Code    │
└─────────────────┘
```

### Component Responsibilities

1. **Config Loader**: Loads configuration from environment variables and/or YAML config file
2. **Process Manager**: Manages Claude Code subprocess lifecycle using PTY
3. **PTY Manager**: Provides transparent terminal emulation with separate stdin/stdout handling
4. **Output Monitor**: Tracks Claude output activity and detects bell characters
5. **Input Handler**: Detects user stdin activity to disable backstop timer
6. **Backstop Notifier**: Sends ONE notification after 30s of Claude inactivity
7. **Ntfy Client**: HTTP client that sends notifications to ntfy.sh

## Key Design Decisions

### 1. Inactivity Detection Logic
The backstop timer implements smart notification behavior:

- **While Claude is outputting**: Timer continuously resets
- **When Claude stops**: 30-second countdown begins
- **If user types (stdin)**: Timer is permanently disabled until Claude responds
- **If Claude sends bell**: Timer is permanently disabled (user already notified)
- **After 30 seconds**: ONE notification sent

Key insight: User input indicates awareness that Claude is waiting, so no notification needed.

### 2. Simplified Architecture
- **No pattern matching**: Removed all regex functionality
- **Single notification type**: "Claude needs attention"
- **Binary state**: Claude is either active or waiting
- **User awareness tracking**: Stdin activity = user knows

### 3. Implementation Approach
- Monitor stdin separately from PTY output using input handler
- Track Claude output for activity (any output resets timer)
- Disable timer permanently on user interaction
- Reset everything when Claude responds (new session)

## Package Structure

```
pkg/
├── config/          # Configuration loading and validation
│   └── config.go    # Config struct and loading logic
├── process/         # Process management and PTY handling
│   ├── manager.go   # Process lifecycle management
│   ├── pty.go       # PTY creation and I/O handling
│   └── interfaces.go # PTY interface definition
├── monitor/         # Output monitoring
│   ├── output_monitor.go     # Activity tracking and bell detection
│   ├── terminal_detector.go  # Terminal sequence detection
│   └── terminal_state.go     # Terminal state management
├── notification/    # Notification system
│   ├── notification.go      # Notification type
│   ├── backstop_notifier.go # Inactivity timer logic
│   ├── ntfy_client.go       # HTTP client for ntfy.sh
│   └── stdout_notifier.go   # Testing/debug notifier
├── interfaces/      # Core interface definitions
│   └── interfaces.go        # Shared interfaces
└── testutil/        # Testing utilities
    └── mocks.go             # Mock implementations
```

## Core Interfaces

```go
// DataHandler processes raw output data
type DataHandler interface {
    HandleData(data []byte)
    HandleLine(line string)
}

// Notifier sends notifications
type Notifier interface {
    Send(notification Notification) error
}

// ActivityMarker marks activity for backstop timer
type ActivityMarker interface {
    MarkActivity()
}
```

## Configuration

### Config File Format
```yaml
# ~/.config/claude-code-ntfy/config.yaml

# Notification settings
ntfy_topic: "my-claude-notifications"
ntfy_server: "https://ntfy.sh"

# Backstop timeout (default: 30s)
backstop_timeout: "30s"

# Disable all notifications
quiet: false

# Path to real claude binary (auto-detected if not set)
claude_path: "/usr/local/bin/claude"
```

### Environment Variables
```bash
export CLAUDE_NOTIFY_TOPIC="my-topic"
export CLAUDE_NOTIFY_SERVER="https://ntfy.sh"
export CLAUDE_NOTIFY_BACKSTOP_TIMEOUT="30s"
export CLAUDE_NOTIFY_QUIET="false"
export CLAUDE_NOTIFY_CLAUDE_PATH="/usr/local/bin/claude"
```

### Configuration Priority
1. Command-line flags (highest priority)
2. Environment variables
3. Config file
4. Default values (lowest priority)

## Implementation Details

### Process Management
- Uses `github.com/creack/pty` for PTY creation
- Full signal forwarding (SIGTERM, SIGINT, SIGWINCH, etc.)
- Transparent I/O copying with separate stdin/stdout handlers
- Terminal size synchronization
- Self-wrap detection via environment variable

### Output Monitoring
- Tracks last output time for activity detection
- Line buffering for bell detection
- Terminal sequence detection for screen clears (new session)
- Thread-safe with mutex protection
- Sends activity signals to backstop notifier

### Backstop Notification
- Single timer per session
- Resets on any Claude output
- Permanently disabled by user input
- Permanently disabled by bell character
- Only sends ONE notification per idle period
- Resets on screen clear (new prompt)

### PTY I/O Handling
- Separate handlers for input and output
- Input detection via wrapper Reader
- Output monitoring via wrapper Reader
- Preserves raw terminal mode
- Full transparency for all terminal features

## Testing Strategy

- All tests use mocks - no real Claude Code execution
- Platform-specific code uses build tags
- Mock implementations for all interfaces
- Table-driven tests for multiple scenarios
- No network calls or file I/O in tests (except config tests)
- ~90% code coverage across packages

## Performance Characteristics

- **Startup time**: < 50ms
- **Memory usage**: ~8MB RSS (without Claude Code)
- **CPU usage**: < 0.1% when idle
- **I/O latency**: < 1ms (transparent passthrough)
- **Zero impact on Claude Code performance**

## Security Considerations

1. **Command Injection**: Using `exec.Command` with separate args
2. **PTY Security**: Proper cleanup and signal handling
3. **No Logging**: Output is never logged or stored
4. **Ntfy Authentication**: Supports auth tokens if needed
5. **Signal Handling**: Proper cleanup on all termination signals