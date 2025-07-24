package sqlite

import (
	"context"

	"github.com/aqz236/port-fly/core/models"
)

// ===== Port Forward Operations =====

func (s *SQLiteStorage) CreatePortForward(ctx context.Context, portForward *models.PortForward) error {
	return s.db.WithContext(ctx).Create(portForward).Error
}

func (s *SQLiteStorage) GetPortForward(ctx context.Context, id uint) (*models.PortForward, error) {
	var portForward models.PortForward
	err := s.db.WithContext(ctx).Preload("Group").Preload("Host").First(&portForward, id).Error
	if err != nil {
		return nil, err
	}
	return &portForward, nil
}

func (s *SQLiteStorage) GetPortForwards(ctx context.Context) ([]models.PortForward, error) {
	var portForwards []models.PortForward
	err := s.db.WithContext(ctx).Preload("Group").Preload("Host").Find(&portForwards).Error
	return portForwards, err
}

func (s *SQLiteStorage) GetPortForwardsByHost(ctx context.Context, hostID uint) ([]models.PortForward, error) {
	var portForwards []models.PortForward
	err := s.db.WithContext(ctx).Preload("Group").Where("host_id = ?", hostID).Find(&portForwards).Error
	return portForwards, err
}

func (s *SQLiteStorage) GetPortForwardsByGroup(ctx context.Context, groupID uint) ([]models.PortForward, error) {
	var portForwards []models.PortForward
	err := s.db.WithContext(ctx).Preload("Host").Where("group_id = ?", groupID).Find(&portForwards).Error
	return portForwards, err
}

func (s *SQLiteStorage) UpdatePortForward(ctx context.Context, portForward *models.PortForward) error {
	return s.db.WithContext(ctx).Save(portForward).Error
}

func (s *SQLiteStorage) DeletePortForward(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.PortForward{}, id).Error
}

func (s *SQLiteStorage) GetPortForwardStats(ctx context.Context, portForwardID uint) (*models.PortForwardStats, error) {
	var stats models.PortForwardStats

	// Count total sessions
	var totalCount int64
	s.db.WithContext(ctx).Model(&models.TunnelSession{}).Where("port_forward_id = ?", portForwardID).Count(&totalCount)
	stats.TotalSessions = int(totalCount)

	// Count active sessions
	var activeCount int64
	s.db.WithContext(ctx).Model(&models.TunnelSession{}).Where("port_forward_id = ? AND status = ?", portForwardID, "active").Count(&activeCount)
	stats.ActiveSessions = int(activeCount)

	// Sum data transferred
	s.db.WithContext(ctx).Model(&models.TunnelSession{}).Where("port_forward_id = ?", portForwardID).Select("COALESCE(SUM(data_transferred), 0)").Scan(&stats.TotalDataTransferred)

	return &stats, nil
}

func (s *SQLiteStorage) SearchPortForwards(ctx context.Context, query string) ([]models.PortForward, error) {
	var portForwards []models.PortForward
	searchPattern := "%" + query + "%"
	err := s.db.WithContext(ctx).Preload("Group").Preload("Host").Where("name LIKE ? OR description LIKE ? OR remote_host LIKE ?", searchPattern, searchPattern, searchPattern).Find(&portForwards).Error
	return portForwards, err
}
