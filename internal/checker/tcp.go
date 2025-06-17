package checker

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
)

// TCPChecker implements health checks for TCP endpoints
type TCPChecker struct {
	timeout time.Duration
}

// NewTCPChecker creates a new TCP checker
func NewTCPChecker(timeout time.Duration) *TCPChecker {
	return &TCPChecker{
		timeout: timeout,
	}
}

// Name returns the checker name
func (t *TCPChecker) Name() string {
	return "TCP"
}

// Check performs a TCP connectivity check
func (t *TCPChecker) Check(check types.CheckConfig) types.Result {
	start := time.Now()
	
	result := types.Result{
		Name:      check.Name,
		URL:       check.URL,
		Timestamp: start,
	}
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), check.Timeout)
	defer cancel()
	
	// Create dialer
	var d net.Dialer
	
	// Attempt connection
	conn, err := d.DialContext(ctx, "tcp", check.URL)
	end := time.Now()
	duration := end.Sub(start)
	result.ResponseTime = duration
	
	if err != nil {
		result.Status = types.StatusDown
		result.Error = fmt.Sprintf("TCP connection failed: %v", err)
		return result
	}
	
	conn.Close()
	
	// Check response time performance
	if check.Expected.ResponseTimeMax > 0 && duration > check.Expected.ResponseTimeMax {
		result.Status = types.StatusSlow
		result.Error = fmt.Sprintf("Connection time %v exceeds maximum %v", duration, check.Expected.ResponseTimeMax)
		return result
	}
	
	result.Status = types.StatusUp
	return result
}