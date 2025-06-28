package errors

import (
	"fmt"
	"time"
)

// ErrorType represents the category of error
type ErrorType string

const (
	// ErrorTypeValidation represents validation errors
	ErrorTypeValidation ErrorType = "validation"
	// ErrorTypeConfiguration represents configuration errors
	ErrorTypeConfiguration ErrorType = "configuration"
	// ErrorTypeNetwork represents network-related errors
	ErrorTypeNetwork ErrorType = "network"
	// ErrorTypeTimeout represents timeout errors
	ErrorTypeTimeout ErrorType = "timeout"
	// ErrorTypeAuthentication represents authentication errors
	ErrorTypeAuthentication ErrorType = "authentication"
	// ErrorTypeAuthorization represents authorization errors
	ErrorTypeAuthorization ErrorType = "authorization"
	// ErrorTypeStorage represents storage/database errors
	ErrorTypeStorage ErrorType = "storage"
	// ErrorTypeRateLimit represents rate limiting errors
	ErrorTypeRateLimit ErrorType = "rate_limit"
	// ErrorTypeCircuitBreaker represents circuit breaker errors
	ErrorTypeCircuitBreaker ErrorType = "circuit_breaker"
	// ErrorTypeInternal represents internal application errors
	ErrorTypeInternal ErrorType = "internal"
	// ErrorTypeExternal represents external service errors
	ErrorTypeExternal ErrorType = "external"
)

// Severity represents the severity level of an error
type Severity string

const (
	// SeverityLow represents low severity errors (warnings)
	SeverityLow Severity = "low"
	// SeverityMedium represents medium severity errors
	SeverityMedium Severity = "medium"
	// SeverityHigh represents high severity errors
	SeverityHigh Severity = "high"
	// SeverityCritical represents critical errors
	SeverityCritical Severity = "critical"
)

// HealthCheckError represents a standardized error in the health check system
type HealthCheckError struct {
	Type        ErrorType              `json:"type"`
	Severity    Severity               `json:"severity"`
	Message     string                 `json:"message"`
	Details     string                 `json:"details,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Cause       error                  `json:"-"` // Original error, not serialized
	Retryable   bool                   `json:"retryable"`
	StatusCode  int                    `json:"status_code,omitempty"`
	Component   string                 `json:"component,omitempty"`
}

// Error implements the error interface
func (e *HealthCheckError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s:%s] %s: %s", e.Type, e.Severity, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Type, e.Severity, e.Message)
}

// Unwrap returns the underlying error for error wrapping
func (e *HealthCheckError) Unwrap() error {
	return e.Cause
}

// IsRetryable returns whether the error is retryable
func (e *HealthCheckError) IsRetryable() bool {
	return e.Retryable
}

// WithContext adds context information to the error
func (e *HealthCheckError) WithContext(key string, value interface{}) *HealthCheckError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithCause sets the underlying cause of the error
func (e *HealthCheckError) WithCause(cause error) *HealthCheckError {
	e.Cause = cause
	return e
}

// WithComponent sets the component where the error occurred
func (e *HealthCheckError) WithComponent(component string) *HealthCheckError {
	e.Component = component
	return e
}

// NewError creates a new HealthCheckError
func NewError(errorType ErrorType, severity Severity, message string) *HealthCheckError {
	return &HealthCheckError{
		Type:      errorType,
		Severity:  severity,
		Message:   message,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
	}
}

// Error creation helpers for common scenarios

// NewValidationError creates a validation error
func NewValidationError(message string, details string) *HealthCheckError {
	return &HealthCheckError{
		Type:      ErrorTypeValidation,
		Severity:  SeverityMedium,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		Retryable: false,
	}
}

// NewConfigurationError creates a configuration error
func NewConfigurationError(message string, details string) *HealthCheckError {
	return &HealthCheckError{
		Type:      ErrorTypeConfiguration,
		Severity:  SeverityHigh,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		Retryable: false,
	}
}

// NewNetworkError creates a network error
func NewNetworkError(message string, statusCode int) *HealthCheckError {
	retryable := statusCode == 0 || statusCode >= 500 || statusCode == 429
	severity := SeverityMedium
	if statusCode >= 500 {
		severity = SeverityHigh
	}
	
	return &HealthCheckError{
		Type:       ErrorTypeNetwork,
		Severity:   severity,
		Message:    message,
		Timestamp:  time.Now(),
		Retryable:  retryable,
		StatusCode: statusCode,
	}
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(message string, duration time.Duration) *HealthCheckError {
	return &HealthCheckError{
		Type:      ErrorTypeTimeout,
		Severity:  SeverityMedium,
		Message:   message,
		Details:   fmt.Sprintf("timeout after %v", duration),
		Timestamp: time.Now(),
		Retryable: true,
		Context: map[string]interface{}{
			"timeout_duration": duration,
		},
	}
}

// NewStorageError creates a storage error
func NewStorageError(message string, cause error) *HealthCheckError {
	return &HealthCheckError{
		Type:      ErrorTypeStorage,
		Severity:  SeverityHigh,
		Message:   message,
		Timestamp: time.Now(),
		Cause:     cause,
		Retryable: true, // Most storage errors are transient
	}
}

// NewRateLimitError creates a rate limit error
func NewRateLimitError(message string, retryAfter time.Duration) *HealthCheckError {
	return &HealthCheckError{
		Type:      ErrorTypeRateLimit,
		Severity:  SeverityLow,
		Message:   message,
		Details:   fmt.Sprintf("retry after %v", retryAfter),
		Timestamp: time.Now(),
		Retryable: true,
		Context: map[string]interface{}{
			"retry_after": retryAfter,
		},
	}
}

// NewCircuitBreakerError creates a circuit breaker error
func NewCircuitBreakerError(message string, state string, retryAfter time.Duration) *HealthCheckError {
	return &HealthCheckError{
		Type:      ErrorTypeCircuitBreaker,
		Severity:  SeverityMedium,
		Message:   message,
		Details:   fmt.Sprintf("circuit breaker is %s, retry after %v", state, retryAfter),
		Timestamp: time.Now(),
		Retryable: true,
		Context: map[string]interface{}{
			"circuit_breaker_state": state,
			"retry_after":           retryAfter,
		},
	}
}

// NewInternalError creates an internal error
func NewInternalError(message string, cause error) *HealthCheckError {
	return &HealthCheckError{
		Type:      ErrorTypeInternal,
		Severity:  SeverityCritical,
		Message:   message,
		Timestamp: time.Now(),
		Cause:     cause,
		Retryable: false,
	}
}

// Error checking utilities

// IsErrorType checks if an error is of a specific type
func IsErrorType(err error, errorType ErrorType) bool {
	var hcErr *HealthCheckError
	if As(err, &hcErr) {
		return hcErr.Type == errorType
	}
	return false
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	var hcErr *HealthCheckError
	if As(err, &hcErr) {
		return hcErr.Retryable
	}
	return false
}

// GetErrorSeverity returns the severity of an error
func GetErrorSeverity(err error) Severity {
	var hcErr *HealthCheckError
	if As(err, &hcErr) {
		return hcErr.Severity
	}
	return SeverityMedium // Default severity for unknown errors
}

// GetErrorContext returns the context of an error
func GetErrorContext(err error) map[string]interface{} {
	var hcErr *HealthCheckError
	if As(err, &hcErr) {
		return hcErr.Context
	}
	return nil
}

// Error aggregation utilities

// ErrorCollector collects multiple errors
type ErrorCollector struct {
	errors []error
}

// NewErrorCollector creates a new error collector
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors: make([]error, 0),
	}
}

// Add adds an error to the collector
func (ec *ErrorCollector) Add(err error) {
	if err != nil {
		ec.errors = append(ec.errors, err)
	}
}

// HasErrors returns true if there are any errors
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// Count returns the number of errors
func (ec *ErrorCollector) Count() int {
	return len(ec.errors)
}

// Errors returns all collected errors
func (ec *ErrorCollector) Errors() []error {
	return ec.errors
}

// FirstError returns the first error or nil
func (ec *ErrorCollector) FirstError() error {
	if len(ec.errors) > 0 {
		return ec.errors[0]
	}
	return nil
}

// ToError returns an aggregated error or nil
func (ec *ErrorCollector) ToError() error {
	if len(ec.errors) == 0 {
		return nil
	}
	if len(ec.errors) == 1 {
		return ec.errors[0]
	}
	
	// Create an aggregated error
	messages := make([]string, len(ec.errors))
	for i, err := range ec.errors {
		messages[i] = err.Error()
	}
	
	return NewError(ErrorTypeInternal, SeverityMedium, 
		fmt.Sprintf("multiple errors occurred: %v", messages))
}

// Compatibility with standard library errors package
var (
	// As is equivalent to errors.As
	As = func(err error, target interface{}) bool {
		return asError(err, target)
	}
	
	// Is is equivalent to errors.Is
	Is = func(err, target error) bool {
		return isError(err, target)
	}
	
	// Unwrap is equivalent to errors.Unwrap
	Unwrap = func(err error) error {
		return unwrapError(err)
	}
)

// Simple implementations for compatibility
func asError(err error, target interface{}) bool {
	if err == nil {
		return false
	}
	
	// Type assertion
	if hcErr, ok := err.(*HealthCheckError); ok {
		if ptr, ok := target.(**HealthCheckError); ok {
			*ptr = hcErr
			return true
		}
	}
	
	// Check wrapped error
	if unwrapped := unwrapError(err); unwrapped != nil {
		return asError(unwrapped, target)
	}
	
	return false
}

func isError(err, target error) bool {
	if err == nil || target == nil {
		return err == target
	}
	
	if err == target {
		return true
	}
	
	// Check wrapped error
	if unwrapped := unwrapError(err); unwrapped != nil {
		return isError(unwrapped, target)
	}
	
	return false
}

func unwrapError(err error) error {
	if hcErr, ok := err.(*HealthCheckError); ok {
		return hcErr.Cause
	}
	return nil
}