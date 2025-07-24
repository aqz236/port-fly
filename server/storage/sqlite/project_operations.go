package sqlite

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/aqz236/port-fly/core/models"
)

// ===== Project Operations =====

func (s *SQLiteStorage) CreateProject(ctx context.Context, project *models.Project) error {
	// 如果设置了父项目，需要计算层级和路径
	if project.ParentID != nil {
		var parent models.Project
		if err := s.db.WithContext(ctx).First(&parent, *project.ParentID).Error; err != nil {
			return fmt.Errorf("parent project not found: %w", err)
		}
		project.Level = parent.Level + 1
	}

	// 创建项目
	if err := s.db.WithContext(ctx).Create(project).Error; err != nil {
		return err
	}

	// 更新路径
	project.Path = project.BuildPath()
	return s.db.WithContext(ctx).Save(project).Error
}

func (s *SQLiteStorage) GetProject(ctx context.Context, id uint) (*models.Project, error) {
	var project models.Project
	err := s.db.WithContext(ctx).Preload("Groups").Preload("Parent").Preload("Children").First(&project, id).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (s *SQLiteStorage) GetProjects(ctx context.Context) ([]models.Project, error) {
	var projects []models.Project
	err := s.db.WithContext(ctx).Preload("Groups").Preload("Parent").Preload("Children").Find(&projects).Error
	return projects, err
}

func (s *SQLiteStorage) GetProjectsByParent(ctx context.Context, parentID *uint, includeChildren bool) ([]models.Project, error) {
	var projects []models.Project
	query := s.db.WithContext(ctx).Preload("Groups").Preload("Parent")

	if includeChildren {
		query = query.Preload("Children")
	}

	if parentID == nil {
		// 获取根项目（没有父项目的）
		query = query.Where("parent_id IS NULL")
	} else {
		// 获取指定父项目的子项目
		query = query.Where("parent_id = ?", *parentID)
	}

	err := query.Order("sort ASC, name ASC").Find(&projects).Error
	return projects, err
}

func (s *SQLiteStorage) GetProjectTree(ctx context.Context, rootID *uint) ([]*models.ProjectTreeNode, error) {
	var projects []models.Project
	query := s.db.WithContext(ctx).Preload("Groups")

	if rootID == nil {
		// 获取所有项目
		query = query.Find(&projects)
	} else {
		// 获取指定根项目及其子树
		var rootProject models.Project
		if err := s.db.WithContext(ctx).First(&rootProject, *rootID).Error; err != nil {
			return nil, fmt.Errorf("root project not found: %w", err)
		}
		// 获取所有路径以rootID开头的项目
		query = query.Where("path LIKE ?", rootProject.Path+"/%").Or("id = ?", *rootID)
		query = query.Find(&projects)
	}

	if query.Error != nil {
		return nil, query.Error
	}

	// 构建树状结构
	return s.buildProjectTree(projects, rootID), nil
}

func (s *SQLiteStorage) buildProjectTree(projects []models.Project, rootID *uint) []*models.ProjectTreeNode {
	// 创建ID到项目的映射
	projectMap := make(map[uint]*models.ProjectTreeNode)
	for i := range projects {
		node := &models.ProjectTreeNode{
			Project:     &projects[i],
			Children:    []*models.ProjectTreeNode{},
			HasChildren: false,
		}
		projectMap[projects[i].ID] = node
	}

	// 构建树状结构
	var roots []*models.ProjectTreeNode
	for _, node := range projectMap {
		if node.ParentID == nil || (rootID != nil && node.ParentID != nil && *node.ParentID == *rootID) {
			// 这是根节点或指定根的直接子节点
			if rootID == nil || node.ID != *rootID {
				roots = append(roots, node)
			}
		} else if parent, exists := projectMap[*node.ParentID]; exists {
			// 添加到父节点的子列表
			parent.Children = append(parent.Children, node)
			parent.HasChildren = true
		}
	}

	return roots
}

func (s *SQLiteStorage) MoveProject(ctx context.Context, params *models.MoveProjectParams) error {
	// 开始事务
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 获取要移动的项目
	var project models.Project
	if err := tx.First(&project, params.ProjectID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("project not found: %w", err)
	}

	// 检查是否移动到自己的子项目下（防止循环引用）
	if params.ParentID != nil {
		var parent models.Project
		if err := tx.First(&parent, *params.ParentID).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("parent project not found: %w", err)
		}

		// 检查是否形成循环
		if strings.Contains(parent.Path, fmt.Sprintf("/%d/", project.ID)) {
			tx.Rollback()
			return fmt.Errorf("cannot move project under its own child")
		}

		project.ParentID = params.ParentID
		project.Level = parent.Level + 1
	} else {
		project.ParentID = nil
		project.Level = 0
	}

	project.Sort = params.Position

	// 保存项目
	if err := tx.Save(&project).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 更新路径
	project.Path = project.BuildPath()
	if err := tx.Save(&project).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 递归更新所有子项目的层级和路径
	if err := s.updateChildrenPaths(tx, project.ID); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *SQLiteStorage) updateChildrenPaths(tx *gorm.DB, parentID uint) error {
	var children []models.Project
	if err := tx.Where("parent_id = ?", parentID).Find(&children).Error; err != nil {
		return err
	}

	for _, child := range children {
		// 获取父项目信息
		var parent models.Project
		if err := tx.First(&parent, child.ParentID).Error; err != nil {
			return err
		}

		child.Level = parent.Level + 1
		child.Path = child.BuildPath()

		if err := tx.Save(&child).Error; err != nil {
			return err
		}

		// 递归更新子项目
		if err := s.updateChildrenPaths(tx, child.ID); err != nil {
			return err
		}
	}

	return nil
}

func (s *SQLiteStorage) GetProjectChildren(ctx context.Context, parentID uint) ([]models.Project, error) {
	var children []models.Project
	err := s.db.WithContext(ctx).
		Preload("Groups").
		Where("parent_id = ?", parentID).
		Order("sort ASC, name ASC").
		Find(&children).Error
	return children, err
}

func (s *SQLiteStorage) UpdateProject(ctx context.Context, project *models.Project) error {
	return s.db.WithContext(ctx).Save(project).Error
}

func (s *SQLiteStorage) DeleteProject(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.Project{}, id).Error
}

func (s *SQLiteStorage) GetProjectStats(ctx context.Context, projectID uint) (*models.ProjectStats, error) {
	var stats models.ProjectStats

	// Count groups
	var groupCount int64
	s.db.WithContext(ctx).Model(&models.Group{}).Where("project_id = ?", projectID).Count(&groupCount)
	stats.TotalGroups = int(groupCount)

	// Count hosts in all groups of this project
	var hostCount int64
	s.db.WithContext(ctx).Table("hosts").
		Joins("JOIN groups ON hosts.group_id = groups.id").
		Where("groups.project_id = ?", projectID).
		Count(&hostCount)
	stats.TotalHosts = int(hostCount)

	// Count port forwards in all groups of this project
	var portCount int64
	s.db.WithContext(ctx).Table("port_forwards").
		Joins("JOIN groups ON port_forwards.group_id = groups.id").
		Where("groups.project_id = ?", projectID).
		Count(&portCount)
	stats.TotalPorts = int(portCount)

	// Count active tunnels
	var tunnelCount int64
	s.db.WithContext(ctx).Table("tunnel_sessions").
		Joins("JOIN port_forwards ON tunnel_sessions.port_forward_id = port_forwards.id").
		Joins("JOIN groups ON port_forwards.group_id = groups.id").
		Where("groups.project_id = ? AND tunnel_sessions.status = ?", projectID, "active").
		Count(&tunnelCount)
	stats.ActiveTunnels = int(tunnelCount)

	return &stats, nil
}
