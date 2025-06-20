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
	"github.com/renancavalcantercb/healthcheck-cli/internal/config"
	"github.com/renancavalcantercb/healthcheck-cli/internal/tui"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
)

// StartQuick starts monitoring a single URL with minimal configuration
func (a *App) StartQuick(url string, interval time.Duration, daemon bool) error {
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
	a.notifier = a.notifier.UpdateConfig(cfg)
	
	fmt.Printf("🔧 Loaded configuration from %s\n", configFile)
	fmt.Printf("📊 Monitoring %d endpoints\n", len(cfg.Checks))
	
	checks := make([]types.CheckConfig, len(cfg.Checks))
	for i, c := range cfg.Checks {
		checks[i] = c.CheckConfig
	}
	
	if daemon {
		fmt.Println("🔄 Running in daemon mode (Press Ctrl+C to stop)")
		return a.runDaemon(checks)
	}
	
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
	
	if !strings.HasPrefix(url, "http") {
		check.Type = types.CheckTypeTCP
		check.Method = ""
	}
	
	result := a.performCheck(check)
	a.printResult(result, verbose)
	
	return nil
}

// ShowStatus shows a status dashboard
func (a *App) ShowStatus(watch bool) error {
	if !watch {
		fmt.Println("📊 Static status not implemented yet")
		fmt.Println("💡 Use --watch for interactive dashboard")
		return nil
	}
	
	var checks []types.CheckConfig
	if len(a.config.Checks) > 0 {
		for _, c := range a.config.Checks {
			checks = append(checks, c.CheckConfig)
		}
	} else {
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
		return config.SaveExample("")
	}
	
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		fmt.Println("\n🛑 Received shutdown signal, stopping...")
		cancel()
	}()
	
	resultChan := make(chan types.Result, len(checks)*2)
	
	for _, check := range checks {
		go a.monitorEndpoint(ctx, check, resultChan)
	}
	
	for {
		select {
		case <-ctx.Done():
			fmt.Println("👋 Shutdown complete")
			return nil
		case result := <-resultChan:
			a.printResult(result, false)
			if err := a.notifier.Notify(result); err != nil {
				fmt.Printf("⚠️  Erro ao enviar notificação: %v\n", err)
			}
		}
	}
}

// runTUIDashboard runs the terminal UI dashboard
func (a *App) runTUIDashboard(checks []types.CheckConfig) error {
	model := tui.New()
	program := tea.NewProgram(model, tea.WithAltScreen())
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	resultsChan := make(chan []types.Result, 10)
	
	for _, check := range checks {
		go a.monitorForTUI(ctx, check, resultsChan)
	}
	
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

// monitorEndpoint monitors a single endpoint continuously
func (a *App) monitorEndpoint(ctx context.Context, check types.CheckConfig, resultChan chan<- types.Result) {
	ticker := time.NewTicker(check.Interval)
	defer ticker.Stop()
	
	result := a.performCheck(check)
	select {
	case resultChan <- result:
	case <-ctx.Done():
		return
	}
	
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

// monitorForTUI monitors a single endpoint for the TUI
func (a *App) monitorForTUI(ctx context.Context, check types.CheckConfig, resultsChan chan<- []types.Result) {
	ticker := time.NewTicker(check.Interval)
	defer ticker.Stop()
	
	result := a.performCheck(check)
	select {
	case resultsChan <- []types.Result{result}:
	case <-ctx.Done():
		return
	}
	
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

// printResult prints a check result to the console
func (a *App) printResult(result types.Result, verbose bool) {
	status := result.Status.Emoji() + " " + result.Status.String()
	if !a.config.Global.DisableColors {
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

// showVersion displays version information
func showVersion() {
	fmt.Printf("healthcheck version %s\n", version)
}