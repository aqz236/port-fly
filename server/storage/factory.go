package storage

import (
	"fmt"
	"strings"
)

// StorageType represents the type of storage backend
type StorageType string

const (
	StorageTypeSQLite   StorageType = "sqlite"
	StorageTypePostgres StorageType = "postgres"
	StorageTypeMySQL    StorageType = "mysql"
)

// DefaultSQLiteConfig returns a default SQLite configuration
func DefaultSQLiteConfig() StorageConfig {
	return StorageConfig{
		Type:     string(StorageTypeSQLite),
		Database: "./data/portfly.db",
		Options: map[string]string{
			"log_level": "warn",
		},
	}
}

// ValidateConfig validates the storage configuration
func ValidateConfig(config StorageConfig) error {
	if config.Type == "" {
		return fmt.Errorf("storage type is required")
	}

	switch strings.ToLower(config.Type) {
	case string(StorageTypeSQLite):
		if config.Database == "" {
			return fmt.Errorf("database file path is required for SQLite")
		}
	case string(StorageTypePostgres), string(StorageTypeMySQL):
		if config.Host == "" {
			return fmt.Errorf("host is required for %s", config.Type)
		}
		if config.Database == "" {
			return fmt.Errorf("database name is required for %s", config.Type)
		}
		if config.Username == "" {
			return fmt.Errorf("username is required for %s", config.Type)
		}
	default:
		return fmt.Errorf("unsupported storage type: %s", config.Type)
	}

	return nil
}

// StorageFactory is a function type for creating storage instances
type StorageFactory func(config StorageConfig) (StorageInterface, error)

// storage factories registry
var storageFactories = make(map[string]StorageFactory)

// RegisterStorageFactory registers a storage factory for a given type
func RegisterStorageFactory(storageType string, factory StorageFactory) {
	storageFactories[strings.ToLower(storageType)] = factory
}

// NewStorage creates a new storage instance based on configuration
func NewStorage(config StorageConfig) (StorageInterface, error) {
	if err := ValidateConfig(config); err != nil {
		return nil, err
	}

	storageType := strings.ToLower(config.Type)

	// Try to find registered factory
	if factory, exists := storageFactories[storageType]; exists {
		return factory(config)
	}

	// Handle known aliases
	switch storageType {
	case "sqlite3":
		if factory, exists := storageFactories["sqlite"]; exists {
			return factory(config)
		}
	}

	return nil, fmt.Errorf("unsupported storage type: %s", config.Type)
}
