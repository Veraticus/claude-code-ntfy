package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for claude-code-ntfy
type Config struct {
	// Notification settings
	NtfyTopic  string `yaml:"ntfy_topic" env:"CLAUDE_NOTIFY_TOPIC"`
	NtfyServer string `yaml:"ntfy_server" env:"CLAUDE_NOTIFY_SERVER"`

	// Behavior flags
	Quiet             bool     `yaml:"quiet" env:"CLAUDE_NOTIFY_QUIET"`
	StartupNotify     bool     `yaml:"startup_notify" env:"CLAUDE_NOTIFY_STARTUP"`
	DefaultClaudeArgs []string `yaml:"default_claude_args"`

	// Backstop notification - send notification after inactivity
	BackstopTimeout time.Duration `yaml:"backstop_timeout" env:"CLAUDE_NOTIFY_BACKSTOP_TIMEOUT"`

	// Claude path configuration
	ClaudePath string `yaml:"claude_path" env:"CLAUDE_NOTIFY_CLAUDE_PATH"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		NtfyServer:      "https://ntfy.sh",
		BackstopTimeout: 30 * time.Second,
		StartupNotify:   true, // Default to true so users know notifications are working
	}
}

// Load loads configuration from file and environment
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Try to load from config file
	configPath := getConfigPath()
	if configPath != "" {
		if err := loadFromFile(cfg, configPath); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	}

	// Override with environment variables
	if err := loadFromEnv(cfg); err != nil {
		return nil, fmt.Errorf("failed to load from environment: %w", err)
	}

	// Validate configuration
	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// getConfigPath returns the config file path
func getConfigPath() string {
	// Check for explicit config path
	if path := os.Getenv("CLAUDE_NOTIFY_CONFIG"); path != "" {
		return path
	}

	// Check XDG config directory
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "claude-code-ntfy", "config.yaml")
	}

	// Fall back to home directory
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "claude-code-ntfy", "config.yaml")
	}

	return ""
}

// loadFromFile loads configuration from a YAML file
func loadFromFile(cfg *Config, path string) error {
	// #nosec G304 - The config file path comes from trusted sources (env var or standard locations)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, cfg)
}

// loadFromEnv loads configuration from environment variables
func loadFromEnv(cfg *Config) error {
	if topic := os.Getenv("CLAUDE_NOTIFY_TOPIC"); topic != "" {
		cfg.NtfyTopic = topic
	}

	if server := os.Getenv("CLAUDE_NOTIFY_SERVER"); server != "" {
		cfg.NtfyServer = server
	}

	if timeout := os.Getenv("CLAUDE_NOTIFY_BACKSTOP_TIMEOUT"); timeout != "" {
		d, err := time.ParseDuration(timeout)
		if err != nil {
			return fmt.Errorf("invalid CLAUDE_NOTIFY_BACKSTOP_TIMEOUT: %w", err)
		}
		cfg.BackstopTimeout = d
	}

	if quiet := os.Getenv("CLAUDE_NOTIFY_QUIET"); quiet != "" {
		switch quiet {
		case "true", "1", "yes":
			cfg.Quiet = true
		case "false", "0", "no":
			cfg.Quiet = false
		default:
			return fmt.Errorf("invalid CLAUDE_NOTIFY_QUIET value: %q (use true/false)", quiet)
		}
	}

	if startup := os.Getenv("CLAUDE_NOTIFY_STARTUP"); startup != "" {
		switch startup {
		case "true", "1", "yes":
			cfg.StartupNotify = true
		case "false", "0", "no":
			cfg.StartupNotify = false
		default:
			return fmt.Errorf("invalid CLAUDE_NOTIFY_STARTUP value: %q (use true/false)", startup)
		}
	}

	if claudePath := os.Getenv("CLAUDE_NOTIFY_CLAUDE_PATH"); claudePath != "" {
		cfg.ClaudePath = claudePath
	}

	if defaultArgs := os.Getenv("CLAUDE_NOTIFY_DEFAULT_ARGS"); defaultArgs != "" {
		// Split by comma and trim whitespace
		args := strings.Split(defaultArgs, ",")
		for i, arg := range args {
			args[i] = strings.TrimSpace(arg)
		}
		// Filter out empty strings
		var filteredArgs []string
		for _, arg := range args {
			if arg != "" {
				filteredArgs = append(filteredArgs, arg)
			}
		}
		cfg.DefaultClaudeArgs = filteredArgs
	}

	return nil
}

// validate validates the configuration
func validate(cfg *Config) error {
	if cfg.NtfyTopic == "" && !cfg.Quiet {
		return fmt.Errorf("ntfy_topic is required when not in quiet mode")
	}

	if cfg.BackstopTimeout < 0 {
		return fmt.Errorf("backstop_timeout must be non-negative")
	}

	return nil
}
