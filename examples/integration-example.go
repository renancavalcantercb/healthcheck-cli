// Example showing how to integrate logging into existing config and main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/renancavalcantercb/healthcheck-cli/internal/config"
	"github.com/renancavalcantercb/healthcheck-cli/internal/logger"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Example Config structure (should be merged with existing config.go)
type Config struct {
	App struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	} `yaml:"app"`
	
	Logging config.LoggingConfig `yaml:"logging"`
	
	Database struct {
		Path string `yaml:"path"`
	} `yaml:"database"`
	
	Checks []HealthCheck `yaml:"checks"`
	
	Notifications struct {
		Discord DiscordConfig `yaml:"discord"`
		Email   EmailConfig   `yaml:"email"`
	} `yaml:"notifications"`
}

type HealthCheck struct {
	ID       string        `yaml:"id"`
	Type     string        `yaml:"type"`
	URL      string        `yaml:"url,omitempty"`
	Host     string        `yaml:"host,omitempty"`
	Port     int           `yaml:"port,omitempty"`
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
}

type DiscordConfig struct {
	WebhookURL string `yaml:"webhook_url"`
	Enabled    bool   `yaml:"enabled"`
}

type EmailConfig struct {
	SMTPHost string   `yaml:"smtp_host"`
	SMTPPort int      `yaml:"smtp_port"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	To       []string `yaml:"to"`
	Enabled  bool     `yaml:"enabled"`
}

// Example of how to integrate logging into main.go
func main() {
	var configFile string
	
	rootCmd := &cobra.Command{
		Use:   "healthcheck",
		Short: "Health monitoring CLI tool",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHealthcheck(configFile)
		},
	}
	
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "config.yml", "Configuration file path")
	
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runHealthcheck(configFile string) error {
	// 1. Load configuration
	cfg, err := loadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	// 2. Initialize logger early in the application lifecycle
	appLogger, err := logger.New(logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		Output: cfg.Logging.Output,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	
	// 3. Log application startup
	appLogger.ApplicationStarted(cfg.App.Version)
	appLogger.ConfigLoaded(configFile, len(cfg.Checks))
	
	// 4. Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// 5. Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		appLogger.Info("Received shutdown signal")
		cancel()
	}()
	
	// 6. Initialize components with logger
	app, err := initializeApp(cfg, appLogger)
	if err != nil {
		appLogger.Error("Failed to initialize application", "error", err.Error())
		return err
	}
	
	// 7. Start application
	appLogger.Info("Starting health monitoring")
	
	if err := app.Run(ctx); err != nil {
		appLogger.Error("Application error", "error", err.Error())
		return err
	}
	
	// 8. Log application shutdown
	appLogger.ApplicationStopped()
	return nil
}

func loadConfig(configFile string) (*Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	
	var cfg Config
	
	// Set defaults
	cfg.Logging = config.DefaultLoggingConfig()
	
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	
	return &cfg, nil
}

// Example of how to initialize app components with logger
func initializeApp(cfg *Config, appLogger *logger.Logger) (*App, error) {
	// Initialize database with logging
	db, err := initializeDatabase(cfg.Database.Path, appLogger)
	if err != nil {
		return nil, err
	}
	
	// Initialize checkers with logging
	checkers := make([]HealthChecker, 0, len(cfg.Checks))
	for _, checkCfg := range cfg.Checks {
		checker, err := createChecker(checkCfg, appLogger)
		if err != nil {
			return nil, fmt.Errorf("failed to create checker %s: %w", checkCfg.ID, err)
		}
		checkers = append(checkers, checker)
	}
	
	// Initialize notifications with logging
	notificationManager := initializeNotifications(cfg.Notifications, appLogger)
	
	return &App{
		logger:              appLogger,
		checkers:           checkers,
		notificationManager: notificationManager,
		database:           db,
	}, nil
}

// Mock types for example (these would be actual implementations)
type App struct {
	logger              *logger.Logger
	checkers           []HealthChecker
	notificationManager NotificationManager
	database           Database
}

type HealthChecker interface {
	Check(ctx context.Context) error
}

type NotificationManager interface {
	Send(message string) error
}

type Database interface {
	SaveResult(result CheckResult) error
}

type CheckResult struct {
	CheckID   string
	Success   bool
	Timestamp time.Time
	Details   map[string]interface{}
}

func (a *App) Run(ctx context.Context) error {
	a.logger.Info("Application started successfully")
	
	// Main application loop would go here
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			a.logger.Info("Shutting down gracefully")
			return nil
		case <-ticker.C:
			a.runHealthChecks(ctx)
		}
	}
}

func (a *App) runHealthChecks(ctx context.Context) {
	for _, checker := range a.checkers {
		go func(c HealthChecker) {
			// This is where you'd add proper health check logging
			// Using the specialized logger methods
		}(checker)
	}
}

// Mock initialization functions
func initializeDatabase(path string, logger *logger.Logger) (Database, error) {
	logger.Info("Initializing database", "path", path)
	return nil, nil
}

func createChecker(cfg HealthCheck, logger *logger.Logger) (HealthChecker, error) {
	logger.Info("Creating health checker", "id", cfg.ID, "type", cfg.Type)
	return nil, nil
}

func initializeNotifications(cfg struct{ Discord DiscordConfig; Email EmailConfig }, logger *logger.Logger) NotificationManager {
	logger.Info("Initializing notification system")
	return nil
}