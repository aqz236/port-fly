package sqlite

import (
	"context"

	"github.com/aqz236/port-fly/core/models"
)

// ===== Group Operations =====

func (s *SQLiteStorage) CreateGroup(ctx context.Context, group *models.Group) error {
	return s.db.WithContext(ctx).Create(group).Error
}

func (s *SQLiteStorage) GetGroup(ctx context.Context, id uint) (*models.Group, error) {
	var group models.Group
	err := s.db.WithContext(ctx).Preload("Project").Preload("Hosts").Preload("PortForwards").First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *SQLiteStorage) GetGroups(ctx context.Context) ([]models.Group, error) {
	var groups []models.Group
	err := s.db.WithContext(ctx).Preload("Project").Preload("Hosts").Preload("PortForwards").Find(&groups).Error
	return groups, err
}

func (s *SQLiteStorage) GetGroupsByProject(ctx context.Context, projectID uint) ([]models.Group, error) {
	var groups []models.Group
	err := s.db.WithContext(ctx).Preload("Hosts").Preload("PortForwards").Where("project_id = ?", projectID).Find(&groups).Error
	return groups, err
}

func (s *SQLiteStorage) UpdateGroup(ctx context.Context, group *models.Group) error {
	return s.db.WithContext(ctx).Save(group).Error
}

func (s *SQLiteStorage) DeleteGroup(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.Group{}, id).Error
}

func (s *SQLiteStorage) GetGroupStats(ctx context.Context, groupID uint) (*models.GroupStats, error) {
	var stats models.GroupStats

	// Count hosts
	var hostCount int64
	s.db.WithContext(ctx).Model(&models.Host{}).Where("group_id = ?", groupID).Count(&hostCount)
	stats.TotalHosts = int(hostCount)

	// Count port forwards
	var portCount int64
	s.db.WithContext(ctx).Model(&models.PortForward{}).Where("group_id = ?", groupID).Count(&portCount)
	stats.TotalPorts = int(portCount)

	// Count connected hosts
	var connectedCount int64
	s.db.WithContext(ctx).Model(&models.Host{}).Where("group_id = ? AND status = ?", groupID, "connected").Count(&connectedCount)
	stats.ConnectedHosts = int(connectedCount)

	// Count active tunnels
	var tunnelCount int64
	s.db.WithContext(ctx).Table("tunnel_sessions").
		Joins("JOIN port_forwards ON tunnel_sessions.port_forward_id = port_forwards.id").
		Where("port_forwards.group_id = ? AND tunnel_sessions.status = ?", groupID, "active").
		Count(&tunnelCount)
	stats.ActiveTunnels = int(tunnelCount)

	return &stats, nil
}
