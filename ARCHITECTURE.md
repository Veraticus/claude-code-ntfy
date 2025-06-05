# Claude Code Ntfy - Design Document

## Overview

Claude Code Ntfy is a transparent wrapper for Claude Code that monitors output and sends notifications via ntfy.sh based on configurable patterns and user activity. The wrapper preserves all Claude Code functionality while adding intelligent notification capabilities.

### Goals
- Zero-impact transparent wrapping of Claude Code
- Intelligent notification triggers based on output patterns
- Activity-aware notification suppression
- Cross-platform support (Linux/macOS)
- 100% test coverage without integration tests

### Non-Goals
- Windows support (future consideration)
- Session recording/replay
- Binary output handling
- Modifying Claude Code behavior

## Architecture

### High-Level Architecture

```
┌─────────────────┐
│   User Input    │
└────────┬────────┘
         │
         ▼
┌────────────────────────────────────────────────┐
│              claude-code-ntfy                    │
│                                                  │
│  ┌─────────────┐  ┌────────────┐  ┌──────────┐ │
│  │   Config    │  │   Process  │  │  Output  │ │
│  │   Loader    │  │   Manager  │  │  Monitor │ │
│  └─────────────┘  └─────┬──────┘  └─────┬────┘ │
│                          │               │      │
│  ┌─────────────┐  ┌─────┴──────┐  ┌────┴────┐ │
│  │    Idle     │  │    PTY     │  │ Pattern │ │
│  │  Detector   │  │  Handler   │  │ Matcher │ │
│  └─────────────┘  └────────────┘  └─────────┘ │
│                                                 │
│  ┌─────────────────────────────────────────┐  │
│  │            Notification Manager          │  │
│  │  ┌──────────┐ ┌────────┐ ┌───────────┐ │  │
│  │  │ Batcher  │ │  Rate  │ │   Ntfy    │ │  │
│  │  │          │ │ Limiter│ │  Client   │ │  │
│  │  └──────────┘ └────────┘ └───────────┘ │  │
│  └─────────────────────────────────────────┘  │
└─────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────┐
│  Claude Code    │
└─────────────────┘
```

### Component Responsibilities

1. **Config Loader**: Loads configuration from environment variables and/or config file
2. **Process Manager**: Manages Claude Code subprocess lifecycle
3. **PTY Handler**: Provides transparent terminal emulation
4. **Output Monitor**: Monitors Claude output for patterns
5. **Pattern Matcher**: Applies regex patterns to output
6. **Idle Detector**: Platform-specific user activity detection
7. **Notification Manager**: Orchestrates notification logic
8. **Batcher**: Groups notifications within time windows
9. **Rate Limiter**: Prevents notification spam
10. **Ntfy Client**: Sends notifications to ntfy.sh

## Detailed Design

### Core Interfaces

```go
// pkg/interfaces/interfaces.go
package interfaces

import (
    "time"
    "regexp"
)

// IdleDetector detects user activity/inactivity
type IdleDetector interface {
    IsUserIdle(threshold time.Duration) (bool, error)
    LastActivity() time.Time
}

// Notifier sends notifications
type Notifier interface {
    Send(notification Notification) error
}

// ProcessWrapper wraps and monitors a process
type ProcessWrapper interface {
    Start(command string, args []string) error
    Wait() error
    ExitCode() int
}

// OutputHandler processes output lines
type OutputHandler interface {
    HandleLine(line string)
}

// PatternMatcher matches patterns in text
type PatternMatcher interface {
    Match(text string) []MatchResult
}

// RateLimiter limits notification frequency
type RateLimiter interface {
    Allow() bool
    Reset()
}
```

### Core Structs

```go
// pkg/config/config.go
package config

import (
    "time"
    "regexp"
)

type Config struct {
    // Notification settings
    NtfyTopic    string        `yaml:"ntfy_topic" env:"CLAUDE_NOTIFY_TOPIC"`
    NtfyServer   string        `yaml:"ntfy_server" env:"CLAUDE_NOTIFY_SERVER"`
    IdleTimeout  time.Duration `yaml:"idle_timeout" env:"CLAUDE_NOTIFY_IDLE_TIMEOUT"`
    
    // Behavior flags
    Quiet        bool          `yaml:"quiet" env:"CLAUDE_NOTIFY_QUIET"`
    ForceNotify  bool          `yaml:"force_notify" env:"CLAUDE_NOTIFY_FORCE"`
    
    // Pattern configuration
    Patterns     []Pattern     `yaml:"patterns"`
    
    // Rate limiting
    RateLimit    RateLimitConfig `yaml:"rate_limit"`
    
    // Batching
    BatchWindow  time.Duration `yaml:"batch_window"`
}

type Pattern struct {
    Name        string `yaml:"name"`
    Regex       string `yaml:"regex"`
    Description string `yaml:"description"`
    Enabled     bool   `yaml:"enabled"`
    compiled    *regexp.Regexp
}

type RateLimitConfig struct {
    Window      time.Duration `yaml:"window"`
    MaxMessages int           `yaml:"max_messages"`
}

// Default configuration
func DefaultConfig() *Config {
    return &Config{
        NtfyServer:  "https://ntfy.sh",
        IdleTimeout: 2 * time.Minute,
        Patterns: []Pattern{
            {
                Name:    "bell",
                Regex:   `\x07`,
                Enabled: true,
            },
            {
                Name:    "question",
                Regex:   `\?\s*$`,
                Enabled: true,
            },
            {
                Name:    "error",
                Regex:   `(?i)(error|failed|exception)`,
                Enabled: true,
            },
            {
                Name:    "completion",
                Regex:   `(?i)(done|finished|completed)`,
                Enabled: true,
            },
        },
        RateLimit: RateLimitConfig{
            Window:      1 * time.Minute,
            MaxMessages: 5,
        },
        BatchWindow: 5 * time.Second,
    }
}
```

```go
// pkg/process/manager.go
package process

import (
    "os"
    "os/exec"
    "github.com/creack/pty"
)

type Manager struct {
    config       *config.Config
    cmd          *exec.Cmd
    pty          *os.File
    outputMonitor *monitor.OutputMonitor
    idleDetector interfaces.IdleDetector
}

type PTYSize struct {
    Rows uint16
    Cols uint16
}
```

```go
// pkg/monitor/output.go
package monitor

import (
    "bufio"
    "sync"
)

type OutputMonitor struct {
    config          *config.Config
    patternMatcher  interfaces.PatternMatcher
    notificationMgr *notification.Manager
    idleDetector    interfaces.IdleDetector
    
    mu              sync.Mutex
    lastOutputTime  time.Time
}

type MatchResult struct {
    PatternName string
    Text        string
    Position    int
}
```

```go
// pkg/notification/manager.go
package notification

import (
    "sync"
    "time"
)

type Manager struct {
    config      *config.Config
    notifier    interfaces.Notifier
    rateLimiter interfaces.RateLimiter
    batcher     *Batcher
    
    mu          sync.Mutex
}

type Notification struct {
    Title       string
    Message     string
    Time        time.Time
    Pattern     string
}

type Batcher struct {
    window       time.Duration
    mu           sync.Mutex
    pending      []Notification
    timer        *time.Timer
}
```

```go
// pkg/idle/detector_linux.go
// +build linux

package idle

type LinuxIdleDetector struct {
    tmuxDetector *TmuxIdleDetector
    fallback     *OutputBasedDetector
}

type TmuxIdleDetector struct {
    sessionName string
}

// pkg/idle/detector_darwin.go
// +build darwin

package idle

type DarwinIdleDetector struct {
    // Uses ioreg to get system idle time
}

// pkg/idle/detector_output.go
package idle

type OutputBasedDetector struct {
    mu           sync.RWMutex
    lastActivity time.Time
}
```

### Key Algorithms

#### Rate Limiting
```go
// Token bucket algorithm
type TokenBucket struct {
    capacity    int
    tokens      int
    refillRate  time.Duration
    lastRefill  time.Time
    mu          sync.Mutex
}

func (tb *TokenBucket) Allow() bool {
    tb.mu.Lock()
    defer tb.mu.Unlock()
    
    // Refill tokens
    now := time.Now()
    elapsed := now.Sub(tb.lastRefill)
    tokensToAdd := int(elapsed / tb.refillRate)
    
    tb.tokens = min(tb.capacity, tb.tokens + tokensToAdd)
    tb.lastRefill = now
    
    // Try to consume
    if tb.tokens > 0 {
        tb.tokens--
        return true
    }
    return false
}
```

#### Notification Batching
```go
func (b *Batcher) Add(n Notification) {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    b.pending = append(b.pending, n)
    
    if b.timer == nil {
        b.timer = time.AfterFunc(b.window, b.flush)
    }
}

func (b *Batcher) flush() {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    if len(b.pending) == 0 {
        return
    }
    
    // Create batched notification
    batched := b.createBatchedNotification(b.pending)
    b.notifier.Send(batched)
    
    b.pending = nil
    b.timer = nil
}
```

## Configuration Schema

### Config File Format
```yaml
# ~/.config/claude-code-ntfy/config.yaml

# Notification settings
ntfy_topic: "my-claude-notifications"
ntfy_server: "https://ntfy.sh"
idle_timeout: "2m"

# Behavior
quiet: false
force_notify: false

# Pattern configuration
patterns:
  - name: "bell"
    regex: '\x07'
    description: "Terminal bell character"
    enabled: true
    
  - name: "question"
    regex: '\?\s*$'
    description: "Lines ending with question mark"
    enabled: true
    
  - name: "error"
    regex: '(?i)(error|failed|exception|panic|fatal)'
    description: "Error indicators"
    enabled: true
    
  - name: "completion"
    regex: '(?i)(done|finished|completed|success)'
    description: "Task completion indicators"
    enabled: true
    
  - name: "custom"
    regex: 'ATTENTION|IMPORTANT'
    description: "Custom attention patterns"
    enabled: false

# Rate limiting
rate_limit:
  window: "1m"
  max_messages: 5

# Batching
batch_window: "5s"
```

### Environment Variables
```bash
# Override any config file setting
export CLAUDE_NOTIFY_TOPIC="my-topic"
export CLAUDE_NOTIFY_SERVER="https://ntfy.sh"
export CLAUDE_NOTIFY_IDLE_TIMEOUT="5m"
export CLAUDE_NOTIFY_QUIET="true"
export CLAUDE_NOTIFY_FORCE="false"
```

## Development Plan

### Phase 1: Core Infrastructure
1. Project setup and structure
2. Config loader with env var and file support
3. Basic process manager with exec.Command
4. Signal forwarding implementation
5. Exit code preservation

### Phase 2: PTY Implementation
1. PTY creation and management
2. Transparent I/O copying
3. Terminal resize handling
4. Signal propagation through PTY

### Phase 3: Pattern Matching
1. Pattern matcher implementation
2. Regex compilation and caching
3. Output monitor with line buffering
4. Match result generation

### Phase 4: Platform-Specific Idle Detection
1. Interface definition
2. Linux tmux implementation
3. macOS ioreg implementation
4. Output-based fallback
5. Platform build tags

### Phase 5: Notification System
1. Notification manager
2. Rate limiter implementation
3. Batcher implementation
4. Ntfy client
5. Quiet mode support

### Phase 6: Integration
1. Wire all components together
2. Command-line argument parsing
3. Self-wrap detection
4. Error handling and graceful shutdown

### Phase 7: Testing and Polish
1. Unit tests for all components
2. Mock implementations
3. Test utilities
4. Documentation
5. Example configurations

## Testing Strategy

### Unit Test Coverage Plan

#### 1. Config Package (100% coverage)
```go
// config_test.go
func TestDefaultConfig(t *testing.T)
func TestLoadFromFile(t *testing.T)
func TestLoadFromEnv(t *testing.T)
func TestConfigMerge(t *testing.T)
func TestPatternCompilation(t *testing.T)
```

#### 2. Process Package (100% coverage)
```go
// Mock PTY for testing
type MockPTY struct {
    ReadData  []byte
    WriteData []byte
    Size      PTYSize
}

func TestProcessStart(t *testing.T)
func TestSignalForwarding(t *testing.T)
func TestExitCodePreservation(t *testing.T)
func TestPTYResize(t *testing.T)
```

#### 3. Monitor Package (100% coverage)
```go
// Mock pattern matcher
type MockPatternMatcher struct {
    Patterns []MatchResult
}

func TestOutputMonitoring(t *testing.T)
func TestLineBuffering(t *testing.T)
func TestPatternDetection(t *testing.T)
func TestIdleTimeout(t *testing.T)
```

#### 4. Notification Package (100% coverage)
```go
// Mock notifier
type MockNotifier struct {
    Sent []Notification
}

func TestNotificationSending(t *testing.T)
func TestRateLimiting(t *testing.T)
func TestBatching(t *testing.T)
func TestQuietMode(t *testing.T)
func TestForceNotify(t *testing.T)
```

#### 5. Idle Package (100% coverage)
```go
// Platform-specific mocks
type MockSystemIdleDetector struct {
    IdleTime time.Duration
}

func TestTmuxDetection(t *testing.T)
func TestDarwinDetection(t *testing.T)
func TestOutputBasedDetection(t *testing.T)
func TestPlatformSelection(t *testing.T)
```

### Test Utilities
```go
// pkg/testutil/testutil.go
package testutil

// Helper to capture output
func CaptureOutput(f func()) string

// Helper to create test config
func TestConfig() *config.Config

// Helper to create temp config file
func TempConfigFile(content string) (string, func())

// Mock time for testing
type MockClock struct {
    Current time.Time
}
```

### Testing Best Practices
1. **No Real Claude Code Execution**: All tests use mocks
2. **Deterministic Tests**: Use fixed time, no sleeps
3. **Table-Driven Tests**: For multiple scenarios
4. **Isolated Tests**: No shared state
5. **Fast Tests**: No network calls, no file I/O (except config tests)

## Error Handling

### Error Categories
1. **Configuration Errors**: Fatal, exit immediately
2. **Process Errors**: Log and propagate exit code
3. **Notification Errors**: Log but don't interrupt
4. **Platform Errors**: Fall back gracefully

### Graceful Degradation
- If idle detection fails, fall back to output-based
- If notifications fail, continue wrapping
- If config file missing, use defaults
- If pattern compile fails, skip that pattern

## Security Considerations

1. **Command Injection**: Use exec.Command properly
2. **Config Validation**: Validate regex patterns
3. **Ntfy Authentication**: Support auth tokens
4. **No Sensitive Data**: Don't log command output
5. **Signal Handling**: Proper cleanup on termination

## Performance Considerations

1. **Zero Buffering**: Direct I/O copying
2. **Efficient Regex**: Compile once, use many
3. **Minimal Overhead**: < 1ms latency target
4. **Memory Usage**: < 10MB RSS
5. **CPU Usage**: < 1% when idle

## Future Considerations

1. **Windows Support**: Would need different idle detection
2. **Multiple Notifiers**: Slack, Discord, etc.
3. **Plugins**: Custom pattern handlers
4. **Metrics**: Usage statistics
5. **Config Hot-Reload**: Update without restart
