package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStatus_String(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   string
	}{
		{"StatusUp", StatusUp, "UP"},
		{"StatusDown", StatusDown, "DOWN"},
		{"StatusSlow", StatusSlow, "SLOW"},
		{"StatusError", StatusError, "ERROR"},
		{"StatusWarning", StatusWarning, "WARNING"},
		{"InvalidStatus", Status(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.String())
		})
	}
}

func TestStatus_Emoji(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   string
	}{
		{"StatusUp", StatusUp, "🟢"},
		{"StatusDown", StatusDown, "🔴"},
		{"StatusSlow", StatusSlow, "🟡"},
		{"StatusError", StatusError, "❌"},
		{"StatusWarning", StatusWarning, "⚠️"},
		{"InvalidStatus", Status(999), "❓"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.Emoji())
		})
	}
}

func TestStatus_Color(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   string
	}{
		{"StatusUp", StatusUp, "\033[32m"},
		{"StatusDown", StatusDown, "\033[31m"},
		{"StatusSlow", StatusSlow, "\033[33m"},
		{"StatusError", StatusError, "\033[91m"},
		{"StatusWarning", StatusWarning, "\033[93m"},
		{"InvalidStatus", Status(999), "\033[37m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.Color())
		})
	}
}

func TestResult_IsHealthy(t *testing.T) {
	tests := []struct {
		name   string
		result Result
		want   bool
	}{
		{
			name: "StatusUp_IsHealthy",
			result: Result{
				Status: StatusUp,
			},
			want: true,
		},
		{
			name: "StatusDown_NotHealthy",
			result: Result{
				Status: StatusDown,
			},
			want: false,
		},
		{
			name: "StatusSlow_IsHealthy",
			result: Result{
				Status: StatusSlow,
			},
			want: true,
		},
		{
			name: "StatusError_NotHealthy",
			result: Result{
				Status: StatusError,
			},
			want: false,
		},
		{
			name: "StatusWarning_NotHealthy",
			result: Result{
				Status: StatusWarning,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.result.IsHealthy())
		})
	}
}

func TestResult_IsCritical(t *testing.T) {
	tests := []struct {
		name   string
		result Result
		want   bool
	}{
		{
			name: "StatusUp_NotCritical",
			result: Result{
				Status: StatusUp,
			},
			want: false,
		},
		{
			name: "StatusDown_IsCritical",
			result: Result{
				Status: StatusDown,
			},
			want: true,
		},
		{
			name: "StatusError_IsCritical",
			result: Result{
				Status: StatusError,
			},
			want: true,
		},
		{
			name: "StatusSlow_NotCritical",
			result: Result{
				Status: StatusSlow,
			},
			want: false,
		},
		{
			name: "StatusWarning_NotCritical",
			result: Result{
				Status: StatusWarning,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.result.IsCritical())
		})
	}
}

func TestCheckType_String(t *testing.T) {
	tests := []struct {
		name      string
		checkType CheckType
		want      string
	}{
		{"CheckTypeHTTP", CheckTypeHTTP, "http"},
		{"CheckTypeTCP", CheckTypeTCP, "tcp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.checkType.String())
		})
	}
}

func TestResult_Creation(t *testing.T) {
	now := time.Now()
	
	result := Result{
		Name:         "Test Service",
		URL:          "https://example.com",
		Status:       StatusUp,
		StatusCode:   200,
		ResponseTime: 100 * time.Millisecond,
		BodySize:     1024,
		Timestamp:    now,
		Error:        "",
		Headers:      map[string]string{"Content-Type": "application/json"},
	}

	assert.Equal(t, "Test Service", result.Name)
	assert.Equal(t, "https://example.com", result.URL)
	assert.Equal(t, StatusUp, result.Status)
	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, 100*time.Millisecond, result.ResponseTime)
	assert.Equal(t, int64(1024), result.BodySize)
	assert.Equal(t, now, result.Timestamp)
	assert.Empty(t, result.Error)
	assert.Contains(t, result.Headers, "Content-Type")
	assert.True(t, result.IsHealthy())
	assert.False(t, result.IsCritical())
}

func TestCheckConfig_Validation(t *testing.T) {
	config := CheckConfig{
		Name:     "Test Check",
		Type:     CheckTypeHTTP,
		URL:      "https://example.com",
		Method:   "GET",
		Timeout:  10 * time.Second,
		Interval: 30 * time.Second,
		Expected: Expected{
			Status:          200,
			ResponseTimeMax: 5 * time.Second,
		},
		Retry: RetryConfig{
			Attempts: 3,
			Delay:    2 * time.Second,
			Backoff:  "exponential",
			MaxDelay: 30 * time.Second,
		},
	}

	assert.Equal(t, "Test Check", config.Name)
	assert.Equal(t, CheckTypeHTTP, config.Type)
	assert.Equal(t, "https://example.com", config.URL)
	assert.Equal(t, "GET", config.Method)
	assert.Equal(t, 10*time.Second, config.Timeout)
	assert.Equal(t, 30*time.Second, config.Interval)
	assert.Equal(t, 200, config.Expected.Status)
	assert.Equal(t, 5*time.Second, config.Expected.ResponseTimeMax)
	assert.Equal(t, 3, config.Retry.Attempts)
	assert.Equal(t, 2*time.Second, config.Retry.Delay)
	assert.Equal(t, "exponential", config.Retry.Backoff)
	assert.Equal(t, 30*time.Second, config.Retry.MaxDelay)
}

func TestServiceStats_UptimeCalculation(t *testing.T) {
	stats := ServiceStats{
		Name:             "Test Service",
		URL:              "https://example.com",
		CheckType:        "http",
		TotalChecks:      100,
		SuccessfulChecks: 95,
		FailedChecks:     5,
		UptimePercent:    95.0,
	}

	assert.Equal(t, "Test Service", stats.Name)
	assert.Equal(t, int64(100), stats.TotalChecks)
	assert.Equal(t, int64(95), stats.SuccessfulChecks)
	assert.Equal(t, int64(5), stats.FailedChecks)
	assert.Equal(t, 95.0, stats.UptimePercent)
	
	// Verify uptime calculation consistency
	calculatedUptime := (float64(stats.SuccessfulChecks) / float64(stats.TotalChecks)) * 100
	assert.Equal(t, stats.UptimePercent, calculatedUptime)
}

func TestCheckResult_DatabaseFields(t *testing.T) {
	now := time.Now()
	
	result := CheckResult{
		ID:             1,
		Name:           "Test Service",
		URL:            "https://example.com",
		CheckType:      "http",
		Status:         int(StatusUp),
		Error:          "",
		ResponseTimeMs: 150,
		StatusCode:     200,
		BodySize:       2048,
		Timestamp:      now,
		CreatedAt:      now,
	}

	assert.Equal(t, int64(1), result.ID)
	assert.Equal(t, "Test Service", result.Name)
	assert.Equal(t, "https://example.com", result.URL)
	assert.Equal(t, "http", result.CheckType)
	assert.Equal(t, int(StatusUp), result.Status)
	assert.Empty(t, result.Error)
	assert.Equal(t, int64(150), result.ResponseTimeMs)
	assert.Equal(t, int(200), result.StatusCode)
	assert.Equal(t, int64(2048), result.BodySize)
	assert.Equal(t, now, result.Timestamp)
	assert.Equal(t, now, result.CreatedAt)
}

func TestGlobalConfig_DefaultValues(t *testing.T) {
	config := GlobalConfig{
		MaxWorkers:      10,
		DefaultTimeout:  10 * time.Second,
		DefaultInterval: 30 * time.Second,
		StoragePath:     "./healthcheck.db",
		LogLevel:        "info",
		DisableColors:   false,
		UserAgent:       "HealthCheck-CLI/1.0",
		MaxRetries:      3,
		RetryDelay:      5 * time.Second,
	}

	assert.Equal(t, 10, config.MaxWorkers)
	assert.Equal(t, 10*time.Second, config.DefaultTimeout)
	assert.Equal(t, 30*time.Second, config.DefaultInterval)
	assert.Equal(t, "./healthcheck.db", config.StoragePath)
	assert.Equal(t, "info", config.LogLevel)
	assert.False(t, config.DisableColors)
	assert.Equal(t, "HealthCheck-CLI/1.0", config.UserAgent)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 5*time.Second, config.RetryDelay)
}