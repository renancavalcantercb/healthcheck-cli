package storage

import (
	"fmt"
	"log"
	"os"

	"github.com/renancavalcantercb/healthcheck-cli/pkg/interfaces"
)

// StorageType represents the type of storage implementation
type StorageType string

const (
	StorageTypeSQLite StorageType = "sqlite"
	StorageTypeMemory StorageType = "memory"
)

// StorageConfig contains configuration for storage initialization
type StorageConfig struct {
	Type           StorageType
	Path           string
	FallbackToMemory bool
}

// NewStorage creates a new storage instance based on the provided configuration
func NewStorage(config StorageConfig) (interfaces.Storage, error) {
	switch config.Type {
	case StorageTypeSQLite:
		return newSQLiteWithFallback(config.Path, config.FallbackToMemory)
	case StorageTypeMemory:
		return NewMemoryStorage(config.Path)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", config.Type)
	}
}

// NewStorageWithDefaults creates a storage instance with intelligent defaults
func NewStorageWithDefaults(dbPath string) (interfaces.Storage, error) {
	config := StorageConfig{
		Type:           StorageTypeSQLite,
		Path:           dbPath,
		FallbackToMemory: true,
	}
	
	return NewStorage(config)
}

// newSQLiteWithFallback tries to create SQLite storage, falls back to memory storage if it fails
func newSQLiteWithFallback(dbPath string, fallback bool) (interfaces.Storage, error) {
	// Try SQLite first
	storage, err := NewSQLiteStorage(dbPath)
	if err == nil {
		log.Printf("✅ SQLite storage initialized successfully: %s", dbPath)
		return storage, nil
	}

	// Check if the error is related to CGO
	if isLikelyCGOError(err) {
		log.Printf("⚠️  SQLite requires CGO but binary was compiled with CGO_ENABLED=0")
		log.Printf("💡 This is common in cross-compiled binaries or some build environments")
		
		if fallback {
			log.Printf("🔄 Falling back to memory storage with file persistence...")
			
			// Use memory storage with file persistence
			memStorage, memErr := NewMemoryStorage(getMemoryStoragePath(dbPath))
			if memErr == nil {
				log.Printf("✅ Memory storage with persistence initialized: %s", getMemoryStoragePath(dbPath))
				return memStorage, nil
			}
			
			log.Printf("⚠️  Failed to initialize memory storage with persistence: %v", memErr)
			log.Printf("🔄 Falling back to pure memory storage (no persistence)...")
			
			// Final fallback: pure memory storage
			pureMemStorage, pureMemErr := NewMemoryStorage("")
			if pureMemErr == nil {
				log.Printf("✅ Pure memory storage initialized (data will not persist between restarts)")
				return pureMemStorage, nil
			}
			
			return nil, fmt.Errorf("all storage options failed - SQLite: %v, Memory w/ persistence: %v, Pure memory: %v", 
				err, memErr, pureMemErr)
		}
	}

	return nil, fmt.Errorf("SQLite storage failed and fallback disabled: %w", err)
}

// isLikelyCGOError checks if the error is likely related to CGO being disabled
func isLikelyCGOError(err error) bool {
	if err == nil {
		return false
	}
	
	errorStr := err.Error()
	cgoIndicators := []string{
		"CGO_ENABLED=0",
		"requires cgo",
		"this is a stub",
		"binary was compiled with",
		"go-sqlite3 requires cgo",
	}
	
	for _, indicator := range cgoIndicators {
		if contains(errorStr, indicator) {
			return true
		}
	}
	
	return false
}

// getMemoryStoragePath converts SQLite path to memory storage path
func getMemoryStoragePath(sqlitePath string) string {
	if sqlitePath == "" {
		return "healthcheck_data.json"
	}
	
	// Replace .db extension with .json
	if len(sqlitePath) > 3 && sqlitePath[len(sqlitePath)-3:] == ".db" {
		return sqlitePath[:len(sqlitePath)-3] + ".json"
	}
	
	return sqlitePath + ".json"
}

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findInString(s, substr))
}

// Helper function to find substring in string
func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// MigrateFromSQLiteToMemory migrates data from SQLite to Memory storage
func MigrateFromSQLiteToMemory(sqlitePath, memoryPath string) error {
	// Try to open SQLite database
	sqliteStorage, err := NewSQLiteStorage(sqlitePath)
	if err != nil {
		return fmt.Errorf("failed to open SQLite database for migration: %w", err)
	}
	defer sqliteStorage.Close()

	// Create memory storage
	memoryStorage, err := NewMemoryStorage(memoryPath)
	if err != nil {
		return fmt.Errorf("failed to create memory storage for migration: %w", err)
	}
	defer memoryStorage.Close()

	// Get all data from SQLite
	info, err := sqliteStorage.GetDatabaseInfo()
	if err != nil {
		return fmt.Errorf("failed to get database info for migration: %w", err)
	}

	totalRecords, ok := info["total_records"].(int64)
	if !ok || totalRecords == 0 {
		log.Printf("No records to migrate from SQLite")
		return nil
	}

	log.Printf("🔄 Migrating %d records from SQLite to Memory storage...", totalRecords)

	// Migration would require additional methods to get all raw results
	// For now, we'll just log that migration is needed but not implemented
	log.Printf("⚠️  Automatic migration not yet implemented")
	log.Printf("💡 You can start fresh with memory storage, or rebuild with CGO_ENABLED=1")

	return nil
}

// GetStorageInfo returns information about the current storage configuration
func GetStorageInfo() map[string]interface{} {
	info := make(map[string]interface{})
	
	// Check if CGO is available
	_, err := NewSQLiteStorage(":memory:")
	if err == nil {
		info["sqlite_available"] = true
	} else {
		info["sqlite_available"] = false
		info["sqlite_error"] = err.Error()
	}
	
	// Memory storage is always available
	info["memory_available"] = true
	
	// Check if file system is writable
	testFile := "healthcheck_test.tmp"
	if file, err := os.Create(testFile); err == nil {
		file.Close()
		os.Remove(testFile)
		info["file_persistence_available"] = true
	} else {
		info["file_persistence_available"] = false
		info["file_persistence_error"] = err.Error()
	}
	
	return info
}