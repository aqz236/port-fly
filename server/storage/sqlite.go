package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/aqz236/port-fly/core/models"
)

// SQLiteStorage implements StorageInterface using SQLite
type SQLiteStorage struct {
	db     *gorm.DB
	config StorageConfig
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(config StorageConfig) (*SQLiteStorage, error) {
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
	
	// Configure SQLite settings
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	
	// Set SQLite pragmas
	if _, err := sqlDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}
	if _, err := sqlDB.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return fmt.Errorf("failed to set WAL mode: %w", err)
	}
	if _, err := sqlDB.Exec("PRAGMA synchronous = NORMAL"); err != nil {
		return fmt.Errorf("failed to set synchronous mode: %w", err)
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
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	
	return sqlDB.Ping()
}

// Migrate runs database migrations
func (s *SQLiteStorage) Migrate() error {
	return s.db.AutoMigrate(
		&models.HostGroup{},
		&models.Host{},
		&models.PortGroup{},
		&models.PortForward{},
		&models.TunnelSession{},
	)
}

// Host Groups

func (s *SQLiteStorage) CreateHostGroup(ctx context.Context, group *models.HostGroup) error {
	return s.db.WithContext(ctx).Create(group).Error
}

func (s *SQLiteStorage) GetHostGroup(ctx context.Context, id uint) (*models.HostGroup, error) {
	var group models.HostGroup
	err := s.db.WithContext(ctx).Preload("Hosts").First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *SQLiteStorage) GetHostGroups(ctx context.Context) ([]models.HostGroup, error) {
	var groups []models.HostGroup
	err := s.db.WithContext(ctx).Preload("Hosts").Find(&groups).Error
	return groups, err
}

func (s *SQLiteStorage) UpdateHostGroup(ctx context.Context, group *models.HostGroup) error {
	return s.db.WithContext(ctx).Save(group).Error
}

func (s *SQLiteStorage) DeleteHostGroup(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.HostGroup{}, id).Error
}

// Hosts

func (s *SQLiteStorage) CreateHost(ctx context.Context, host *models.Host) error {
	return s.db.WithContext(ctx).Create(host).Error
}

func (s *SQLiteStorage) GetHost(ctx context.Context, id uint) (*models.Host, error) {
	var host models.Host
	err := s.db.WithContext(ctx).Preload("HostGroup").Preload("PortForwards").First(&host, id).Error
	if err != nil {
		return nil, err
	}
	return &host, nil
}

func (s *SQLiteStorage) GetHosts(ctx context.Context) ([]models.Host, error) {
	var hosts []models.Host
	err := s.db.WithContext(ctx).Preload("HostGroup").Preload("PortForwards").Find(&hosts).Error
	return hosts, err
}

func (s *SQLiteStorage) GetHostsByGroup(ctx context.Context, groupID uint) ([]models.Host, error) {
	var hosts []models.Host
	err := s.db.WithContext(ctx).Preload("HostGroup").Preload("PortForwards").Where("host_group_id = ?", groupID).Find(&hosts).Error
	return hosts, err
}

func (s *SQLiteStorage) UpdateHost(ctx context.Context, host *models.Host) error {
	return s.db.WithContext(ctx).Save(host).Error
}

func (s *SQLiteStorage) DeleteHost(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.Host{}, id).Error
}

// Port Groups

func (s *SQLiteStorage) CreatePortGroup(ctx context.Context, group *models.PortGroup) error {
	return s.db.WithContext(ctx).Create(group).Error
}

func (s *SQLiteStorage) GetPortGroup(ctx context.Context, id uint) (*models.PortGroup, error) {
	var group models.PortGroup
	err := s.db.WithContext(ctx).Preload("PortForwards").Preload("PortForwards.Host").First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *SQLiteStorage) GetPortGroups(ctx context.Context) ([]models.PortGroup, error) {
	var groups []models.PortGroup
	err := s.db.WithContext(ctx).Preload("PortForwards").Preload("PortForwards.Host").Find(&groups).Error
	return groups, err
}

func (s *SQLiteStorage) UpdatePortGroup(ctx context.Context, group *models.PortGroup) error {
	return s.db.WithContext(ctx).Save(group).Error
}

func (s *SQLiteStorage) DeletePortGroup(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.PortGroup{}, id).Error
}

// Port Forwards

func (s *SQLiteStorage) CreatePortForward(ctx context.Context, portForward *models.PortForward) error {
	return s.db.WithContext(ctx).Create(portForward).Error
}

func (s *SQLiteStorage) GetPortForward(ctx context.Context, id uint) (*models.PortForward, error) {
	var portForward models.PortForward
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortGroup").First(&portForward, id).Error
	if err != nil {
		return nil, err
	}
	return &portForward, nil
}

func (s *SQLiteStorage) GetPortForwards(ctx context.Context) ([]models.PortForward, error) {
	var portForwards []models.PortForward
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortGroup").Find(&portForwards).Error
	return portForwards, err
}

func (s *SQLiteStorage) GetPortForwardsByHost(ctx context.Context, hostID uint) ([]models.PortForward, error) {
	var portForwards []models.PortForward
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortGroup").Where("host_id = ?", hostID).Find(&portForwards).Error
	return portForwards, err
}

func (s *SQLiteStorage) GetPortForwardsByGroup(ctx context.Context, groupID uint) ([]models.PortForward, error) {
	var portForwards []models.PortForward
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortGroup").Where("port_group_id = ?", groupID).Find(&portForwards).Error
	return portForwards, err
}

func (s *SQLiteStorage) UpdatePortForward(ctx context.Context, portForward *models.PortForward) error {
	return s.db.WithContext(ctx).Save(portForward).Error
}

func (s *SQLiteStorage) DeletePortForward(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.PortForward{}, id).Error
}

// Tunnel Sessions

func (s *SQLiteStorage) CreateTunnelSession(ctx context.Context, session *models.TunnelSession) error {
	return s.db.WithContext(ctx).Create(session).Error
}

func (s *SQLiteStorage) GetTunnelSession(ctx context.Context, id string) (*models.TunnelSession, error) {
	var session models.TunnelSession
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortForward").Preload("PortGroup").First(&session, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *SQLiteStorage) GetTunnelSessions(ctx context.Context) ([]models.TunnelSession, error) {
	var sessions []models.TunnelSession
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortForward").Preload("PortGroup").Find(&sessions).Error
	return sessions, err
}

func (s *SQLiteStorage) GetActiveTunnelSessions(ctx context.Context) ([]models.TunnelSession, error) {
	var sessions []models.TunnelSession
	activeStates := []string{"connecting", "connected", "active"}
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortForward").Preload("PortGroup").Where("status IN ?", activeStates).Find(&sessions).Error
	return sessions, err
}

func (s *SQLiteStorage) UpdateTunnelSession(ctx context.Context, session *models.TunnelSession) error {
	return s.db.WithContext(ctx).Save(session).Error
}

func (s *SQLiteStorage) DeleteTunnelSession(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&models.TunnelSession{}, "id = ?", id).Error
}

// Statistics and Search

func (s *SQLiteStorage) GetHostGroupStats(ctx context.Context, groupID uint) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Count hosts in group
	var hostCount int64
	s.db.WithContext(ctx).Model(&models.Host{}).Where("host_group_id = ?", groupID).Count(&hostCount)
	stats["host_count"] = hostCount
	
	// Count port forwards for hosts in group
	var portForwardCount int64
	s.db.WithContext(ctx).Model(&models.PortForward{}).
		Joins("JOIN hosts ON hosts.id = port_forwards.host_id").
		Where("hosts.host_group_id = ?", groupID).
		Count(&portForwardCount)
	stats["port_forward_count"] = portForwardCount
	
	// Count active sessions for hosts in group
	var activeSessionCount int64
	activeStates := []string{"connecting", "connected", "active"}
	s.db.WithContext(ctx).Model(&models.TunnelSession{}).
		Joins("JOIN hosts ON hosts.id = tunnel_sessions.host_id").
		Where("hosts.host_group_id = ? AND tunnel_sessions.status IN ?", groupID, activeStates).
		Count(&activeSessionCount)
	stats["active_session_count"] = activeSessionCount
	
	return stats, nil
}

func (s *SQLiteStorage) GetPortGroupStats(ctx context.Context, groupID uint) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Count port forwards in group
	var portForwardCount int64
	s.db.WithContext(ctx).Model(&models.PortForward{}).Where("port_group_id = ?", groupID).Count(&portForwardCount)
	stats["port_forward_count"] = portForwardCount
	
	// Count active sessions for port forwards in group
	var activeSessionCount int64
	activeStates := []string{"connecting", "connected", "active"}
	s.db.WithContext(ctx).Model(&models.TunnelSession{}).Where("port_group_id = ? AND status IN ?", groupID, activeStates).Count(&activeSessionCount)
	stats["active_session_count"] = activeSessionCount
	
	return stats, nil
}

func (s *SQLiteStorage) SearchHosts(ctx context.Context, query string) ([]models.Host, error) {
	var hosts []models.Host
	searchTerm := "%" + query + "%"
	err := s.db.WithContext(ctx).Preload("HostGroup").Preload("PortForwards").
		Where("name LIKE ? OR description LIKE ? OR hostname LIKE ? OR username LIKE ?", 
			searchTerm, searchTerm, searchTerm, searchTerm).
		Find(&hosts).Error
	return hosts, err
}

func (s *SQLiteStorage) SearchPortForwards(ctx context.Context, query string) ([]models.PortForward, error) {
	var portForwards []models.PortForward
	searchTerm := "%" + query + "%"
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortGroup").
		Where("name LIKE ? OR description LIKE ? OR remote_host LIKE ?", 
			searchTerm, searchTerm, searchTerm).
		Find(&portForwards).Error
	return portForwards, err
}
