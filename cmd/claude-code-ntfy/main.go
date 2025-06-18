package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/Veraticus/claude-code-ntfy/pkg/config"
	flag "github.com/spf13/pflag"
)

func main() {
	// Parse our flags and separate Claude's flags
	var (
		configPath string
		quiet      bool
		help       bool
	)

	// Manually parse arguments to separate our flags from Claude's
	ourArgs := []string{}
	claudeArgs := []string{}
	
	i := 1 // Skip program name
	for i < len(os.Args) {
		arg := os.Args[i]
		
		// Check if it's one of our flags
		switch arg {
		case "--config", "-config":
			ourArgs = append(ourArgs, arg)
			if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "-") {
				ourArgs = append(ourArgs, os.Args[i+1])
				i++
			}
		case "--quiet", "-quiet":
			ourArgs = append(ourArgs, arg)
		case "--help", "-help":
			ourArgs = append(ourArgs, arg)
		default:
			// Handle --flag=value format for our flags
			if strings.HasPrefix(arg, "--config=") || strings.HasPrefix(arg, "-config=") {
				ourArgs = append(ourArgs, arg)
			} else {
				// Everything else goes to Claude
				claudeArgs = append(claudeArgs, arg)
			}
		}
		i++
	}

	// Define our flags first
	flag.StringVar(&configPath, "config", "", "Path to config file")
	flag.BoolVar(&quiet, "quiet", false, "Disable all notifications") 
	flag.BoolVar(&help, "help", false, "Show help message")
	
	// Parse only our flags
	flag.CommandLine.Parse(ourArgs)

	// Only show our help if --help was provided without other Claude args
	if help && len(claudeArgs) == 0 {
		printUsage()
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Override config with command line flags
	if configPath != "" {
		if err := os.Setenv("CLAUDE_NOTIFY_CONFIG", configPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting config path: %v\n", err)
			os.Exit(1)
		}
	}
	if quiet {
		cfg.Quiet = true
	}

	// Use the manually parsed Claude args
	userArgs := claudeArgs
	var command string

	// Determine claude path
	if cfg.ClaudePath != "" {
		// Use configured path directly - don't validate, let it fail at execution if wrong
		command = cfg.ClaudePath
		if os.Getenv("CLAUDE_NOTIFY_DEBUG") == "1" {
			fmt.Fprintf(os.Stderr, "claude-code-ntfy: Using configured claude path: %s\n", command)
		}
	} else {
		// Try to find claude in PATH, excluding ourselves
		claudePath, err := findClaude()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "\nYou can fix this by:\n")
			fmt.Fprintf(os.Stderr, "1. Setting claude_path in your config file (~/.config/claude-code-ntfy/config.yaml)\n")
			fmt.Fprintf(os.Stderr, "2. Setting CLAUDE_NOTIFY_CLAUDE_PATH environment variable\n")
			fmt.Fprintf(os.Stderr, "3. Ensuring the real claude is in your PATH\n")
			os.Exit(1)
		}
		command = claudePath
		if os.Getenv("CLAUDE_NOTIFY_DEBUG") == "1" {
			fmt.Fprintf(os.Stderr, "claude-code-ntfy: Found claude in PATH at: %s\n", command)
		}
	}

	// Merge default args with user args
	var args []string
	if len(cfg.DefaultClaudeArgs) > 0 {
		args = append(args, cfg.DefaultClaudeArgs...)
	}
	args = append(args, userArgs...)

	// Create dependencies
	deps, err := NewDependencies(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating dependencies: %v\n", err)
		os.Exit(1)
	}
	defer deps.Close()

	// Create application
	app := NewApplication(deps)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Ensure terminal restoration on panic
	defer func() {
		if r := recover(); r != nil {
			_ = app.Stop() // Best effort terminal restoration
			panic(r)       // Re-panic
		}
	}()

	go func() {
		<-sigChan
		// Attempt graceful shutdown
		if err := app.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "Error stopping process: %v\n", err)
		}
		// Exit with standard interrupt code
		os.Exit(130)
	}()

	// Debug output if verbose
	if os.Getenv("CLAUDE_NOTIFY_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "claude-code-ntfy: Starting claude with args: %v\n", args)
		fmt.Fprintf(os.Stderr, "claude-code-ntfy: Config: quiet=%v, topic=%q\n", cfg.Quiet, cfg.NtfyTopic)
	}

	// Run the application
	if err := app.Run(command, args); err != nil {
		// Check if it's an exec.ExitError
		if _, ok := err.(*exec.ExitError); !ok {
			// Only log if it's not an expected exit error
			fmt.Fprintf(os.Stderr, "Error running claude: %v\n", err)
		}
	}

	// Exit with the same code as the wrapped process
	os.Exit(app.ExitCode())
}

func printUsage() {
	fmt.Println("claude-code-ntfy - Claude Code wrapper with notifications")
	fmt.Println()
	fmt.Println("Usage: claude-code-ntfy [OPTIONS] [CLAUDE_ARGS...]")
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("All unknown flags are passed through to Claude Code")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  CLAUDE_NOTIFY_TOPIC       Ntfy topic for notifications")
	fmt.Println("  CLAUDE_NOTIFY_SERVER      Ntfy server URL (default: https://ntfy.sh)")
	fmt.Println("  CLAUDE_NOTIFY_BACKSTOP_TIMEOUT  Inactivity timeout (default: 30s)")
	fmt.Println("  CLAUDE_NOTIFY_QUIET       Disable notifications (true/false)")
	fmt.Println("  CLAUDE_NOTIFY_STARTUP     Send startup notification (default: true)")
	fmt.Println("  CLAUDE_NOTIFY_DEFAULT_ARGS  Default Claude args (comma-separated)")
	fmt.Println("  CLAUDE_NOTIFY_CONFIG      Path to config file")
	fmt.Println("  CLAUDE_NOTIFY_CLAUDE_PATH  Path to the real claude binary")
	fmt.Println()
	fmt.Println("Configuration file: ~/.config/claude-code-ntfy/config.yaml")
}

// findClaude searches for the real claude binary in PATH, excluding ourselves
func findClaude() (string, error) {
	// Get our own executable path to exclude it
	ourPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get our executable path: %w", err)
	}
	ourPath, err = filepath.EvalSymlinks(ourPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve our executable path: %w", err)
	}

	// Search PATH for claude
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return "", fmt.Errorf("PATH environment variable is empty")
	}

	for _, dir := range filepath.SplitList(pathEnv) {
		claudePath := filepath.Join(dir, "claude")

		// Check if file exists and is executable
		info, err := os.Stat(claudePath)
		if err != nil {
			continue // Not found in this directory
		}

		if info.Mode().IsRegular() && info.Mode()&0111 != 0 {
			// Resolve symlinks to check if it's us
			resolvedPath, err := filepath.EvalSymlinks(claudePath)
			if err != nil {
				continue
			}

			// Skip if it's our own binary
			if resolvedPath == ourPath {
				continue
			}

			// Found a different claude binary
			return claudePath, nil
		}
	}

	return "", fmt.Errorf("claude not found in PATH (excluding claude-code-ntfy wrapper)")
}
