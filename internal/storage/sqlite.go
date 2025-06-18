package storage

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/renancavalcantercb/healthcheck-cli/pkg/types"
)

// SQLiteStorage implements storage using SQLite database
type SQLiteStorage struct {
	db   *sql.DB
	path string
}

// CheckResult represents a stored check result
type CheckResult struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	URL            string    `json:"url"`
	CheckType      string    `json:"check_type"`
	Status         int       `json:"status"`
	Error          string    `json:"error"`
	ResponseTimeMs int64     `json:"response_time_ms"`
	StatusCode     int       `json:"status_code"`
	BodySize       int64     `json:"body_size"`
	Timestamp      time.Time `json:"timestamp"`
	CreatedAt      time.Time `json:"created_at"`
}

// ServiceStats represents aggregated statistics for a service
type ServiceStats struct {
	Name              string        `json:"name"`
	URL               string        `json:"url"`
	CheckType         string        `json:"check_type"`
	TotalChecks       int64         `json:"total_checks"`
	SuccessfulChecks  int64         `json:"successful_checks"`
	FailedChecks      int64         `json:"failed_checks"`
	AvgResponseTimeMs float64       `json:"avg_response_time_ms"`
	MinResponseTimeMs int64         `json:"min_response_time_ms"`
	MaxResponseTimeMs int64         `json:"max_response_time_ms"`
	UptimePercent     float64       `json:"uptime_percent"`
	LastCheck         time.Time     `json:"last_check"`
	LastSuccess       time.Time     `json:"last_success"`
	LastFailure       time.Time     `json:"last_failure"`
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_timeout=5000&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &SQLiteStorage{
		db:   db,
		path: dbPath,
	}

	if err := storage.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	if err := storage.createIndexes(); err != nil {
		log.Printf("Warning: failed to create indexes: %v", err)
	}

	return storage, nil
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// createTables creates the necessary database tables
func (s *SQLiteStorage) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS check_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		check_type TEXT NOT NULL,
		status INTEGER NOT NULL,
		error TEXT,
		response_time_ms INTEGER NOT NULL,
		status_code INTEGER,
		body_size INTEGER,
		timestamp DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS service_metadata (
		name TEXT PRIMARY KEY,
		url TEXT NOT NULL,
		check_type TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := s.db.Exec(query)
	return err
}

// createIndexes creates database indexes for better performance
func (s *SQLiteStorage) createIndexes() error {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_check_results_name ON check_results(name)",
		"CREATE INDEX IF NOT EXISTS idx_check_results_timestamp ON check_results(timestamp)",
		"CREATE INDEX IF NOT EXISTS idx_check_results_name_timestamp ON check_results(name, timestamp)",
		"CREATE INDEX IF NOT EXISTS idx_check_results_status ON check_results(status)",
		"CREATE INDEX IF NOT EXISTS idx_check_results_created_at ON check_results(created_at)",
	}

	for _, index := range indexes {
		if _, err := s.db.Exec(index); err != nil {
			return err
		}
	}

	return nil
}

// SaveResult saves a check result to the database
func (s *SQLiteStorage) SaveResult(result types.Result) error {
	query := `
	INSERT INTO check_results (
		name, url, check_type, status, error, response_time_ms, 
		status_code, body_size, timestamp
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	checkType := "http"
	if result.URL != "" && !contains(result.URL, "http") {
		checkType = "tcp"
	}

	_, err := s.db.Exec(query,
		result.Name,
		result.URL,
		checkType,
		int(result.Status),
		result.Error,
		result.ResponseTime.Milliseconds(),
		result.StatusCode,
		result.BodySize,
		result.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to save result: %w", err)
	}

	// Update service metadata
	s.updateServiceMetadata(result.Name, result.URL, checkType)

	return nil
}

// updateServiceMetadata updates or inserts service metadata
func (s *SQLiteStorage) updateServiceMetadata(name, url, checkType string) error {
	query := `
	INSERT INTO service_metadata (name, url, check_type, updated_at)
	VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(name) DO UPDATE SET
		url = excluded.url,
		check_type = excluded.check_type,
		updated_at = excluded.updated_at`

	_, err := s.db.Exec(query, name, url, checkType)
	return err
}

// GetServiceStats returns aggregated statistics for a service
func (s *SQLiteStorage) GetServiceStats(name string, since time.Time) (*ServiceStats, error) {
	query := `
	SELECT 
		name,
		url,
		check_type,
		COUNT(*) as total_checks,
		SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END) as successful_checks,
		SUM(CASE WHEN status != 0 THEN 1 ELSE 0 END) as failed_checks,
		AVG(response_time_ms) as avg_response_time_ms,
		MIN(response_time_ms) as min_response_time_ms,
		MAX(response_time_ms) as max_response_time_ms,
		MAX(timestamp) as last_check,
		MAX(CASE WHEN status = 0 THEN timestamp END) as last_success,
		MAX(CASE WHEN status != 0 THEN timestamp END) as last_failure
	FROM check_results 
	WHERE name = ? AND timestamp >= ?
	GROUP BY name, url, check_type`

	var stats ServiceStats
	var lastSuccess, lastFailure sql.NullTime

	err := s.db.QueryRow(query, name, since).Scan(
		&stats.Name,
		&stats.URL,
		&stats.CheckType,
		&stats.TotalChecks,
		&stats.SuccessfulChecks,
		&stats.FailedChecks,
		&stats.AvgResponseTimeMs,
		&stats.MinResponseTimeMs,
		&stats.MaxResponseTimeMs,
		&stats.LastCheck,
		&lastSuccess,
		&lastFailure,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no data found for service %s since %v", name, since)
		}
		return nil, fmt.Errorf("failed to get service stats: %w", err)
	}

	// Calculate uptime percentage
	if stats.TotalChecks > 0 {
		stats.UptimePercent = (float64(stats.SuccessfulChecks) / float64(stats.TotalChecks)) * 100
	}

	// Handle nullable timestamps
	if lastSuccess.Valid {
		stats.LastSuccess = lastSuccess.Time
	}
	if lastFailure.Valid {
		stats.LastFailure = lastFailure.Time
	}

	return &stats, nil
}

// GetAllServiceStats returns stats for all services
func (s *SQLiteStorage) GetAllServiceStats(since time.Time) ([]ServiceStats, error) {
	query := `
	SELECT DISTINCT name FROM service_metadata ORDER BY name`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get service names: %w", err)
	}
	defer rows.Close()

	var allStats []ServiceStats
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}

		stats, err := s.GetServiceStats(name, since)
		if err != nil {
			log.Printf("Warning: failed to get stats for service %s: %v", name, err)
			continue
		}

		allStats = append(allStats, *stats)
	}

	return allStats, nil
}

// GetRecentResults returns recent results for all services
func (s *SQLiteStorage) GetRecentResults(limit int) ([]CheckResult, error) {
	query := `
	SELECT id, name, url, check_type, status, error, response_time_ms, 
		   status_code, body_size, timestamp, created_at
	FROM check_results 
	ORDER BY timestamp DESC 
	LIMIT ?`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent results: %w", err)
	}
	defer rows.Close()

	var results []CheckResult
	for rows.Next() {
		var result CheckResult
		var errorStr sql.NullString
		var statusCode, bodySize sql.NullInt64

		err := rows.Scan(
			&result.ID,
			&result.Name,
			&result.URL,
			&result.CheckType,
			&result.Status,
			&errorStr,
			&result.ResponseTimeMs,
			&statusCode,
			&bodySize,
			&result.Timestamp,
			&result.CreatedAt,
		)

		if err != nil {
			log.Printf("Warning: failed to scan result: %v", err)
			continue
		}

		// Handle nullable fields
		if errorStr.Valid {
			result.Error = errorStr.String
		}
		if statusCode.Valid {
			result.StatusCode = int(statusCode.Int64)
		}
		if bodySize.Valid {
			result.BodySize = bodySize.Int64
		}

		results = append(results, result)
	}

	return results, nil
}

// GetServiceHistory returns historical data for a specific service
func (s *SQLiteStorage) GetServiceHistory(name string, since time.Time, limit int) ([]CheckResult, error) {
	query := `
	SELECT id, name, url, check_type, status, error, response_time_ms, 
		   status_code, body_size, timestamp, created_at
	FROM check_results 
	WHERE name = ? AND timestamp >= ?
	ORDER BY timestamp DESC 
	LIMIT ?`

	rows, err := s.db.Query(query, name, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get service history: %w", err)
	}
	defer rows.Close()

	var results []CheckResult
	for rows.Next() {
		var result CheckResult
		var errorStr sql.NullString
		var statusCode, bodySize sql.NullInt64

		err := rows.Scan(
			&result.ID,
			&result.Name,
			&result.URL,
			&result.CheckType,
			&result.Status,
			&errorStr,
			&result.ResponseTimeMs,
			&statusCode,
			&bodySize,
			&result.Timestamp,
			&result.CreatedAt,
		)

		if err != nil {
			continue
		}

		if errorStr.Valid {
			result.Error = errorStr.String
		}
		if statusCode.Valid {
			result.StatusCode = int(statusCode.Int64)
		}
		if bodySize.Valid {
			result.BodySize = bodySize.Int64
		}

		results = append(results, result)
	}

	return results, nil
}

// CleanupOldData removes data older than the specified duration
func (s *SQLiteStorage) CleanupOldData(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	
	query := "DELETE FROM check_results WHERE created_at < ?"
	result, err := s.db.Exec(query, cutoff)
	if err != nil {
		return fmt.Errorf("failed to cleanup old data: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("Cleaned up %d old check results (older than %v)", rowsAffected, olderThan)
	}

	// Vacuum to reclaim space
	if _, err := s.db.Exec("VACUUM"); err != nil {
		log.Printf("Warning: failed to vacuum database: %v", err)
	}

	return nil
}

// GetDatabaseInfo returns information about the database
func (s *SQLiteStorage) GetDatabaseInfo() (map[string]interface{}, error) {
	info := make(map[string]interface{})
	
	// Total records
	var totalRecords int64
	err := s.db.QueryRow("SELECT COUNT(*) FROM check_results").Scan(&totalRecords)
	if err != nil {
		return nil, err
	}
	info["total_records"] = totalRecords

	// Database size
	var pageCount, pageSize int64
	err = s.db.QueryRow("PRAGMA page_count").Scan(&pageCount)
	if err == nil {
		err = s.db.QueryRow("PRAGMA page_size").Scan(&pageSize)
		if err == nil {
			info["database_size_bytes"] = pageCount * pageSize
		}
	}

	// Oldest and newest records
	var oldest, newest time.Time
	err = s.db.QueryRow("SELECT MIN(created_at), MAX(created_at) FROM check_results").Scan(&oldest, &newest)
	if err == nil {
		info["oldest_record"] = oldest
		info["newest_record"] = newest
	}

	// Number of services
	var serviceCount int64
	err = s.db.QueryRow("SELECT COUNT(*) FROM service_metadata").Scan(&serviceCount)
	if err == nil {
		info["total_services"] = serviceCount
	}

	info["database_path"] = s.path

	return info, nil
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 findInString(s, substr))))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}