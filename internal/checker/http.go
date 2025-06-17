package checker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
)

// HTTPChecker implements health checks for HTTP/HTTPS endpoints
type HTTPChecker struct {
	client *http.Client
}

// NewHTTPChecker creates a new HTTP checker
func NewHTTPChecker(timeout time.Duration) *HTTPChecker {
	return &HTTPChecker{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// Name returns the checker name
func (h *HTTPChecker) Name() string {
	return "HTTP"
}

// Check performs an HTTP health check
func (h *HTTPChecker) Check(check types.CheckConfig) types.Result {
	start := time.Now()
	
	result := types.Result{
		Name:      check.Name,
		URL:       check.URL,
		Timestamp: start,
	}
	
	// Create request context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), check.Timeout)
	defer cancel()
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, check.Method, check.URL, strings.NewReader(check.Body))
	if err != nil {
		result.Status = types.StatusError
		result.Error = fmt.Sprintf("Failed to create request: %v", err)
		result.ResponseTime = time.Since(start)
		return result
	}
	
	// Add headers
	for key, value := range check.Headers {
		req.Header.Set(key, value)
	}
	
	// Set User-Agent if not provided
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "HealthCheck-CLI/1.0")
	}
	
	// Perform request
	resp, err := h.client.Do(req)
	duration := time.Since(start)
	result.ResponseTime = duration
	
	if err != nil {
		result.Status = types.StatusDown
		result.Error = fmt.Sprintf("Request failed: %v", err)
		return result
	}
	defer resp.Body.Close()
	
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Status = types.StatusError
		result.Error = fmt.Sprintf("Failed to read response body: %v", err)
		result.StatusCode = resp.StatusCode
		return result
	}
	
	result.StatusCode = resp.StatusCode
	result.BodySize = int64(len(body))
	result.Headers = make(map[string]string)
	
	// Copy important headers
	for _, header := range []string{"Content-Type", "Content-Length", "Server", "Cache-Control"} {
		if value := resp.Header.Get(header); value != "" {
			result.Headers[header] = value
		}
	}
	
	// Validate response
	if err := h.validateResponse(resp, body, check.Expected); err != nil {
		// Check if it's a performance issue or actual failure
		if duration > check.Expected.ResponseTimeMax && check.Expected.ResponseTimeMax > 0 {
			result.Status = types.StatusSlow
		} else {
			result.Status = types.StatusDown
		}

	}

	result.Status = types.StatusUp
	return result
}

// validateResponse validates the response against the expected criteria
func (h *HTTPChecker) validateResponse(resp *http.Response, body []byte, expected types.Expected) error {
	if expected.Status != 0 && resp.StatusCode != expected.Status {
		return fmt.Errorf("expected status %d but got %d", expected.Status, resp.StatusCode)
	}
	return nil
}