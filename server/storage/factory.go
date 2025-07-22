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

// DefaultPostgresConfig returns a default PostgreSQL configuration
func DefaultPostgresConfig() StorageConfig {
	return StorageConfig{
		Type:     string(StorageTypePostgres),
		Host:     "localhost",
		Port:     5432,
		Database: "portfly",
		Username: "postgres",
		Password: "",
		Options: map[string]string{
			"sslmode":        "disable",
			"timezone":       "UTC",
			"log_level":      "warn",
			"max_open_conns": "25",
			"max_idle_conns": "10",
		},
	}
}

// DefaultMySQLConfig returns a default MySQL configuration
func DefaultMySQLConfig() StorageConfig {
	return StorageConfig{
		Type:     string(StorageTypeMySQL),
		Host:     "localhost",
		Port:     3306,
		Database: "portfly",
		Username: "root",
		Password: "",
		Options: map[string]string{
			"charset":        "utf8mb4",
			"parseTime":      "True",
			"loc":            "Local",
			"log_level":      "warn",
			"max_open_conns": "25",
			"max_idle_conns": "10",
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
