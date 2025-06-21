package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
)

// MemoryStorage implements storage using in-memory data structures with optional file persistence
type MemoryStorage struct {
	mu           sync.RWMutex
	results      []types.CheckResult
	services     map[string]*ServiceInfo
	path         string
	maxResults   int
	autoSave     bool
	saveInterval time.Duration
	stopSave     chan struct{}
}

// ServiceInfo tracks metadata about a service
type ServiceInfo struct {
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	CheckType string    `json:"check_type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MemoryStorageData represents the persistent data structure
type MemoryStorageData struct {
	Results  []types.CheckResult    `json:"results"`
	Services map[string]*ServiceInfo `json:"services"`
	Version  string                 `json:"version"`
	SavedAt  time.Time              `json:"saved_at"`
}

// NewMemoryStorage creates a new in-memory storage with optional file persistence
func NewMemoryStorage(filePath string) (*MemoryStorage, error) {
	storage := &MemoryStorage{
		results:      make([]types.CheckResult, 0),
		services:     make(map[string]*ServiceInfo),
		path:         filePath,
		maxResults:   10000, // Keep last 10k results
		autoSave:     filePath != "",
		saveInterval: 30 * time.Second,
		stopSave:     make(chan struct{}),
	}

	// Load existing data if file exists
	if filePath != "" {
		if err := storage.loadFromFile(); err != nil {
			// Don't fail initialization if load fails, just log warning
			fmt.Printf("⚠️  Warning: Failed to load existing data from %s: %v\n", filePath, err)
		}

		// Start auto-save goroutine if enabled
		if storage.autoSave {
			go storage.autoSaveWorker()
		}
	}

	return storage, nil
}

// autoSaveWorker periodically saves data to file
func (m *MemoryStorage) autoSaveWorker() {
	ticker := time.NewTicker(m.saveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := m.saveToFile(); err != nil {
				fmt.Printf("⚠️  Warning: Auto-save failed: %v\n", err)
			}
		case <-m.stopSave:
			return
		}
	}
}

// loadFromFile loads data from the persistence file
func (m *MemoryStorage) loadFromFile() error {
	if m.path == "" {
		return nil
	}

	data, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, that's OK
		}
		return fmt.Errorf("failed to read storage file: %w", err)
	}

	var storageData MemoryStorageData
	if err := json.Unmarshal(data, &storageData); err != nil {
		return fmt.Errorf("failed to unmarshal storage data: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.results = storageData.Results
	m.services = storageData.Services
	if m.services == nil {
		m.services = make(map[string]*ServiceInfo)
	}

	fmt.Printf("📁 Loaded %d results and %d services from %s\n", 
		len(m.results), len(m.services), m.path)

	return nil
}

// saveToFile saves current data to the persistence file
func (m *MemoryStorage) saveToFile() error {
	if m.path == "" {
		return nil
	}

	m.mu.RLock()
	storageData := MemoryStorageData{
		Results:  m.results,
		Services: m.services,
		Version:  "1.0",
		SavedAt:  time.Now(),
	}
	m.mu.RUnlock()

	data, err := json.MarshalIndent(storageData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal storage data: %w", err)
	}

	// Write to temporary file first, then rename (atomic operation)
	tempPath := m.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	if err := os.Rename(tempPath, m.path); err != nil {
		os.Remove(tempPath) // Clean up on failure
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// Close closes the storage and saves any pending data
func (m *MemoryStorage) Close() error {
	if m.autoSave {
		// Stop auto-save worker
		close(m.stopSave)

		// Final save
		if err := m.saveToFile(); err != nil {
			return fmt.Errorf("failed to save data on close: %w", err)
		}
	}

	return nil
}

// SaveResult saves a check result
func (m *MemoryStorage) SaveResult(result types.Result) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Convert types.Result to types.CheckResult
	checkResult := types.CheckResult{
		ID:             int64(len(m.results) + 1),
		Name:           result.Name,
		URL:            result.URL,
		CheckType:      m.determineCheckType(result.URL),
		Status:         int(result.Status),
		Error:          result.Error,
		ResponseTimeMs: result.ResponseTime.Milliseconds(),
		StatusCode:     result.StatusCode,
		BodySize:       result.BodySize,
		Timestamp:      result.Timestamp,
		CreatedAt:      time.Now(),
	}

	// Add to results
	m.results = append(m.results, checkResult)

	// Maintain size limit
	if len(m.results) > m.maxResults {
		// Remove oldest 10% when limit is reached
		removeCount := m.maxResults / 10
		m.results = m.results[removeCount:]
		
		// Update IDs
		for i := range m.results {
			m.results[i].ID = int64(i + 1)
		}
	}

	// Update service metadata
	m.updateServiceInfo(result.Name, result.URL, checkResult.CheckType)

	return nil
}

// determineCheckType determines the check type based on URL
func (m *MemoryStorage) determineCheckType(url string) string {
	if url == "" {
		return "unknown"
	}
	
	if len(url) >= 4 && (url[:4] == "http" || (len(url) >= 5 && url[:5] == "https")) {
		return "http"
	}
	
	return "tcp"
}

// updateServiceInfo updates service metadata
func (m *MemoryStorage) updateServiceInfo(name, url, checkType string) {
	now := time.Now()
	
	if service, exists := m.services[name]; exists {
		service.URL = url
		service.CheckType = checkType
		service.UpdatedAt = now
	} else {
		m.services[name] = &ServiceInfo{
			Name:      name,
			URL:       url,
			CheckType: checkType,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}
}

// GetServiceStats returns aggregated statistics for a service
func (m *MemoryStorage) GetServiceStats(name string, since time.Time) (*types.ServiceStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	serviceInfo, exists := m.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	stats := &types.ServiceStats{
		Name:      name,
		URL:       serviceInfo.URL,
		CheckType: serviceInfo.CheckType,
	}

	var totalChecks, successfulChecks, failedChecks int64
	var responseTimes []int64
	var lastCheck, lastSuccess, lastFailure time.Time

	for _, result := range m.results {
		if result.Name != name || result.Timestamp.Before(since) {
			continue
		}

		totalChecks++
		responseTimes = append(responseTimes, result.ResponseTimeMs)

		if result.Timestamp.After(lastCheck) {
			lastCheck = result.Timestamp
		}

		if result.Status == int(types.StatusUp) {
			successfulChecks++
			if result.Timestamp.After(lastSuccess) {
				lastSuccess = result.Timestamp
			}
		} else {
			failedChecks++
			if result.Timestamp.After(lastFailure) {
				lastFailure = result.Timestamp
			}
		}
	}

	if totalChecks == 0 {
		return nil, fmt.Errorf("no data found for service %s since %v", name, since)
	}

	stats.TotalChecks = totalChecks
	stats.SuccessfulChecks = successfulChecks
	stats.FailedChecks = failedChecks
	stats.LastCheck = lastCheck
	stats.LastSuccess = lastSuccess
	stats.LastFailure = lastFailure

	// Calculate uptime percentage
	if totalChecks > 0 {
		stats.UptimePercent = (float64(successfulChecks) / float64(totalChecks)) * 100
	}

	// Calculate response time statistics
	if len(responseTimes) > 0 {
		sort.Slice(responseTimes, func(i, j int) bool { return responseTimes[i] < responseTimes[j] })
		
		stats.MinResponseTimeMs = responseTimes[0]
		stats.MaxResponseTimeMs = responseTimes[len(responseTimes)-1]
		
		var sum int64
		for _, rt := range responseTimes {
			sum += rt
		}
		stats.AvgResponseTimeMs = float64(sum) / float64(len(responseTimes))
	}

	return stats, nil
}

// GetAllServiceStats returns stats for all services
func (m *MemoryStorage) GetAllServiceStats(since time.Time) ([]types.ServiceStats, error) {
	m.mu.RLock()
	services := make([]string, 0, len(m.services))
	for name := range m.services {
		services = append(services, name)
	}
	m.mu.RUnlock()

	var allStats []types.ServiceStats
	for _, name := range services {
		stats, err := m.GetServiceStats(name, since)
		if err != nil {
			continue // Skip services with no data
		}
		allStats = append(allStats, *stats)
	}

	// Sort by name
	sort.Slice(allStats, func(i, j int) bool {
		return allStats[i].Name < allStats[j].Name
	})

	return allStats, nil
}

// GetServiceHistory returns historical data for a specific service
func (m *MemoryStorage) GetServiceHistory(name string, since time.Time, limit int) ([]types.CheckResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []types.CheckResult
	for _, result := range m.results {
		if result.Name == name && result.Timestamp.After(since) {
			results = append(results, result)
		}
	}

	// Sort by timestamp descending (newest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// CleanupOldData removes data older than the specified duration
func (m *MemoryStorage) CleanupOldData(olderThan time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	
	// Count items to remove
	removeCount := 0
	for _, result := range m.results {
		if result.CreatedAt.Before(cutoff) {
			removeCount++
		} else {
			break // Results are sorted by creation time
		}
	}

	if removeCount > 0 {
		m.results = m.results[removeCount:]
		fmt.Printf("🧹 Cleaned up %d old check results (older than %v)\n", removeCount, olderThan)
	}

	return nil
}

// GetDatabaseInfo returns information about the storage
func (m *MemoryStorage) GetDatabaseInfo() (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info := make(map[string]interface{})
	
	info["storage_type"] = "memory"
	info["database_path"] = m.path
	info["total_records"] = int64(len(m.results))
	info["total_services"] = int64(len(m.services))
	info["max_records"] = int64(m.maxResults)
	info["auto_save_enabled"] = m.autoSave
	info["save_interval"] = m.saveInterval.String()

	if len(m.results) > 0 {
		// Sort results by created_at to find oldest and newest
		oldest := m.results[0].CreatedAt
		newest := m.results[0].CreatedAt
		
		for _, result := range m.results {
			if result.CreatedAt.Before(oldest) {
				oldest = result.CreatedAt
			}
			if result.CreatedAt.After(newest) {
				newest = result.CreatedAt
			}
		}
		
		info["oldest_record"] = oldest
		info["newest_record"] = newest
	}

	// Calculate memory usage estimate
	dataSize := len(m.results) * 200 // Rough estimate per result
	for name, service := range m.services {
		dataSize += len(name) + len(service.URL) + len(service.CheckType) + 100
	}
	info["memory_usage_bytes"] = int64(dataSize)

	return info, nil
}