package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Limiter defines the interface for rate limiting functionality
type Limiter interface {
	// Allow returns true if the request is allowed under the rate limit
	Allow(key string) bool
	// Wait blocks until the request can proceed under the rate limit
	Wait(ctx context.Context, key string) error
	// SetLimit updates the rate limit for a specific key
	SetLimit(key string, limit rate.Limit, burst int)
	// RemoveLimit removes the rate limit for a specific key
	RemoveLimit(key string)
	// Stats returns current statistics for a key
	Stats(key string) (*Stats, error)
}

// Stats represents rate limiter statistics
type Stats struct {
	Key           string        `json:"key"`
	Limit         rate.Limit    `json:"limit"`
	Burst         int           `json:"burst"`
	Available     float64       `json:"available"`
	LastRequest   time.Time     `json:"last_request"`
	TotalRequests int64         `json:"total_requests"`
	Rejected      int64         `json:"rejected"`
}

// Config represents rate limiter configuration
type Config struct {
	DefaultLimit rate.Limit `yaml:"default_limit" json:"default_limit"`
	DefaultBurst int        `yaml:"default_burst" json:"default_burst"`
	Enabled      bool       `yaml:"enabled" json:"enabled"`
}

// PerEndpointLimiter implements rate limiting on a per-endpoint basis
type PerEndpointLimiter struct {
	limiters map[string]*endpointLimiter
	config   Config
	mu       sync.RWMutex
}

type endpointLimiter struct {
	limiter       *rate.Limiter
	stats         *Stats
	mu            sync.RWMutex
	totalRequests int64
	rejected      int64
}

// NewPerEndpointLimiter creates a new per-endpoint rate limiter
func NewPerEndpointLimiter(config Config) *PerEndpointLimiter {
	return &PerEndpointLimiter{
		limiters: make(map[string]*endpointLimiter),
		config:   config,
	}
}

// Allow checks if a request for the given key is allowed
func (p *PerEndpointLimiter) Allow(key string) bool {
	if !p.config.Enabled {
		return true
	}

	limiter := p.getOrCreateLimiter(key)
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	limiter.totalRequests++
	limiter.stats.TotalRequests = limiter.totalRequests
	limiter.stats.LastRequest = time.Now()
	limiter.stats.Available = limiter.limiter.Tokens()

	allowed := limiter.limiter.Allow()
	if !allowed {
		limiter.rejected++
		limiter.stats.Rejected = limiter.rejected
	}

	return allowed
}

// Wait blocks until the request can proceed under the rate limit
func (p *PerEndpointLimiter) Wait(ctx context.Context, key string) error {
	if !p.config.Enabled {
		return nil
	}

	limiter := p.getOrCreateLimiter(key)
	limiter.mu.Lock()
	limiter.totalRequests++
	limiter.stats.TotalRequests = limiter.totalRequests
	limiter.stats.LastRequest = time.Now()
	limiter.mu.Unlock()

	return limiter.limiter.Wait(ctx)
}

// SetLimit updates the rate limit for a specific key
func (p *PerEndpointLimiter) SetLimit(key string, limit rate.Limit, burst int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if existing, exists := p.limiters[key]; exists {
		existing.mu.Lock()
		existing.limiter.SetLimit(limit)
		existing.limiter.SetBurst(burst)
		existing.stats.Limit = limit
		existing.stats.Burst = burst
		existing.mu.Unlock()
	} else {
		p.limiters[key] = &endpointLimiter{
			limiter: rate.NewLimiter(limit, burst),
			stats: &Stats{
				Key:   key,
				Limit: limit,
				Burst: burst,
			},
		}
	}
}

// RemoveLimit removes the rate limit for a specific key
func (p *PerEndpointLimiter) RemoveLimit(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.limiters, key)
}

// Stats returns current statistics for a key
func (p *PerEndpointLimiter) Stats(key string) (*Stats, error) {
	p.mu.RLock()
	limiter, exists := p.limiters[key]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no rate limiter found for key: %s", key)
	}

	limiter.mu.RLock()
	defer limiter.mu.RUnlock()

	// Create a copy to avoid race conditions
	statsCopy := *limiter.stats
	statsCopy.Available = limiter.limiter.Tokens()
	
	return &statsCopy, nil
}

// GetAllStats returns statistics for all rate limiters
func (p *PerEndpointLimiter) GetAllStats() map[string]*Stats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := make(map[string]*Stats)
	for key, limiter := range p.limiters {
		limiter.mu.RLock()
		statsCopy := *limiter.stats
		statsCopy.Available = limiter.limiter.Tokens()
		stats[key] = &statsCopy
		limiter.mu.RUnlock()
	}

	return stats
}

// getOrCreateLimiter gets an existing limiter or creates a new one with default settings
func (p *PerEndpointLimiter) getOrCreateLimiter(key string) *endpointLimiter {
	p.mu.RLock()
	if limiter, exists := p.limiters[key]; exists {
		p.mu.RUnlock()
		return limiter
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists := p.limiters[key]; exists {
		return limiter
	}

	// Create new limiter with default settings
	p.limiters[key] = &endpointLimiter{
		limiter: rate.NewLimiter(p.config.DefaultLimit, p.config.DefaultBurst),
		stats: &Stats{
			Key:   key,
			Limit: p.config.DefaultLimit,
			Burst: p.config.DefaultBurst,
		},
	}

	return p.limiters[key]
}

// DefaultConfig returns a sensible default rate limiting configuration
func DefaultConfig() Config {
	return Config{
		DefaultLimit: rate.Every(time.Second), // 1 request per second by default
		DefaultBurst: 5,                       // Allow burst of 5 requests
		Enabled:      true,
	}
}

// DisabledConfig returns a configuration with rate limiting disabled
func DisabledConfig() Config {
	return Config{
		Enabled: false,
	}
}