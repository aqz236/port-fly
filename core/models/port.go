package models

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Port validation errors
var (
	ErrInvalidName     = errors.New("port name cannot be empty")
	ErrInvalidPort     = errors.New("port number must be between 1 and 65535")
	ErrInvalidPortType = errors.New("invalid port type")
	ErrGroupRequired   = errors.New("group ID is required")
)

// PortType 端口类型
type PortType string

const (
	PortTypeRemote PortType = "remote_port" // 远程端口（源端口）
	PortTypeLocal  PortType = "local_port"  // 本地端口（目标端口）
)

// PortStatus 端口状态
type PortStatus string

const (
	PortStatusAvailable   PortStatus = "available"   // 可用
	PortStatusUnavailable PortStatus = "unavailable" // 不可用
	PortStatusActive      PortStatus = "active"      // 活跃（正在转发）
	PortStatusError       PortStatus = "error"       // 错误
	PortStatusConnecting  PortStatus = "connecting"  // 连接中
)

// Port 端口配置 - 独立模型
type Port struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 基本信息
	Name        string     `gorm:"not null;size:100" json:"name"`
	Type        PortType   `gorm:"not null;size:20" json:"type"`
	Port        int        `gorm:"not null" json:"port"`
	BindAddress string     `gorm:"size:255;default:127.0.0.1" json:"bind_address"`
	Description string     `gorm:"size:500" json:"description"`

	// 状态信息
	Status         PortStatus `gorm:"size:20;default:unavailable" json:"status"`
	LastTested     *time.Time `json:"last_tested,omitempty"`
	LastActive     *time.Time `json:"last_active,omitempty"`
	ConnectionTest bool       `gorm:"default:false" json:"connection_test"` // Host连线测试结果

	// 显示配置
	Color     string `gorm:"size:20;default:#3b82f6" json:"color"`
	Icon      string `gorm:"size:50;default:port" json:"icon"`
	IsVisible bool   `gorm:"default:true" json:"is_visible"`

	// 配置选项
	AutoStart bool `gorm:"default:false" json:"auto_start"`

	// 元数据
	Tags     []string `gorm:"type:text;serializer:json" json:"tags,omitempty"`
	Metadata string   `gorm:"type:text" json:"metadata,omitempty"` // JSON string

	// 外键关联
	GroupID uint  `gorm:"not null;index" json:"group_id"`
	Group   Group `gorm:"constraint:OnDelete:CASCADE" json:"group,omitempty"`

	// 可选关联Host（用于连通性测试）
	HostID *uint `gorm:"index" json:"host_id,omitempty"`
	Host   *Host `gorm:"constraint:OnDelete:SET NULL" json:"host,omitempty"`

	// 端口转发关联（Remote_Port -> Local_Port）
	// 如果当前是Remote_Port，则指向目标Local_Port
	TargetPortID *uint `gorm:"index" json:"target_port_id,omitempty"`
	TargetPort   *Port `gorm:"foreignKey:TargetPortID;constraint:OnDelete:SET NULL" json:"target_port,omitempty"`

	// 如果当前是Local_Port，则可以被多个Remote_Port指向
	SourcePorts []Port `gorm:"foreignKey:TargetPortID" json:"source_ports,omitempty"`

	// 关联的隧道会话
	TunnelSessions []TunnelSession `gorm:"foreignKey:PortID;constraint:OnDelete:CASCADE" json:"tunnel_sessions,omitempty"`
}

// PortStats 端口统计信息
type PortStats struct {
	TotalConnections     int        `json:"total_connections"`
	ActiveConnections    int        `json:"active_connections"`
	TotalDataTransferred int64      `json:"total_data_transferred"`
	BytesSent            int64      `json:"bytes_sent"`
	BytesReceived        int64      `json:"bytes_received"`
	LastUsed             *time.Time `json:"last_used,omitempty"`
	UptimePercentage     float64    `json:"uptime_percentage"`
	SuccessRate          float64    `json:"success_rate"`
}

// PortConnection 端口连接信息（用于Remote_Port -> Local_Port转发）
type PortConnection struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 端口关联
	RemotePortID uint `gorm:"not null;index" json:"remote_port_id"`
	RemotePort   Port `gorm:"foreignKey:RemotePortID;constraint:OnDelete:CASCADE" json:"remote_port"`

	LocalPortID uint `gorm:"not null;index" json:"local_port_id"`
	LocalPort   Port `gorm:"foreignKey:LocalPortID;constraint:OnDelete:CASCADE" json:"local_port"`

	// 转发状态
	Status    PortStatus `gorm:"size:20;default:inactive" json:"status"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	StoppedAt *time.Time `json:"stopped_at,omitempty"`

	// 关联的隧道会话
	TunnelSessionID *uint            `gorm:"index" json:"tunnel_session_id,omitempty"`
	TunnelSession   *TunnelSession   `gorm:"constraint:OnDelete:SET NULL" json:"tunnel_session,omitempty"`

	// 统计信息
	Stats PortStats `gorm:"embedded;embeddedPrefix:stats_" json:"stats"`
}

// Validate 验证端口配置
func (p *Port) Validate() error {
	if p.Name == "" {
		return ErrInvalidName
	}

	if p.Port <= 0 || p.Port > 65535 {
		return ErrInvalidPort
	}

	if p.Type != PortTypeRemote && p.Type != PortTypeLocal {
		return ErrInvalidPortType
	}

	if p.GroupID == 0 {
		return ErrGroupRequired
	}

	return nil
}

// IsRemotePort 检查是否为远程端口
func (p *Port) IsRemotePort() bool {
	return p.Type == PortTypeRemote
}

// IsLocalPort 检查是否为本地端口
func (p *Port) IsLocalPort() bool {
	return p.Type == PortTypeLocal
}

// IsAvailable 检查端口是否可用
func (p *Port) IsAvailable() bool {
	return p.Status == PortStatusAvailable
}

// IsActive 检查端口是否活跃
func (p *Port) IsActive() bool {
	return p.Status == PortStatusActive
}

// GetDisplayName 获取显示名称
func (p *Port) GetDisplayName() string {
	if p.Name != "" {
		return p.Name
	}
	return fmt.Sprintf("%s:%d", p.Type, p.Port)
}

// GetBindAddress 获取绑定地址
func (p *Port) GetBindAddress() string {
	if p.BindAddress != "" {
		return p.BindAddress
	}
	return "127.0.0.1"
}

// GetFullAddress 获取完整地址
func (p *Port) GetFullAddress() string {
	return fmt.Sprintf("%s:%d", p.GetBindAddress(), p.Port)
}

// UpdateStatus 更新端口状态
func (p *Port) UpdateStatus(status PortStatus) {
	p.Status = status
	now := time.Now()
	
	switch status {
	case PortStatusActive:
		p.LastActive = &now
	case PortStatusAvailable, PortStatusUnavailable:
		p.LastTested = &now
	}
}

// SetConnectionTestResult 设置连接测试结果
func (p *Port) SetConnectionTestResult(success bool) {
	p.ConnectionTest = success
	now := time.Now()
	p.LastTested = &now
	
	if success {
		p.Status = PortStatusAvailable
	} else {
		p.Status = PortStatusUnavailable
	}
}
