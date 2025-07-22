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
		&models.PortForward{},
		&models.TunnelSession{},
	)
}

// ===== Project Operations =====

func (s *SQLiteStorage) CreateProject(ctx context.Context, project *models.Project) error {
	return s.db.WithContext(ctx).Create(project).Error
}

func (s *SQLiteStorage) GetProject(ctx context.Context, id uint) (*models.Project, error) {
	var project models.Project
	err := s.db.WithContext(ctx).Preload("Groups").First(&project, id).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (s *SQLiteStorage) GetProjects(ctx context.Context) ([]models.Project, error) {
	var projects []models.Project
	err := s.db.WithContext(ctx).Preload("Groups").Find(&projects).Error
	return projects, err
}

func (s *SQLiteStorage) UpdateProject(ctx context.Context, project *models.Project) error {
	return s.db.WithContext(ctx).Save(project).Error
}

func (s *SQLiteStorage) DeleteProject(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.Project{}, id).Error
}

func (s *SQLiteStorage) GetProjectStats(ctx context.Context, projectID uint) (*models.ProjectStats, error) {
	var stats models.ProjectStats
	
	// Count groups
	var groupCount int64
	s.db.WithContext(ctx).Model(&models.Group{}).Where("project_id = ?", projectID).Count(&groupCount)
	stats.TotalGroups = int(groupCount)
	
	// Count hosts in all groups of this project
	var hostCount int64
	s.db.WithContext(ctx).Table("hosts").
		Joins("JOIN groups ON hosts.group_id = groups.id").
		Where("groups.project_id = ?", projectID).
		Count(&hostCount)
	stats.TotalHosts = int(hostCount)
	
	// Count port forwards in all groups of this project
	var portCount int64
	s.db.WithContext(ctx).Table("port_forwards").
		Joins("JOIN groups ON port_forwards.group_id = groups.id").
		Where("groups.project_id = ?", projectID).
		Count(&portCount)
	stats.TotalPorts = int(portCount)
	
	// Count active tunnels
	var tunnelCount int64
	s.db.WithContext(ctx).Table("tunnel_sessions").
		Joins("JOIN port_forwards ON tunnel_sessions.port_forward_id = port_forwards.id").
		Joins("JOIN groups ON port_forwards.group_id = groups.id").
		Where("groups.project_id = ? AND tunnel_sessions.status = ?", projectID, "active").
		Count(&tunnelCount)
	stats.ActiveTunnels = int(tunnelCount)
	
	return &stats, nil
}

// ===== Group Operations =====

func (s *SQLiteStorage) CreateGroup(ctx context.Context, group *models.Group) error {
	return s.db.WithContext(ctx).Create(group).Error
}

func (s *SQLiteStorage) GetGroup(ctx context.Context, id uint) (*models.Group, error) {
	var group models.Group
	err := s.db.WithContext(ctx).Preload("Project").Preload("Hosts").Preload("PortForwards").First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *SQLiteStorage) GetGroups(ctx context.Context) ([]models.Group, error) {
	var groups []models.Group
	err := s.db.WithContext(ctx).Preload("Project").Preload("Hosts").Preload("PortForwards").Find(&groups).Error
	return groups, err
}

func (s *SQLiteStorage) GetGroupsByProject(ctx context.Context, projectID uint) ([]models.Group, error) {
	var groups []models.Group
	err := s.db.WithContext(ctx).Preload("Hosts").Preload("PortForwards").Where("project_id = ?", projectID).Find(&groups).Error
	return groups, err
}

func (s *SQLiteStorage) UpdateGroup(ctx context.Context, group *models.Group) error {
	return s.db.WithContext(ctx).Save(group).Error
}

func (s *SQLiteStorage) DeleteGroup(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.Group{}, id).Error
}

func (s *SQLiteStorage) GetGroupStats(ctx context.Context, groupID uint) (*models.GroupStats, error) {
	var stats models.GroupStats
	
	// Count hosts
	var hostCount int64
	s.db.WithContext(ctx).Model(&models.Host{}).Where("group_id = ?", groupID).Count(&hostCount)
	stats.TotalHosts = int(hostCount)
	
	// Count port forwards
	var portCount int64
	s.db.WithContext(ctx).Model(&models.PortForward{}).Where("group_id = ?", groupID).Count(&portCount)
	stats.TotalPorts = int(portCount)
	
	// Count connected hosts
	var connectedCount int64
	s.db.WithContext(ctx).Model(&models.Host{}).Where("group_id = ? AND status = ?", groupID, "connected").Count(&connectedCount)
	stats.ConnectedHosts = int(connectedCount)
	
	// Count active tunnels
	var tunnelCount int64
	s.db.WithContext(ctx).Table("tunnel_sessions").
		Joins("JOIN port_forwards ON tunnel_sessions.port_forward_id = port_forwards.id").
		Where("port_forwards.group_id = ? AND tunnel_sessions.status = ?", groupID, "active").
		Count(&tunnelCount)
	stats.ActiveTunnels = int(tunnelCount)
	
	return &stats, nil
}

// ===== Host Operations =====

func (s *SQLiteStorage) CreateHost(ctx context.Context, host *models.Host) error {
	return s.db.WithContext(ctx).Create(host).Error
}

func (s *SQLiteStorage) GetHost(ctx context.Context, id uint) (*models.Host, error) {
	var host models.Host
	err := s.db.WithContext(ctx).Preload("Group").Preload("PortForwards").First(&host, id).Error
	if err != nil {
		return nil, err
	}
	return &host, nil
}

func (s *SQLiteStorage) GetHosts(ctx context.Context) ([]models.Host, error) {
	var hosts []models.Host
	err := s.db.WithContext(ctx).Preload("Group").Find(&hosts).Error
	return hosts, err
}

func (s *SQLiteStorage) GetHostsByGroup(ctx context.Context, groupID uint) ([]models.Host, error) {
	var hosts []models.Host
	err := s.db.WithContext(ctx).Where("group_id = ?", groupID).Find(&hosts).Error
	return hosts, err
}

func (s *SQLiteStorage) UpdateHost(ctx context.Context, host *models.Host) error {
	return s.db.WithContext(ctx).Save(host).Error
}

func (s *SQLiteStorage) DeleteHost(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.Host{}, id).Error
}

func (s *SQLiteStorage) GetHostStats(ctx context.Context, hostID uint) (*models.HostStats, error) {
	var stats models.HostStats
	
	// Get host info
	var host models.Host
	if err := s.db.WithContext(ctx).First(&host, hostID).Error; err != nil {
		return nil, err
	}
	
	stats.TotalConnections = host.ConnectionCount
	
	// Count active tunnels
	var tunnelCount int64
	s.db.WithContext(ctx).Model(&models.TunnelSession{}).Where("host_id = ? AND status = ?", hostID, "active").Count(&tunnelCount)
	stats.ActiveTunnels = int(tunnelCount)
	
	// Calculate uptime percentage (simplified)
	stats.UptimePercentage = 95.0 // TODO: implement real calculation
	
	return &stats, nil
}

func (s *SQLiteStorage) SearchHosts(ctx context.Context, query string) ([]models.Host, error) {
	var hosts []models.Host
	searchPattern := "%" + query + "%"
	err := s.db.WithContext(ctx).Preload("Group").Where("name LIKE ? OR hostname LIKE ? OR description LIKE ?", searchPattern, searchPattern, searchPattern).Find(&hosts).Error
	return hosts, err
}

// ===== Port Forward Operations =====

func (s *SQLiteStorage) CreatePortForward(ctx context.Context, portForward *models.PortForward) error {
	return s.db.WithContext(ctx).Create(portForward).Error
}

func (s *SQLiteStorage) GetPortForward(ctx context.Context, id uint) (*models.PortForward, error) {
	var portForward models.PortForward
	err := s.db.WithContext(ctx).Preload("Group").Preload("Host").First(&portForward, id).Error
	if err != nil {
		return nil, err
	}
	return &portForward, nil
}

func (s *SQLiteStorage) GetPortForwards(ctx context.Context) ([]models.PortForward, error) {
	var portForwards []models.PortForward
	err := s.db.WithContext(ctx).Preload("Group").Preload("Host").Find(&portForwards).Error
	return portForwards, err
}

func (s *SQLiteStorage) GetPortForwardsByHost(ctx context.Context, hostID uint) ([]models.PortForward, error) {
	var portForwards []models.PortForward
	err := s.db.WithContext(ctx).Preload("Group").Where("host_id = ?", hostID).Find(&portForwards).Error
	return portForwards, err
}

func (s *SQLiteStorage) GetPortForwardsByGroup(ctx context.Context, groupID uint) ([]models.PortForward, error) {
	var portForwards []models.PortForward
	err := s.db.WithContext(ctx).Preload("Host").Where("group_id = ?", groupID).Find(&portForwards).Error
	return portForwards, err
}

func (s *SQLiteStorage) UpdatePortForward(ctx context.Context, portForward *models.PortForward) error {
	return s.db.WithContext(ctx).Save(portForward).Error
}

func (s *SQLiteStorage) DeletePortForward(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.PortForward{}, id).Error
}

func (s *SQLiteStorage) GetPortForwardStats(ctx context.Context, portForwardID uint) (*models.PortForwardStats, error) {
	var stats models.PortForwardStats
	
	// Count total sessions
	var totalCount int64
	s.db.WithContext(ctx).Model(&models.TunnelSession{}).Where("port_forward_id = ?", portForwardID).Count(&totalCount)
	stats.TotalSessions = int(totalCount)
	
	// Count active sessions
	var activeCount int64
	s.db.WithContext(ctx).Model(&models.TunnelSession{}).Where("port_forward_id = ? AND status = ?", portForwardID, "active").Count(&activeCount)
	stats.ActiveSessions = int(activeCount)
	
	// Sum data transferred
	s.db.WithContext(ctx).Model(&models.TunnelSession{}).Where("port_forward_id = ?", portForwardID).Select("COALESCE(SUM(data_transferred), 0)").Scan(&stats.TotalDataTransferred)
	
	return &stats, nil
}

func (s *SQLiteStorage) SearchPortForwards(ctx context.Context, query string) ([]models.PortForward, error) {
	var portForwards []models.PortForward
	searchPattern := "%" + query + "%"
	err := s.db.WithContext(ctx).Preload("Group").Preload("Host").Where("name LIKE ? OR description LIKE ? OR remote_host LIKE ?", searchPattern, searchPattern, searchPattern).Find(&portForwards).Error
	return portForwards, err
}

// ===== Tunnel Session Operations =====

func (s *SQLiteStorage) CreateTunnelSession(ctx context.Context, session *models.TunnelSession) error {
	return s.db.WithContext(ctx).Create(session).Error
}

func (s *SQLiteStorage) GetTunnelSession(ctx context.Context, id uint) (*models.TunnelSession, error) {
	var session models.TunnelSession
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortForward").First(&session, id).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *SQLiteStorage) GetTunnelSessions(ctx context.Context) ([]models.TunnelSession, error) {
	var sessions []models.TunnelSession
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortForward").Find(&sessions).Error
	return sessions, err
}

func (s *SQLiteStorage) GetActiveTunnelSessions(ctx context.Context) ([]models.TunnelSession, error) {
	var sessions []models.TunnelSession
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortForward").Where("status = ?", "active").Find(&sessions).Error
	return sessions, err
}

func (s *SQLiteStorage) UpdateTunnelSession(ctx context.Context, session *models.TunnelSession) error {
	return s.db.WithContext(ctx).Save(session).Error
}

func (s *SQLiteStorage) DeleteTunnelSession(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.TunnelSession{}, id).Error
}

func (s *SQLiteStorage) GetSessionStats(ctx context.Context) (*models.SessionStats, error) {
	var stats models.SessionStats
	
	// Count total connections
	var totalCount int64
	s.db.WithContext(ctx).Model(&models.TunnelSession{}).Count(&totalCount)
	stats.TotalConnections = totalCount
	
	// Count active connections
	var activeCount int64
	s.db.WithContext(ctx).Model(&models.TunnelSession{}).Where("status = ?", "active").Count(&activeCount)
	stats.ActiveConnections = activeCount
	
	// Count failed connections
	var failedCount int64
	s.db.WithContext(ctx).Model(&models.TunnelSession{}).Where("status = ?", "error").Count(&failedCount)
	stats.FailedConnections = failedCount
	
	// Sum bytes sent and received (using data_transferred as total)
	var totalTransferred int64
	s.db.WithContext(ctx).Model(&models.TunnelSession{}).Select("COALESCE(SUM(data_transferred), 0)").Scan(&totalTransferred)
	stats.BytesSent = totalTransferred / 2 // Simplified: assume half sent, half received
	stats.BytesReceived = totalTransferred / 2
	
	return &stats, nil
}
