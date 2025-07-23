package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Project 项目/工作空间 - 支持树状结构的容器
type Project struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name        string `gorm:"not null;size:100" json:"name"`
	Description string `gorm:"size:500" json:"description"`
	Color       string `gorm:"size:20;default:#6366f1" json:"color"`
	Icon        string `gorm:"size:50;default:folder" json:"icon"`
	IsDefault   bool   `gorm:"default:false" json:"is_default"`
	Metadata    string `gorm:"type:text" json:"metadata,omitempty"` // JSON string

	// 树状结构支持
	ParentID *uint  `gorm:"index" json:"parent_id,omitempty"` // 父项目ID，为空表示根项目
	Level    int    `gorm:"default:0" json:"level"`           // 层级深度，0为根项目
	Path     string `gorm:"size:500" json:"path,omitempty"`   // 层级路径，如 "/1/2/3"
	Sort     int    `gorm:"default:0" json:"sort"`            // 同级排序

	// 关联关系
	Parent   *Project  `gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE" json:"parent,omitempty"`
	Children []Project `gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE" json:"children,omitempty"`
	Groups   []Group   `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE" json:"groups,omitempty"`
}

// 统计信息结构

type ProjectStats struct {
	TotalGroups   int        `json:"total_groups"`
	TotalHosts    int        `json:"total_hosts"`
	TotalPorts    int        `json:"total_ports"`
	ActiveTunnels int        `json:"active_tunnels"`
	LastUsed      *time.Time `json:"last_used,omitempty"`
}

// 项目树节点，用于前端展示
type ProjectTreeNode struct {
	*Project
	Children    []*ProjectTreeNode `json:"children,omitempty"`
	HasChildren bool               `json:"has_children"`
	IsExpanded  bool               `json:"is_expanded,omitempty"`
}

// 项目移动参数
type MoveProjectParams struct {
	ProjectID uint  `json:"project_id"`
	ParentID  *uint `json:"parent_id,omitempty"` // null表示移动到根级别
	Position  int   `json:"position"`            // 在目标位置的排序
}

// 辅助方法

// IsRoot 判断是否为根项目
func (p *Project) IsRoot() bool {
	return p.ParentID == nil
}

// HasChildren 判断是否有子项目
func (p *Project) HasChildren() bool {
	return len(p.Children) > 0
}

// GetFullPath 获取完整路径名称
func (p *Project) GetFullPath() string {
	if p.Parent == nil {
		return p.Name
	}
	return p.Parent.GetFullPath() + " / " + p.Name
}

// BuildPath 构建数字路径
func (p *Project) BuildPath() string {
	if p.ParentID == nil {
		return fmt.Sprintf("/%d", p.ID)
	}
	if p.Parent != nil {
		return p.Parent.BuildPath() + fmt.Sprintf("/%d", p.ID)
	}
	return fmt.Sprintf("/%d", p.ID)
}
