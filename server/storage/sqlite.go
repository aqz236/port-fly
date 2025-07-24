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
	// 如果设置了父项目，需要计算层级和路径
	if project.ParentID != nil {
		var parent models.Project
		if err := s.db.WithContext(ctx).First(&parent, *project.ParentID).Error; err != nil {
			return fmt.Errorf("parent project not found: %w", err)
		}
		project.Level = parent.Level + 1
	}
	
	// 创建项目
	if err := s.db.WithContext(ctx).Create(project).Error; err != nil {
		return err
	}
	
	// 更新路径
	project.Path = project.BuildPath()
	return s.db.WithContext(ctx).Save(project).Error
}

func (s *SQLiteStorage) GetProject(ctx context.Context, id uint) (*models.Project, error) {
	var project models.Project
	err := s.db.WithContext(ctx).Preload("Groups").Preload("Parent").Preload("Children").First(&project, id).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (s *SQLiteStorage) GetProjects(ctx context.Context) ([]models.Project, error) {
	var projects []models.Project
	err := s.db.WithContext(ctx).Preload("Groups").Preload("Parent").Preload("Children").Find(&projects).Error
	return projects, err
}

func (s *SQLiteStorage) GetProjectsByParent(ctx context.Context, parentID *uint, includeChildren bool) ([]models.Project, error) {
	var projects []models.Project
	query := s.db.WithContext(ctx).Preload("Groups").Preload("Parent")
	
	if includeChildren {
		query = query.Preload("Children")
	}
	
	if parentID == nil {
		// 获取根项目（没有父项目的）
		query = query.Where("parent_id IS NULL")
	} else {
		// 获取指定父项目的子项目
		query = query.Where("parent_id = ?", *parentID)
	}
	
	err := query.Order("sort ASC, name ASC").Find(&projects).Error
	return projects, err
}

func (s *SQLiteStorage) GetProjectTree(ctx context.Context, rootID *uint) ([]*models.ProjectTreeNode, error) {
	var projects []models.Project
	query := s.db.WithContext(ctx).Preload("Groups")
	
	if rootID == nil {
		// 获取所有项目
		query = query.Find(&projects)
	} else {
		// 获取指定根项目及其子树
		var rootProject models.Project
		if err := s.db.WithContext(ctx).First(&rootProject, *rootID).Error; err != nil {
			return nil, fmt.Errorf("root project not found: %w", err)
		}
		// 获取所有路径以rootID开头的项目
		query = query.Where("path LIKE ?", rootProject.Path+"/%").Or("id = ?", *rootID)
		query = query.Find(&projects)
	}
	
	if query.Error != nil {
		return nil, query.Error
	}
	
	// 构建树状结构
	return s.buildProjectTree(projects, rootID), nil
}

func (s *SQLiteStorage) buildProjectTree(projects []models.Project, rootID *uint) []*models.ProjectTreeNode {
	// 创建ID到项目的映射
	projectMap := make(map[uint]*models.ProjectTreeNode)
	for i := range projects {
		node := &models.ProjectTreeNode{
			Project:     &projects[i],
			Children:    []*models.ProjectTreeNode{},
			HasChildren: false,
		}
		projectMap[projects[i].ID] = node
	}
	
	// 构建树状结构
	var roots []*models.ProjectTreeNode
	for _, node := range projectMap {
		if node.ParentID == nil || (rootID != nil && node.ParentID != nil && *node.ParentID == *rootID) {
			// 这是根节点或指定根的直接子节点
			if rootID == nil || node.ID != *rootID {
				roots = append(roots, node)
			}
		} else if parent, exists := projectMap[*node.ParentID]; exists {
			// 添加到父节点的子列表
			parent.Children = append(parent.Children, node)
			parent.HasChildren = true
		}
	}
	
	return roots
}

func (s *SQLiteStorage) MoveProject(ctx context.Context, params *models.MoveProjectParams) error {
	// 开始事务
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	// 获取要移动的项目
	var project models.Project
	if err := tx.First(&project, params.ProjectID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("project not found: %w", err)
	}
	
	// 检查是否移动到自己的子项目下（防止循环引用）
	if params.ParentID != nil {
		var parent models.Project
		if err := tx.First(&parent, *params.ParentID).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("parent project not found: %w", err)
		}
		
		// 检查是否形成循环
		if strings.Contains(parent.Path, fmt.Sprintf("/%d/", project.ID)) {
			tx.Rollback()
			return fmt.Errorf("cannot move project under its own child")
		}
		
		project.ParentID = params.ParentID
		project.Level = parent.Level + 1
	} else {
		project.ParentID = nil
		project.Level = 0
	}
	
	project.Sort = params.Position
	
	// 保存项目
	if err := tx.Save(&project).Error; err != nil {
		tx.Rollback()
		return err
	}
	
	// 更新路径
	project.Path = project.BuildPath()
	if err := tx.Save(&project).Error; err != nil {
		tx.Rollback()
		return err
	}
	
	// 递归更新所有子项目的层级和路径
	if err := s.updateChildrenPaths(tx, project.ID); err != nil {
		tx.Rollback()
		return err
	}
	
	return tx.Commit().Error
}

func (s *SQLiteStorage) updateChildrenPaths(tx *gorm.DB, parentID uint) error {
	var children []models.Project
	if err := tx.Where("parent_id = ?", parentID).Find(&children).Error; err != nil {
		return err
	}
	
	for _, child := range children {
		// 获取父项目信息
		var parent models.Project
		if err := tx.First(&parent, child.ParentID).Error; err != nil {
			return err
		}
		
		child.Level = parent.Level + 1
		child.Path = child.BuildPath()
		
		if err := tx.Save(&child).Error; err != nil {
			return err
		}
		
		// 递归更新子项目
		if err := s.updateChildrenPaths(tx, child.ID); err != nil {
			return err
		}
	}
	
	return nil
}

func (s *SQLiteStorage) GetProjectChildren(ctx context.Context, parentID uint) ([]models.Project, error) {
	var children []models.Project
	err := s.db.WithContext(ctx).
		Preload("Groups").
		Where("parent_id = ?", parentID).
		Order("sort ASC, name ASC").
		Find(&children).Error
	return children, err
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

// ===== Port Operations (V2) =====

func (s *SQLiteStorage) CreatePort(ctx context.Context, port *models.Port) error {
	return s.db.WithContext(ctx).Create(port).Error
}

func (s *SQLiteStorage) GetPort(ctx context.Context, id uint) (*models.Port, error) {
	var port models.Port
	err := s.db.WithContext(ctx).Preload("Group").Preload("Host").First(&port, id).Error
	if err != nil {
		return nil, err
	}
	return &port, nil
}

func (s *SQLiteStorage) GetPorts(ctx context.Context) ([]models.Port, error) {
	var ports []models.Port
	err := s.db.WithContext(ctx).Preload("Group").Preload("Host").Find(&ports).Error
	return ports, err
}

func (s *SQLiteStorage) GetPortsByGroup(ctx context.Context, groupID uint) ([]models.Port, error) {
	var ports []models.Port
	err := s.db.WithContext(ctx).Preload("Host").Where("group_id = ?", groupID).Find(&ports).Error
	return ports, err
}

func (s *SQLiteStorage) GetPortsByHost(ctx context.Context, hostID uint) ([]models.Port, error) {
	var ports []models.Port
	err := s.db.WithContext(ctx).Preload("Group").Where("host_id = ?", hostID).Find(&ports).Error
	return ports, err
}

func (s *SQLiteStorage) UpdatePort(ctx context.Context, port *models.Port) error {
	return s.db.WithContext(ctx).Save(port).Error
}

func (s *SQLiteStorage) DeletePort(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.Port{}, id).Error
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
