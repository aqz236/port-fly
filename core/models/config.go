package models

import (
	"time"
)

// Config represents the main configuration structure
type Config struct {
	// Server configuration
	Server ServerConfig `json:"server" yaml:"server"`
	
	// SSH configuration
	SSH SSHConfig `json:"ssh" yaml:"ssh"`
	
	// Logging configuration
	Logging LoggingConfig `json:"logging" yaml:"logging"`
	
	// Storage configuration
	Storage StorageConfig `json:"storage" yaml:"storage"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Host         string        `json:"host" yaml:"host"`
	Port         int           `json:"port" yaml:"port"`
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout" yaml:"idle_timeout"`
	TLS          TLSConfig     `json:"tls" yaml:"tls"`
	CORS         CORSConfig    `json:"cors" yaml:"cors"`
}

// TLSConfig contains TLS/SSL configuration
type TLSConfig struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	CertFile string `json:"cert_file" yaml:"cert_file"`
	KeyFile  string `json:"key_file" yaml:"key_file"`
}

// CORSConfig contains CORS configuration
type CORSConfig struct {
	AllowedOrigins []string `json:"allowed_origins" yaml:"allowed_origins"`
	AllowedMethods []string `json:"allowed_methods" yaml:"allowed_methods"`
	AllowedHeaders []string `json:"allowed_headers" yaml:"allowed_headers"`
}

// SSHConfig contains SSH client configuration
type SSHConfig struct {
	// Connection settings
	ConnectTimeout   time.Duration `json:"connect_timeout" yaml:"connect_timeout"`
	KeepAliveTimeout time.Duration `json:"keepalive_timeout" yaml:"keepalive_timeout"`
	MaxRetries       int           `json:"max_retries" yaml:"max_retries"`
	RetryInterval    time.Duration `json:"retry_interval" yaml:"retry_interval"`
	
	// Connection pool settings
	MaxConnections    int           `json:"max_connections" yaml:"max_connections"`
	ConnectionTimeout time.Duration `json:"connection_timeout" yaml:"connection_timeout"`
	IdleTimeout       time.Duration `json:"idle_timeout" yaml:"idle_timeout"`
	
	// Security settings
	HostKeyCallback string   `json:"host_key_callback" yaml:"host_key_callback"`
	Ciphers         []string `json:"ciphers" yaml:"ciphers"`
	MACs            []string `json:"macs" yaml:"macs"`
	KeyExchanges    []string `json:"key_exchanges" yaml:"key_exchanges"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level      string `json:"level" yaml:"level"`
	Format     string `json:"format" yaml:"format"`
	Output     string `json:"output" yaml:"output"`
	MaxSize    int    `json:"max_size" yaml:"max_size"`     // megabytes
	MaxBackups int    `json:"max_backups" yaml:"max_backups"`
	MaxAge     int    `json:"max_age" yaml:"max_age"`       // days
	Compress   bool   `json:"compress" yaml:"compress"`
}

// StorageConfig contains storage configuration
type StorageConfig struct {
	Type     string            `json:"type" yaml:"type"` // "boltdb", "memory", etc.
	Path     string            `json:"path" yaml:"path"`
	Options  map[string]string `json:"options" yaml:"options"`
	Backup   BackupConfig      `json:"backup" yaml:"backup"`
}

// BackupConfig contains backup configuration
type BackupConfig struct {
	Enabled  bool          `json:"enabled" yaml:"enabled"`
	Interval time.Duration `json:"interval" yaml:"interval"`
	Path     string        `json:"path" yaml:"path"`
	MaxFiles int           `json:"max_files" yaml:"max_files"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
			TLS: TLSConfig{
				Enabled: false,
			},
			CORS: CORSConfig{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders: []string{"*"},
			},
		},
		SSH: SSHConfig{
			ConnectTimeout:    30 * time.Second,
			KeepAliveTimeout:  30 * time.Second,
			MaxRetries:        3,
			RetryInterval:     5 * time.Second,
			MaxConnections:    10,
			ConnectionTimeout: 60 * time.Second,
			IdleTimeout:       300 * time.Second,
			HostKeyCallback:   "ask", // ask, accept, strict
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   true,
		},
		Storage: StorageConfig{
			Type: "boltdb",
			Path: "./data/portfly.db",
			Backup: BackupConfig{
				Enabled:  true,
				Interval: 24 * time.Hour,
				Path:     "./data/backups",
				MaxFiles: 7,
			},
		},
	}
}
