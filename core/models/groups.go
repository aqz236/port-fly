package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// HostGroup represents a group of SSH hosts
type HostGroup struct {
	ID          uint                   `json:"id" gorm:"primaryKey"`
	Name        string                 `json:"name" gorm:"uniqueIndex;not null"`
	Description string                 `json:"description"`
	Color       string                 `json:"color" gorm:"default:'#3B82F6'"` // Tailwind blue-500
	Icon        string                 `json:"icon" gorm:"default:'server'"`
	Tags        string                 `json:"tags"`                           // JSON array stored as string
	Metadata    string                 `json:"metadata"`                       // JSON object stored as string
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	DeletedAt   gorm.DeletedAt         `json:"deleted_at,omitempty" gorm:"index"`
	
	// Relationships
	Hosts       []Host                 `json:"hosts" gorm:"foreignKey:HostGroupID"`
}

// Host represents an SSH host configuration
type Host struct {
	ID          uint                   `json:"id" gorm:"primaryKey"`
	Name        string                 `json:"name" gorm:"not null"`
	Description string                 `json:"description"`
	
	// SSH Connection Details
	Hostname    string                 `json:"hostname" gorm:"not null"`
	Port        int                    `json:"port" gorm:"default:22"`
	Username    string                 `json:"username" gorm:"not null"`
	
	// Authentication
	AuthMethod  string                 `json:"auth_method" gorm:"not null;default:'private_key'"` // password, private_key, agent
	Password    string                 `json:"password,omitempty"`                                // Encrypted
	PrivateKey  string                 `json:"private_key,omitempty"`                             // Encrypted or path
	Passphrase  string                 `json:"passphrase,omitempty"`                              // Encrypted
	
	// Advanced Options
	HostKeyCallback string             `json:"host_key_callback" gorm:"default:'ask'"`
	KnownHostsFile  string             `json:"known_hosts_file"`
	ConnectTimeout  int                `json:"connect_timeout" gorm:"default:30"`    // seconds
	KeepAlive       int                `json:"keep_alive" gorm:"default:30"`         // seconds
	MaxRetries      int                `json:"max_retries" gorm:"default:3"`
	RetryInterval   int                `json:"retry_interval" gorm:"default:5"`      // seconds
	
	// Metadata
	Tags        string                 `json:"tags"`           // JSON array stored as string
	Color       string                 `json:"color"`          // Custom color for UI
	Icon        string                 `json:"icon"`           // Custom icon
	LastUsed    *time.Time             `json:"last_used,omitempty"`
	UseCount    int                    `json:"use_count" gorm:"default:0"`
	
	// Group Association
	HostGroupID *uint                  `json:"host_group_id"`
	HostGroup   *HostGroup             `json:"host_group,omitempty"`
	
	// Timestamps
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	DeletedAt   gorm.DeletedAt         `json:"deleted_at,omitempty" gorm:"index"`
	
	// Relationships
	PortForwards []PortForward         `json:"port_forwards" gorm:"foreignKey:HostID"`
}

// PortGroup represents a group of port forwarding configurations
type PortGroup struct {
	ID          uint                   `json:"id" gorm:"primaryKey"`
	Name        string                 `json:"name" gorm:"uniqueIndex;not null"`
	Description string                 `json:"description"`
	Color       string                 `json:"color" gorm:"default:'#10B981'"` // Tailwind green-500
	Icon        string                 `json:"icon" gorm:"default:'network'"`
	
	// Group Settings
	AutoStart   bool                   `json:"auto_start" gorm:"default:false"`
	Schedule    string                 `json:"schedule"`       // Cron expression for auto-start
	MaxConcurrent int                  `json:"max_concurrent" gorm:"default:10"`
	
	// Metadata
	Tags        string                 `json:"tags"`           // JSON array stored as string
	Metadata    string                 `json:"metadata"`       // JSON object stored as string
	
	// Statistics
	TotalUse    int                    `json:"total_use" gorm:"default:0"`
	LastUsed    *time.Time             `json:"last_used,omitempty"`
	
	// Timestamps
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	DeletedAt   gorm.DeletedAt         `json:"deleted_at,omitempty" gorm:"index"`
	
	// Relationships
	PortForwards []PortForward         `json:"port_forwards" gorm:"foreignKey:PortGroupID"`
}

// PortForward represents a port forwarding configuration
type PortForward struct {
	ID          uint                   `json:"id" gorm:"primaryKey"`
	Name        string                 `json:"name" gorm:"not null"`
	Description string                 `json:"description"`
	
	// Tunnel Configuration
	Type        string                 `json:"type" gorm:"not null"` // local, remote, dynamic
	
	// Local Forwarding (SSH -L)
	LocalBindAddress  string           `json:"local_bind_address" gorm:"default:'127.0.0.1'"`
	LocalPort         int              `json:"local_port"`
	RemoteHost        string           `json:"remote_host"`
	RemotePort        int              `json:"remote_port"`
	
	// Remote Forwarding (SSH -R)
	RemoteBindAddress string           `json:"remote_bind_address" gorm:"default:'127.0.0.1'"`
	
	// Dynamic Forwarding (SOCKS)
	SOCKSBindAddress  string           `json:"socks_bind_address" gorm:"default:'127.0.0.1'"`
	SOCKSPort         int              `json:"socks_port"`
	SOCKSVersion      int              `json:"socks_version" gorm:"default:5"`
	
	// Advanced Options
	AllowRemoteConnections bool        `json:"allow_remote_connections" gorm:"default:false"`
	MaxConnections        int          `json:"max_connections" gorm:"default:100"`
	IdleTimeout           int          `json:"idle_timeout" gorm:"default:300"` // seconds
	
	// Metadata
	Tags        string                 `json:"tags"`           // JSON array stored as string
	Color       string                 `json:"color"`          // Custom color for UI
	Enabled     bool                   `json:"enabled" gorm:"default:true"`
	
	// Statistics
	UseCount    int                    `json:"use_count" gorm:"default:0"`
	LastUsed    *time.Time             `json:"last_used,omitempty"`
	
	// Associations
	HostID      uint                   `json:"host_id" gorm:"not null"`
	Host        Host                   `json:"host"`
	PortGroupID *uint                  `json:"port_group_id"`
	PortGroup   *PortGroup             `json:"port_group,omitempty"`
	
	// Timestamps
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	DeletedAt   gorm.DeletedAt         `json:"deleted_at,omitempty" gorm:"index"`
}

// TunnelSession represents an active tunnel session (extends original Session model)
type TunnelSession struct {
	ID          string                 `json:"id" gorm:"primaryKey"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	
	// Status
	Status      string                 `json:"status"` // created, connecting, connected, active, stopping, stopped, error
	LastError   string                 `json:"last_error"`
	
	// Timing
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	ConnectedAt   *time.Time           `json:"connected_at,omitempty"`
	DisconnectedAt *time.Time          `json:"disconnected_at,omitempty"`
	
	// Associations
	HostID        uint                 `json:"host_id"`
	Host          Host                 `json:"host"`
	PortForwardID uint                 `json:"port_forward_id"`
	PortForward   PortForward          `json:"port_forward"`
	PortGroupID   *uint                `json:"port_group_id"`
	PortGroup     *PortGroup           `json:"port_group,omitempty"`
	
	// Statistics (JSON stored as string)
	Stats         string               `json:"stats"` // JSON serialized SessionStats
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Type     string            `json:"type" yaml:"type"`         // sqlite, postgres, mysql
	Host     string            `json:"host" yaml:"host"`
	Port     int               `json:"port" yaml:"port"`
	Database string            `json:"database" yaml:"database"`
	Username string            `json:"username" yaml:"username"`
	Password string            `json:"password" yaml:"password"`
	SSLMode  string            `json:"ssl_mode" yaml:"ssl_mode"`
	Charset  string            `json:"charset" yaml:"charset"`
	Options  map[string]string `json:"options" yaml:"options"`
}

// DefaultDatabaseConfig returns default database configuration
func DefaultDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Type:     "sqlite",
		Database: "./data/portfly.db",
		Charset:  "utf8mb4",
		Options: map[string]string{
			"parseTime": "true",
			"loc":       "Local",
		},
	}
}

// Validation methods

// TableName returns the table name for HostGroup
func (HostGroup) TableName() string {
	return "host_groups"
}

// TableName returns the table name for Host
func (Host) TableName() string {
	return "hosts"
}

// TableName returns the table name for PortGroup
func (PortGroup) TableName() string {
	return "port_groups"
}

// TableName returns the table name for PortForward
func (PortForward) TableName() string {
	return "port_forwards"
}

// TableName returns the table name for TunnelSession
func (TunnelSession) TableName() string {
	return "tunnel_sessions"
}

// GetTunnelDescription returns a human-readable description of the port forward
func (pf *PortForward) GetTunnelDescription() string {
	switch pf.Type {
	case "local":
		return fmt.Sprintf("Local %s:%d -> %s:%d", 
			pf.LocalBindAddress, pf.LocalPort, pf.RemoteHost, pf.RemotePort)
	case "remote":
		return fmt.Sprintf("Remote %s:%d -> %s:%d", 
			pf.RemoteBindAddress, pf.LocalPort, pf.RemoteHost, pf.RemotePort)
	case "dynamic":
		return fmt.Sprintf("SOCKS%d proxy on %s:%d", 
			pf.SOCKSVersion, pf.SOCKSBindAddress, pf.SOCKSPort)
	default:
		return "Unknown tunnel type"
	}
}

// ToSSHConnectionConfig converts Host to SSHConnectionConfig
func (h *Host) ToSSHConnectionConfig() SSHConnectionConfig {
	return SSHConnectionConfig{
		Host:             h.Hostname,
		Port:             h.Port,
		Username:         h.Username,
		AuthMethod:       AuthMethod(h.AuthMethod),
		Password:         h.Password,
		PrivateKeyPath:   h.PrivateKey,
		HostKeyCallback:  h.HostKeyCallback,
		KnownHostsFile:   h.KnownHostsFile,
		ConnectTimeout:   time.Duration(h.ConnectTimeout) * time.Second,
		KeepAliveTimeout: time.Duration(h.KeepAlive) * time.Second,
		MaxRetries:       h.MaxRetries,
		RetryInterval:    time.Duration(h.RetryInterval) * time.Second,
	}
}

// ToTunnelConfig converts PortForward to TunnelConfig
func (pf *PortForward) ToTunnelConfig() TunnelConfig {
	config := TunnelConfig{
		Type:                   TunnelType(pf.Type),
		AllowRemoteConnections: pf.AllowRemoteConnections,
		MaxConnections:         pf.MaxConnections,
		IdleTimeout:           time.Duration(pf.IdleTimeout) * time.Second,
	}
	
	switch pf.Type {
	case "local":
		config.LocalBindAddress = pf.LocalBindAddress
		config.LocalPort = pf.LocalPort
		config.RemoteHost = pf.RemoteHost
		config.RemotePort = pf.RemotePort
	case "remote":
		config.RemoteBindAddress = pf.RemoteBindAddress
		config.LocalPort = pf.LocalPort
		config.RemoteHost = pf.RemoteHost
		config.RemotePort = pf.RemotePort
	case "dynamic":
		config.SOCKSBindAddress = pf.SOCKSBindAddress
		config.SOCKSPort = pf.SOCKSPort
		config.SOCKSVersion = pf.SOCKSVersion
	}
	
	return config
}
