package config

import "github.com/renancavalcantercb/healthcheck-cli/internal/logger"

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level  logger.Level `yaml:"level"`
	Format string       `yaml:"format"`
	Output string       `yaml:"output"`
}

// DefaultLoggingConfig returns default logging configuration
func DefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		Level:  logger.LevelInfo,
		Format: "text",
		Output: "stdout",
	}
}