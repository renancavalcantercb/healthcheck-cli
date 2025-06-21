package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/renancavalcantercb/healthcheck-cli/internal/checker"
	"github.com/renancavalcantercb/healthcheck-cli/internal/config"
	"github.com/renancavalcantercb/healthcheck-cli/internal/notifications"
	"github.com/renancavalcantercb/healthcheck-cli/internal/services"
	"github.com/renancavalcantercb/healthcheck-cli/internal/storage"
	"github.com/renancavalcantercb/healthcheck-cli/internal/tui"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/interfaces"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"

	tea "github.com/charmbracelet/bubbletea"
)

// Application implements the main application with service layer architecture
type Application struct {
	// Core services
	healthCheckService interfaces.HealthCheckService
	statsService       interfaces.StatsService
	configService      interfaces.ConfigService
	
	// Infrastructure
	storage  interfaces.Storage
	notifier interfaces.NotificationManager
	
	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc
}

// Dependencies contains all the dependencies needed to create an Application
type Dependencies struct {
	Storage  interfaces.Storage
	Notifier interfaces.NotificationManager
	Checkers map[types.CheckType]interfaces.Checker
}

// NewApplication creates a new application instance with dependency injection
func NewApplication(deps Dependencies) *Application {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create services
	healthCheckService := services.NewHealthCheckService(deps.Checkers, deps.Storage, deps.Notifier)
	statsService := services.NewStatsService(deps.Storage)
	configService := services.NewConfigService()
	
	return &Application{
		healthCheckService: healthCheckService,
		statsService:       statsService,
		configService:      configService,
		storage:            deps.Storage,
		notifier:           deps.Notifier,
		ctx:                ctx,
		cancel:             cancel,
	}
}

// NewApplicationWithDefaults creates an application with default dependencies
func NewApplicationWithDefaults() (*Application, error) {
	// Initialize storage
	storage, err := storage.NewSQLiteStorage("./healthcheck.db")
	if err != nil {
		log.Printf("⚠️  Warning: Failed to initialize storage: %v", err)
		log.Println("💡 Continuing without storage (data won't be persisted)")
	}
	
	// Initialize checkers
	checkers := map[types.CheckType]interfaces.Checker{
		types.CheckTypeHTTP: checker.NewHTTPChecker(30 * time.Second),
		types.CheckTypeTCP:  checker.NewTCPChecker(10 * time.Second),
	}
	
	// Initialize notification manager
	defaultConfig := config.DefaultConfig()
	notifier := notifications.NewManager(defaultConfig)
	
	deps := Dependencies{
		Storage:  storage,
		Notifier: notifier,
		Checkers: checkers,
	}
	
	app := NewApplication(deps)
	
	// Start background cleanup if storage is available
	if storage != nil {
		go app.startBackgroundCleanup()
	}
	
	return app, nil
}

// Start starts the application
func (a *Application) Start(ctx context.Context) error {
	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		log.Println("\n🛑 Received shutdown signal, stopping...")
		a.cancel()
	}()
	
	// Wait for context cancellation
	<-ctx.Done()
	return nil
}

// Stop stops the application
func (a *Application) Stop() error {
	a.cancel()
	
	if a.storage != nil {
		return a.storage.Close()
	}
	
	return nil
}

// HealthCheck returns the health check service
func (a *Application) HealthCheck() interfaces.HealthCheckService {
	return a.healthCheckService
}

// Stats returns the stats service
func (a *Application) Stats() interfaces.StatsService {
	return a.statsService
}

// Config returns the config service
func (a *Application) Config() interfaces.ConfigService {
	return a.configService
}

// TestEndpoint tests a single endpoint immediately
func (a *Application) TestEndpoint(url string, timeout time.Duration, verbose bool) error {
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
	
	// Determine check type based on URL
	if !isHTTPURL(url) {
		check.Type = types.CheckTypeTCP
		check.Method = ""
	}
	
	result, err := a.healthCheckService.ExecuteCheck(a.ctx, check)
	if err != nil {
		return fmt.Errorf("check execution failed: %w", err)
	}
	
	a.printResult(result, verbose)
	return nil
}

// QuickCheck starts monitoring a single URL with minimal configuration
func (a *Application) QuickCheck(url string, interval time.Duration, daemon bool) error {
	if interval == 0 {
		interval = 30 * time.Second
	}
	
	fmt.Printf("🚀 Starting health check for %s\n", url)
	fmt.Printf("📊 Check interval: %v\n", interval)
	
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
	if !isHTTPURL(url) {
		check.Type = types.CheckTypeTCP
		check.Method = ""
	}
	
	if daemon {
		fmt.Println("🔄 Running in daemon mode (Press Ctrl+C to stop)")
		resultsChan, err := a.healthCheckService.MonitorEndpoint(a.ctx, check)
		if err != nil {
			return fmt.Errorf("failed to start monitoring: %w", err)
		}
		
		// Process results
		for result := range resultsChan {
			a.printResult(result, false)
		}
		
		return nil
	}
	
	// Single run
	result, err := a.healthCheckService.ExecuteCheck(a.ctx, check)
	if err != nil {
		return fmt.Errorf("check execution failed: %w", err)
	}
	
	a.printResult(result, false)
	
	if !result.IsHealthy() {
		return fmt.Errorf("check failed")
	}
	
	return nil
}

// LoadConfigAndRun loads configuration and runs health checks
func (a *Application) LoadConfigAndRun(configFile string, daemon bool) error {
	return a.LoadConfigAndRunWithEnv(configFile, "", daemon)
}

// LoadConfigAndRunWithEnv loads configuration with environment variables and runs health checks
func (a *Application) LoadConfigAndRunWithEnv(configFile, envFile string, daemon bool) error {
	// Load configuration with environment variables
	cfg, err := config.LoadConfigWithEnv(configFile, envFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	// Update notification manager with new config
	a.notifier = a.notifier.UpdateConfig(cfg)
	
	// Convert to CheckConfig format
	checks := make([]types.CheckConfig, len(cfg.Checks))
	for i, check := range cfg.Checks {
		checks[i] = check.CheckConfig
	}
	
	fmt.Printf("🔧 Loaded configuration from %s\n", configFile)
	if envFile != "" {
		fmt.Printf("🔧 Loaded environment variables from %s\n", envFile)
	}
	fmt.Printf("📊 Monitoring %d endpoints\n", len(checks))
	
	if daemon {
		fmt.Println("🔄 Running in daemon mode (Press Ctrl+C to stop)")
		return a.healthCheckService.StartMonitoring(a.ctx, checks)
	}
	
	// Single run
	fmt.Println("🏃 Running all checks once...")
	results, err := a.healthCheckService.ExecuteChecks(a.ctx, checks)
	if err != nil {
		return fmt.Errorf("checks execution failed: %w", err)
	}
	
	for _, result := range results {
		a.printResult(result, false)
	}
	
	return nil
}

// ShowStatus shows a status dashboard
func (a *Application) ShowStatus(watch bool) error {
	if !watch {
		fmt.Println("📊 Static status not implemented yet")
		fmt.Println("💡 Use --watch for interactive dashboard")
		return nil
	}
	
	checks := a.configService.GetChecks()
	if len(checks) == 0 {
		// Default example check
		checks = []types.CheckConfig{
			{
				Name:     "Example",
				URL:      "https://httpbin.org/get",
				Type:     types.CheckTypeHTTP,
				Method:   "GET",
				Timeout:  10 * time.Second,
				Expected: types.Expected{Status: 200},
			},
		}
	}
	
	return a.runTUIDashboard(checks)
}

// startBackgroundCleanup performs periodic cleanup of old data
func (a *Application) startBackgroundCleanup() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			if err := a.statsService.CleanupOldData(30 * 24 * time.Hour); err != nil {
				log.Printf("Warning: cleanup failed: %v", err)
			}
		}
	}
}

// Helper functions

func isHTTPURL(url string) bool {
	return len(url) > 4 && (url[:4] == "http" || url[:5] == "https")
}

func (a *Application) printResult(result types.Result, verbose bool) {
	globalConfig := a.configService.GetGlobalConfig()
	
	status := result.Status.Emoji() + " " + result.Status.String()
	if !globalConfig.DisableColors {
		status = result.Status.Color() + status + "\033[0m"
	}
	
	fmt.Printf("[%s] %s %s - %v",
		result.Timestamp.Format("15:04:05"),
		status,
		result.Name,
		result.ResponseTime,
	)
	
	if result.StatusCode > 0 {
		fmt.Printf(" (HTTP %d)", result.StatusCode)
	}
	
	if result.Error != "" {
		fmt.Printf(" - %s", result.Error)
	}
	
	fmt.Println()
	
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

func (a *Application) runTUIDashboard(checks []types.CheckConfig) error {
	model := tui.New()
	program := tea.NewProgram(model, tea.WithAltScreen())
	
	resultsChan := make(chan []types.Result, 10)
	
	// Start monitoring for TUI
	for _, check := range checks {
		go a.monitorForTUI(check, resultsChan)
	}
	
	go func() {
		resultsMap := make(map[string]types.Result)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-a.ctx.Done():
				return
			case result := <-resultsChan:
				for _, r := range result {
					resultsMap[r.Name] = r
				}
			case <-ticker.C:
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
	
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("TUI program failed: %w", err)
	}
	
	return nil
}

func (a *Application) monitorForTUI(check types.CheckConfig, resultsChan chan<- []types.Result) {
	resultStream, err := a.healthCheckService.MonitorEndpoint(a.ctx, check)
	if err != nil {
		log.Printf("Error starting monitoring for TUI: %v", err)
		return
	}
	
	for result := range resultStream {
		select {
		case resultsChan <- []types.Result{result}:
		case <-a.ctx.Done():
			return
		}
	}
}