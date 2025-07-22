package storage

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/aqz236/port-fly/core/models"
)

// MySQLStorage implements StorageInterface using MySQL
type MySQLStorage struct {
	db     *gorm.DB
	config StorageConfig
}

// NewMySQLStorage creates a new MySQL storage instance
func NewMySQLStorage(config StorageConfig) (*MySQLStorage, error) {
	storage := &MySQLStorage{
		config: config,
	}
	
	if err := storage.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize MySQL storage: %w", err)
	}
	
	return storage, nil
}

// Initialize initializes the MySQL database connection
func (s *MySQLStorage) Initialize() error {
	dsn := s.buildDSN()
	
	// Configure GORM logger
	gormLogger := logger.Default
	if strings.ToLower(s.config.Options["log_level"]) == "silent" {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}
	
	// Open database connection
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL database: %w", err)
	}
	
	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	
	// Set connection pool settings
	if maxOpen, ok := s.config.Options["max_open_conns"]; ok {
		if val, err := strconv.Atoi(maxOpen); err == nil {
			sqlDB.SetMaxOpenConns(val)
		}
	} else {
		sqlDB.SetMaxOpenConns(25)
	}
	
	if maxIdle, ok := s.config.Options["max_idle_conns"]; ok {
		if val, err := strconv.Atoi(maxIdle); err == nil {
			sqlDB.SetMaxIdleConns(val)
		}
	} else {
		sqlDB.SetMaxIdleConns(10)
	}
	
	s.db = db
	return nil
}

// buildDSN builds the MySQL data source name
func (s *MySQLStorage) buildDSN() string {
	host := s.config.Host
	if host == "" {
		host = "localhost"
	}
	
	port := s.config.Port
	if port == 0 {
		port = 3306
	}
	
	dbname := s.config.Database
	if dbname == "" {
		dbname = "portfly"
	}
	
	charset := s.config.Options["charset"]
	if charset == "" {
		charset = "utf8mb4"
	}
	
	parseTime := s.config.Options["parseTime"]
	if parseTime == "" {
		parseTime = "True"
	}
	
	loc := s.config.Options["loc"]
	if loc == "" {
		loc = "Local"
	}
	
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%s&loc=%s",
		s.config.Username, s.config.Password, host, port, dbname, charset, parseTime, loc)
}

// Close closes the database connection
func (s *MySQLStorage) Close() error {
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
func (s *MySQLStorage) Health() error {
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
func (s *MySQLStorage) Migrate() error {
	return s.db.AutoMigrate(
		&models.HostGroup{},
		&models.Host{},
		&models.PortGroup{},
		&models.PortForward{},
		&models.TunnelSession{},
	)
}

// Host Groups

func (s *MySQLStorage) CreateHostGroup(ctx context.Context, group *models.HostGroup) error {
	return s.db.WithContext(ctx).Create(group).Error
}

func (s *MySQLStorage) GetHostGroup(ctx context.Context, id uint) (*models.HostGroup, error) {
	var group models.HostGroup
	err := s.db.WithContext(ctx).Preload("Hosts").First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *MySQLStorage) GetHostGroups(ctx context.Context) ([]models.HostGroup, error) {
	var groups []models.HostGroup
	err := s.db.WithContext(ctx).Preload("Hosts").Find(&groups).Error
	return groups, err
}

func (s *MySQLStorage) UpdateHostGroup(ctx context.Context, group *models.HostGroup) error {
	return s.db.WithContext(ctx).Save(group).Error
}

func (s *MySQLStorage) DeleteHostGroup(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.HostGroup{}, id).Error
}

// Hosts

func (s *MySQLStorage) CreateHost(ctx context.Context, host *models.Host) error {
	return s.db.WithContext(ctx).Create(host).Error
}

func (s *MySQLStorage) GetHost(ctx context.Context, id uint) (*models.Host, error) {
	var host models.Host
	err := s.db.WithContext(ctx).Preload("HostGroup").Preload("PortForwards").First(&host, id).Error
	if err != nil {
		return nil, err
	}
	return &host, nil
}

func (s *MySQLStorage) GetHosts(ctx context.Context) ([]models.Host, error) {
	var hosts []models.Host
	err := s.db.WithContext(ctx).Preload("HostGroup").Preload("PortForwards").Find(&hosts).Error
	return hosts, err
}

func (s *MySQLStorage) GetHostsByGroup(ctx context.Context, groupID uint) ([]models.Host, error) {
	var hosts []models.Host
	err := s.db.WithContext(ctx).Preload("HostGroup").Preload("PortForwards").Where("host_group_id = ?", groupID).Find(&hosts).Error
	return hosts, err
}

func (s *MySQLStorage) UpdateHost(ctx context.Context, host *models.Host) error {
	return s.db.WithContext(ctx).Save(host).Error
}

func (s *MySQLStorage) DeleteHost(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.Host{}, id).Error
}

// Port Groups

func (s *MySQLStorage) CreatePortGroup(ctx context.Context, group *models.PortGroup) error {
	return s.db.WithContext(ctx).Create(group).Error
}

func (s *MySQLStorage) GetPortGroup(ctx context.Context, id uint) (*models.PortGroup, error) {
	var group models.PortGroup
	err := s.db.WithContext(ctx).Preload("PortForwards").Preload("PortForwards.Host").First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *MySQLStorage) GetPortGroups(ctx context.Context) ([]models.PortGroup, error) {
	var groups []models.PortGroup
	err := s.db.WithContext(ctx).Preload("PortForwards").Preload("PortForwards.Host").Find(&groups).Error
	return groups, err
}

func (s *MySQLStorage) UpdatePortGroup(ctx context.Context, group *models.PortGroup) error {
	return s.db.WithContext(ctx).Save(group).Error
}

func (s *MySQLStorage) DeletePortGroup(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.PortGroup{}, id).Error
}

// Port Forwards

func (s *MySQLStorage) CreatePortForward(ctx context.Context, portForward *models.PortForward) error {
	return s.db.WithContext(ctx).Create(portForward).Error
}

func (s *MySQLStorage) GetPortForward(ctx context.Context, id uint) (*models.PortForward, error) {
	var portForward models.PortForward
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortGroup").First(&portForward, id).Error
	if err != nil {
		return nil, err
	}
	return &portForward, nil
}

func (s *MySQLStorage) GetPortForwards(ctx context.Context) ([]models.PortForward, error) {
	var portForwards []models.PortForward
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortGroup").Find(&portForwards).Error
	return portForwards, err
}

func (s *MySQLStorage) GetPortForwardsByHost(ctx context.Context, hostID uint) ([]models.PortForward, error) {
	var portForwards []models.PortForward
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortGroup").Where("host_id = ?", hostID).Find(&portForwards).Error
	return portForwards, err
}

func (s *MySQLStorage) GetPortForwardsByGroup(ctx context.Context, groupID uint) ([]models.PortForward, error) {
	var portForwards []models.PortForward
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortGroup").Where("port_group_id = ?", groupID).Find(&portForwards).Error
	return portForwards, err
}

func (s *MySQLStorage) UpdatePortForward(ctx context.Context, portForward *models.PortForward) error {
	return s.db.WithContext(ctx).Save(portForward).Error
}

func (s *MySQLStorage) DeletePortForward(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.PortForward{}, id).Error
}

// Tunnel Sessions

func (s *MySQLStorage) CreateTunnelSession(ctx context.Context, session *models.TunnelSession) error {
	return s.db.WithContext(ctx).Create(session).Error
}

func (s *MySQLStorage) GetTunnelSession(ctx context.Context, id string) (*models.TunnelSession, error) {
	var session models.TunnelSession
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortForward").Preload("PortGroup").First(&session, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *MySQLStorage) GetTunnelSessions(ctx context.Context) ([]models.TunnelSession, error) {
	var sessions []models.TunnelSession
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortForward").Preload("PortGroup").Find(&sessions).Error
	return sessions, err
}

func (s *MySQLStorage) GetActiveTunnelSessions(ctx context.Context) ([]models.TunnelSession, error) {
	var sessions []models.TunnelSession
	activeStates := []string{"connecting", "connected", "active"}
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortForward").Preload("PortGroup").Where("status IN ?", activeStates).Find(&sessions).Error
	return sessions, err
}

func (s *MySQLStorage) UpdateTunnelSession(ctx context.Context, session *models.TunnelSession) error {
	return s.db.WithContext(ctx).Save(session).Error
}

func (s *MySQLStorage) DeleteTunnelSession(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&models.TunnelSession{}, "id = ?", id).Error
}

// Statistics and Search

func (s *MySQLStorage) GetHostGroupStats(ctx context.Context, groupID uint) (map[string]interface{}, error) {
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

func (s *MySQLStorage) GetPortGroupStats(ctx context.Context, groupID uint) (map[string]interface{}, error) {
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

func (s *MySQLStorage) SearchHosts(ctx context.Context, query string) ([]models.Host, error) {
	var hosts []models.Host
	searchTerm := "%" + query + "%"
	err := s.db.WithContext(ctx).Preload("HostGroup").Preload("PortForwards").
		Where("name LIKE ? OR description LIKE ? OR hostname LIKE ? OR username LIKE ?", 
			searchTerm, searchTerm, searchTerm, searchTerm).
		Find(&hosts).Error
	return hosts, err
}

func (s *MySQLStorage) SearchPortForwards(ctx context.Context, query string) ([]models.PortForward, error) {
	var portForwards []models.PortForward
	searchTerm := "%" + query + "%"
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortGroup").
		Where("name LIKE ? OR description LIKE ? OR remote_host LIKE ?", 
			searchTerm, searchTerm, searchTerm).
		Find(&portForwards).Error
	return portForwards, err
}
