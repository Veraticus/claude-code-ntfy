package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/Veraticus/claude-code-ntfy/pkg/config"
)

const (
	defaultClaudeCommand = "claude"
)

func main() {
	// Parse command line flags
	var (
		configPath    string
		quiet         bool
		forceNotify   bool
		startupNotify bool
		help          bool
	)

	flag.StringVar(&configPath, "config", "", "Path to config file")
	flag.BoolVar(&quiet, "quiet", false, "Disable all notifications")
	flag.BoolVar(&forceNotify, "force", false, "Force notifications even when user is active")
	flag.BoolVar(&startupNotify, "startup", false, "Send notification on startup with current directory")
	flag.BoolVar(&help, "help", false, "Show help message")
	flag.Parse()

	// Only show our help if no args were provided with --help
	if help && len(flag.Args()) == 0 {
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
	if forceNotify {
		cfg.ForceNotify = true
	}
	if startupNotify {
		cfg.StartupNotify = true
	}

	// Get Claude command and args
	userArgs := flag.Args()
	command := defaultClaudeCommand

	// Merge default args with user args
	var args []string
	if len(cfg.DefaultClaudeArgs) > 0 {
		args = append(args, cfg.DefaultClaudeArgs...)
	}
	args = append(args, userArgs...)

	// Debug: Check if claude command exists
	if _, err := exec.LookPath(command); err != nil {
		fmt.Fprintf(os.Stderr, "Error: claude command not found in PATH\n")
		fmt.Fprintf(os.Stderr, "Make sure 'claude' is installed and in your PATH\n")
		os.Exit(1)
	}

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
	fmt.Println("Environment Variables:")
	fmt.Println("  CLAUDE_NOTIFY_TOPIC       Ntfy topic for notifications")
	fmt.Println("  CLAUDE_NOTIFY_SERVER      Ntfy server URL (default: https://ntfy.sh)")
	fmt.Println("  CLAUDE_NOTIFY_IDLE_TIMEOUT  User idle timeout (default: 2m)")
	fmt.Println("  CLAUDE_NOTIFY_QUIET       Disable notifications (true/false)")
	fmt.Println("  CLAUDE_NOTIFY_FORCE       Force notifications (true/false)")
	fmt.Println("  CLAUDE_NOTIFY_STARTUP     Send startup notification (true/false)")
	fmt.Println("  CLAUDE_NOTIFY_DEFAULT_ARGS  Default Claude args (comma-separated)")
	fmt.Println("  CLAUDE_NOTIFY_CONFIG      Path to config file")
}
