package sqlite

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/aqz236/port-fly/core/models"
	"github.com/aqz236/port-fly/server/storage"
)

// SQLiteStorage implements StorageInterface using SQLite
type SQLiteStorage struct {
	db     *gorm.DB
	config storage.StorageConfig
}

func init() {
	// Register SQLite storage factory
	storage.RegisterStorageFactory("sqlite", func(config storage.StorageConfig) (storage.StorageInterface, error) {
		return NewSQLiteStorage(config)
	})
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(config storage.StorageConfig) (*SQLiteStorage, error) {
	storage := &SQLiteStorage{
		config: config,
	}

	if err := storage.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize SQLite storage: %w", err)
	}

	return storage, nil
}

// Initialize initializes the SQLite database connection
func (s *SQLiteStorage) Initialize() error {
	dbPath := s.config.Database
	if dbPath == "" {
		dbPath = "./data/portfly.db"
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Configure GORM logger
	gormLogger := logger.Default
	if strings.ToLower(s.config.Options["log_level"]) == "silent" {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}

	// Open database connection
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to SQLite database: %w", err)
	}

	s.db = db
	return nil
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// Health checks the database connection health
func (s *SQLiteStorage) Health() error {
	if s.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Ping()
}

// Migrate runs database migrations
func (s *SQLiteStorage) Migrate() error {
	return s.db.AutoMigrate(
		&models.Project{},
		&models.Group{},
		&models.Host{},
		&models.Port{},
		&models.PortConnection{},
		&models.PortForward{},
		&models.TunnelSession{},
	)
}
