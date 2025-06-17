package app

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
	"github.com/renancavalcantercb/healthcheck-cli/internal/tui"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
)

// App represents the main application
type App struct {
	httpChecker *checker.HTTPChecker
	tcpChecker  *checker.TCPChecker
	config      *config.Config
}

// New creates a new App instance
func New() *App {
	return &App{
		httpChecker: checker.NewHTTPChecker(30 * time.Second),
		tcpChecker:  checker.NewTCPChecker(10 * time.Second),
		config:      config.DefaultConfig(),
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