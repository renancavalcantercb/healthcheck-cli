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
		result.Error = fmt.Sprintf("Response validation failed: %v", err)
		return result
	}
	
	// Check response time performance (even if other validations passed)
	if check.Expected.ResponseTimeMax > 0 && duration > check.Expected.ResponseTimeMax {
		result.Status = types.StatusSlow
		result.Error = fmt.Sprintf("Response time %v exceeds maximum %v", duration, check.Expected.ResponseTimeMax)
		return result
	}
	
	// All checks passed
	result.Status = types.StatusUp
	return result
}

// validateResponse validates the HTTP response against expected criteria
func (h *HTTPChecker) validateResponse(resp *http.Response, body []byte, expected types.Expected) error {
	// Check status code
	if expected.Status > 0 && resp.StatusCode != expected.Status {
		// Check if status range is defined
		if len(expected.StatusRange) == 2 {
			if resp.StatusCode < expected.StatusRange[0] || resp.StatusCode > expected.StatusRange[1] {
				return fmt.Errorf("status code %d not in expected range %d-%d", 
					resp.StatusCode, expected.StatusRange[0], expected.StatusRange[1])
			}
		} else {
			return fmt.Errorf("expected status %d, got %d", expected.Status, resp.StatusCode)
		}
	}
	
	bodyStr := string(body)
	
	// Check if body contains expected content
	if expected.BodyContains != "" && !strings.Contains(bodyStr, expected.BodyContains) {
		return fmt.Errorf("response body does not contain '%s'", expected.BodyContains)
	}
	
	// Check if body does NOT contain unwanted content
	if expected.BodyNotContains != "" && strings.Contains(bodyStr, expected.BodyNotContains) {
		return fmt.Errorf("response body contains unwanted content '%s'", expected.BodyNotContains)
	}
	
	// Check content type
	if expected.ContentType != "" {
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, expected.ContentType) {
			return fmt.Errorf("expected content type '%s', got '%s'", expected.ContentType, contentType)
		}
	}
	
	// Check minimum body size
	if expected.MinBodySize > 0 && int64(len(body)) < expected.MinBodySize {
		return fmt.Errorf("response body size %d bytes is less than minimum %d bytes", 
			len(body), expected.MinBodySize)
	}
	
	return nil
}
