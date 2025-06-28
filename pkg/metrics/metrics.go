package metrics

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
)

// MetricType represents the type of metric
type MetricType string

const (
	// MetricTypeCounter represents a counter metric (monotonically increasing)
	MetricTypeCounter MetricType = "counter"
	// MetricTypeGauge represents a gauge metric (can go up and down)
	MetricTypeGauge MetricType = "gauge"
	// MetricTypeHistogram represents a histogram metric (distribution of values)
	MetricTypeHistogram MetricType = "histogram"
	// MetricTypeSummary represents a summary metric (quantiles)
	MetricTypeSummary MetricType = "summary"
)

// Metric represents a single metric data point
type Metric struct {
	Name        string                 `json:"name"`
	Type        MetricType             `json:"type"`
	Value       float64                `json:"value"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Description string                 `json:"description,omitempty"`
	Unit        string                 `json:"unit,omitempty"`
}

// HealthCheckMetrics contains metrics specific to health checking
type HealthCheckMetrics struct {
	// Request metrics
	RequestsTotal       *Counter   `json:"requests_total"`
	RequestDuration     *Histogram `json:"request_duration"`
	RequestsInFlight    *Gauge     `json:"requests_in_flight"`
	
	// Response metrics
	ResponsesTotal      *Counter   `json:"responses_total"`
	ResponseSizeBytes   *Histogram `json:"response_size_bytes"`
	
	// Error metrics
	ErrorsTotal         *Counter   `json:"errors_total"`
	TimeoutsTotal       *Counter   `json:"timeouts_total"`
	
	// Circuit breaker metrics
	CircuitBreakerState *Gauge     `json:"circuit_breaker_state"`
	CircuitBreakerTrips *Counter   `json:"circuit_breaker_trips"`
	
	// Rate limiter metrics
	RateLimitedTotal    *Counter   `json:"rate_limited_total"`
	RateLimitWaitTime   *Histogram `json:"rate_limit_wait_time"`
	
	// System metrics
	MemoryUsageBytes    *Gauge     `json:"memory_usage_bytes"`
	GoroutinesActive    *Gauge     `json:"goroutines_active"`
	
	mu sync.RWMutex
}

// Counter represents a monotonically increasing counter
type Counter struct {
	value  float64
	labels map[string]string
	mu     sync.RWMutex
}

// Gauge represents a value that can go up and down
type Gauge struct {
	value  float64
	labels map[string]string
	mu     sync.RWMutex
}

// Histogram represents a distribution of values
type Histogram struct {
	buckets map[float64]float64 // bucket -> count
	sum     float64
	count   float64
	labels  map[string]string
	mu      sync.RWMutex
}

// Summary represents quantile estimates
type Summary struct {
	sum     float64
	count   float64
	samples []float64
	labels  map[string]string
	mu      sync.RWMutex
}

// MetricsCollector collects and manages metrics
type MetricsCollector struct {
	metrics map[string]*Metric
	mu      sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]*Metric),
	}
}

// NewHealthCheckMetrics creates a new health check metrics instance
func NewHealthCheckMetrics() *HealthCheckMetrics {
	return &HealthCheckMetrics{
		RequestsTotal:       NewCounter(),
		RequestDuration:     NewHistogram([]float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}),
		RequestsInFlight:    NewGauge(),
		ResponsesTotal:      NewCounter(),
		ResponseSizeBytes:   NewHistogram([]float64{100, 500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000}),
		ErrorsTotal:         NewCounter(),
		TimeoutsTotal:       NewCounter(),
		CircuitBreakerState: NewGauge(),
		CircuitBreakerTrips: NewCounter(),
		RateLimitedTotal:    NewCounter(),
		RateLimitWaitTime:   NewHistogram([]float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1}),
		MemoryUsageBytes:    NewGauge(),
		GoroutinesActive:    NewGauge(),
	}
}

// Counter methods
func NewCounter() *Counter {
	return &Counter{
		labels: make(map[string]string),
	}
}

func (c *Counter) Inc() {
	c.Add(1)
}

func (c *Counter) Add(value float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value += value
}

func (c *Counter) Value() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.value
}

func (c *Counter) WithLabels(labels map[string]string) *Counter {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range labels {
		c.labels[k] = v
	}
	return c
}

// Gauge methods
func NewGauge() *Gauge {
	return &Gauge{
		labels: make(map[string]string),
	}
}

func (g *Gauge) Set(value float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value = value
}

func (g *Gauge) Inc() {
	g.Add(1)
}

func (g *Gauge) Dec() {
	g.Add(-1)
}

func (g *Gauge) Add(value float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value += value
}

func (g *Gauge) Value() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.value
}

func (g *Gauge) WithLabels(labels map[string]string) *Gauge {
	g.mu.Lock()
	defer g.mu.Unlock()
	for k, v := range labels {
		g.labels[k] = v
	}
	return g
}

// Histogram methods
func NewHistogram(buckets []float64) *Histogram {
	h := &Histogram{
		buckets: make(map[float64]float64),
		labels:  make(map[string]string),
	}
	for _, bucket := range buckets {
		h.buckets[bucket] = 0
	}
	return h
}

func (h *Histogram) Observe(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	h.sum += value
	h.count++
	
	// Increment bucket counters
	for bucket := range h.buckets {
		if value <= bucket {
			h.buckets[bucket]++
		}
	}
}

func (h *Histogram) Sum() float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.sum
}

func (h *Histogram) Count() float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.count
}

func (h *Histogram) Buckets() map[float64]float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	buckets := make(map[float64]float64)
	for k, v := range h.buckets {
		buckets[k] = v
	}
	return buckets
}

func (h *Histogram) WithLabels(labels map[string]string) *Histogram {
	h.mu.Lock()
	defer h.mu.Unlock()
	for k, v := range labels {
		h.labels[k] = v
	}
	return h
}

// MetricsCollector methods
func (mc *MetricsCollector) RecordMetric(name string, metricType MetricType, value float64, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.metrics[name] = &Metric{
		Name:      name,
		Type:      metricType,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}
}

func (mc *MetricsCollector) GetMetric(name string) (*Metric, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	metric, exists := mc.metrics[name]
	return metric, exists
}

func (mc *MetricsCollector) GetAllMetrics() map[string]*Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	metrics := make(map[string]*Metric)
	for k, v := range mc.metrics {
		metrics[k] = v
	}
	return metrics
}

func (mc *MetricsCollector) Clear() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics = make(map[string]*Metric)
}

// HealthCheckMetrics methods
func (hcm *HealthCheckMetrics) RecordRequest(endpoint string, duration time.Duration, result types.Result) {
	labels := map[string]string{
		"endpoint": endpoint,
		"status":   result.Status.String(),
	}

	// Record basic request metrics
	hcm.RequestsTotal.WithLabels(labels).Inc()
	hcm.RequestDuration.WithLabels(labels).Observe(duration.Seconds())
	
	// Record response metrics
	if result.StatusCode > 0 {
		responseLabels := map[string]string{
			"endpoint":    endpoint,
			"status_code": fmt.Sprintf("%d", result.StatusCode),
		}
		hcm.ResponsesTotal.WithLabels(responseLabels).Inc()
		hcm.ResponseSizeBytes.WithLabels(responseLabels).Observe(float64(result.BodySize))
	}
	
	// Record error metrics
	if !result.IsHealthy() {
		errorLabels := map[string]string{
			"endpoint": endpoint,
			"type":     string(result.Status),
		}
		hcm.ErrorsTotal.WithLabels(errorLabels).Inc()
		
		// Check if it's a timeout
		if result.Status == types.StatusDown && strings.Contains(result.Error, "timeout") {
			hcm.TimeoutsTotal.WithLabels(errorLabels).Inc()
		}
	}
}

func (hcm *HealthCheckMetrics) RecordCircuitBreakerState(endpoint string, state string) {
	labels := map[string]string{
		"endpoint": endpoint,
		"state":    state,
	}
	
	var stateValue float64
	switch state {
	case "CLOSED":
		stateValue = 0
	case "OPEN":
		stateValue = 1
	case "HALF_OPEN":
		stateValue = 0.5
	}
	
	hcm.CircuitBreakerState.WithLabels(labels).Set(stateValue)
}

func (hcm *HealthCheckMetrics) RecordCircuitBreakerTrip(endpoint string) {
	labels := map[string]string{
		"endpoint": endpoint,
	}
	hcm.CircuitBreakerTrips.WithLabels(labels).Inc()
}

func (hcm *HealthCheckMetrics) RecordRateLimit(endpoint string, waitTime time.Duration) {
	labels := map[string]string{
		"endpoint": endpoint,
	}
	hcm.RateLimitedTotal.WithLabels(labels).Inc()
	hcm.RateLimitWaitTime.WithLabels(labels).Observe(waitTime.Seconds())
}

func (hcm *HealthCheckMetrics) GetSummary() map[string]interface{} {
	hcm.mu.RLock()
	defer hcm.mu.RUnlock()

	return map[string]interface{}{
		"requests_total":        hcm.RequestsTotal.Value(),
		"requests_in_flight":    hcm.RequestsInFlight.Value(),
		"responses_total":       hcm.ResponsesTotal.Value(),
		"errors_total":          hcm.ErrorsTotal.Value(),
		"timeouts_total":        hcm.TimeoutsTotal.Value(),
		"circuit_breaker_trips": hcm.CircuitBreakerTrips.Value(),
		"rate_limited_total":    hcm.RateLimitedTotal.Value(),
		"memory_usage_bytes":    hcm.MemoryUsageBytes.Value(),
		"goroutines_active":     hcm.GoroutinesActive.Value(),
		"request_duration_sum":  hcm.RequestDuration.Sum(),
		"request_duration_count": hcm.RequestDuration.Count(),
	}
}

// Global metrics instance
var (
	DefaultMetrics *HealthCheckMetrics
	once           sync.Once
)

// GetDefaultMetrics returns the default metrics instance (singleton)
func GetDefaultMetrics() *HealthCheckMetrics {
	once.Do(func() {
		DefaultMetrics = NewHealthCheckMetrics()
	})
	return DefaultMetrics
}

// Helper functions for common operations
func RecordCheckDuration(endpoint string, duration time.Duration, result types.Result) {
	GetDefaultMetrics().RecordRequest(endpoint, duration, result)
}

func RecordCircuitBreakerEvent(endpoint string, state string) {
	GetDefaultMetrics().RecordCircuitBreakerState(endpoint, state)
}

func RecordRateLimitEvent(endpoint string, waitTime time.Duration) {
	GetDefaultMetrics().RecordRateLimit(endpoint, waitTime)
}