package validator

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/renancavalcantercb/healthcheck-cli/pkg/errors"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
)

// ConfigValidator provides enhanced configuration validation
type ConfigValidator struct {
	errorCollector *errors.ErrorCollector
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		errorCollector: errors.NewErrorCollector(),
	}
}

// ValidateGlobalConfig validates global configuration settings
func (v *ConfigValidator) ValidateGlobalConfig(config types.GlobalConfig) error {
	v.errorCollector = errors.NewErrorCollector()

	// Validate worker count
	if config.MaxWorkers <= 0 {
		v.errorCollector.Add(errors.NewValidationError(
			"Invalid max_workers", 
			"max_workers must be greater than 0",
		).WithContext("value", config.MaxWorkers))
	} else if config.MaxWorkers > 1000 {
		v.errorCollector.Add(errors.NewValidationError(
			"Excessive max_workers",
			"max_workers should not exceed 1000 for stability",
		).WithContext("value", config.MaxWorkers))
	}

	// Validate timeout settings
	if config.DefaultTimeout <= 0 {
		v.errorCollector.Add(errors.NewValidationError(
			"Invalid default_timeout",
			"default_timeout must be greater than 0",
		).WithContext("value", config.DefaultTimeout))
	} else if config.DefaultTimeout > 5*time.Minute {
		v.errorCollector.Add(errors.NewValidationError(
			"Excessive default_timeout",
			"default_timeout should not exceed 5 minutes",
		).WithContext("value", config.DefaultTimeout))
	}

	// Validate interval settings
	if config.DefaultInterval <= 0 {
		v.errorCollector.Add(errors.NewValidationError(
			"Invalid default_interval",
			"default_interval must be greater than 0",
		).WithContext("value", config.DefaultInterval))
	} else if config.DefaultInterval < time.Second {
		v.errorCollector.Add(errors.NewValidationError(
			"Too frequent default_interval",
			"default_interval should be at least 1 second",
		).WithContext("value", config.DefaultInterval))
	}

	// Validate timeout vs interval relationship
	if config.DefaultTimeout >= config.DefaultInterval {
		v.errorCollector.Add(errors.NewValidationError(
			"Invalid timeout/interval relationship",
			"default_timeout should be less than default_interval",
		).WithContext("timeout", config.DefaultTimeout).
			WithContext("interval", config.DefaultInterval))
	}

	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error", "fatal"}
	if !contains(validLogLevels, strings.ToLower(config.LogLevel)) {
		v.errorCollector.Add(errors.NewValidationError(
			"Invalid log_level",
			fmt.Sprintf("log_level must be one of: %v", validLogLevels),
		).WithContext("value", config.LogLevel))
	}

	// Validate User-Agent
	if config.UserAgent == "" {
		v.errorCollector.Add(errors.NewValidationError(
			"Empty user_agent",
			"user_agent should not be empty",
		))
	}

	// Validate retry settings
	if config.MaxRetries < 0 {
		v.errorCollector.Add(errors.NewValidationError(
			"Invalid max_retries",
			"max_retries cannot be negative",
		).WithContext("value", config.MaxRetries))
	} else if config.MaxRetries > 10 {
		v.errorCollector.Add(errors.NewValidationError(
			"Excessive max_retries",
			"max_retries should not exceed 10",
		).WithContext("value", config.MaxRetries))
	}

	if config.RetryDelay < 0 {
		v.errorCollector.Add(errors.NewValidationError(
			"Invalid retry_delay",
			"retry_delay cannot be negative",
		).WithContext("value", config.RetryDelay))
	}

	// Validate rate limit configuration
	v.validateRateLimitConfig(config.RateLimit)

	// Validate circuit breaker configuration
	v.validateCircuitBreakerConfig(config.CircuitBreaker)

	// Validate memory management configuration
	v.validateMemoryManagementConfig(config.MemoryManagement)

	return v.errorCollector.ToError()
}

// ValidateCheckConfig validates a single check configuration
func (v *ConfigValidator) ValidateCheckConfig(check types.CheckConfig, index int) error {
	v.errorCollector = errors.NewErrorCollector()

	prefix := fmt.Sprintf("check[%d]", index)

	// Validate name
	if check.Name == "" {
		v.errorCollector.Add(errors.NewValidationError(
			fmt.Sprintf("%s: missing name", prefix),
			"check name is required",
		))
	} else if len(check.Name) > 100 {
		v.errorCollector.Add(errors.NewValidationError(
			fmt.Sprintf("%s: name too long", prefix),
			"check name should not exceed 100 characters",
		).WithContext("name", check.Name))
	}

	// Validate URL
	if check.URL == "" {
		v.errorCollector.Add(errors.NewValidationError(
			fmt.Sprintf("%s: missing URL", prefix),
			"check URL is required",
		))
	} else {
		if err := v.validateURL(check.URL, check.Type); err != nil {
			v.errorCollector.Add(err.(*errors.HealthCheckError).
				WithContext("check_index", index).
				WithContext("check_name", check.Name))
		}
	}

	// Validate check type
	validTypes := []types.CheckType{types.CheckTypeHTTP, types.CheckTypeTCP, types.CheckTypeSSL}
	if !containsCheckType(validTypes, check.Type) {
		v.errorCollector.Add(errors.NewValidationError(
			fmt.Sprintf("%s: invalid type", prefix),
			fmt.Sprintf("check type must be one of: %v", validTypes),
		).WithContext("type", check.Type))
	}

	// Validate timing configuration
	if check.Interval <= 0 {
		v.errorCollector.Add(errors.NewValidationError(
			fmt.Sprintf("%s: invalid interval", prefix),
			"check interval must be greater than 0",
		).WithContext("interval", check.Interval))
	}

	if check.Timeout <= 0 {
		v.errorCollector.Add(errors.NewValidationError(
			fmt.Sprintf("%s: invalid timeout", prefix),
			"check timeout must be greater than 0",
		).WithContext("timeout", check.Timeout))
	}

	if check.Timeout >= check.Interval {
		v.errorCollector.Add(errors.NewValidationError(
			fmt.Sprintf("%s: timeout exceeds interval", prefix),
			"check timeout must be less than interval",
		).WithContext("timeout", check.Timeout).
			WithContext("interval", check.Interval))
	}

	// Validate HTTP-specific settings
	if check.Type == types.CheckTypeHTTP {
		if err := v.validateHTTPSettings(check, prefix); err != nil {
			v.errorCollector.Add(err)
		}
	}

	// Validate expected settings
	if err := v.validateExpectedSettings(check.Expected, prefix); err != nil {
		v.errorCollector.Add(err)
	}

	// Validate retry settings
	if err := v.validateRetrySettings(check.Retry, prefix); err != nil {
		v.errorCollector.Add(err)
	}

	return v.errorCollector.ToError()
}

// validateURL validates URL format based on check type
func (v *ConfigValidator) validateURL(rawURL string, checkType types.CheckType) error {
	switch checkType {
	case types.CheckTypeHTTP:
		if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
			return errors.NewValidationError(
				"Invalid HTTP URL",
				"HTTP checks require http:// or https:// URL",
			).WithContext("url", rawURL)
		}
		
		parsed, err := url.Parse(rawURL)
		if err != nil {
			return errors.NewValidationError(
				"Malformed URL",
				err.Error(),
			).WithContext("url", rawURL)
		}
		
		if parsed.Host == "" {
			return errors.NewValidationError(
				"Missing hostname",
				"URL must include a hostname",
			).WithContext("url", rawURL)
		}

	case types.CheckTypeTCP, types.CheckTypeSSL:
		if strings.Contains(rawURL, "://") {
			return errors.NewValidationError(
				"Invalid TCP/SSL URL",
				"TCP and SSL checks should use host:port format",
			).WithContext("url", rawURL)
		}
		
		// Validate host:port format
		hostPortRegex := regexp.MustCompile(`^[a-zA-Z0-9.-]+:[0-9]+$`)
		if !hostPortRegex.MatchString(rawURL) {
			return errors.NewValidationError(
				"Invalid host:port format",
				"URL should be in format host:port",
			).WithContext("url", rawURL)
		}
	}

	return nil
}

// validateHTTPSettings validates HTTP-specific configuration
func (v *ConfigValidator) validateHTTPSettings(check types.CheckConfig, prefix string) error {
	// Validate HTTP method
	validMethods := []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS", "PATCH"}
	if check.Method != "" && !contains(validMethods, strings.ToUpper(check.Method)) {
		return errors.NewValidationError(
			fmt.Sprintf("%s: invalid HTTP method", prefix),
			fmt.Sprintf("HTTP method must be one of: %v", validMethods),
		).WithContext("method", check.Method)
	}

	// Validate headers
	for key, value := range check.Headers {
		if key == "" {
			return errors.NewValidationError(
				fmt.Sprintf("%s: empty header name", prefix),
				"header names cannot be empty",
			)
		}
		if len(value) > 1000 {
			return errors.NewValidationError(
				fmt.Sprintf("%s: header value too long", prefix),
				"header values should not exceed 1000 characters",
			).WithContext("header", key).WithContext("value_length", len(value))
		}
	}

	return nil
}

// validateExpectedSettings validates expected response criteria
func (v *ConfigValidator) validateExpectedSettings(expected types.Expected, prefix string) error {
	// Validate status code
	if expected.Status != 0 && (expected.Status < 100 || expected.Status > 599) {
		return errors.NewValidationError(
			fmt.Sprintf("%s: invalid expected status", prefix),
			"expected status must be a valid HTTP status code (100-599)",
		).WithContext("status", expected.Status)
	}

	// Validate status range
	if len(expected.StatusRange) == 2 {
		if expected.StatusRange[0] < 100 || expected.StatusRange[0] > 599 ||
			expected.StatusRange[1] < 100 || expected.StatusRange[1] > 599 {
			return errors.NewValidationError(
				fmt.Sprintf("%s: invalid status range", prefix),
				"status range values must be valid HTTP status codes (100-599)",
			).WithContext("status_range", expected.StatusRange)
		}
		if expected.StatusRange[0] > expected.StatusRange[1] {
			return errors.NewValidationError(
				fmt.Sprintf("%s: invalid status range", prefix),
				"status range start must be less than or equal to end",
			).WithContext("status_range", expected.StatusRange)
		}
	} else if len(expected.StatusRange) > 0 {
		return errors.NewValidationError(
			fmt.Sprintf("%s: invalid status range", prefix),
			"status range must have exactly 2 values",
		).WithContext("status_range", expected.StatusRange)
	}

	// Validate response time maximum
	if expected.ResponseTimeMax < 0 {
		return errors.NewValidationError(
			fmt.Sprintf("%s: invalid response time max", prefix),
			"response time max cannot be negative",
		).WithContext("response_time_max", expected.ResponseTimeMax)
	}

	// Validate minimum body size
	if expected.MinBodySize < 0 {
		return errors.NewValidationError(
			fmt.Sprintf("%s: invalid min body size", prefix),
			"minimum body size cannot be negative",
		).WithContext("min_body_size", expected.MinBodySize)
	}

	return nil
}

// validateRetrySettings validates retry configuration
func (v *ConfigValidator) validateRetrySettings(retry types.RetryConfig, prefix string) error {
	if retry.Attempts < 0 {
		return errors.NewValidationError(
			fmt.Sprintf("%s: invalid retry attempts", prefix),
			"retry attempts cannot be negative",
		).WithContext("attempts", retry.Attempts)
	}
	if retry.Attempts > 10 {
		return errors.NewValidationError(
			fmt.Sprintf("%s: excessive retry attempts", prefix),
			"retry attempts should not exceed 10",
		).WithContext("attempts", retry.Attempts)
	}

	if retry.Delay < 0 {
		return errors.NewValidationError(
			fmt.Sprintf("%s: invalid retry delay", prefix),
			"retry delay cannot be negative",
		).WithContext("delay", retry.Delay)
	}

	validBackoffs := []string{"", "linear", "exponential"}
	if !contains(validBackoffs, retry.Backoff) {
		return errors.NewValidationError(
			fmt.Sprintf("%s: invalid backoff strategy", prefix),
			fmt.Sprintf("backoff must be one of: %v", validBackoffs),
		).WithContext("backoff", retry.Backoff)
	}

	return nil
}

// validateRateLimitConfig validates rate limiting configuration
func (v *ConfigValidator) validateRateLimitConfig(config types.RateLimitConfig) {
	if config.Enabled {
		if config.DefaultLimit <= 0 {
			v.errorCollector.Add(errors.NewValidationError(
				"Invalid rate limit",
				"default_limit must be greater than 0 when rate limiting is enabled",
			).WithContext("default_limit", config.DefaultLimit))
		}
		if config.DefaultBurst <= 0 {
			v.errorCollector.Add(errors.NewValidationError(
				"Invalid rate limit burst",
				"default_burst must be greater than 0 when rate limiting is enabled",
			).WithContext("default_burst", config.DefaultBurst))
		}
	}
}

// validateCircuitBreakerConfig validates circuit breaker configuration
func (v *ConfigValidator) validateCircuitBreakerConfig(config types.CircuitBreakerConfig) {
	if config.Enabled {
		if config.MaxFailures <= 0 {
			v.errorCollector.Add(errors.NewValidationError(
				"Invalid circuit breaker max failures",
				"max_failures must be greater than 0 when circuit breaker is enabled",
			).WithContext("max_failures", config.MaxFailures))
		}
		if config.Timeout <= 0 {
			v.errorCollector.Add(errors.NewValidationError(
				"Invalid circuit breaker timeout",
				"timeout must be greater than 0 when circuit breaker is enabled",
			).WithContext("timeout", config.Timeout))
		}
		if config.SuccessThreshold <= 0 {
			v.errorCollector.Add(errors.NewValidationError(
				"Invalid circuit breaker success threshold",
				"success_threshold must be greater than 0 when circuit breaker is enabled",
			).WithContext("success_threshold", config.SuccessThreshold))
		}
	}
}

// validateMemoryManagementConfig validates memory management configuration
func (v *ConfigValidator) validateMemoryManagementConfig(config types.MemoryManagementConfig) {
	if config.Enabled {
		if config.MaxHistoryPerService <= 0 {
			v.errorCollector.Add(errors.NewValidationError(
				"Invalid memory management history limit",
				"max_history_per_service must be greater than 0 when memory management is enabled",
			).WithContext("max_history_per_service", config.MaxHistoryPerService))
		}
		if config.MaxHistoryAge <= 0 {
			v.errorCollector.Add(errors.NewValidationError(
				"Invalid memory management age limit",
				"max_history_age must be greater than 0 when memory management is enabled",
			).WithContext("max_history_age", config.MaxHistoryAge))
		}
		if config.CleanupInterval <= 0 {
			v.errorCollector.Add(errors.NewValidationError(
				"Invalid memory management cleanup interval",
				"cleanup_interval must be greater than 0 when memory management is enabled",
			).WithContext("cleanup_interval", config.CleanupInterval))
		}
	}
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsCheckType(slice []types.CheckType, item types.CheckType) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}