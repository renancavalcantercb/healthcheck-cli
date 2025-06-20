package main

import (
	"fmt"
	"os"
	"time"

	"github.com/renancavalcantercb/healthcheck-cli/internal/checker"
	"github.com/renancavalcantercb/healthcheck-cli/internal/config"
	"github.com/renancavalcantercb/healthcheck-cli/internal/notifications"
	"github.com/renancavalcantercb/healthcheck-cli/internal/storage"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
)

// version will be set during build
var version = "dev"

// App represents the main application
type App struct {
	httpChecker *checker.HTTPChecker
	tcpChecker  *checker.TCPChecker
	storage     *storage.SQLiteStorage
	config      *config.Config
	notifier    *notifications.Manager
}

// New creates a new App instance
func New() *App {
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

	app.notifier = notifications.NewManager(app.config)

	if storage != nil {
		go app.backgroundCleanup()
	}

	return app
}

// main is the entry point of the application
func main() {
	app := New()
	defer app.Close()

	rootCmd := setupCommands(app)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// performCheck executes a health check with retry logic
func (a *App) performCheck(check types.CheckConfig) types.Result {
	var result types.Result
	
	maxAttempts := check.Retry.Attempts
	if maxAttempts == 0 {
		maxAttempts = 1
	}
	
	for attempt := 1; attempt <= maxAttempts; attempt++ {
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
		
		if result.IsHealthy() || attempt >= maxAttempts {
			break
		}
		
		if attempt < maxAttempts {
			delay := a.calculateRetryDelay(check.Retry, attempt)
			time.Sleep(delay)
		}
	}
	
	if a.storage != nil {
		if err := a.storage.SaveResult(result); err != nil {
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
		delay := baseDelay * time.Duration(1<<uint(attempt-1))
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
	default:
		return baseDelay
	}
}

// backgroundCleanup performs periodic cleanup of old data
func (a *App) backgroundCleanup() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		if a.storage != nil {
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