package services

import (
	"fmt"

	"github.com/renancavalcantercb/healthcheck-cli/internal/config"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
)

// ConfigService implements configuration management functionality
type ConfigService struct {
	config *config.Config
}

// NewConfigService creates a new config service
func NewConfigService() *ConfigService {
	return &ConfigService{
		config: config.DefaultConfig(),
	}
}

// Load loads configuration from a file
func (s *ConfigService) Load(filePath string) error {
	cfg, err := config.LoadConfig(filePath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	s.config = cfg
	return nil
}

// Save saves configuration to a file
func (s *ConfigService) Save(filePath string) error {
	return fmt.Errorf("save functionality not yet implemented")
}

// Validate validates the current configuration
func (s *ConfigService) Validate() error {
	if s.config == nil {
		return fmt.Errorf("no configuration loaded")
	}
	
	return s.config.Validate()
}

// GetChecks returns the list of health checks
func (s *ConfigService) GetChecks() []types.CheckConfig {
	if s.config == nil {
		return nil
	}
	
	checks := make([]types.CheckConfig, len(s.config.Checks))
	for i, c := range s.config.Checks {
		checks[i] = c.CheckConfig
	}
	
	return checks
}

// GetGlobalConfig returns the global configuration
func (s *ConfigService) GetGlobalConfig() types.GlobalConfig {
	if s.config == nil {
		return types.GlobalConfig{}
	}
	
	return types.GlobalConfig{
		MaxWorkers:      s.config.Global.MaxWorkers,
		DefaultTimeout:  s.config.Global.DefaultTimeout,
		DefaultInterval: s.config.Global.DefaultInterval,
		StoragePath:     s.config.Global.StoragePath,
		LogLevel:        s.config.Global.LogLevel,
		DisableColors:   s.config.Global.DisableColors,
		UserAgent:       s.config.Global.UserAgent,
		MaxRetries:      s.config.Global.MaxRetries,
		RetryDelay:      s.config.Global.RetryDelay,
	}
}

// GetNotificationConfig returns the notification configuration
func (s *ConfigService) GetNotificationConfig() interface{} {
	if s.config == nil {
		return nil
	}
	
	return s.config.Notifications
}

// GetConfig returns the raw configuration (for backward compatibility)
func (s *ConfigService) GetConfig() *config.Config {
	return s.config
}

// SetConfig sets the configuration (for dependency injection)
func (s *ConfigService) SetConfig(cfg *config.Config) {
	s.config = cfg
}