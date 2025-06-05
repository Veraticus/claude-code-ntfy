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

### Go Install

```bash
go install github.com/Veraticus/claude-code-ntfy/cmd/claude-code-ntfy@latest
```

### Build from Source

```bash
git clone https://github.com/Veraticus/claude-code-ntfy
cd claude-code-ntfy
make build
```

### NixOS / Nix

Claude Code Ntfy can be installed on NixOS or any system with Nix package manager. This installation method provides a wrapper that automatically intercepts the `claude` command.

#### Using Nix Flakes

```bash
# Run directly without installation
nix run github:Veraticus/claude-code-ntfy -- --help

# Install to user profile (hijacks the claude command)
nix profile install github:Veraticus/claude-code-ntfy

# Or in your flake.nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    claude-code-ntfy.url = "github:Veraticus/claude-code-ntfy";
  };

  outputs = { self, nixpkgs, claude-code-ntfy }: {
    # For NixOS system configuration
    nixosConfigurations.mysystem = nixpkgs.lib.nixosSystem {
      modules = [
        claude-code-ntfy.nixosModules.default
        {
          programs.claude-code-ntfy.enable = true;
        }
      ];
    };
  };
}
```

#### Using Traditional Nix

```bash
# Build from source
nix-build -E 'with import <nixpkgs> {}; callPackage ./default.nix {}'

# Install using nix-env
nix-env -f ./default.nix -i

# Or add to configuration.nix
{ pkgs, ... }:
let
  claude-code-ntfy = pkgs.callPackage (pkgs.fetchFromGitHub {
    owner = "Veraticus";
    repo = "claude-code-ntfy";
    rev = "ba76a6ce3b0bce2b17e5b9d528b8f4f80ec93cf8";
    sha256 = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";
  } + "/default.nix") { };
in
{
  environment.systemPackages = [ claude-code-ntfy ];
}
```

#### Home Manager Configuration

For user-level installation with configuration management:

```nix
{
  programs.claude-code-ntfy = {
    enable = true;
    
    # Optional: Define configuration
    settings = {
      ntfy_topic = "my-claude-notifications";
      ntfy_server = "https://ntfy.sh";
      idle_timeout = "2m";
      patterns = [
        {
          name = "error";
          regex = "(?i)(error|failed|exception)";
          enabled = true;
        }
      ];
      rate_limit = {
        window = "1m";
        max_messages = 5;
      };
    };
  };
}
```

The Nix installation:
- **Automatically hijacks the `claude` command** - no need to use `claude-code-ntfy`
- Finds and wraps the original Claude Code installation from npm
- Works with both NixOS system-wide and Home Manager user installations
- Creates config at `~/.config/claude-code-ntfy/config.yaml` if settings are provided

## Quick Start

1. Set your ntfy topic:
   ```bash
   export CLAUDE_NOTIFY_TOPIC="my-claude-notifications"
   ```

2. Run Claude Code:
   ```bash
   # If installed via Nix (automatically uses wrapper)
   claude
   
   # If installed via Go or built from source
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