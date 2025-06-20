package services

import (
	"fmt"
	"time"

	"github.com/renancavalcantercb/healthcheck-cli/pkg/interfaces"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
)

// StatsService implements statistics and analytics functionality
type StatsService struct {
	storage interfaces.Storage
}

// NewStatsService creates a new stats service
func NewStatsService(storage interfaces.Storage) *StatsService {
	return &StatsService{
		storage: storage,
	}
}

// GetServiceStats retrieves statistics for a specific service
func (s *StatsService) GetServiceStats(serviceName string, since time.Time) (*types.ServiceStats, error) {
	if s.storage == nil {
		return nil, fmt.Errorf("storage not available - stats require data persistence")
	}
	
	return s.storage.GetServiceStats(serviceName, since)
}

// GetAllStats retrieves statistics for all services
func (s *StatsService) GetAllStats(since time.Time) ([]types.ServiceStats, error) {
	if s.storage == nil {
		return nil, fmt.Errorf("storage not available - stats require data persistence")
	}
	
	return s.storage.GetAllServiceStats(since)
}

// GetHistory retrieves historical data for a service
func (s *StatsService) GetHistory(serviceName string, since time.Time, limit int) ([]types.CheckResult, error) {
	if s.storage == nil {
		return nil, fmt.Errorf("storage not available - history requires data persistence")
	}
	
	return s.storage.GetServiceHistory(serviceName, since, limit)
}

// GetDatabaseInfo retrieves information about the database
func (s *StatsService) GetDatabaseInfo() (map[string]interface{}, error) {
	if s.storage == nil {
		return nil, fmt.Errorf("storage not available")
	}
	
	return s.storage.GetDatabaseInfo()
}

// CleanupOldData removes old data from storage
func (s *StatsService) CleanupOldData(maxAge time.Duration) error {
	if s.storage == nil {
		return fmt.Errorf("storage not available")
	}
	
	return s.storage.CleanupOldData(maxAge)
}