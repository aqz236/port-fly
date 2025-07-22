package models

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

// SessionStatus represents the status of a session
type SessionStatus string

const (
	StatusCreated      SessionStatus = "created"
	StatusConnecting   SessionStatus = "connecting"
	StatusConnected    SessionStatus = "connected"
	StatusActive       SessionStatus = "active"
	StatusStopping     SessionStatus = "stopping"
	StatusStopped      SessionStatus = "stopped"
	StatusError        SessionStatus = "error"
	StatusDisconnected SessionStatus = "disconnected"
)

// TunnelType represents the type of tunnel
type TunnelType string

const (
	TunnelTypeLocal   TunnelType = "local"   // Local port forwarding
	TunnelTypeRemote  TunnelType = "remote"  // Remote port forwarding
	TunnelTypeDynamic TunnelType = "dynamic" // Dynamic port forwarding (SOCKS)
)

// AuthMethod represents SSH authentication method
type AuthMethod string

const (
	AuthMethodPassword    AuthMethod = "password"
	AuthMethodPrivateKey  AuthMethod = "private_key"
	AuthMethodAgent       AuthMethod = "agent"
	AuthMethodInteractive AuthMethod = "interactive"
)

// Session represents a tunnel session
type Session struct {
	// Basic information
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`

	// Status and state
	Status         SessionStatus `json:"status" db:"status"`
	LastError      string        `json:"last_error,omitempty" db:"last_error"`
	ConnectedAt    *time.Time    `json:"connected_at,omitempty" db:"connected_at"`
	DisconnectedAt *time.Time    `json:"disconnected_at,omitempty" db:"disconnected_at"`

	// SSH connection details
	SSHConfig SSHConnectionConfig `json:"ssh_config" db:"ssh_config"`

	// Tunnel configuration
	TunnelConfig TunnelConfig `json:"tunnel_config" db:"tunnel_config"`

	// Statistics
	Stats SessionStats `json:"stats" db:"stats"`

	// Runtime information (not persisted)
	SSHClient   *ssh.Client    `json:"-" db:"-"`
	Listeners   []net.Listener `json:"-" db:"-"`
	Connections []net.Conn     `json:"-" db:"-"`
	StopChan    chan bool      `json:"-" db:"-"`
}

// SSHConnectionConfig contains SSH connection configuration
type SSHConnectionConfig struct {
	// Connection details
	Host     string `json:"host" db:"host"`
	Port     int    `json:"port" db:"port"`
	Username string `json:"username" db:"username"`

	// Authentication
	AuthMethod     AuthMethod `json:"auth_method" db:"auth_method"`
	Password       string     `json:"password,omitempty" db:"password"`
	PrivateKeyPath string     `json:"private_key_path,omitempty" db:"private_key_path"`
	PrivateKeyData []byte     `json:"private_key_data,omitempty" db:"private_key_data"`
	Passphrase     string     `json:"passphrase,omitempty" db:"passphrase"`

	// Advanced options
	HostKeyCallback string            `json:"host_key_callback" db:"host_key_callback"`
	KnownHostsFile  string            `json:"known_hosts_file,omitempty" db:"known_hosts_file"`
	ClientVersion   string            `json:"client_version,omitempty" db:"client_version"`
	Extensions      map[string]string `json:"extensions,omitempty" db:"extensions"`

	// Connection settings
	ConnectTimeout   time.Duration `json:"connect_timeout" db:"connect_timeout"`
	KeepAliveTimeout time.Duration `json:"keepalive_timeout" db:"keepalive_timeout"`
	MaxRetries       int           `json:"max_retries" db:"max_retries"`
	RetryInterval    time.Duration `json:"retry_interval" db:"retry_interval"`
}

// TunnelConfig contains tunnel configuration
type TunnelConfig struct {
	Type TunnelType `json:"type" db:"type"`

	// Local forwarding: -L [bind_address:]port:host:hostport
	// Remote forwarding: -R [bind_address:]port:host:hostport
	LocalBindAddress string `json:"local_bind_address,omitempty" db:"local_bind_address"`
	LocalPort        int    `json:"local_port,omitempty" db:"local_port"`
	RemoteHost       string `json:"remote_host,omitempty" db:"remote_host"`
	RemotePort       int    `json:"remote_port,omitempty" db:"remote_port"`

	// Remote forwarding specific
	RemoteBindAddress string `json:"remote_bind_address,omitempty" db:"remote_bind_address"`

	// Dynamic forwarding (SOCKS proxy)
	SOCKSBindAddress string `json:"socks_bind_address,omitempty" db:"socks_bind_address"`
	SOCKSPort        int    `json:"socks_port,omitempty" db:"socks_port"`
	SOCKSVersion     int    `json:"socks_version,omitempty" db:"socks_version"` // 4 or 5

	// Advanced options
	AllowRemoteConnections bool          `json:"allow_remote_connections" db:"allow_remote_connections"`
	MaxConnections         int           `json:"max_connections" db:"max_connections"`
	IdleTimeout            time.Duration `json:"idle_timeout" db:"idle_timeout"`
}

// SessionStats contains session statistics
type SessionStats struct {
	// Connection statistics
	TotalConnections  int64 `json:"total_connections" db:"total_connections"`
	ActiveConnections int64 `json:"active_connections" db:"active_connections"`
	FailedConnections int64 `json:"failed_connections" db:"failed_connections"`

	// Data transfer statistics
	BytesSent     int64 `json:"bytes_sent" db:"bytes_sent"`
	BytesReceived int64 `json:"bytes_received" db:"bytes_received"`

	// Time statistics
	TotalUptime    time.Duration `json:"total_uptime" db:"total_uptime"`
	LastActivityAt *time.Time    `json:"last_activity_at,omitempty" db:"last_activity_at"`

	// Error statistics
	ReconnectCount  int64      `json:"reconnect_count" db:"reconnect_count"`
	LastReconnectAt *time.Time `json:"last_reconnect_at,omitempty" db:"last_reconnect_at"`
}

// TunnelSession represents a database model for tunnel sessions
type TunnelSession struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	Status           SessionStatus  `json:"status" gorm:"not null;default:'pending'"`
	StartTime        *time.Time     `json:"start_time,omitempty"`
	EndTime          *time.Time     `json:"end_time,omitempty"`
	ErrorMessage     string         `json:"error_message,omitempty" gorm:"type:text"`
	DataTransferred  int64          `json:"data_transferred" gorm:"default:0"`
	PID              *int           `json:"pid,omitempty"`
	LocalAddress     string         `json:"local_address,omitempty" gorm:"size:255"`
	RemoteAddress    string         `json:"remote_address,omitempty" gorm:"size:255"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        *time.Time     `json:"deleted_at,omitempty" gorm:"index"`
	
	// Foreign keys - will be updated when we integrate with new models
	HostID           uint           `json:"host_id" gorm:"not null;index"`
	PortForwardID    uint           `json:"port_forward_id" gorm:"not null;index"`
}

// Rule represents a forwarding rule template
type Rule struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`

	// Configuration template
	SSHConfig    SSHConnectionConfig `json:"ssh_config" db:"ssh_config"`
	TunnelConfig TunnelConfig        `json:"tunnel_config" db:"tunnel_config"`

	// Rule metadata
	Tags     []string          `json:"tags" db:"tags"`
	Metadata map[string]string `json:"metadata" db:"metadata"`

	// Auto-start configuration
	AutoStart bool   `json:"auto_start" db:"auto_start"`
	Schedule  string `json:"schedule,omitempty" db:"schedule"` // Cron expression
}

// IsActive returns true if the session is in an active state
func (s *Session) IsActive() bool {
	return s.Status == StatusConnected || s.Status == StatusActive
}

// IsConnected returns true if the session has an active SSH connection
func (s *Session) IsConnected() bool {
	return s.SSHClient != nil && s.Status != StatusStopped && s.Status != StatusError
}

// GetTunnelDescription returns a human-readable description of the tunnel
func (tc *TunnelConfig) GetTunnelDescription() string {
	switch tc.Type {
	case TunnelTypeLocal:
		return fmt.Sprintf("Local %s:%d -> %s:%d",
			tc.LocalBindAddress, tc.LocalPort, tc.RemoteHost, tc.RemotePort)
	case TunnelTypeRemote:
		return fmt.Sprintf("Remote %s:%d -> %s:%d",
			tc.RemoteBindAddress, tc.LocalPort, tc.RemoteHost, tc.RemotePort)
	case TunnelTypeDynamic:
		return fmt.Sprintf("SOCKS%d proxy on %s:%d",
			tc.SOCKSVersion, tc.SOCKSBindAddress, tc.SOCKSPort)
	default:
		return "Unknown tunnel type"
	}
}

// Validate validates the tunnel configuration
func (tc *TunnelConfig) Validate() error {
	switch tc.Type {
	case TunnelTypeLocal:
		if tc.LocalPort <= 0 || tc.LocalPort > 65535 {
			return fmt.Errorf("invalid local port: %d", tc.LocalPort)
		}
		if tc.RemoteHost == "" {
			return fmt.Errorf("remote host is required for local forwarding")
		}
		if tc.RemotePort <= 0 || tc.RemotePort > 65535 {
			return fmt.Errorf("invalid remote port: %d", tc.RemotePort)
		}
	case TunnelTypeRemote:
		if tc.LocalPort <= 0 || tc.LocalPort > 65535 {
			return fmt.Errorf("invalid local port: %d", tc.LocalPort)
		}
		if tc.RemoteHost == "" {
			return fmt.Errorf("remote host is required for remote forwarding")
		}
		if tc.RemotePort <= 0 || tc.RemotePort > 65535 {
			return fmt.Errorf("invalid remote port: %d", tc.RemotePort)
		}
	case TunnelTypeDynamic:
		if tc.SOCKSPort <= 0 || tc.SOCKSPort > 65535 {
			return fmt.Errorf("invalid SOCKS port: %d", tc.SOCKSPort)
		}
		if tc.SOCKSVersion != 4 && tc.SOCKSVersion != 5 {
			return fmt.Errorf("invalid SOCKS version: %d", tc.SOCKSVersion)
		}
	default:
		return fmt.Errorf("unknown tunnel type: %s", tc.Type)
	}
	return nil
}
