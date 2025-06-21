package storage

import (
	"os"
	"testing"
	"time"

	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_Basic(t *testing.T) {
	// Create memory storage without persistence
	storage, err := NewMemoryStorage("")
	require.NoError(t, err)
	defer storage.Close()

	// Test saving a result
	result := types.Result{
		Name:         "Test Service",
		URL:          "https://example.com",
		Status:       types.StatusUp,
		ResponseTime: 100 * time.Millisecond,
		StatusCode:   200,
		Timestamp:    time.Now(),
	}

	err = storage.SaveResult(result)
	assert.NoError(t, err)

	// Test getting service stats
	stats, err := storage.GetServiceStats("Test Service", time.Now().Add(-1*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, "Test Service", stats.Name)
	assert.Equal(t, "https://example.com", stats.URL)
	assert.Equal(t, "http", stats.CheckType)
	assert.Equal(t, int64(1), stats.TotalChecks)
	assert.Equal(t, int64(1), stats.SuccessfulChecks)
	assert.Equal(t, int64(0), stats.FailedChecks)
	assert.Equal(t, 100.0, stats.UptimePercent)
}

func TestMemoryStorage_Persistence(t *testing.T) {
	tempFile := "/tmp/test_memory_storage.json"
	defer os.Remove(tempFile)

	// Create storage with persistence
	storage1, err := NewMemoryStorage(tempFile)
	require.NoError(t, err)

	// Save some data
	result := types.Result{
		Name:         "Persistent Service",
		URL:          "https://example.com",
		Status:       types.StatusUp,
		ResponseTime: 150 * time.Millisecond,
		StatusCode:   200,
		Timestamp:    time.Now(),
	}

	err = storage1.SaveResult(result)
	require.NoError(t, err)

	// Close storage (should save data)
	err = storage1.Close()
	require.NoError(t, err)

	// Create new storage instance and load data
	storage2, err := NewMemoryStorage(tempFile)
	require.NoError(t, err)
	defer storage2.Close()

	// Verify data was loaded
	stats, err := storage2.GetServiceStats("Persistent Service", time.Now().Add(-1*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, "Persistent Service", stats.Name)
	assert.Equal(t, int64(1), stats.TotalChecks)
}

func TestMemoryStorage_MultipleServices(t *testing.T) {
	storage, err := NewMemoryStorage("")
	require.NoError(t, err)
	defer storage.Close()

	now := time.Now()

	// Add results for multiple services
	services := []struct {
		name   string
		url    string
		status types.Status
	}{
		{"Service A", "https://a.example.com", types.StatusUp},
		{"Service B", "https://b.example.com", types.StatusDown},
		{"Service C", "tcp://c.example.com:8080", types.StatusSlow},
	}

	for _, svc := range services {
		result := types.Result{
			Name:         svc.name,
			URL:          svc.url,
			Status:       svc.status,
			ResponseTime: 100 * time.Millisecond,
			StatusCode:   200,
			Timestamp:    now,
		}
		err = storage.SaveResult(result)
		require.NoError(t, err)
	}

	// Test GetAllServiceStats
	allStats, err := storage.GetAllServiceStats(now.Add(-1 * time.Hour))
	require.NoError(t, err)
	assert.Len(t, allStats, 3)

	// Verify services are sorted by name
	assert.Equal(t, "Service A", allStats[0].Name)
	assert.Equal(t, "Service B", allStats[1].Name)
	assert.Equal(t, "Service C", allStats[2].Name)

	// Verify check types are detected correctly
	assert.Equal(t, "http", allStats[0].CheckType)
	assert.Equal(t, "http", allStats[1].CheckType)
	assert.Equal(t, "tcp", allStats[2].CheckType)
}

func TestMemoryStorage_UptimeCalculation(t *testing.T) {
	storage, err := NewMemoryStorage("")
	require.NoError(t, err)
	defer storage.Close()

	serviceName := "Uptime Test Service"
	baseTime := time.Now()

	// Add 7 successful and 3 failed checks
	for i := 0; i < 10; i++ {
		status := types.StatusUp
		if i < 3 {
			status = types.StatusDown
		}

		result := types.Result{
			Name:         serviceName,
			URL:          "https://example.com",
			Status:       status,
			ResponseTime: time.Duration(50+i*10) * time.Millisecond,
			StatusCode:   200,
			Timestamp:    baseTime.Add(time.Duration(i) * time.Minute),
		}
		
		err = storage.SaveResult(result)
		require.NoError(t, err)
	}

	// Get stats
	stats, err := storage.GetServiceStats(serviceName, baseTime.Add(-1*time.Hour))
	require.NoError(t, err)

	assert.Equal(t, int64(10), stats.TotalChecks)
	assert.Equal(t, int64(7), stats.SuccessfulChecks)
	assert.Equal(t, int64(3), stats.FailedChecks)
	assert.Equal(t, 70.0, stats.UptimePercent)
	assert.Equal(t, int64(50), stats.MinResponseTimeMs)
	assert.Equal(t, int64(140), stats.MaxResponseTimeMs)
	assert.Equal(t, 95.0, stats.AvgResponseTimeMs) // (50+60+...+140)/10 = 95
}

func TestMemoryStorage_History(t *testing.T) {
	storage, err := NewMemoryStorage("")
	require.NoError(t, err)
	defer storage.Close()

	serviceName := "History Test Service"
	baseTime := time.Now()

	// Add multiple results
	for i := 0; i < 5; i++ {
		result := types.Result{
			Name:         serviceName,
			URL:          "https://example.com",
			Status:       types.StatusUp,
			ResponseTime: time.Duration(100+i*50) * time.Millisecond,
			Timestamp:    baseTime.Add(time.Duration(i) * time.Minute),
		}
		
		err = storage.SaveResult(result)
		require.NoError(t, err)
	}

	// Get history
	history, err := storage.GetServiceHistory(serviceName, baseTime.Add(-1*time.Hour), 3)
	require.NoError(t, err)

	// Should return 3 most recent results (newest first)
	assert.Len(t, history, 3)
	assert.True(t, history[0].Timestamp.After(history[1].Timestamp))
	assert.True(t, history[1].Timestamp.After(history[2].Timestamp))
}

func TestMemoryStorage_Cleanup(t *testing.T) {
	storage, err := NewMemoryStorage("")
	require.NoError(t, err)
	defer storage.Close()

	now := time.Now()
	serviceName := "Cleanup Test Service"

	// Add old results
	for i := 0; i < 3; i++ {
		result := types.Result{
			Name:         serviceName,
			URL:          "https://example.com",
			Status:       types.StatusUp,
			ResponseTime: 100 * time.Millisecond,
			Timestamp:    now.Add(-2 * time.Hour),
		}
		
		// Manually set created_at to simulate old data
		storage.SaveResult(result)
		
		// Modify the last added result to have old created_at
		storage.mu.Lock()
		if len(storage.results) > 0 {
			storage.results[len(storage.results)-1].CreatedAt = now.Add(-2 * time.Hour)
		}
		storage.mu.Unlock()
	}

	// Add recent results
	for i := 0; i < 2; i++ {
		result := types.Result{
			Name:         serviceName,
			URL:          "https://example.com",
			Status:       types.StatusUp,
			ResponseTime: 100 * time.Millisecond,
			Timestamp:    now,
		}
		storage.SaveResult(result)
	}

	// Check initial count
	info, err := storage.GetDatabaseInfo()
	require.NoError(t, err)
	assert.Equal(t, int64(5), info["total_records"])

	// Cleanup data older than 1 hour
	err = storage.CleanupOldData(1 * time.Hour)
	require.NoError(t, err)

	// Check remaining count
	info, err = storage.GetDatabaseInfo()
	require.NoError(t, err)
	assert.Equal(t, int64(2), info["total_records"])
}

func TestMemoryStorage_DatabaseInfo(t *testing.T) {
	storage, err := NewMemoryStorage("")
	require.NoError(t, err)
	defer storage.Close()

	// Add some data
	result := types.Result{
		Name:         "Info Test Service",
		URL:          "https://example.com",
		Status:       types.StatusUp,
		ResponseTime: 100 * time.Millisecond,
		Timestamp:    time.Now(),
	}
	storage.SaveResult(result)

	// Get database info
	info, err := storage.GetDatabaseInfo()
	require.NoError(t, err)

	assert.Equal(t, "memory", info["storage_type"])
	assert.Equal(t, int64(1), info["total_records"])
	assert.Equal(t, int64(1), info["total_services"])
	assert.Equal(t, int64(10000), info["max_records"])
	assert.False(t, info["auto_save_enabled"].(bool))
	assert.Contains(t, info, "memory_usage_bytes")
	assert.Contains(t, info, "oldest_record")
	assert.Contains(t, info, "newest_record")
}