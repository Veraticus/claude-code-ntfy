# claude-code-ntfy

A transparent wrapper for Claude Code that monitors output and sends notifications via [ntfy.sh](https://ntfy.sh) based on configurable patterns and user activity.

## Features

- = Smart notifications based on output patterns
- >� Transparent wrapping - preserves all Claude Code functionality
- =� User idle detection to avoid interruptions
- <� Configurable regex patterns for triggers
- =� Rate limiting and notification batching
- =' Cross-platform support (Linux/macOS)

## Installation

```bash
go install github.com/Veraticus/claude-code-ntfy/cmd/claude-code-ntfy@latest
```

Or build from source:

```bash
git clone https://github.com/Veraticus/claude-code-ntfy
cd claude-code-ntfy
make build
```

## Quick Start

1. Set your ntfy topic:
   ```bash
   export CLAUDE_NOTIFY_TOPIC="my-claude-notifications"
   ```

2. Run Claude Code through the wrapper:
   ```bash
   claude-code-ntfy
   ```

## Configuration

Configure via environment variables:

- `CLAUDE_NOTIFY_TOPIC` - Ntfy topic for notifications (required)
- `CLAUDE_NOTIFY_SERVER` - Ntfy server URL (default: https://ntfy.sh)
- `CLAUDE_NOTIFY_IDLE_TIMEOUT` - User idle timeout (default: 2m)
- `CLAUDE_NOTIFY_QUIET` - Disable notifications (true/false)
- `CLAUDE_NOTIFY_FORCE` - Force notifications even when active (true/false)

Or use a config file at `~/.config/claude-code-ntfy/config.yaml`:

```yaml
ntfy_topic: "my-claude-notifications"
ntfy_server: "https://ntfy.sh"
idle_timeout: "2m"

patterns:
  - name: "bell"
    regex: '\x07'
    enabled: true
  - name: "question"
    regex: '\?\s*$'
    enabled: true
  - name: "error"
    regex: '(?i)(error|failed|exception)'
    enabled: true

rate_limit:
  window: "1m"
  max_messages: 5

batch_window: "5s"
```

## Development

Simple development workflow:

```bash
# One-time setup
make install-tools  # Install required development tools

# During development
make test          # Run tests with race detection

# Before committing
make fix           # Auto-fix issues
make verify        # Run all checks
```

See [docs/development.md](docs/development.md) for detailed development guide.

## License

MIT