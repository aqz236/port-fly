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
		&models.Project{},
		&models.Group{},
		&models.Host{},
		&models.PortForward{},
		&models.TunnelSession{},
	)
}

// ===== Project Operations =====

func (s *MySQLStorage) CreateProject(ctx context.Context, project *models.Project) error {
	return s.db.WithContext(ctx).Create(project).Error
}

func (s *MySQLStorage) GetProject(ctx context.Context, id uint) (*models.Project, error) {
	var project models.Project
	err := s.db.WithContext(ctx).Preload("Groups").First(&project, id).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (s *MySQLStorage) GetProjects(ctx context.Context) ([]models.Project, error) {
	var projects []models.Project
	err := s.db.WithContext(ctx).Preload("Groups").Find(&projects).Error
	return projects, err
}

func (s *MySQLStorage) UpdateProject(ctx context.Context, project *models.Project) error {
	return s.db.WithContext(ctx).Save(project).Error
}

func (s *MySQLStorage) DeleteProject(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.Project{}, id).Error
}

func (s *MySQLStorage) GetProjectStats(ctx context.Context, projectID uint) (*models.ProjectStats, error) {
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

func (s *MySQLStorage) CreateGroup(ctx context.Context, group *models.Group) error {
	return s.db.WithContext(ctx).Create(group).Error
}

func (s *MySQLStorage) GetGroup(ctx context.Context, id uint) (*models.Group, error) {
	var group models.Group
	err := s.db.WithContext(ctx).Preload("Project").Preload("Hosts").Preload("PortForwards").First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *MySQLStorage) GetGroups(ctx context.Context) ([]models.Group, error) {
	var groups []models.Group
	err := s.db.WithContext(ctx).Preload("Project").Preload("Hosts").Preload("PortForwards").Find(&groups).Error
	return groups, err
}

func (s *MySQLStorage) GetGroupsByProject(ctx context.Context, projectID uint) ([]models.Group, error) {
	var groups []models.Group
	err := s.db.WithContext(ctx).Preload("Hosts").Preload("PortForwards").Where("project_id = ?", projectID).Find(&groups).Error
	return groups, err
}

func (s *MySQLStorage) UpdateGroup(ctx context.Context, group *models.Group) error {
	return s.db.WithContext(ctx).Save(group).Error
}

func (s *MySQLStorage) DeleteGroup(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.Group{}, id).Error
}

func (s *MySQLStorage) GetGroupStats(ctx context.Context, groupID uint) (*models.GroupStats, error) {
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

func (s *MySQLStorage) CreateHost(ctx context.Context, host *models.Host) error {
	return s.db.WithContext(ctx).Create(host).Error
}

func (s *MySQLStorage) GetHost(ctx context.Context, id uint) (*models.Host, error) {
	var host models.Host
	err := s.db.WithContext(ctx).Preload("Group").Preload("PortForwards").First(&host, id).Error
	if err != nil {
		return nil, err
	}
	return &host, nil
}

func (s *MySQLStorage) GetHosts(ctx context.Context) ([]models.Host, error) {
	var hosts []models.Host
	err := s.db.WithContext(ctx).Preload("Group").Find(&hosts).Error
	return hosts, err
}

func (s *MySQLStorage) GetHostsByGroup(ctx context.Context, groupID uint) ([]models.Host, error) {
	var hosts []models.Host
	err := s.db.WithContext(ctx).Where("group_id = ?", groupID).Find(&hosts).Error
	return hosts, err
}

func (s *MySQLStorage) UpdateHost(ctx context.Context, host *models.Host) error {
	return s.db.WithContext(ctx).Save(host).Error
}

func (s *MySQLStorage) DeleteHost(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.Host{}, id).Error
}

func (s *MySQLStorage) GetHostStats(ctx context.Context, hostID uint) (*models.HostStats, error) {
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

func (s *MySQLStorage) SearchHosts(ctx context.Context, query string) ([]models.Host, error) {
	var hosts []models.Host
	searchPattern := "%" + query + "%"
	err := s.db.WithContext(ctx).Preload("Group").Where("name LIKE ? OR hostname LIKE ? OR description LIKE ?", searchPattern, searchPattern, searchPattern).Find(&hosts).Error
	return hosts, err
}

// ===== Port Forward Operations =====

func (s *MySQLStorage) CreatePortForward(ctx context.Context, portForward *models.PortForward) error {
	return s.db.WithContext(ctx).Create(portForward).Error
}

func (s *MySQLStorage) GetPortForward(ctx context.Context, id uint) (*models.PortForward, error) {
	var portForward models.PortForward
	err := s.db.WithContext(ctx).Preload("Group").Preload("Host").First(&portForward, id).Error
	if err != nil {
		return nil, err
	}
	return &portForward, nil
}

func (s *MySQLStorage) GetPortForwards(ctx context.Context) ([]models.PortForward, error) {
	var portForwards []models.PortForward
	err := s.db.WithContext(ctx).Preload("Group").Preload("Host").Find(&portForwards).Error
	return portForwards, err
}

func (s *MySQLStorage) GetPortForwardsByHost(ctx context.Context, hostID uint) ([]models.PortForward, error) {
	var portForwards []models.PortForward
	err := s.db.WithContext(ctx).Preload("Group").Where("host_id = ?", hostID).Find(&portForwards).Error
	return portForwards, err
}

func (s *MySQLStorage) GetPortForwardsByGroup(ctx context.Context, groupID uint) ([]models.PortForward, error) {
	var portForwards []models.PortForward
	err := s.db.WithContext(ctx).Preload("Host").Where("group_id = ?", groupID).Find(&portForwards).Error
	return portForwards, err
}

func (s *MySQLStorage) UpdatePortForward(ctx context.Context, portForward *models.PortForward) error {
	return s.db.WithContext(ctx).Save(portForward).Error
}

func (s *MySQLStorage) DeletePortForward(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.PortForward{}, id).Error
}

func (s *MySQLStorage) GetPortForwardStats(ctx context.Context, portForwardID uint) (*models.PortForwardStats, error) {
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

func (s *MySQLStorage) SearchPortForwards(ctx context.Context, query string) ([]models.PortForward, error) {
	var portForwards []models.PortForward
	searchPattern := "%" + query + "%"
	err := s.db.WithContext(ctx).Preload("Group").Preload("Host").Where("name LIKE ? OR description LIKE ? OR remote_host LIKE ?", searchPattern, searchPattern, searchPattern).Find(&portForwards).Error
	return portForwards, err
}

// ===== Tunnel Session Operations =====

func (s *MySQLStorage) CreateTunnelSession(ctx context.Context, session *models.TunnelSession) error {
	return s.db.WithContext(ctx).Create(session).Error
}

func (s *MySQLStorage) GetTunnelSession(ctx context.Context, id uint) (*models.TunnelSession, error) {
	var session models.TunnelSession
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortForward").First(&session, id).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *MySQLStorage) GetTunnelSessions(ctx context.Context) ([]models.TunnelSession, error) {
	var sessions []models.TunnelSession
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortForward").Find(&sessions).Error
	return sessions, err
}

func (s *MySQLStorage) GetActiveTunnelSessions(ctx context.Context) ([]models.TunnelSession, error) {
	var sessions []models.TunnelSession
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortForward").Where("status = ?", "active").Find(&sessions).Error
	return sessions, err
}

func (s *MySQLStorage) UpdateTunnelSession(ctx context.Context, session *models.TunnelSession) error {
	return s.db.WithContext(ctx).Save(session).Error
}

func (s *MySQLStorage) DeleteTunnelSession(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.TunnelSession{}, id).Error
}

func (s *MySQLStorage) GetSessionStats(ctx context.Context) (*models.SessionStats, error) {
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
