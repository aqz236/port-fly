package storage

import (
	"context"

	"github.com/aqz236/port-fly/core/models"
)

// StorageInterface defines the contract for data storage
type StorageInterface interface {
	// Database management
	Initialize() error
	Close() error
	Health() error
	Migrate() error

	// ===== Project Operations =====
	CreateProject(ctx context.Context, project *models.Project) error
	GetProject(ctx context.Context, id uint) (*models.Project, error)
	GetProjects(ctx context.Context) ([]models.Project, error)
	GetProjectsByParent(ctx context.Context, parentID *uint, includeChildren bool) ([]models.Project, error)
	GetProjectTree(ctx context.Context, rootID *uint) ([]*models.ProjectTreeNode, error)
	MoveProject(ctx context.Context, params *models.MoveProjectParams) error
	UpdateProject(ctx context.Context, project *models.Project) error
	DeleteProject(ctx context.Context, id uint) error
	GetProjectStats(ctx context.Context, projectID uint) (*models.ProjectStats, error)
	GetProjectChildren(ctx context.Context, parentID uint) ([]models.Project, error)

	// ===== Group Operations =====
	CreateGroup(ctx context.Context, group *models.Group) error
	GetGroup(ctx context.Context, id uint) (*models.Group, error)
	GetGroups(ctx context.Context) ([]models.Group, error)
	GetGroupsByProject(ctx context.Context, projectID uint) ([]models.Group, error)
	UpdateGroup(ctx context.Context, group *models.Group) error
	DeleteGroup(ctx context.Context, id uint) error
	GetGroupStats(ctx context.Context, groupID uint) (*models.GroupStats, error)

	// ===== Host Operations =====
	CreateHost(ctx context.Context, host *models.Host) error
	GetHost(ctx context.Context, id uint) (*models.Host, error)
	GetHosts(ctx context.Context) ([]models.Host, error)
	GetHostsByGroup(ctx context.Context, groupID uint) ([]models.Host, error)
	UpdateHost(ctx context.Context, host *models.Host) error
	DeleteHost(ctx context.Context, id uint) error
	GetHostStats(ctx context.Context, hostID uint) (*models.HostStats, error)
	SearchHosts(ctx context.Context, query string) ([]models.Host, error)

	// ===== Port Operations =====
	CreatePort(ctx context.Context, port *models.Port) error
	GetPort(ctx context.Context, id uint) (*models.Port, error)
	GetPorts(ctx context.Context) ([]models.Port, error)
	GetPortsByGroup(ctx context.Context, groupID uint) ([]models.Port, error)
	GetPortsByHost(ctx context.Context, hostID uint) ([]models.Port, error)
	UpdatePort(ctx context.Context, port *models.Port) error
	DeletePort(ctx context.Context, id uint) error
	GetPortStats(ctx context.Context, portID uint) (*models.PortStats, error)
	SearchPorts(ctx context.Context, query string) ([]models.Port, error)
	UpdatePortStatus(ctx context.Context, portID uint, status models.PortStatus) error

	// ===== Port Connection Operations =====
	CreatePortConnection(ctx context.Context, connection *models.PortConnection) error
	GetPortConnection(ctx context.Context, id uint) (*models.PortConnection, error)
	GetPortConnections(ctx context.Context) ([]models.PortConnection, error)
	GetActivePortConnections(ctx context.Context) ([]models.PortConnection, error)
	UpdatePortConnection(ctx context.Context, connection *models.PortConnection) error
	DeletePortConnection(ctx context.Context, id uint) error
	GetPortConnectionByPorts(ctx context.Context, remotePortID, localPortID uint) (*models.PortConnection, error)

	// ===== Tunnel Session Operations =====
	CreateTunnelSession(ctx context.Context, session *models.TunnelSession) error
	GetTunnelSession(ctx context.Context, id uint) (*models.TunnelSession, error)
	GetTunnelSessions(ctx context.Context) ([]models.TunnelSession, error)
	GetActiveTunnelSessions(ctx context.Context) ([]models.TunnelSession, error)
	UpdateTunnelSession(ctx context.Context, session *models.TunnelSession) error
	DeleteTunnelSession(ctx context.Context, id uint) error
	GetSessionStats(ctx context.Context) (*models.SessionStats, error)
}

// StorageConfig contains storage configuration
type StorageConfig struct {
	Type     string            `json:"type"`
	Host     string            `json:"host"`
	Port     int               `json:"port"`
	Database string            `json:"database"`
	Username string            `json:"username"`
	Password string            `json:"password"`
	SSLMode  string            `json:"ssl_mode"`
	Options  map[string]string `json:"options"`
}
