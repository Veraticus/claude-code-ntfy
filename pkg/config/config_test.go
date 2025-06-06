package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Check default values
	if cfg.NtfyServer != "https://ntfy.sh" {
		t.Errorf("expected NtfyServer to be https://ntfy.sh but got %s", cfg.NtfyServer)
	}
	if cfg.BackstopTimeout != 30*time.Second {
		t.Errorf("expected BackstopTimeout to be 30s but got %v", cfg.BackstopTimeout)
	}
	if !cfg.StartupNotify {
		t.Error("expected StartupNotify to be true by default")
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Save original env and restore after test
	origTopic := os.Getenv("CLAUDE_NOTIFY_TOPIC")
	origServer := os.Getenv("CLAUDE_NOTIFY_SERVER")
	origBackstopTimeout := os.Getenv("CLAUDE_NOTIFY_BACKSTOP_TIMEOUT")
	origQuiet := os.Getenv("CLAUDE_NOTIFY_QUIET")
	origClaudePath := os.Getenv("CLAUDE_NOTIFY_CLAUDE_PATH")
	origDefaultArgs := os.Getenv("CLAUDE_NOTIFY_DEFAULT_ARGS")
	origStartup := os.Getenv("CLAUDE_NOTIFY_STARTUP")
	defer func() {
		_ = os.Setenv("CLAUDE_NOTIFY_TOPIC", origTopic)
		_ = os.Setenv("CLAUDE_NOTIFY_SERVER", origServer)
		_ = os.Setenv("CLAUDE_NOTIFY_BACKSTOP_TIMEOUT", origBackstopTimeout)
		_ = os.Setenv("CLAUDE_NOTIFY_QUIET", origQuiet)
		_ = os.Setenv("CLAUDE_NOTIFY_CLAUDE_PATH", origClaudePath)
		_ = os.Setenv("CLAUDE_NOTIFY_DEFAULT_ARGS", origDefaultArgs)
		_ = os.Setenv("CLAUDE_NOTIFY_STARTUP", origStartup)
	}()

	tests := []struct {
		name      string
		envVars   map[string]string
		checkFunc func(*testing.T, *Config)
		wantErr   bool
	}{
		{
			name: "valid environment variables",
			envVars: map[string]string{
				"CLAUDE_NOTIFY_TOPIC":            "test-topic",
				"CLAUDE_NOTIFY_SERVER":           "https://test.server",
				"CLAUDE_NOTIFY_BACKSTOP_TIMEOUT": "5m",
				"CLAUDE_NOTIFY_QUIET":            "true",
				"CLAUDE_NOTIFY_CLAUDE_PATH":      "/usr/local/bin/claude",
			},
			checkFunc: func(t *testing.T, cfg *Config) {
				if cfg.NtfyTopic != "test-topic" {
					t.Errorf("expected NtfyTopic to be test-topic but got %s", cfg.NtfyTopic)
				}
				if cfg.NtfyServer != "https://test.server" {
					t.Errorf("expected NtfyServer to be https://test.server but got %s", cfg.NtfyServer)
				}
				if cfg.BackstopTimeout != 5*time.Minute {
					t.Errorf("expected BackstopTimeout to be 5m but got %v", cfg.BackstopTimeout)
				}
				if !cfg.Quiet {
					t.Error("expected Quiet to be true")
				}
				if cfg.ClaudePath != "/usr/local/bin/claude" {
					t.Errorf("expected ClaudePath to be /usr/local/bin/claude but got %s", cfg.ClaudePath)
				}
			},
		},
		{
			name: "invalid timeout",
			envVars: map[string]string{
				"CLAUDE_NOTIFY_BACKSTOP_TIMEOUT": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid quiet value",
			envVars: map[string]string{
				"CLAUDE_NOTIFY_QUIET": "maybe",
			},
			wantErr: true,
		},
		{
			name: "boolean variations",
			envVars: map[string]string{
				"CLAUDE_NOTIFY_QUIET": "yes",
			},
			checkFunc: func(t *testing.T, cfg *Config) {
				if !cfg.Quiet {
					t.Error("expected Quiet to be true for 'yes'")
				}
			},
		},
		{
			name: "default args parsing",
			envVars: map[string]string{
				"CLAUDE_NOTIFY_TOPIC":        "test-topic",
				"CLAUDE_NOTIFY_DEFAULT_ARGS": "--verbose,--model,gpt-4,--timeout,30",
			},
			checkFunc: func(t *testing.T, cfg *Config) {
				expected := []string{"--verbose", "--model", "gpt-4", "--timeout", "30"}
				if len(cfg.DefaultClaudeArgs) != len(expected) {
					t.Errorf("expected %d default args but got %d", len(expected), len(cfg.DefaultClaudeArgs))
				}
				for i, arg := range expected {
					if i < len(cfg.DefaultClaudeArgs) && cfg.DefaultClaudeArgs[i] != arg {
						t.Errorf("expected default arg[%d] to be %q but got %q", i, arg, cfg.DefaultClaudeArgs[i])
					}
				}
			},
		},
		{
			name: "default args with spaces",
			envVars: map[string]string{
				"CLAUDE_NOTIFY_TOPIC":        "test-topic",
				"CLAUDE_NOTIFY_DEFAULT_ARGS": " --verbose , --debug , --quiet ",
			},
			checkFunc: func(t *testing.T, cfg *Config) {
				expected := []string{"--verbose", "--debug", "--quiet"}
				if len(cfg.DefaultClaudeArgs) != len(expected) {
					t.Errorf("expected %d default args but got %d", len(expected), len(cfg.DefaultClaudeArgs))
				}
				for i, arg := range expected {
					if i < len(cfg.DefaultClaudeArgs) && cfg.DefaultClaudeArgs[i] != arg {
						t.Errorf("expected default arg[%d] to be %q but got %q", i, arg, cfg.DefaultClaudeArgs[i])
					}
				}
			},
		},
		{
			name: "empty default args",
			envVars: map[string]string{
				"CLAUDE_NOTIFY_TOPIC":        "test-topic",
				"CLAUDE_NOTIFY_DEFAULT_ARGS": ",,",
			},
			checkFunc: func(t *testing.T, cfg *Config) {
				if len(cfg.DefaultClaudeArgs) != 0 {
					t.Errorf("expected no default args but got %d", len(cfg.DefaultClaudeArgs))
				}
			},
		},
		{
			name: "startup notification parsing",
			envVars: map[string]string{
				"CLAUDE_NOTIFY_TOPIC":   "test-topic",
				"CLAUDE_NOTIFY_STARTUP": "false",
			},
			checkFunc: func(t *testing.T, cfg *Config) {
				if cfg.StartupNotify {
					t.Error("expected StartupNotify to be false")
				}
			},
		},
		{
			name: "invalid startup value",
			envVars: map[string]string{
				"CLAUDE_NOTIFY_STARTUP": "maybe",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars first
			_ = os.Unsetenv("CLAUDE_NOTIFY_TOPIC")
			_ = os.Unsetenv("CLAUDE_NOTIFY_SERVER")
			_ = os.Unsetenv("CLAUDE_NOTIFY_BACKSTOP_TIMEOUT")
			_ = os.Unsetenv("CLAUDE_NOTIFY_QUIET")
			_ = os.Unsetenv("CLAUDE_NOTIFY_CLAUDE_PATH")
			_ = os.Unsetenv("CLAUDE_NOTIFY_DEFAULT_ARGS")
			_ = os.Unsetenv("CLAUDE_NOTIFY_STARTUP")
			_ = os.Unsetenv("CLAUDE_NOTIFY_CONFIG")

			// Set test env vars
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}

			// Set a non-existent config path to prevent loading user's config
			if _, hasConfig := tt.envVars["CLAUDE_NOTIFY_CONFIG"]; !hasConfig {
				_ = os.Setenv("CLAUDE_NOTIFY_CONFIG", "/tmp/non-existent-test-config.yaml")
			}

			// Load config
			cfg, err := Load()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.checkFunc != nil && cfg != nil {
					tt.checkFunc(t, cfg)
				}
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create a temporary directory for test configs
	tmpDir, err := os.MkdirTemp("", "claude-notify-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tests := []struct {
		name      string
		content   string
		checkFunc func(*testing.T, *Config)
		wantErr   bool
	}{
		{
			name: "valid config file",
			content: `
ntfy_topic: "file-topic"
ntfy_server: "https://file.server"
backstop_timeout: "10s"
quiet: true
claude_path: "/opt/claude/bin/claude"
default_claude_args:
  - "--verbose"
  - "--model"
  - "gpt-4"
`,
			checkFunc: func(t *testing.T, cfg *Config) {
				if cfg.NtfyTopic != "file-topic" {
					t.Errorf("expected NtfyTopic to be file-topic but got %s", cfg.NtfyTopic)
				}
				if cfg.BackstopTimeout != 10*time.Second {
					t.Errorf("expected BackstopTimeout to be 10s but got %v", cfg.BackstopTimeout)
				}
				if !cfg.Quiet {
					t.Errorf("expected Quiet to be true")
				}
				if cfg.ClaudePath != "/opt/claude/bin/claude" {
					t.Errorf("expected ClaudePath to be /opt/claude/bin/claude but got %s", cfg.ClaudePath)
				}
				expected := []string{"--verbose", "--model", "gpt-4"}
				if len(cfg.DefaultClaudeArgs) != len(expected) {
					t.Errorf("expected %d default args but got %d", len(expected), len(cfg.DefaultClaudeArgs))
				}
				for i, arg := range expected {
					if i < len(cfg.DefaultClaudeArgs) && cfg.DefaultClaudeArgs[i] != arg {
						t.Errorf("expected default arg[%d] to be %q but got %q", i, arg, cfg.DefaultClaudeArgs[i])
					}
				}
			},
		},
		{
			name:    "invalid yaml",
			content: "invalid: yaml: content:\n  bad indentation",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config file
			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.content), 0600); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			// Set config path env var
			_ = os.Setenv("CLAUDE_NOTIFY_CONFIG", configPath)
			defer func() { _ = os.Unsetenv("CLAUDE_NOTIFY_CONFIG") }()

			// Clear other env vars to avoid interference
			_ = os.Unsetenv("CLAUDE_NOTIFY_TOPIC")
			_ = os.Unsetenv("CLAUDE_NOTIFY_SERVER")

			// Load config
			cfg, err := Load()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.checkFunc != nil && cfg != nil {
					tt.checkFunc(t, cfg)
				}
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Config
		wantErr  bool
		errorMsg string
	}{
		{
			name: "valid config",
			cfg: &Config{
				NtfyTopic:       "test-topic",
				BackstopTimeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing topic when not quiet",
			cfg: &Config{
				NtfyTopic: "",
				Quiet:     false,
			},
			wantErr:  true,
			errorMsg: "ntfy_topic is required",
		},
		{
			name: "missing topic allowed when quiet",
			cfg: &Config{
				NtfyTopic: "",
				Quiet:     true,
			},
			wantErr: false,
		},
		{
			name: "negative backstop timeout",
			cfg: &Config{
				NtfyTopic:       "test",
				BackstopTimeout: -1 * time.Second,
			},
			wantErr:  true,
			errorMsg: "must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q but got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	// Save original env and restore after test
	origConfig := os.Getenv("CLAUDE_NOTIFY_CONFIG")
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	origHome := os.Getenv("HOME")
	defer func() {
		_ = os.Setenv("CLAUDE_NOTIFY_CONFIG", origConfig)
		_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		_ = os.Setenv("HOME", origHome)
	}()

	tests := []struct {
		name        string
		envVars     map[string]string
		wantContain string
	}{
		{
			name: "explicit config path",
			envVars: map[string]string{
				"CLAUDE_NOTIFY_CONFIG": "/custom/path/config.yaml",
			},
			wantContain: "/custom/path/config.yaml",
		},
		{
			name: "XDG config path",
			envVars: map[string]string{
				"XDG_CONFIG_HOME": "/xdg/config",
			},
			wantContain: "/xdg/config/claude-code-ntfy/config.yaml",
		},
		{
			name:        "home directory fallback",
			envVars:     map[string]string{},
			wantContain: ".config/claude-code-ntfy/config.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear env vars
			_ = os.Unsetenv("CLAUDE_NOTIFY_CONFIG")
			_ = os.Unsetenv("XDG_CONFIG_HOME")

			// Set test env vars
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}

			path := getConfigPath()
			if !contains(path, tt.wantContain) {
				t.Errorf("expected path to contain %q but got %q", tt.wantContain, path)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
