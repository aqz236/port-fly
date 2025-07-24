package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Port 端口配置 - 基础端口模型
type Port struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name        string `gorm:"not null;size:100" json:"name"`
	Type        string `gorm:"not null;size:20" json:"type"` // local, remote
	Port        int    `gorm:"not null" json:"port"`         // 端口号
	TargetHost  string `gorm:"size:255" json:"target_host"`  // 目标主机 (local: localhost, remote: 0.0.0.0)
	TargetPort  int    `json:"target_port"`                  // 目标端口
	Description string `gorm:"size:500" json:"description"`

	// 配置选项
	AutoStart bool   `gorm:"default:false" json:"auto_start"`
	Status    string `gorm:"size:20;default:inactive" json:"status"` // active, inactive, error, connecting

	// 样式配置
	Color     string `gorm:"size:20;default:#3B82F6" json:"color"` // 节点颜色
	Icon      string `gorm:"size:50;default:network" json:"icon"`  // 图标名称
	IsVisible bool   `gorm:"default:true" json:"is_visible"`       // 是否可见

	// 元数据
	Tags     []string `gorm:"type:text;serializer:json" json:"tags,omitempty"`
	Metadata string   `gorm:"type:text" json:"metadata,omitempty"` // JSON string

	// 外键
	GroupID uint  `gorm:"not null;index" json:"group_id"`
	Group   Group `gorm:"constraint:OnDelete:CASCADE" json:"group,omitempty"`
	HostID  uint  `gorm:"not null;index" json:"host_id"`
	Host    Host  `gorm:"constraint:OnDelete:CASCADE" json:"host,omitempty"`

	// 关联关系
	TunnelSessions []TunnelSession `gorm:"foreignKey:PortID;constraint:OnDelete:CASCADE" json:"tunnel_sessions,omitempty"`
}

// PortStats 端口统计信息
type PortStats struct {
	TotalSessions        int        `json:"total_sessions"`
	ActiveSessions       int        `json:"active_sessions"`
	TotalDataTransferred int64      `json:"total_data_transferred"`
	LastUsed             *time.Time `json:"last_used,omitempty"`
	Uptime               int64      `json:"uptime"` // 运行时间（秒）
}

// CreatePortRequest 创建端口请求
type CreatePortRequest struct {
	Name        string   `json:"name" binding:"required,min=1,max=100"`
	Type        string   `json:"type" binding:"required,oneof=local remote"`
	Port        int      `json:"port" binding:"required,min=1,max=65535"`
	TargetHost  string   `json:"target_host,omitempty"`
	TargetPort  int      `json:"target_port,omitempty"`
	Description string   `json:"description,omitempty"`
	AutoStart   bool     `json:"auto_start,omitempty"`
	Color       string   `json:"color,omitempty"`
	Icon        string   `json:"icon,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	GroupID     uint     `json:"group_id" binding:"required"`
	HostID      uint     `json:"host_id" binding:"required"`
}

// UpdatePortRequest 更新端口请求
type UpdatePortRequest struct {
	Name        string   `json:"name,omitempty"`
	Type        string   `json:"type,omitempty"`
	Port        int      `json:"port,omitempty"`
	TargetHost  string   `json:"target_host,omitempty"`
	TargetPort  int      `json:"target_port,omitempty"`
	Description string   `json:"description,omitempty"`
	AutoStart   bool     `json:"auto_start,omitempty"`
	Color       string   `json:"color,omitempty"`
	Icon        string   `json:"icon,omitempty"`
	IsVisible   bool     `json:"is_visible,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// PortControlRequest 端口控制请求
type PortControlRequest struct {
	Action string `json:"action" binding:"required,oneof=start stop restart"`
}

// PortResponse 端口响应
type PortResponse struct {
	Port
	Stats *PortStats `json:"stats,omitempty"`
}

// Validate 验证端口配置
func (p *Port) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("name is required")
	}
	if p.Type != "local" && p.Type != "remote" {
		return fmt.Errorf("type must be 'local' or 'remote'")
	}
	if p.Port <= 0 || p.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if p.TargetPort <= 0 || p.TargetPort > 65535 {
		return fmt.Errorf("target_port must be between 1 and 65535")
	}
	if p.TargetHost == "" {
		return fmt.Errorf("target_host is required")
	}
	return nil
}

// GetStats 获取端口统计信息
func (p *Port) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"id":          p.ID,
		"name":        p.Name,
		"type":        p.Type,
		"port":        p.Port,
		"target_host": p.TargetHost,
		"target_port": p.TargetPort,
		"status":      p.Status,
		"auto_start":  p.AutoStart,
		"is_visible":  p.IsVisible,
		"created_at":  p.CreatedAt,
		"updated_at":  p.UpdatedAt,
	}

	// 添加运行时统计（这里是模拟数据）
	if p.Status == "active" {
		stats["connections"] = 0
		stats["bytes_sent"] = 0
		stats["bytes_received"] = 0
		stats["uptime"] = 0
	}

	return stats
}
