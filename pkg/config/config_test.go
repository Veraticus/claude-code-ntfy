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
	if cfg.IdleTimeout != 2*time.Minute {
		t.Errorf("expected IdleTimeout to be 2m but got %v", cfg.IdleTimeout)
	}
	if len(cfg.Patterns) != 4 {
		t.Errorf("expected 4 default patterns but got %d", len(cfg.Patterns))
	}
	if cfg.RateLimit.Window != 1*time.Minute {
		t.Errorf("expected RateLimit.Window to be 1m but got %v", cfg.RateLimit.Window)
	}
	if cfg.RateLimit.MaxMessages != 5 {
		t.Errorf("expected RateLimit.MaxMessages to be 5 but got %d", cfg.RateLimit.MaxMessages)
	}
	if cfg.BatchWindow != 5*time.Second {
		t.Errorf("expected BatchWindow to be 5s but got %v", cfg.BatchWindow)
	}

	// Check pattern defaults
	expectedPatterns := []struct {
		name  string
		regex string
	}{
		{"bell", `\x07`},
		{"question", `\?\s*$`},
		{"error", `(?i)(\berror:|\bfailed:|\bexception:|✗|❌)\s`},
		{"completion", `(?i)(✓|✅|task completed|build succeeded|all tests pass)`},
	}

	for i, expected := range expectedPatterns {
		if i >= len(cfg.Patterns) {
			t.Errorf("pattern %d missing", i)
			continue
		}
		pattern := cfg.Patterns[i]
		if pattern.Name != expected.name {
			t.Errorf("pattern %d: expected name %s but got %s", i, expected.name, pattern.Name)
		}
		if pattern.Regex != expected.regex {
			t.Errorf("pattern %d: expected regex %s but got %s", i, expected.regex, pattern.Regex)
		}
		if !pattern.Enabled {
			t.Errorf("pattern %d: expected to be enabled", i)
		}
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Save original env and restore after test
	origTopic := os.Getenv("CLAUDE_NOTIFY_TOPIC")
	origServer := os.Getenv("CLAUDE_NOTIFY_SERVER")
	origTimeout := os.Getenv("CLAUDE_NOTIFY_IDLE_TIMEOUT")
	origQuiet := os.Getenv("CLAUDE_NOTIFY_QUIET")
	origForce := os.Getenv("CLAUDE_NOTIFY_FORCE")
	origStartup := os.Getenv("CLAUDE_NOTIFY_STARTUP")
	origDefaultArgs := os.Getenv("CLAUDE_NOTIFY_DEFAULT_ARGS")
	defer func() {
		_ = os.Setenv("CLAUDE_NOTIFY_TOPIC", origTopic)
		_ = os.Setenv("CLAUDE_NOTIFY_SERVER", origServer)
		_ = os.Setenv("CLAUDE_NOTIFY_IDLE_TIMEOUT", origTimeout)
		_ = os.Setenv("CLAUDE_NOTIFY_QUIET", origQuiet)
		_ = os.Setenv("CLAUDE_NOTIFY_FORCE", origForce)
		_ = os.Setenv("CLAUDE_NOTIFY_STARTUP", origStartup)
		_ = os.Setenv("CLAUDE_NOTIFY_DEFAULT_ARGS", origDefaultArgs)
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
				"CLAUDE_NOTIFY_TOPIC":        "test-topic",
				"CLAUDE_NOTIFY_SERVER":       "https://test.server",
				"CLAUDE_NOTIFY_IDLE_TIMEOUT": "5m",
				"CLAUDE_NOTIFY_QUIET":        "true",
				"CLAUDE_NOTIFY_FORCE":        "false",
				"CLAUDE_NOTIFY_STARTUP":      "true",
			},
			checkFunc: func(t *testing.T, cfg *Config) {
				if cfg.NtfyTopic != "test-topic" {
					t.Errorf("expected NtfyTopic to be test-topic but got %s", cfg.NtfyTopic)
				}
				if cfg.NtfyServer != "https://test.server" {
					t.Errorf("expected NtfyServer to be https://test.server but got %s", cfg.NtfyServer)
				}
				if cfg.IdleTimeout != 5*time.Minute {
					t.Errorf("expected IdleTimeout to be 5m but got %v", cfg.IdleTimeout)
				}
				if !cfg.Quiet {
					t.Error("expected Quiet to be true")
				}
				if cfg.ForceNotify {
					t.Error("expected ForceNotify to be false")
				}
				if !cfg.StartupNotify {
					t.Error("expected StartupNotify to be true")
				}
			},
		},
		{
			name: "invalid timeout",
			envVars: map[string]string{
				"CLAUDE_NOTIFY_IDLE_TIMEOUT": "invalid",
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
			name: "invalid force value",
			envVars: map[string]string{
				"CLAUDE_NOTIFY_FORCE": "perhaps",
			},
			wantErr: true,
		},
		{
			name: "invalid startup value",
			envVars: map[string]string{
				"CLAUDE_NOTIFY_STARTUP": "maybe",
			},
			wantErr: true,
		},
		{
			name: "empty default args",
			envVars: map[string]string{
				"CLAUDE_NOTIFY_TOPIC":        "test-topic",
				"CLAUDE_NOTIFY_DEFAULT_ARGS": ",,,",
			},
			checkFunc: func(t *testing.T, cfg *Config) {
				if len(cfg.DefaultClaudeArgs) != 0 {
					t.Errorf("expected no default args for empty string, got %v", cfg.DefaultClaudeArgs)
				}
			},
		},
		{
			name: "boolean variations",
			envVars: map[string]string{
				"CLAUDE_NOTIFY_QUIET":        "yes",
				"CLAUDE_NOTIFY_FORCE":        "1",
				"CLAUDE_NOTIFY_STARTUP":      "no",
				"CLAUDE_NOTIFY_DEFAULT_ARGS": "--model,claude-3-opus,--verbose",
			},
			checkFunc: func(t *testing.T, cfg *Config) {
				if !cfg.Quiet {
					t.Error("expected Quiet to be true for 'yes'")
				}
				if !cfg.ForceNotify {
					t.Error("expected ForceNotify to be true for '1'")
				}
				if cfg.StartupNotify {
					t.Error("expected StartupNotify to be false for 'no'")
				}
				if len(cfg.DefaultClaudeArgs) != 3 {
					t.Errorf("expected 3 default args, got %d", len(cfg.DefaultClaudeArgs))
				} else {
					expected := []string{"--model", "claude-3-opus", "--verbose"}
					for i, arg := range expected {
						if cfg.DefaultClaudeArgs[i] != arg {
							t.Errorf("expected arg[%d] to be %q, got %q", i, arg, cfg.DefaultClaudeArgs[i])
						}
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars first
			_ = os.Unsetenv("CLAUDE_NOTIFY_TOPIC")
			_ = os.Unsetenv("CLAUDE_NOTIFY_SERVER")
			_ = os.Unsetenv("CLAUDE_NOTIFY_IDLE_TIMEOUT")
			_ = os.Unsetenv("CLAUDE_NOTIFY_QUIET")
			_ = os.Unsetenv("CLAUDE_NOTIFY_FORCE")
			_ = os.Unsetenv("CLAUDE_NOTIFY_STARTUP")
			_ = os.Unsetenv("CLAUDE_NOTIFY_DEFAULT_ARGS")
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
idle_timeout: "10m"
quiet: true
force_notify: false
startup_notify: true
default_claude_args:
  - "--model"
  - "claude-3-opus"
patterns:
  - name: "custom"
    regex: "CUSTOM"
    description: "Custom pattern"
    enabled: true
rate_limit:
  window: "2m"
  max_messages: 10
batch_window: "10s"
`,
			checkFunc: func(t *testing.T, cfg *Config) {
				if cfg.NtfyTopic != "file-topic" {
					t.Errorf("expected NtfyTopic to be file-topic but got %s", cfg.NtfyTopic)
				}
				if cfg.IdleTimeout != 10*time.Minute {
					t.Errorf("expected IdleTimeout to be 10m but got %v", cfg.IdleTimeout)
				}
				if !cfg.StartupNotify {
					t.Errorf("expected StartupNotify to be true")
				}
				if len(cfg.DefaultClaudeArgs) != 2 || cfg.DefaultClaudeArgs[0] != "--model" || cfg.DefaultClaudeArgs[1] != "claude-3-opus" {
					t.Errorf("expected default args [--model claude-3-opus], got %v", cfg.DefaultClaudeArgs)
				}
				if len(cfg.Patterns) != 1 {
					t.Errorf("expected 1 pattern but got %d", len(cfg.Patterns))
				} else {
					if cfg.Patterns[0].Name != "custom" {
						t.Errorf("expected pattern name to be custom but got %s", cfg.Patterns[0].Name)
					}
				}
				if cfg.RateLimit.Window != 2*time.Minute {
					t.Errorf("expected RateLimit.Window to be 2m but got %v", cfg.RateLimit.Window)
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

func TestCompilePatterns(t *testing.T) {
	tests := []struct {
		name     string
		patterns []Pattern
		wantErr  bool
		errorMsg string
	}{
		{
			name: "valid patterns",
			patterns: []Pattern{
				{Name: "test1", Regex: `\d+`, Enabled: true},
				{Name: "test2", Regex: `[a-z]+`, Enabled: true},
			},
			wantErr: false,
		},
		{
			name: "invalid regex",
			patterns: []Pattern{
				{Name: "bad", Regex: `[`, Enabled: true},
			},
			wantErr:  true,
			errorMsg: "failed to compile pattern",
		},
		{
			name: "disabled pattern",
			patterns: []Pattern{
				{Name: "disabled", Regex: `[`, Enabled: false},
			},
			wantErr: false, // Should not compile disabled patterns
		},
		{
			name: "empty regex",
			patterns: []Pattern{
				{Name: "empty", Regex: "", Enabled: true},
			},
			wantErr: false, // Empty regex is valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Patterns: tt.patterns,
			}

			err := compilePatterns(cfg)

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
				// Check that enabled patterns were compiled
				for _, p := range cfg.Patterns {
					if p.Enabled && p.Regex != "" {
						if p.CompiledRegex() == nil {
							t.Errorf("pattern %s was not compiled", p.Name)
						}
					}
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
				NtfyTopic: "test-topic",
				RateLimit: RateLimitConfig{
					Window:      1 * time.Minute,
					MaxMessages: 5,
				},
				BatchWindow: 5 * time.Second,
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
			name: "negative max messages",
			cfg: &Config{
				NtfyTopic: "test",
				RateLimit: RateLimitConfig{
					MaxMessages: -1,
				},
			},
			wantErr:  true,
			errorMsg: "must be non-negative",
		},
		{
			name: "negative window",
			cfg: &Config{
				NtfyTopic: "test",
				RateLimit: RateLimitConfig{
					Window: -1 * time.Second,
				},
			},
			wantErr:  true,
			errorMsg: "must be non-negative",
		},
		{
			name: "negative batch window",
			cfg: &Config{
				NtfyTopic:   "test",
				BatchWindow: -1 * time.Second,
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

// Additional helper to test pattern matching after compilation
func TestCompiledPatternMatching(t *testing.T) {
	cfg := DefaultConfig()
	err := compilePatterns(cfg)
	if err != nil {
		t.Fatalf("failed to compile patterns: %v", err)
	}

	tests := []struct {
		patternName string
		input       string
		shouldMatch bool
	}{
		{"bell", "test\x07", true},
		{"bell", "no bell here", false},
		{"question", "Is this a question?", true},
		{"question", "This is a statement.", false},
		{"error", "error: file not found ", true},
		{"error", "Failed: build error ", true},
		{"error", "Exception: null pointer ", true},
		{"error", "All is well", false},
		{"error", "✗ Test failed ", true},
		{"error", "❌ Build failed ", true},
		{"completion", "✓ Done", true},
		{"completion", "✅ All tests pass", true},
		{"completion", "task completed successfully", true},
		{"completion", "build succeeded", true},
		{"completion", "Still working", false},
	}

	for _, tt := range tests {
		t.Run(tt.patternName+"_"+tt.input, func(t *testing.T) {
			// Find the pattern
			var pattern *Pattern
			for i := range cfg.Patterns {
				if cfg.Patterns[i].Name == tt.patternName {
					pattern = &cfg.Patterns[i]
					break
				}
			}
			if pattern == nil {
				t.Fatalf("pattern %s not found", tt.patternName)
			}

			// Test the match
			regex := pattern.CompiledRegex()
			if regex == nil {
				t.Fatal("compiled regex is nil")
			}

			matches := regex.MatchString(tt.input)
			if matches != tt.shouldMatch {
				t.Errorf("pattern %s: expected match=%v for input %q", tt.patternName, tt.shouldMatch, tt.input)
			}
		})
	}
}
