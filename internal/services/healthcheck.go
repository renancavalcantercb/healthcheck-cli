package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/renancavalcantercb/healthcheck-cli/pkg/circuitbreaker"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/interfaces"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/ratelimit"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
	"golang.org/x/time/rate"
)

// HealthCheckService implements the core health checking business logic
type HealthCheckService struct {
	checkers        map[types.CheckType]interfaces.Checker
	storage         interfaces.Storage
	notifier        interfaces.NotificationManager
	rateLimiter     ratelimit.Limiter
	circuitBreakers *circuitbreaker.Manager
	mu              sync.RWMutex
}

// NewHealthCheckService creates a new health check service
func NewHealthCheckService(
	checkers map[types.CheckType]interfaces.Checker,
	storage interfaces.Storage,
	notifier interfaces.NotificationManager,
) *HealthCheckService {
	// Create default rate limiter configuration
	rateLimitConfig := ratelimit.Config{
		DefaultLimit: rate.Every(time.Second), // 1 request per second
		DefaultBurst: 5,
		Enabled:      true,
	}
	
	return &HealthCheckService{
		checkers:    checkers,
		storage:     storage,
		notifier:    notifier,
		rateLimiter: ratelimit.NewPerEndpointLimiter(rateLimitConfig),
	}
}

// NewHealthCheckServiceWithRateLimit creates a new health check service with custom rate limiting
func NewHealthCheckServiceWithRateLimit(
	checkers map[types.CheckType]interfaces.Checker,
	storage interfaces.Storage,
	notifier interfaces.NotificationManager,
	rateLimitConfig types.RateLimitConfig,
) *HealthCheckService {
	var limiter ratelimit.Limiter
	
	if rateLimitConfig.Enabled {
		config := ratelimit.Config{
			DefaultLimit: rate.Limit(rateLimitConfig.DefaultLimit),
			DefaultBurst: rateLimitConfig.DefaultBurst,
			Enabled:      rateLimitConfig.Enabled,
		}
		limiter = ratelimit.NewPerEndpointLimiter(config)
	} else {
		// Use disabled rate limiter
		limiter = ratelimit.NewPerEndpointLimiter(ratelimit.DisabledConfig())
	}
	
	// Initialize circuit breaker manager with default config
	cbConfig := circuitbreaker.Config{
		MaxFailures:      5,
		Timeout:          60 * time.Second,
		SuccessThreshold: 3,
		Enabled:          true,
	}
	cbManager := circuitbreaker.NewManager(cbConfig)
	
	return &HealthCheckService{
		checkers:        checkers,
		storage:         storage,
		notifier:        notifier,
		rateLimiter:     limiter,
		circuitBreakers: cbManager,
	}
}

// NewHealthCheckServiceWithConfig creates a new health check service with full configuration
func NewHealthCheckServiceWithConfig(
	checkers map[types.CheckType]interfaces.Checker,
	storage interfaces.Storage,
	notifier interfaces.NotificationManager,
	rateLimitConfig types.RateLimitConfig,
	circuitBreakerConfig types.CircuitBreakerConfig,
) *HealthCheckService {
	var limiter ratelimit.Limiter
	
	if rateLimitConfig.Enabled {
		config := ratelimit.Config{
			DefaultLimit: rate.Limit(rateLimitConfig.DefaultLimit),
			DefaultBurst: rateLimitConfig.DefaultBurst,
			Enabled:      rateLimitConfig.Enabled,
		}
		limiter = ratelimit.NewPerEndpointLimiter(config)
	} else {
		limiter = ratelimit.NewPerEndpointLimiter(ratelimit.DisabledConfig())
	}
	
	// Initialize circuit breaker manager
	var cbManager *circuitbreaker.Manager
	if circuitBreakerConfig.Enabled {
		cbConfig := circuitbreaker.Config{
			MaxFailures:      circuitBreakerConfig.MaxFailures,
			Timeout:          circuitBreakerConfig.Timeout,
			SuccessThreshold: circuitBreakerConfig.SuccessThreshold,
			Enabled:          circuitBreakerConfig.Enabled,
		}
		cbManager = circuitbreaker.NewManager(cbConfig)
	} else {
		cbManager = circuitbreaker.NewManager(circuitbreaker.DisabledConfig())
	}
	
	return &HealthCheckService{
		checkers:        checkers,
		storage:         storage,
		notifier:        notifier,
		rateLimiter:     limiter,
		circuitBreakers: cbManager,
	}
}

// ExecuteCheck performs a single health check with retry logic
func (s *HealthCheckService) ExecuteCheck(ctx context.Context, check types.CheckConfig) (types.Result, error) {
	s.mu.RLock()
	checker, exists := s.checkers[check.Type]
	s.mu.RUnlock()
	
	if !exists {
		return types.Result{}, fmt.Errorf("unsupported check type: %s", check.Type)
	}

	// Apply rate limiting
	rateLimitKey := check.URL
	if s.rateLimiter != nil {
		if err := s.rateLimiter.Wait(ctx, rateLimitKey); err != nil {
			return types.Result{}, fmt.Errorf("rate limit wait failed: %w", err)
		}
	}

	var result types.Result
	maxAttempts := check.Retry.Attempts
	if maxAttempts == 0 {
		maxAttempts = 1
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return types.Result{}, ctx.Err()
		default:
		}

		// Execute check with circuit breaker protection
		if s.circuitBreakers != nil {
			breaker := s.circuitBreakers.GetBreaker(check.URL)
			err := breaker.Execute(ctx, func() error {
				result = checker.Check(check)
				// Consider the check failed if it's not healthy
				if !result.IsHealthy() {
					return fmt.Errorf("health check failed: %s", result.Error)
				}
				return nil
			})
			
			// If circuit breaker failed, create an appropriate result
			if err != nil {
				if circuitbreaker.IsCircuitBreakerOpen(err) {
					result = types.Result{
						Name:         check.Name,
						URL:          check.URL,
						Status:       types.StatusDown,
						Error:        err.Error(),
						Timestamp:    time.Now(),
						ResponseTime: 0, // No actual request was made
					}
				}
				// If it's not a circuit breaker error, the result is already set by the checker
			}
		} else {
			// No circuit breaker, execute directly
			result = checker.Check(check)
		}
		
		// If healthy or max attempts reached, break
		if result.IsHealthy() || attempt >= maxAttempts {
			break
		}
		
		// Calculate retry delay
		if attempt < maxAttempts {
			delay := s.calculateRetryDelay(check.Retry, attempt)
			select {
			case <-ctx.Done():
				return types.Result{}, ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	// Store result if storage is available
	if s.storage != nil {
		if err := s.storage.SaveResult(result); err != nil {
			log.Printf("Warning: failed to save result to storage: %v", err)
		}
	}

	return result, nil
}

// ExecuteChecks performs multiple health checks concurrently
func (s *HealthCheckService) ExecuteChecks(ctx context.Context, checks []types.CheckConfig) ([]types.Result, error) {
	if len(checks) == 0 {
		return nil, fmt.Errorf("no checks provided")
	}

	resultsChan := make(chan types.Result, len(checks))
	errorsChan := make(chan error, len(checks))
	
	var wg sync.WaitGroup
	
	for _, check := range checks {
		wg.Add(1)
		go func(c types.CheckConfig) {
			defer wg.Done()
			
			result, err := s.ExecuteCheck(ctx, c)
			if err != nil {
				errorsChan <- fmt.Errorf("check %s failed: %w", c.Name, err)
				return
			}
			
			resultsChan <- result
		}(check)
	}
	
	// Wait for all checks to complete
	wg.Wait()
	close(resultsChan)
	close(errorsChan)
	
	// Collect results
	var results []types.Result
	for result := range resultsChan {
		results = append(results, result)
	}
	
	// Collect errors
	var errors []error
	for err := range errorsChan {
		errors = append(errors, err)
	}
	
	// Return error if any checks failed
	if len(errors) > 0 {
		return results, fmt.Errorf("some checks failed: %v", errors)
	}
	
	return results, nil
}

// MonitorEndpoint starts continuous monitoring of a single endpoint
func (s *HealthCheckService) MonitorEndpoint(ctx context.Context, check types.CheckConfig) (<-chan types.Result, error) {
	if check.Interval == 0 {
		return nil, fmt.Errorf("check interval cannot be zero")
	}

	resultsChan := make(chan types.Result, 10) // Buffer for results
	
	go func() {
		defer close(resultsChan)
		
		ticker := time.NewTicker(check.Interval)
		defer ticker.Stop()
		
		// Perform initial check
		result, err := s.ExecuteCheck(ctx, check)
		if err != nil {
			log.Printf("Error in initial check for %s: %v", check.Name, err)
		} else {
			select {
			case resultsChan <- result:
			case <-ctx.Done():
				return
			}
			
			// Send notification
			if s.notifier != nil {
				if err := s.notifier.Notify(result); err != nil {
					log.Printf("Warning: failed to send notification: %v", err)
				}
			}
		}
		
		// Continue monitoring
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				result, err := s.ExecuteCheck(ctx, check)
				if err != nil {
					log.Printf("Error in check for %s: %v", check.Name, err)
					continue
				}
				
				select {
				case resultsChan <- result:
				case <-ctx.Done():
					return
				}
				
				// Send notification
				if s.notifier != nil {
					if err := s.notifier.Notify(result); err != nil {
						log.Printf("Warning: failed to send notification: %v", err)
					}
				}
			}
		}
	}()
	
	return resultsChan, nil
}

// StartMonitoring starts monitoring multiple endpoints
func (s *HealthCheckService) StartMonitoring(ctx context.Context, checks []types.CheckConfig) error {
	if len(checks) == 0 {
		return fmt.Errorf("no checks provided")
	}

	var wg sync.WaitGroup
	
	for _, check := range checks {
		wg.Add(1)
		go func(c types.CheckConfig) {
			defer wg.Done()
			
			resultsChan, err := s.MonitorEndpoint(ctx, c)
			if err != nil {
				log.Printf("Error starting monitoring for %s: %v", c.Name, err)
				return
			}
			
			// Consume results (they're already processed in MonitorEndpoint)
			for range resultsChan {
				// Results are handled in MonitorEndpoint
			}
		}(check)
	}
	
	wg.Wait()
	return nil
}

// calculateRetryDelay calculates the delay before retry based on backoff strategy
func (s *HealthCheckService) calculateRetryDelay(retry types.RetryConfig, attempt int) time.Duration {
	baseDelay := retry.Delay
	if baseDelay == 0 {
		baseDelay = 2 * time.Second
	}
	
	switch retry.Backoff {
	case "exponential":
		delay := baseDelay * time.Duration(1<<uint(attempt-1))
		if retry.MaxDelay > 0 && delay > retry.MaxDelay {
			return retry.MaxDelay
		}
		return delay
	case "linear":
		delay := baseDelay * time.Duration(attempt)
		if retry.MaxDelay > 0 && delay > retry.MaxDelay {
			return retry.MaxDelay
		}
		return delay
	default:
		return baseDelay
	}
}

// AddChecker adds a new checker to the service
func (s *HealthCheckService) AddChecker(checkType types.CheckType, checker interfaces.Checker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkers[checkType] = checker
}

// RemoveChecker removes a checker from the service
func (s *HealthCheckService) RemoveChecker(checkType types.CheckType) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.checkers, checkType)
}