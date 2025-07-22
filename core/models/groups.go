package models

import (
	"time"
	"gorm.io/gorm"
)

// Host 主机配置 - 完整版本
type Host struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	
	Name        string `gorm:"not null;size:100" json:"name"`
	Hostname    string `gorm:"not null;size:255" json:"hostname"`
	Port        int    `gorm:"default:22" json:"port"`
	Username    string `gorm:"not null;size:100" json:"username"`
	Description string `gorm:"size:500" json:"description"`
	
	// 认证相关
	AuthMethod string `gorm:"not null;size:20;default:password" json:"auth_method"` // password, key, agent
	PrivateKey string `gorm:"type:text" json:"private_key,omitempty"`
	Password   string `gorm:"type:text" json:"password,omitempty"` // 加密存储
	
	// 状态信息
	Status           string     `gorm:"size:20;default:unknown" json:"status"` // connected, disconnected, connecting, error, unknown
	LastConnected    *time.Time `json:"last_connected,omitempty"`
	ConnectionCount  int        `gorm:"default:0" json:"connection_count"`
	
	// 元数据
	Tags     string `gorm:"type:text" json:"tags,omitempty"` // JSON array string
	Metadata string `gorm:"type:text" json:"metadata,omitempty"` // JSON string
	
	// 外键
	GroupID uint  `gorm:"not null;index" json:"group_id"`
	Group   Group `gorm:"constraint:OnDelete:CASCADE" json:"group,omitempty"`
	
	// 关联关系
	PortForwards   []PortForward   `gorm:"foreignKey:HostID;constraint:OnDelete:CASCADE" json:"port_forwards,omitempty"`
	TunnelSessions []TunnelSession `gorm:"foreignKey:HostID;constraint:OnDelete:CASCADE" json:"tunnel_sessions,omitempty"`
}

// PortForward 端口转发配置 - 完整版本
type PortForward struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	
	Name        string `gorm:"not null;size:100" json:"name"`
	Type        string `gorm:"not null;size:20" json:"type"` // local (-L), remote (-R), dynamic (-D)
	LocalPort   int    `gorm:"not null" json:"local_port"`
	RemoteHost  string `gorm:"not null;size:255" json:"remote_host"`
	RemotePort  int    `gorm:"not null" json:"remote_port"`
	Description string `gorm:"size:500" json:"description"`
	
	// 配置选项
	AutoStart bool   `gorm:"default:false" json:"auto_start"`
	Status    string `gorm:"size:20;default:inactive" json:"status"` // active, inactive, error
	
	// 元数据
	Tags     string `gorm:"type:text" json:"tags,omitempty"` // JSON array string
	Metadata string `gorm:"type:text" json:"metadata,omitempty"` // JSON string
	
	// 外键
	GroupID uint  `gorm:"not null;index" json:"group_id"`
	Group   Group `gorm:"constraint:OnDelete:CASCADE" json:"group,omitempty"`
	HostID  uint  `gorm:"not null;index" json:"host_id"`
	Host    Host  `gorm:"constraint:OnDelete:CASCADE" json:"host,omitempty"`
	
	// 关联关系
	TunnelSessions []TunnelSession `gorm:"foreignKey:PortForwardID;constraint:OnDelete:CASCADE" json:"tunnel_sessions,omitempty"`
}
