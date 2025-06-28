package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
)

type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

type Config struct {
	Level  Level  `yaml:"level"`
	Format string `yaml:"format"` // json, text
	Output string `yaml:"output"` // stdout, stderr, file path
}

type Logger struct {
	*slog.Logger
}

func New(config Config) (*Logger, error) {
	var level slog.Level
	switch config.Level {
	case LevelDebug:
		level = slog.LevelDebug
	case LevelInfo:
		level = slog.LevelInfo
	case LevelWarn:
		level = slog.LevelWarn
	case LevelError:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var writer io.Writer
	switch config.Output {
	case "stdout", "":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	default:
		file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		writer = file
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: level,
		AddSource: true,
	}

	switch config.Format {
	case "json":
		handler = slog.NewJSONHandler(writer, opts)
	case "text", "":
		handler = slog.NewTextHandler(writer, opts)
	default:
		handler = slog.NewTextHandler(writer, opts)
	}

	logger := slog.New(handler)
	return &Logger{Logger: logger}, nil
}

func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{Logger: l.Logger.With()}
}

func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{Logger: l.Logger.With("component", component)}
}

func (l *Logger) WithCheckID(checkID string) *Logger {
	return &Logger{Logger: l.Logger.With("check_id", checkID)}
}

func (l *Logger) WithError(err error) *Logger {
	return &Logger{Logger: l.Logger.With("error", err.Error())}
}

func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return &Logger{Logger: l.Logger.With(args...)}
}

func (l *Logger) HealthCheckStarted(checkID, checkType, target string) {
	l.Info("Health check started",
		"check_id", checkID,
		"check_type", checkType,
		"target", target,
	)
}

func (l *Logger) HealthCheckCompleted(checkID string, success bool, duration string, details map[string]interface{}) {
	args := []interface{}{
		"check_id", checkID,
		"success", success,
		"duration", duration,
	}
	
	for k, v := range details {
		args = append(args, k, v)
	}

	if success {
		l.Info("Health check completed successfully", args...)
	} else {
		l.Error("Health check failed", args...)
	}
}

func (l *Logger) NotificationSent(notifier, checkID string, success bool) {
	l.Info("Notification sent",
		"notifier", notifier,
		"check_id", checkID,
		"success", success,
	)
}

func (l *Logger) DatabaseOperation(operation string, table string, success bool, err error) {
	args := []interface{}{
		"operation", operation,
		"table", table,
		"success", success,
	}
	
	if err != nil {
		args = append(args, "error", err.Error())
		l.Error("Database operation failed", args...)
	} else {
		l.Info("Database operation completed", args...)
	}
}

func (l *Logger) ConfigLoaded(configPath string, checksCount int) {
	l.Info("Configuration loaded",
		"config_path", configPath,
		"checks_count", checksCount,
	)
}

func (l *Logger) ApplicationStarted(version string) {
	l.Info("Healthcheck CLI started",
		"version", version,
		"pid", os.Getpid(),
	)
}

func (l *Logger) ApplicationStopped() {
	l.Info("Healthcheck CLI stopped")
}