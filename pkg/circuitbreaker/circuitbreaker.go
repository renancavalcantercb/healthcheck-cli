package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// State represents the circuit breaker state
type State int

const (
	// StateClosed means the circuit breaker is allowing requests through
	StateClosed State = iota
	// StateOpen means the circuit breaker is rejecting requests
	StateOpen
	// StateHalfOpen means the circuit breaker is testing if the service has recovered
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// Config represents circuit breaker configuration
type Config struct {
	// MaxFailures is the maximum number of failures before opening the circuit
	MaxFailures int `yaml:"max_failures" json:"max_failures"`
	// Timeout is how long to wait before attempting to close the circuit again
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
	// SuccessThreshold is the number of successful calls needed to close the circuit from half-open state
	SuccessThreshold int `yaml:"success_threshold" json:"success_threshold"`
	// Enabled determines if circuit breaking is active
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// Metrics represents circuit breaker metrics
type Metrics struct {
	State            State     `json:"state"`
	Failures         int       `json:"failures"`
	Successes        int       `json:"successes"`
	TotalRequests    int64     `json:"total_requests"`
	ConsecutiveSuccesses int   `json:"consecutive_successes"`
	LastFailureTime  time.Time `json:"last_failure_time"`
	LastSuccessTime  time.Time `json:"last_success_time"`
	LastStateChange  time.Time `json:"last_state_change"`
}

// CircuitBreaker interface defines the circuit breaker contract
type CircuitBreaker interface {
	// Execute executes the given function with circuit breaker protection
	Execute(ctx context.Context, fn func() error) error
	// Call executes a function and returns both result and error
	Call(ctx context.Context, fn func() (interface{}, error)) (interface{}, error)
	// State returns the current state of the circuit breaker
	State() State
	// Metrics returns current metrics
	Metrics() *Metrics
	// Reset resets the circuit breaker to closed state
	Reset()
}

// circuitBreaker implements the CircuitBreaker interface
type circuitBreaker struct {
	config   Config
	state    State
	failures int
	successes int
	consecutiveSuccesses int
	totalRequests       int64
	lastFailureTime     time.Time
	lastSuccessTime     time.Time
	lastStateChange     time.Time
	mu                  sync.RWMutex
}

// New creates a new circuit breaker with the given configuration
func New(config Config) CircuitBreaker {
	return &circuitBreaker{
		config:          config,
		state:           StateClosed,
		lastStateChange: time.Now(),
	}
}

// DefaultConfig returns a default circuit breaker configuration
func DefaultConfig() Config {
	return Config{
		MaxFailures:      5,
		Timeout:          60 * time.Second,
		SuccessThreshold: 3,
		Enabled:          true,
	}
}

// DisabledConfig returns a configuration with circuit breaking disabled
func DisabledConfig() Config {
	return Config{
		Enabled: false,
	}
}

// Execute executes the given function with circuit breaker protection
func (cb *circuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if !cb.config.Enabled {
		return fn()
	}

	// Check if circuit breaker should allow the request
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	// Execute the function
	err := fn()

	// Record the result
	cb.afterRequest(err)

	return err
}

// Call executes a function and returns both result and error
func (cb *circuitBreaker) Call(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	if !cb.config.Enabled {
		return fn()
	}

	// Check if circuit breaker should allow the request
	if err := cb.beforeRequest(); err != nil {
		return nil, err
	}

	// Execute the function
	result, err := fn()

	// Record the result
	cb.afterRequest(err)

	return result, err
}

// State returns the current state of the circuit breaker
func (cb *circuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Metrics returns current metrics
func (cb *circuitBreaker) Metrics() *Metrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return &Metrics{
		State:                cb.state,
		Failures:             cb.failures,
		Successes:            cb.successes,
		TotalRequests:        cb.totalRequests,
		ConsecutiveSuccesses: cb.consecutiveSuccesses,
		LastFailureTime:      cb.lastFailureTime,
		LastSuccessTime:      cb.lastSuccessTime,
		LastStateChange:      cb.lastStateChange,
	}
}

// Reset resets the circuit breaker to closed state
func (cb *circuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.consecutiveSuccesses = 0
	cb.lastStateChange = time.Now()
}

// beforeRequest checks if the request should be allowed
func (cb *circuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalRequests++

	switch cb.state {
	case StateClosed:
		// Allow request
		return nil
	case StateOpen:
		// Check if timeout has passed
		if time.Since(cb.lastStateChange) > cb.config.Timeout {
			cb.state = StateHalfOpen
			cb.consecutiveSuccesses = 0
			cb.lastStateChange = time.Now()
			return nil
		}
		// Circuit is open, reject request
		return &CircuitBreakerOpenError{
			State:           cb.state,
			LastFailure:     cb.lastFailureTime,
			TimeUntilRetry:  cb.config.Timeout - time.Since(cb.lastStateChange),
		}
	case StateHalfOpen:
		// Allow limited requests to test the service
		return nil
	default:
		return fmt.Errorf("unknown circuit breaker state: %v", cb.state)
	}
}

// afterRequest records the result of the request
func (cb *circuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.onFailure()
	} else {
		cb.onSuccess()
	}
}

// onSuccess handles a successful request
func (cb *circuitBreaker) onSuccess() {
	cb.successes++
	cb.consecutiveSuccesses++
	cb.lastSuccessTime = time.Now()

	switch cb.state {
	case StateHalfOpen:
		// Check if we have enough consecutive successes to close the circuit
		if cb.consecutiveSuccesses >= cb.config.SuccessThreshold {
			cb.state = StateClosed
			cb.failures = 0
			cb.lastStateChange = time.Now()
		}
	case StateClosed:
		// Reset failure count on success in closed state
		cb.failures = 0
	}
}

// onFailure handles a failed request
func (cb *circuitBreaker) onFailure() {
	cb.failures++
	cb.consecutiveSuccesses = 0
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		// Check if we should open the circuit
		if cb.failures >= cb.config.MaxFailures {
			cb.state = StateOpen
			cb.lastStateChange = time.Now()
		}
	case StateHalfOpen:
		// Any failure in half-open state should immediately open the circuit
		cb.state = StateOpen
		cb.lastStateChange = time.Now()
	}
}

// CircuitBreakerOpenError is returned when the circuit breaker is open
type CircuitBreakerOpenError struct {
	State          State
	LastFailure    time.Time
	TimeUntilRetry time.Duration
}

func (e *CircuitBreakerOpenError) Error() string {
	return fmt.Sprintf("circuit breaker is %s (will retry in %v)", 
		e.State.String(), e.TimeUntilRetry.Round(time.Second))
}

// IsCircuitBreakerOpen checks if an error is a circuit breaker open error
func IsCircuitBreakerOpen(err error) bool {
	var cbErr *CircuitBreakerOpenError
	return errors.As(err, &cbErr)
}

// Manager manages multiple circuit breakers for different endpoints
type Manager struct {
	breakers map[string]CircuitBreaker
	config   Config
	mu       sync.RWMutex
}

// NewManager creates a new circuit breaker manager
func NewManager(config Config) *Manager {
	return &Manager{
		breakers: make(map[string]CircuitBreaker),
		config:   config,
	}
}

// GetBreaker returns a circuit breaker for the given key, creating one if it doesn't exist
func (m *Manager) GetBreaker(key string) CircuitBreaker {
	m.mu.RLock()
	if breaker, exists := m.breakers[key]; exists {
		m.mu.RUnlock()
		return breaker
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if breaker, exists := m.breakers[key]; exists {
		return breaker
	}

	// Create new circuit breaker
	m.breakers[key] = New(m.config)
	return m.breakers[key]
}

// GetAllMetrics returns metrics for all circuit breakers
func (m *Manager) GetAllMetrics() map[string]*Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := make(map[string]*Metrics)
	for key, breaker := range m.breakers {
		metrics[key] = breaker.Metrics()
	}
	return metrics
}

// Reset resets all circuit breakers
func (m *Manager) Reset() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, breaker := range m.breakers {
		breaker.Reset()
	}
}

// RemoveBreaker removes a circuit breaker for the given key
func (m *Manager) RemoveBreaker(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.breakers, key)
}