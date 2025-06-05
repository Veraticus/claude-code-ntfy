package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Veraticus/claude-code-ntfy/pkg/interfaces"
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
		{"error", `(?i)(error|failed|exception)`},
		{"completion", `(?i)(done|finished|completed)`},
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
	defer func() {
		os.Setenv("CLAUDE_NOTIFY_TOPIC", origTopic)
		os.Setenv("CLAUDE_NOTIFY_SERVER", origServer)
		os.Setenv("CLAUDE_NOTIFY_IDLE_TIMEOUT", origTimeout)
		os.Setenv("CLAUDE_NOTIFY_QUIET", origQuiet)
		os.Setenv("CLAUDE_NOTIFY_FORCE", origForce)
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
			name: "boolean variations",
			envVars: map[string]string{
				"CLAUDE_NOTIFY_QUIET": "yes",
				"CLAUDE_NOTIFY_FORCE": "1",
			},
			checkFunc: func(t *testing.T, cfg *Config) {
				if !cfg.Quiet {
					t.Error("expected Quiet to be true for 'yes'")
				}
				if !cfg.ForceNotify {
					t.Error("expected ForceNotify to be true for '1'")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars first
			os.Unsetenv("CLAUDE_NOTIFY_TOPIC")
			os.Unsetenv("CLAUDE_NOTIFY_SERVER")
			os.Unsetenv("CLAUDE_NOTIFY_IDLE_TIMEOUT")
			os.Unsetenv("CLAUDE_NOTIFY_QUIET")
			os.Unsetenv("CLAUDE_NOTIFY_FORCE")
			os.Unsetenv("CLAUDE_NOTIFY_CONFIG")

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
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
	defer os.RemoveAll(tmpDir)

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
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			// Set config path env var
			os.Setenv("CLAUDE_NOTIFY_CONFIG", configPath)
			defer os.Unsetenv("CLAUDE_NOTIFY_CONFIG")

			// Clear other env vars to avoid interference
			os.Unsetenv("CLAUDE_NOTIFY_TOPIC")
			os.Unsetenv("CLAUDE_NOTIFY_SERVER")

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
		patterns []interfaces.Pattern
		wantErr  bool
		errorMsg string
	}{
		{
			name: "valid patterns",
			patterns: []interfaces.Pattern{
				{Name: "test1", Regex: `\d+`, Enabled: true},
				{Name: "test2", Regex: `[a-z]+`, Enabled: true},
			},
			wantErr: false,
		},
		{
			name: "invalid regex",
			patterns: []interfaces.Pattern{
				{Name: "bad", Regex: `[`, Enabled: true},
			},
			wantErr:  true,
			errorMsg: "failed to compile pattern",
		},
		{
			name: "disabled pattern",
			patterns: []interfaces.Pattern{
				{Name: "disabled", Regex: `[`, Enabled: false},
			},
			wantErr: false, // Should not compile disabled patterns
		},
		{
			name: "empty regex",
			patterns: []interfaces.Pattern{
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
		os.Setenv("CLAUDE_NOTIFY_CONFIG", origConfig)
		os.Setenv("XDG_CONFIG_HOME", origXDG)
		os.Setenv("HOME", origHome)
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
			os.Unsetenv("CLAUDE_NOTIFY_CONFIG")
			os.Unsetenv("XDG_CONFIG_HOME")

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
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
		{"error", "An error occurred", true},
		{"error", "Failed to process", true},
		{"error", "Exception thrown", true},
		{"error", "All is well", false},
		{"completion", "Task done", true},
		{"completion", "Process finished", true},
		{"completion", "Job completed", true},
		{"completion", "Still working", false},
	}

	for _, tt := range tests {
		t.Run(tt.patternName+"_"+tt.input, func(t *testing.T) {
			// Find the pattern
			var pattern *interfaces.Pattern
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
