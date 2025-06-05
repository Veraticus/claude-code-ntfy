package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for claude-code-ntfy
type Config struct {
	// Notification settings
	NtfyTopic   string        `yaml:"ntfy_topic" env:"CLAUDE_NOTIFY_TOPIC"`
	NtfyServer  string        `yaml:"ntfy_server" env:"CLAUDE_NOTIFY_SERVER"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env:"CLAUDE_NOTIFY_IDLE_TIMEOUT"`

	// Behavior flags
	Quiet       bool `yaml:"quiet" env:"CLAUDE_NOTIFY_QUIET"`
	ForceNotify bool `yaml:"force_notify" env:"CLAUDE_NOTIFY_FORCE"`

	// Pattern configuration
	Patterns []Pattern `yaml:"patterns"`

	// Rate limiting
	RateLimit RateLimitConfig `yaml:"rate_limit"`

	// Batching
	BatchWindow time.Duration `yaml:"batch_window"`
}

// Pattern represents a configurable pattern.
type Pattern struct {
	Name        string         `yaml:"name"`
	Regex       string         `yaml:"regex"`
	Description string         `yaml:"description"`
	Enabled     bool           `yaml:"enabled"`
	compiled    *regexp.Regexp `yaml:"-"`
}

// CompiledRegex returns the compiled regular expression
func (p *Pattern) CompiledRegex() *regexp.Regexp {
	return p.compiled
}

// SetCompiledRegex sets the compiled regular expression
func (p *Pattern) SetCompiledRegex(re *regexp.Regexp) {
	p.compiled = re
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Window      time.Duration `yaml:"window"`
	MaxMessages int           `yaml:"max_messages"`
}

// DefaultConfig returns the default configuration
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

	// Compile regex patterns
	if err := compilePatterns(cfg); err != nil {
		return nil, fmt.Errorf("failed to compile patterns: %w", err)
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

	if timeout := os.Getenv("CLAUDE_NOTIFY_IDLE_TIMEOUT"); timeout != "" {
		d, err := time.ParseDuration(timeout)
		if err != nil {
			return fmt.Errorf("invalid CLAUDE_NOTIFY_IDLE_TIMEOUT: %w", err)
		}
		cfg.IdleTimeout = d
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

	if force := os.Getenv("CLAUDE_NOTIFY_FORCE"); force != "" {
		switch force {
		case "true", "1", "yes":
			cfg.ForceNotify = true
		case "false", "0", "no":
			cfg.ForceNotify = false
		default:
			return fmt.Errorf("invalid CLAUDE_NOTIFY_FORCE value: %q (use true/false)", force)
		}
	}

	return nil
}

// compilePatterns compiles all regex patterns
func compilePatterns(cfg *Config) error {
	for i := range cfg.Patterns {
		pattern := &cfg.Patterns[i]
		if pattern.Enabled && pattern.Regex != "" {
			re, err := regexp.Compile(pattern.Regex)
			if err != nil {
				return fmt.Errorf("failed to compile pattern %q: %w", pattern.Name, err)
			}
			pattern.SetCompiledRegex(re)
		}
	}
	return nil
}

// validate validates the configuration
func validate(cfg *Config) error {
	if cfg.NtfyTopic == "" && !cfg.Quiet {
		return fmt.Errorf("ntfy_topic is required when not in quiet mode")
	}

	if cfg.RateLimit.MaxMessages < 0 {
		return fmt.Errorf("rate_limit.max_messages must be non-negative")
	}

	if cfg.RateLimit.Window < 0 {
		return fmt.Errorf("rate_limit.window must be non-negative")
	}

	if cfg.BatchWindow < 0 {
		return fmt.Errorf("batch_window must be non-negative")
	}

	return nil
}
