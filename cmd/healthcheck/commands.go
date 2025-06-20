package main

import (
	"time"

	"github.com/spf13/cobra"
)

// setupCommands creates and configures all CLI commands
func setupCommands(app *App) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "healthcheck",
		Short: "A powerful health checking CLI tool",
		Long:  "Monitor the health of your endpoints with HTTP and TCP checks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Quick check command
	quickCmd := &cobra.Command{
		Use:   "quick [URL]",
		Short: "Quickly check a single endpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			interval, _ := cmd.Flags().GetDuration("interval")
			daemon, _ := cmd.Flags().GetBool("daemon")
			return app.StartQuick(url, interval, daemon)
		},
	}
	quickCmd.Flags().DurationP("interval", "i", 30*time.Second, "Check interval")
	quickCmd.Flags().BoolP("daemon", "d", false, "Run in daemon mode")

	// Config-based monitoring
	monitorCmd := &cobra.Command{
		Use:   "monitor [config-file]",
		Short: "Monitor endpoints using a configuration file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile := args[0]
			daemon, _ := cmd.Flags().GetBool("daemon")
			return app.StartWithConfig(configFile, daemon)
		},
	}
	monitorCmd.Flags().BoolP("daemon", "d", false, "Run in daemon mode")

	// Test command
	testCmd := &cobra.Command{
		Use:   "test [URL]",
		Short: "Test a single endpoint immediately",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			timeout, _ := cmd.Flags().GetDuration("timeout")
			verbose, _ := cmd.Flags().GetBool("verbose")
			return app.TestEndpoint(url, timeout, verbose)
		},
	}
	testCmd.Flags().DurationP("timeout", "t", 10*time.Second, "Request timeout")
	testCmd.Flags().BoolP("verbose", "v", false, "Verbose output")

	// Status dashboard
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show status dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			watch, _ := cmd.Flags().GetBool("watch")
			configFile, _ := cmd.Flags().GetString("config")
			
			if configFile != "" {
				if err := app.LoadConfigForStatus(configFile); err != nil {
					return err
				}
			}
			
			return app.ShowStatus(watch)
		},
	}
	statusCmd.Flags().BoolP("watch", "w", false, "Interactive dashboard")
	statusCmd.Flags().StringP("config", "c", "", "Configuration file")

	// Configuration commands
	configCmd := setupConfigCommands(app)

	// Statistics commands
	statsCmd := setupStatsCommands(app)

	// History command
	historyCmd := &cobra.Command{
		Use:   "history [service-name]",
		Short: "Show historical data for a service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serviceName := args[0]
			limit, _ := cmd.Flags().GetInt("limit")
			since, _ := cmd.Flags().GetString("since")
			return app.ShowHistory(serviceName, limit, since)
		},
	}
	historyCmd.Flags().IntP("limit", "l", 50, "Maximum number of records to show")
	historyCmd.Flags().StringP("since", "s", "24h", "Show history since duration")

	// Database info command
	dbInfoCmd := &cobra.Command{
		Use:   "db-info",
		Short: "Show database information",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.ShowDatabaseInfo()
		},
	}

	// Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			showVersion()
		},
	}

	// Add all commands to root
	rootCmd.AddCommand(
		quickCmd,
		monitorCmd,
		testCmd,
		statusCmd,
		configCmd,
		statsCmd,
		historyCmd,
		dbInfoCmd,
		versionCmd,
	)

	return rootCmd
}

// setupConfigCommands creates configuration-related commands
func setupConfigCommands(app *App) *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
	}

	validateCmd := &cobra.Command{
		Use:   "validate [config-file]",
		Short: "Validate a configuration file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.ValidateConfig(args[0])
		},
	}

	exampleCmd := &cobra.Command{
		Use:   "example [output-file]",
		Short: "Generate an example configuration file",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var outputFile string
			if len(args) > 0 {
				outputFile = args[0]
			}
			return app.GenerateExampleConfig(outputFile)
		},
	}

	configCmd.AddCommand(validateCmd, exampleCmd)
	return configCmd
}

// setupStatsCommands creates statistics-related commands
func setupStatsCommands(app *App) *cobra.Command {
	statsCmd := &cobra.Command{
		Use:   "stats [service-name]",
		Short: "Show statistics from stored data",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var serviceName string
			if len(args) > 0 {
				serviceName = args[0]
			}
			since, _ := cmd.Flags().GetString("since")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			return app.ShowStats(serviceName, since, jsonOutput)
		},
	}
	statsCmd.Flags().StringP("since", "s", "24h", "Show stats since duration (e.g., 1h, 24h, 7d)")
	statsCmd.Flags().BoolP("json", "j", false, "Output in JSON format")

	return statsCmd
}