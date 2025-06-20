package interfaces

import (
	"context"
	"time"

	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
)

// Storage defines the interface for data persistence
type Storage interface {
	SaveResult(result types.Result) error
	GetServiceStats(serviceName string, since time.Time) (*types.ServiceStats, error)
	GetAllServiceStats(since time.Time) ([]types.ServiceStats, error)
	GetServiceHistory(serviceName string, since time.Time, limit int) ([]types.CheckResult, error)
	GetDatabaseInfo() (map[string]interface{}, error)
	CleanupOldData(maxAge time.Duration) error
	Close() error
}

// Checker defines the interface for health check implementations
type Checker interface {
	Check(check types.CheckConfig) types.Result
	Name() string
}

// NotificationManager defines the interface for notification handling
type NotificationManager interface {
	Notify(result types.Result) error
	UpdateConfig(config interface{}) NotificationManager
}

// HealthCheckService defines the core business logic interface
type HealthCheckService interface {
	ExecuteCheck(ctx context.Context, check types.CheckConfig) (types.Result, error)
	ExecuteChecks(ctx context.Context, checks []types.CheckConfig) ([]types.Result, error)
	MonitorEndpoint(ctx context.Context, check types.CheckConfig) (<-chan types.Result, error)
	StartMonitoring(ctx context.Context, checks []types.CheckConfig) error
}

// ConfigService defines the interface for configuration management
type ConfigService interface {
	Load(filePath string) error
	Save(filePath string) error
	Validate() error
	GetChecks() []types.CheckConfig
	GetGlobalConfig() types.GlobalConfig
	GetNotificationConfig() interface{}
}

// StatsService defines the interface for statistics and analytics
type StatsService interface {
	GetServiceStats(serviceName string, since time.Time) (*types.ServiceStats, error)
	GetAllStats(since time.Time) ([]types.ServiceStats, error)
	GetHistory(serviceName string, since time.Time, limit int) ([]types.CheckResult, error)
	GetDatabaseInfo() (map[string]interface{}, error)
	CleanupOldData(maxAge time.Duration) error
}

// Application defines the main application interface
type Application interface {
	// Core operations
	Start(ctx context.Context) error
	Stop() error
	
	// Services
	HealthCheck() HealthCheckService
	Stats() StatsService
	Config() ConfigService
	
	// Quick operations
	TestEndpoint(url string, timeout time.Duration, verbose bool) error
	QuickCheck(url string, interval time.Duration, daemon bool) error
	
	// Configuration-based operations
	LoadConfigAndRun(configFile string, daemon bool) error
	ShowStatus(watch bool) error
}