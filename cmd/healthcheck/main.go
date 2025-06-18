package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/renancavalcantercb/healthcheck-cli/internal/checker"
	"github.com/renancavalcantercb/healthcheck-cli/internal/config"
	"github.com/renancavalcantercb/healthcheck-cli/internal/storage"
	"github.com/renancavalcantercb/healthcheck-cli/internal/tui"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
	"github.com/spf13/cobra"
)

// version will be set during build
var version = "dev"

// App represents the main application
type App struct {
	httpChecker *checker.HTTPChecker
	tcpChecker  *checker.TCPChecker
	storage     *storage.SQLiteStorage
	config      *config.Config
}

// New creates a new App instance
func New() *App {
	// Initialize storage
	storage, err := storage.NewSQLiteStorage("./healthcheck.db")
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to initialize storage: %v\n", err)
		fmt.Println("💡 Continuing without storage (data won't be persisted)")
	}

	app := &App{
		httpChecker: checker.NewHTTPChecker(30 * time.Second),
		tcpChecker:  checker.NewTCPChecker(10 * time.Second),
		storage:     storage,
		config:      config.DefaultConfig(),
	}

	// Start background cleanup routine
	if storage != nil {
		go app.backgroundCleanup()
	}

	return app
}

// main is the entry point of the application
func main() {
	app := New()
	defer app.Close()

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

	// Statistics commands
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
			fmt.Printf("healthcheck version %s\n", version)
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

	// Execute the CLI
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// StartQuick starts monitoring a single URL with minimal configuration
func (a *App) StartQuick(url string, interval time.Duration, daemon bool) error {
	// Apply default interval if not specified
	if interval == 0 {
		interval = 30 * time.Second
	}
	
	fmt.Printf("🚀 Starting health check for %s\n", url)
	fmt.Printf("📊 Check interval: %v\n", interval)
	
	// Create a simple check configuration
	check := types.CheckConfig{
		Name:     "Quick Check",
		URL:      url,
		Interval: interval,
		Timeout:  10 * time.Second,
		Method:   "GET",
		Type:     types.CheckTypeHTTP,
		Expected: types.Expected{
			Status:          200,
			ResponseTimeMax: 5 * time.Second,
		},
		Retry: types.RetryConfig{
			Attempts: 3,
			Delay:    2 * time.Second,
			Backoff:  "exponential",
		},
	}
	
	// Determine check type based on URL
	if !strings.HasPrefix(url, "http") {
		check.Type = types.CheckTypeTCP
		check.Method = ""
	}
	
	if daemon {
		fmt.Println("🔄 Running in daemon mode (Press Ctrl+C to stop)")
		return a.runDaemon([]types.CheckConfig{check})
	}
	
	return a.runOnce(check)
}

// StartWithConfig starts monitoring using a configuration file
func (a *App) StartWithConfig(configFile string, daemon bool) error {
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	a.config = cfg
	
	fmt.Printf("🔧 Loaded configuration from %s\n", configFile)
	fmt.Printf("📊 Monitoring %d endpoints\n", len(cfg.Checks))
	
	// Convert config checks to types.CheckConfig
	checks := make([]types.CheckConfig, len(cfg.Checks))
	for i, c := range cfg.Checks {
		checks[i] = c.CheckConfig
	}
	
	if daemon {
		fmt.Println("🔄 Running in daemon mode (Press Ctrl+C to stop)")
		return a.runDaemon(checks)
	}
	
	// Run all checks once
	fmt.Println("🏃 Running all checks once...")
	for _, check := range checks {
		if err := a.runOnce(check); err != nil {
			fmt.Printf("❌ Check %s failed: %v\n", check.Name, err)
		}
	}
	
	return nil
}

// TestEndpoint tests a single endpoint immediately
func (a *App) TestEndpoint(url string, timeout time.Duration, verbose bool) error {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	
	fmt.Printf("🧪 Testing %s...\n", url)
	
	check := types.CheckConfig{
		Name:    "Test",
		URL:     url,
		Timeout: timeout,
		Method:  "GET",
		Type:    types.CheckTypeHTTP,
		Expected: types.Expected{
			Status: 200,
		},
	}
	
	// Determine check type
	if !strings.HasPrefix(url, "http") {
		check.Type = types.CheckTypeTCP
		check.Method = ""
	}
	
	// Perform the check directly
	result := a.performCheck(check)
	a.printResult(result, verbose)
	
	// Don't return error for failed checks in test mode
	// Just print the result and return success
	return nil
}

// ShowStatus shows a status dashboard
func (a *App) ShowStatus(watch bool) error {
	if !watch {
		// Static status (não implementado ainda)
		fmt.Println("📊 Static status not implemented yet")
		fmt.Println("💡 Use --watch for interactive dashboard")
		return nil
	}
	
	// Load config if we have one, otherwise use current checks
	var checks []types.CheckConfig
	if len(a.config.Checks) > 0 {
		for _, c := range a.config.Checks {
			checks = append(checks, c.CheckConfig)
		}
	} else {
		// Default check if no config
		checks = []types.CheckConfig{
			{
				Name: "Example",
				URL:  "https://httpbin.org/get",
				Type: types.CheckTypeHTTP,
				Method: "GET",
				Timeout: 10 * time.Second,
				Expected: types.Expected{Status: 200},
			},
		}
	}
	
	return a.runTUIDashboard(checks)
}

// ValidateConfig validates a configuration file
func (a *App) ValidateConfig(configFile string) error {
	fmt.Printf("🔍 Validating configuration file: %s\n", configFile)
	
	_, err := config.LoadConfig(configFile)
	if err != nil {
		fmt.Printf("❌ Configuration validation failed: %v\n", err)
		return err
	}
	
	fmt.Println("✅ Configuration is valid!")
	return nil
}

// LoadConfigForStatus loads configuration for status dashboard
func (a *App) LoadConfigForStatus(configFile string) error {
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return err
	}
	a.config = cfg
	return nil
}

// GenerateExampleConfig generates an example configuration file
func (a *App) GenerateExampleConfig(outputFile string) error {
	if outputFile == "" {
		// Output to stdout
		return config.SaveExample("")
	}
	
	// Output to file
	if err := config.SaveExample(outputFile); err != nil {
		return fmt.Errorf("failed to generate example config: %w", err)
	}
	
	fmt.Printf("✅ Example configuration saved to %s\n", outputFile)
	return nil
}

// ShowStats displays statistics from stored data
func (a *App) ShowStats(serviceName, sinceStr string, jsonOutput bool) error {
	if a.storage == nil {
		return fmt.Errorf("storage not available - stats require data persistence")
	}

	// Parse since duration
	since := time.Now().Add(-24 * time.Hour) // Default to 24 hours
	if sinceStr != "" {
		duration, err := time.ParseDuration(sinceStr)
		if err != nil {
			return fmt.Errorf("invalid duration format: %s (use: 1h, 24h, 7d, etc.)", sinceStr)
		}
		since = time.Now().Add(-duration)
	}

	if serviceName != "" {
		// Show stats for specific service
		return a.showServiceStats(serviceName, since, jsonOutput)
	} else {
		// Show stats for all services
		return a.showAllStats(since, jsonOutput)
	}
}

// ShowHistory displays historical data for a service
func (a *App) ShowHistory(serviceName string, limit int, sinceStr string) error {
	if a.storage == nil {
		return fmt.Errorf("storage not available - history requires data persistence")
	}

	// Parse since duration
	since := time.Now().Add(-24 * time.Hour) // Default to 24 hours
	if sinceStr != "" {
		duration, err := time.ParseDuration(sinceStr)
		if err != nil {
			return fmt.Errorf("invalid duration format: %s", sinceStr)
		}
		since = time.Now().Add(-duration)
	}

	history, err := a.storage.GetServiceHistory(serviceName, since, limit)
	if err != nil {
		return fmt.Errorf("failed to get history: %w", err)
	}

	if len(history) == 0 {
		fmt.Printf("📊 No history found for service '%s'\n", serviceName)
		return nil
	}

	fmt.Printf("📈 History for %s (last %d checks)\n", serviceName, len(history))
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("%-19s %-8s %-12s %-30s\n", "TIMESTAMP", "STATUS", "RESPONSE", "ERROR")
	fmt.Printf("───────────────────────────────────────────────────────────────\n")

	for _, record := range history {
		timestamp := record.Timestamp.Format("01-02 15:04:05")
		
		var status string
		switch record.Status {
		case 0: // StatusUp
			status = "🟢 UP"
		case 1: // StatusDown
			status = "🔴 DOWN"
		case 2: // StatusSlow
			status = "🟡 SLOW"
		default:
			status = "❓ UNK"
		}

		response := fmt.Sprintf("%dms", record.ResponseTimeMs)
		if record.StatusCode > 0 {
			response += fmt.Sprintf(" (%d)", record.StatusCode)
		}

		errorMsg := truncateString(record.Error, 28)

		fmt.Printf("%-19s %-8s %-12s %-30s\n", timestamp, status, response, errorMsg)
	}

	return nil
}

// ShowDatabaseInfo displays information about the database
func (a *App) ShowDatabaseInfo() error {
	if a.storage == nil {
		return fmt.Errorf("storage not available")
	}

	info, err := a.storage.GetDatabaseInfo()
	if err != nil {
		return fmt.Errorf("failed to get database info: %w", err)
	}

	fmt.Println("🗄️  Database Information")
	fmt.Println("═══════════════════════════════════")
	
	if path, ok := info["database_path"].(string); ok {
		fmt.Printf("📁 Path:            %s\n", path)
	}
	
	if totalRecords, ok := info["total_records"].(int64); ok {
		fmt.Printf("📊 Total Records:   %d\n", totalRecords)
	}
	
	if totalServices, ok := info["total_services"].(int64); ok {
		fmt.Printf("🏷️  Services:        %d\n", totalServices)
	}
	
	if sizeBytes, ok := info["database_size_bytes"].(int64); ok {
		sizeKB := float64(sizeBytes) / 1024
		sizeMB := sizeKB / 1024
		if sizeMB > 1 {
			fmt.Printf("💾 Size:            %.1f MB\n", sizeMB)
		} else {
			fmt.Printf("💾 Size:            %.1f KB\n", sizeKB)
		}
	}
	
	if oldest, ok := info["oldest_record"].(time.Time); ok {
		fmt.Printf("📅 Oldest Record:   %s\n", oldest.Format("2006-01-02 15:04:05"))
	}
	
	if newest, ok := info["newest_record"].(time.Time); ok {
		fmt.Printf("🕐 Newest Record:   %s\n", newest.Format("2006-01-02 15:04:05"))
	}

	return nil
}

// backgroundCleanup performs periodic cleanup of old data
func (a *App) backgroundCleanup() {
	ticker := time.NewTicker(24 * time.Hour) // Run daily
	defer ticker.Stop()

	for range ticker.C {
		if a.storage != nil {
			// Keep data for 30 days
			if err := a.storage.CleanupOldData(30 * 24 * time.Hour); err != nil {
				fmt.Printf("Warning: cleanup failed: %v\n", err)
			}
		}
	}
}

// Close gracefully closes the app
func (a *App) Close() error {
	if a.storage != nil {
		return a.storage.Close()
	}
	return nil
}

// runOnce executes a single health check
func (a *App) runOnce(check types.CheckConfig) error {
	result := a.performCheck(check)
	a.printResult(result, false)
	
	if !result.IsHealthy() {
		return fmt.Errorf("check failed")
	}
	
	return nil
}

// runDaemon runs health checks continuously
func (a *App) runDaemon(checks []types.CheckConfig) error {
	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		fmt.Println("\n🛑 Received shutdown signal, stopping...")
		cancel()
	}()
	
	// Create channels for results
	resultChan := make(chan types.Result, len(checks)*2)
	
	// Start monitoring goroutines for each check
	for _, check := range checks {
		go a.monitorEndpoint(ctx, check, resultChan)
	}
	
	// Process results
	for {
		select {
		case <-ctx.Done():
			fmt.Println("👋 Shutdown complete")
			return nil
		case result := <-resultChan:
			a.printResult(result, false)
		}
	}
}

// monitorEndpoint monitors a single endpoint continuously
func (a *App) monitorEndpoint(ctx context.Context, check types.CheckConfig, resultChan chan<- types.Result) {
	ticker := time.NewTicker(check.Interval)
	defer ticker.Stop()
	
	// Run initial check immediately
	result := a.performCheck(check)
	select {
	case resultChan <- result:
	case <-ctx.Done():
		return
	}
	
	// Continue monitoring
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result := a.performCheck(check)
			select {
			case resultChan <- result:
			case <-ctx.Done():
				return
			}
		}
	}
}

// performCheck executes a health check with retry logic
func (a *App) performCheck(check types.CheckConfig) types.Result {
	var result types.Result
	
	maxAttempts := check.Retry.Attempts
	if maxAttempts == 0 {
		maxAttempts = 1 // At least one attempt
	}
	
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Perform the actual check
		switch check.Type {
		case types.CheckTypeHTTP:
			if a.httpChecker == nil {
				a.httpChecker = checker.NewHTTPChecker(check.Timeout)
			}
			result = a.httpChecker.Check(check)
		case types.CheckTypeTCP:
			if a.tcpChecker == nil {
				a.tcpChecker = checker.NewTCPChecker(check.Timeout)
			}
			result = a.tcpChecker.Check(check)
		default:
			result = types.Result{
				Name:      check.Name,
				URL:       check.URL,
				Status:    types.StatusError,
				Error:     fmt.Sprintf("unsupported check type: %s", check.Type),
				Timestamp: time.Now(),
			}
		}
		
		// If check succeeded or we're out of attempts, return
		if result.IsHealthy() || attempt >= maxAttempts {
			break
		}
		
		// Wait before retry (with backoff)
		if attempt < maxAttempts {
			delay := a.calculateRetryDelay(check.Retry, attempt)
			time.Sleep(delay)
		}
	}
	
	// Save result to storage
	if a.storage != nil {
		if err := a.storage.SaveResult(result); err != nil {
			// Log error but don't fail the check
			fmt.Printf("Warning: failed to save result to storage: %v\n", err)
		}
	}
	
	return result
}

// calculateRetryDelay calculates the delay before retry based on backoff strategy
func (a *App) calculateRetryDelay(retry types.RetryConfig, attempt int) time.Duration {
	baseDelay := retry.Delay
	if baseDelay == 0 {
		baseDelay = 2 * time.Second
	}
	
	switch retry.Backoff {
	case "exponential":
		delay := baseDelay * time.Duration(1<<uint(attempt-1)) // 2^(attempt-1)
		if retry.MaxDelay > 0 && delay > retry.MaxDelay {
			return retry.MaxDelay
		}
		return delay
	case "linear":
		delay := baseDelay * time.Duration(attempt)
		if retry.MaxDelay > 0 && delay > retry.MaxDelay {
			return retry.MaxDelay
		}
		return delay
	default: // "none" or unknown
		return baseDelay
	}
}

// printResult prints a check result to the console
func (a *App) printResult(result types.Result, verbose bool) {
	// Get color codes (check if colors are disabled)
	status := result.Status.Emoji() + " " + result.Status.String()
	if !a.config.Global.DisableColors {
		status = result.Status.Color() + status + "\033[0m" // Reset color
	}
	
	// Basic info
	fmt.Printf("[%s] %s %s - %v",
		result.Timestamp.Format("15:04:05"),
		status,
		result.Name,
		result.ResponseTime,
	)
	
	// Add status code for HTTP checks
	if result.StatusCode > 0 {
		fmt.Printf(" (HTTP %d)", result.StatusCode)
	}
	
	// Add error if present
	if result.Error != "" {
		fmt.Printf(" - %s", result.Error)
	}
	
	fmt.Println()
	
	// Verbose output
	if verbose {
		fmt.Printf("  URL: %s\n", result.URL)
		if result.BodySize > 0 {
			fmt.Printf("  Body Size: %d bytes\n", result.BodySize)
		}
		if len(result.Headers) > 0 {
			fmt.Println("  Headers:")
			for key, value := range result.Headers {
				fmt.Printf("    %s: %s\n", key, value)
			}
		}
		fmt.Println()
	}
}

// runTUIDashboard runs the terminal UI dashboard
func (a *App) runTUIDashboard(checks []types.CheckConfig) error {
	// Create TUI model
	model := tui.New()
	
	// Create a program
	program := tea.NewProgram(model, tea.WithAltScreen())
	
	// Start background monitoring
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Channel for sending results to TUI
	resultsChan := make(chan []types.Result, 10)
	
	// Start monitoring goroutines
	for _, check := range checks {
		go a.monitorForTUI(ctx, check, resultsChan)
	}
	
	// Goroutine to collect and send results to TUI
	go func() {
		resultsMap := make(map[string]types.Result)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case result := <-resultsChan:
				for _, r := range result {
					resultsMap[r.Name] = r
				}
			case <-ticker.C:
				// Send current results to TUI
				var results []types.Result
				for _, result := range resultsMap {
					results = append(results, result)
				}
				if len(results) > 0 {
					program.Send(results)
				}
			}
		}
	}()
	
	// Run the TUI
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("TUI program failed: %w", err)
	}
	
	return nil
}

// monitorForTUI monitors a single endpoint for the TUI
func (a *App) monitorForTUI(ctx context.Context, check types.CheckConfig, resultsChan chan<- []types.Result) {
	ticker := time.NewTicker(check.Interval)
	defer ticker.Stop()
	
	// Run initial check
	result := a.performCheck(check)
	select {
	case resultsChan <- []types.Result{result}:
	case <-ctx.Done():
		return
	}
	
	// Continue monitoring
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result := a.performCheck(check)
			select {
			case resultsChan <- []types.Result{result}:
			case <-ctx.Done():
				return
			}
		}
	}
}

// Helper functions

// showServiceStats shows detailed stats for a specific service
func (a *App) showServiceStats(serviceName string, since time.Time, jsonOutput bool) error {
	stats, err := a.storage.GetServiceStats(serviceName, since)
	if err != nil {
		return fmt.Errorf("failed to get stats for %s: %w", serviceName, err)
	}

	if jsonOutput {
		// TODO: Implement JSON output
		fmt.Printf("{\"service\":\"%s\",\"stats\":%+v}\n", serviceName, stats)
		return nil
	}

	// Pretty print stats
	fmt.Printf("📊 Statistics for %s\n", stats.Name)
	fmt.Printf("═══════════════════════════════════════\n")
	fmt.Printf("🔗 URL:              %s\n", stats.URL)
	fmt.Printf("📝 Type:             %s\n", strings.ToUpper(stats.CheckType))
	fmt.Printf("📈 Uptime:           %.2f%%\n", stats.UptimePercent)
	fmt.Printf("✅ Successful:       %d\n", stats.SuccessfulChecks)
	fmt.Printf("❌ Failed:           %d\n", stats.FailedChecks)
	fmt.Printf("📊 Total Checks:     %d\n", stats.TotalChecks)
	fmt.Printf("⚡ Avg Response:     %.0fms\n", stats.AvgResponseTimeMs)
	fmt.Printf("🚀 Min Response:     %dms\n", stats.MinResponseTimeMs)
	fmt.Printf("🐌 Max Response:     %dms\n", stats.MaxResponseTimeMs)
	fmt.Printf("🕐 Last Check:       %s\n", stats.LastCheck.Format("2006-01-02 15:04:05"))

	if !stats.LastSuccess.IsZero() {
		fmt.Printf("✅ Last Success:     %s\n", stats.LastSuccess.Format("2006-01-02 15:04:05"))
	}
	if !stats.LastFailure.IsZero() {
		fmt.Printf("❌ Last Failure:     %s\n", stats.LastFailure.Format("2006-01-02 15:04:05"))
	}

	return nil
}

// showAllStats shows stats for all services
func (a *App) showAllStats(since time.Time, jsonOutput bool) error {
	allStats, err := a.storage.GetAllServiceStats(since)
	if err != nil {
		return fmt.Errorf("failed to get all stats: %w", err)
	}

	if len(allStats) == 0 {
		fmt.Println("📊 No statistics available yet")
		fmt.Println("💡 Run some checks first to generate stats")
		return nil
	}

	if jsonOutput {
		// TODO: Implement JSON output
		fmt.Printf("{\"services\":%+v}\n", allStats)
		return nil
	}

	fmt.Printf("📊 Service Statistics (since %s)\n", since.Format("2006-01-02 15:04"))
	fmt.Printf("═══════════════════════════════════════════════════════════════════════\n")
	fmt.Printf("%-20s %-12s %-8s %-10s %-12s %-15s\n", 
		"SERVICE", "TYPE", "UPTIME", "CHECKS", "AVG RT", "LAST CHECK")
	fmt.Printf("───────────────────────────────────────────────────────────────────────\n")

	for _, stats := range allStats {
		name := truncateString(stats.Name, 18)
		checkType := strings.ToUpper(stats.CheckType)
		uptime := fmt.Sprintf("%.1f%%", stats.UptimePercent)
		checks := fmt.Sprintf("%d", stats.TotalChecks)
		avgRT := fmt.Sprintf("%.0fms", stats.AvgResponseTimeMs)
		lastCheck := stats.LastCheck.Format("15:04:05")

		// Color coding
		uptimeColor := ""
		if stats.UptimePercent >= 99.0 {
			uptimeColor = "🟢"
		} else if stats.UptimePercent >= 95.0 {
			uptimeColor = "🟡"
		} else {
			uptimeColor = "🔴"
		}

		fmt.Printf("%-20s %-12s %s%-7s %-10s %-12s %-15s\n", 
			name, checkType, uptimeColor, uptime, checks, avgRT, lastCheck)
	}

	return nil
}

// truncateString truncates a string to a specified length
func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}