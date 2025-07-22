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
	
	// Host Groups
	CreateHostGroup(ctx context.Context, group *models.HostGroup) error
	GetHostGroup(ctx context.Context, id uint) (*models.HostGroup, error)
	GetHostGroups(ctx context.Context) ([]models.HostGroup, error)
	UpdateHostGroup(ctx context.Context, group *models.HostGroup) error
	DeleteHostGroup(ctx context.Context, id uint) error
	
	// Hosts
	CreateHost(ctx context.Context, host *models.Host) error
	GetHost(ctx context.Context, id uint) (*models.Host, error)
	GetHosts(ctx context.Context) ([]models.Host, error)
	GetHostsByGroup(ctx context.Context, groupID uint) ([]models.Host, error)
	UpdateHost(ctx context.Context, host *models.Host) error
	DeleteHost(ctx context.Context, id uint) error
	
	// Port Groups
	CreatePortGroup(ctx context.Context, group *models.PortGroup) error
	GetPortGroup(ctx context.Context, id uint) (*models.PortGroup, error)
	GetPortGroups(ctx context.Context) ([]models.PortGroup, error)
	UpdatePortGroup(ctx context.Context, group *models.PortGroup) error
	DeletePortGroup(ctx context.Context, id uint) error
	
	// Port Forwards
	CreatePortForward(ctx context.Context, portForward *models.PortForward) error
	GetPortForward(ctx context.Context, id uint) (*models.PortForward, error)
	GetPortForwards(ctx context.Context) ([]models.PortForward, error)
	GetPortForwardsByHost(ctx context.Context, hostID uint) ([]models.PortForward, error)
	GetPortForwardsByGroup(ctx context.Context, groupID uint) ([]models.PortForward, error)
	UpdatePortForward(ctx context.Context, portForward *models.PortForward) error
	DeletePortForward(ctx context.Context, id uint) error
	
	// Tunnel Sessions
	CreateTunnelSession(ctx context.Context, session *models.TunnelSession) error
	GetTunnelSession(ctx context.Context, id string) (*models.TunnelSession, error)
	GetTunnelSessions(ctx context.Context) ([]models.TunnelSession, error)
	GetActiveTunnelSessions(ctx context.Context) ([]models.TunnelSession, error)
	UpdateTunnelSession(ctx context.Context, session *models.TunnelSession) error
	DeleteTunnelSession(ctx context.Context, id string) error
	
	// Statistics and Search
	GetHostGroupStats(ctx context.Context, groupID uint) (map[string]interface{}, error)
	GetPortGroupStats(ctx context.Context, groupID uint) (map[string]interface{}, error)
	SearchHosts(ctx context.Context, query string) ([]models.Host, error)
	SearchPortForwards(ctx context.Context, query string) ([]models.PortForward, error)
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

// NewStorage creates a new storage instance based on configuration
func NewStorage(config StorageConfig) (StorageInterface, error) {
	switch config.Type {
	case "sqlite", "sqlite3":
		return NewSQLiteStorage(config)
	case "postgres", "postgresql":
		return NewPostgresStorage(config)
	case "mysql":
		return NewMySQLStorage(config)
	default:
		return NewSQLiteStorage(config) // Default to SQLite
	}
}
