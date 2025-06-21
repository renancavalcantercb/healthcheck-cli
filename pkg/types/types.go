package types

import (
	"time"
)

// GlobalConfig contains global application settings
type GlobalConfig struct {
	MaxWorkers        int           `yaml:"max_workers"`
	DefaultTimeout    time.Duration `yaml:"default_timeout"`
	DefaultInterval   time.Duration `yaml:"default_interval"`
	StoragePath       string        `yaml:"storage_path"`
	LogLevel          string        `yaml:"log_level"`
	DisableColors     bool          `yaml:"disable_colors"`
	UserAgent         string        `yaml:"user_agent"`
	MaxRetries        int           `yaml:"max_retries"`
	RetryDelay        time.Duration `yaml:"retry_delay"`
}

// ServiceStats represents aggregated statistics for a service
type ServiceStats struct {
	Name              string        `json:"name"`
	URL               string        `json:"url"`
	CheckType         string        `json:"check_type"`
	TotalChecks       int64         `json:"total_checks"`
	SuccessfulChecks  int64         `json:"successful_checks"`
	FailedChecks      int64         `json:"failed_checks"`
	AvgResponseTimeMs float64       `json:"avg_response_time_ms"`
	MinResponseTimeMs int64         `json:"min_response_time_ms"`
	MaxResponseTimeMs int64         `json:"max_response_time_ms"`
	UptimePercent     float64       `json:"uptime_percent"`
	LastCheck         time.Time     `json:"last_check"`
	LastSuccess       time.Time     `json:"last_success"`
	LastFailure       time.Time     `json:"last_failure"`
}

// CheckResult represents a stored check result
type CheckResult struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	URL            string    `json:"url"`
	CheckType      string    `json:"check_type"`
	Status         int       `json:"status"`
	Error          string    `json:"error"`
	ResponseTimeMs int64     `json:"response_time_ms"`
	StatusCode     int       `json:"status_code"`
	BodySize       int64     `json:"body_size"`
	Timestamp      time.Time `json:"timestamp"`
	CreatedAt      time.Time `json:"created_at"`
}

// Status represents the health status of an endpoint
type Status int

const (
	StatusUp Status = iota
	StatusDown
	StatusSlow
	StatusError
	StatusWarning
)

func (s Status) String() string {
	switch s {
	case StatusUp:
		return "UP"
	case StatusDown:
		return "DOWN"
	case StatusSlow:
		return "SLOW"
	case StatusError:
		return "ERROR"
	case StatusWarning:
		return "WARNING"
	default:
		return "UNKNOWN"
	}
}

// Emoji returns the emoji representation of the status
func (s Status) Emoji() string {
	switch s {
	case StatusUp:
		return "🟢"
	case StatusDown:
		return "🔴"
	case StatusSlow:
		return "🟡"
	case StatusError:
		return "❌"
	case StatusWarning:
		return "⚠️"
	default:
		return "❓"
	}
}

// Color returns the ANSI color code for the status
func (s Status) Color() string {
	switch s {
	case StatusUp:
		return "\033[32m" // Green
	case StatusDown:
		return "\033[31m" // Red
	case StatusSlow:
		return "\033[33m" // Yellow
	case StatusError:
		return "\033[91m" // Bright Red
	case StatusWarning:
		return "\033[93m" // Bright Yellow
	default:
		return "\033[37m" // White
	}
}

// Result represents the result of a health check
type Result struct {
	Name         string            `json:"name"`
	URL          string            `json:"url"`
	Status       Status            `json:"status"`
	Error        string            `json:"error,omitempty"`
	ResponseTime time.Duration     `json:"response_time"`
	StatusCode   int               `json:"status_code,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
	Headers      map[string]string `json:"headers,omitempty"`
	BodySize     int64             `json:"body_size,omitempty"`
	CertInfo     *CertInfo         `json:"cert_info,omitempty"`
}

// CertInfo represents SSL certificate information
type CertInfo struct {
	Subject     string    `json:"subject"`
	Issuer      string    `json:"issuer"`
	ExpiryDate  time.Time `json:"expiry_date"`
	DaysToExpiry int      `json:"days_to_expiry"`
	IsValid     bool      `json:"is_valid"`
	CommonName  string    `json:"common_name"`
	DNSNames    []string  `json:"dns_names"`
}

// IsHealthy returns true if the status indicates a healthy endpoint
func (r *Result) IsHealthy() bool {
	return r.Status == StatusUp || r.Status == StatusSlow
}

// IsCritical returns true if the status indicates a critical issue
func (r *Result) IsCritical() bool {
	return r.Status == StatusDown || r.Status == StatusError
}

// CheckType represents the type of health check
type CheckType string

const (
	CheckTypeHTTP CheckType = "http"
	CheckTypeTCP  CheckType = "tcp"
	CheckTypePing CheckType = "ping"
	CheckTypeSSL  CheckType = "ssl"
)

// String returns the string representation of CheckType
func (c CheckType) String() string {
	return string(c)
}

// Checker interface for different types of health checks
type Checker interface {
	Check(check CheckConfig) Result
	Name() string
}

// CheckConfig represents the configuration for a health check
type CheckConfig struct {
	Name     string            `yaml:"name" json:"name"`
	Type     CheckType         `yaml:"type" json:"type"`
	URL      string            `yaml:"url" json:"url"`
	Interval time.Duration     `yaml:"interval" json:"interval"`
	Timeout  time.Duration     `yaml:"timeout" json:"timeout"`
	Method   string            `yaml:"method" json:"method"`
	Headers  map[string]string `yaml:"headers" json:"headers"`
	Body     string            `yaml:"body" json:"body"`
	Expected Expected          `yaml:"expected" json:"expected"`
	Retry    RetryConfig       `yaml:"retry" json:"retry"`
	Tags     []string          `yaml:"tags" json:"tags"`
}

// Expected defines what constitutes a successful check
type Expected struct {
	Status           int           `yaml:"status" json:"status"`
	StatusRange      []int         `yaml:"status_range" json:"status_range"`
	BodyContains     string        `yaml:"body_contains" json:"body_contains"`
	BodyNotContains  string        `yaml:"body_not_contains" json:"body_not_contains"`
	ResponseTimeMax  time.Duration `yaml:"response_time_max" json:"response_time_max"`
	ContentType      string        `yaml:"content_type" json:"content_type"`
	MinBodySize      int64         `yaml:"min_body_size" json:"min_body_size"`
	CertExpiryDays   int           `yaml:"cert_expiry_days" json:"cert_expiry_days"`
	CertValidDomains []string      `yaml:"cert_valid_domains" json:"cert_valid_domains"`
}

// RetryConfig defines retry behavior
type RetryConfig struct {
	Attempts int           `yaml:"attempts" json:"attempts"`
	Delay    time.Duration `yaml:"delay" json:"delay"`
	Backoff  string        `yaml:"backoff" json:"backoff"` // linear, exponential
	MaxDelay time.Duration `yaml:"max_delay" json:"max_delay"`
}

// Notification represents a notification to be sent
type Notification struct {
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// Stats represents aggregated statistics for an endpoint
type Stats struct {
	Name             string        `json:"name"`
	URL              string        `json:"url"`
	TotalChecks      int64         `json:"total_checks"`
	SuccessfulChecks int64         `json:"successful_checks"`
	FailedChecks     int64         `json:"failed_checks"`
	AvgResponseTime  time.Duration `json:"avg_response_time"`
	MinResponseTime  time.Duration `json:"min_response_time"`
	MaxResponseTime  time.Duration `json:"max_response_time"`
	UptimePercent    float64       `json:"uptime_percent"`
	LastCheck        time.Time     `json:"last_check"`
	LastSuccess      time.Time     `json:"last_success"`
	LastFailure      time.Time     `json:"last_failure"`
}

// CalculateUptime calculates uptime percentage
func (s *Stats) CalculateUptime() float64 {
	if s.TotalChecks == 0 {
		return 0
	}
	return (float64(s.SuccessfulChecks) / float64(s.TotalChecks)) * 100
}
