package main

import (
	"fmt"
	"os"

	"github.com/renancavalcantercb/healthcheck-cli/internal/app"
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "healthcheck",
	Short: "A self-hosted health checking service",
	Long: `🏥 HealthCheck CLI - A simple, powerful health checking service

Monitor your websites, APIs, and services with ease.
Get alerts when things go wrong, track uptime, and keep your services healthy.

Examples:
  healthcheck test https://google.com
  healthcheck start https://api.example.com
  healthcheck start --config sites.yaml --daemon`,
	Version: version,
}

var startCmd = &cobra.Command{
	Use:   "start [url]",
	Short: "Start monitoring endpoints",
	Long: `Start monitoring one or more endpoints.

Examples:
  healthcheck start https://api.example.com           # Quick start mode
  healthcheck start --config sites.yaml              # Config file mode
  healthcheck start --config sites.yaml --daemon     # Daemon mode`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configFile, _ := cmd.Flags().GetString("config")
		daemon, _ := cmd.Flags().GetBool("daemon")
		interval, _ := cmd.Flags().GetDuration("interval")
		
		app := app.New()
		
		// Quick start mode
		if len(args) == 1 {
			return app.StartQuick(args[0], interval, daemon)
		}
		
		// Config file mode
		if configFile != "" {
			return app.StartWithConfig(configFile, daemon)
		}
		
		return fmt.Errorf("either provide a URL or use --config flag")
	},
}

var testCmd = &cobra.Command{
	Use:   "test <url>",
	Short: "Test a single endpoint",
	Long: `Test a single endpoint and show the result immediately.

Examples:
  healthcheck test https://google.com
  healthcheck test https://api.example.com/health
  healthcheck test https://httpbin.org/delay/2`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		timeout, _ := cmd.Flags().GetDuration("timeout")
		verbose, _ := cmd.Flags().GetBool("verbose")
		
		app := app.New()
		return app.TestEndpoint(args[0], timeout, verbose)
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status dashboard",
	Long: `Show a real-time status dashboard of all monitored endpoints.

Examples:
  healthcheck status              # Show current status
  healthcheck status --watch      # Watch mode with auto-refresh
  healthcheck status --config config.yaml --watch  # Watch with specific config`,
	RunE: func(cmd *cobra.Command, args []string) error {
		watch, _ := cmd.Flags().GetBool("watch")
		configFile, _ := cmd.Flags().GetString("config")
		app := app.New()
		
		// Load config if provided
		if configFile != "" {
			if err := app.LoadConfigForStatus(configFile); err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
		}
		
		return app.ShowStatus(watch)
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate <config-file>",
	Short: "Validate configuration file",
	Long: `Validate the syntax and content of a configuration file.

Examples:
  healthcheck validate sites.yaml
  healthcheck validate config.json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := app.New()
		return app.ValidateConfig(args[0])
	},
}

var exampleCmd = &cobra.Command{
	Use:   "example-config",
	Short: "Generate example configuration file",
	Long: `Generate an example configuration file with common settings.

Examples:
  healthcheck example-config > healthcheck.yaml
  healthcheck example-config --output config.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		output, _ := cmd.Flags().GetString("output")
		app := app.New()
		return app.GenerateExampleConfig(output)
	},
}

func init() {
	// Start command flags
	startCmd.Flags().StringP("config", "c", "", "Configuration file path")
	startCmd.Flags().BoolP("daemon", "d", false, "Run as daemon in background")
	startCmd.Flags().DurationP("interval", "i", 0, "Check interval (e.g., 30s, 1m)")
	
	// Test command flags
	testCmd.Flags().DurationP("timeout", "t", 0, "Request timeout (default: 10s)")
	testCmd.Flags().BoolP("verbose", "v", false, "Verbose output")
	
	// Status command flags
	statusCmd.Flags().BoolP("watch", "w", false, "Watch mode (auto-refresh every 5s)")
	statusCmd.Flags().StringP("config", "c", "", "Configuration file path")
	
	// Example config command flags
	exampleCmd.Flags().StringP("output", "o", "", "Output file (default: stdout)")
	
	// Add commands to root
	rootCmd.AddCommand(startCmd, testCmd, statusCmd, validateCmd, exampleCmd)
	
	// Global flags
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable colored output")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}