package models

import (
	"time"

	"gorm.io/gorm"
)

// Group Reactflow画布 - 可以包含主机和端口 node
type Group struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name        string   `gorm:"not null;size:100" json:"name"`
	Description string   `gorm:"size:500" json:"description"`
	Color       string   `gorm:"size:20;default:#10b981" json:"color"`
	Icon        string   `gorm:"size:50;default:folder" json:"icon"`
	Tags        []string `gorm:"type:text;serializer:json" json:"tags,omitempty"`
	Metadata    string   `gorm:"type:text" json:"metadata,omitempty"` // JSON string

	// 外键
	ProjectID uint    `gorm:"not null;index" json:"project_id"`
	Project   Project `gorm:"constraint:OnDelete:CASCADE" json:"project,omitempty"`

	// 关联关系
	Hosts        []Host        `gorm:"foreignKey:GroupID;constraint:OnDelete:CASCADE" json:"hosts,omitempty"`
	PortForwards []PortForward `gorm:"foreignKey:GroupID;constraint:OnDelete:CASCADE" json:"port_forwards,omitempty"`
}

type GroupStats struct {
	TotalHosts     int        `json:"total_hosts"`
	TotalPorts     int        `json:"total_ports"`
	ConnectedHosts int        `json:"connected_hosts"`
	ActiveTunnels  int        `json:"active_tunnels"`
	LastUsed       *time.Time `json:"last_used,omitempty"`
}
