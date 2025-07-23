package models

import (
	"time"

	"gorm.io/gorm"
)

// PortForward 端口转发配置 - 完整版本
type PortForward struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

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
	Tags     []string `gorm:"type:text;serializer:json" json:"tags,omitempty"`
	Metadata string   `gorm:"type:text" json:"metadata,omitempty"` // JSON string

	// 外键
	GroupID uint  `gorm:"not null;index" json:"group_id"`
	Group   Group `gorm:"constraint:OnDelete:CASCADE" json:"group,omitempty"`
	HostID  uint  `gorm:"not null;index" json:"host_id"`
	Host    Host  `gorm:"constraint:OnDelete:CASCADE" json:"host,omitempty"`

	// 关联关系
	TunnelSessions []TunnelSession `gorm:"foreignKey:PortForwardID;constraint:OnDelete:CASCADE" json:"tunnel_sessions,omitempty"`
}

type PortForwardStats struct {
	TotalSessions        int        `json:"total_sessions"`
	ActiveSessions       int        `json:"active_sessions"`
	TotalDataTransferred int64      `json:"total_data_transferred"`
	LastUsed             *time.Time `json:"last_used,omitempty"`
}
