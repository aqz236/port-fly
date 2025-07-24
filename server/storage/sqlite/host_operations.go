package sqlite

import (
	"context"

	"github.com/aqz236/port-fly/core/models"
)

// ===== Host Operations =====

func (s *SQLiteStorage) CreateHost(ctx context.Context, host *models.Host) error {
	return s.db.WithContext(ctx).Create(host).Error
}

func (s *SQLiteStorage) GetHost(ctx context.Context, id uint) (*models.Host, error) {
	var host models.Host
	err := s.db.WithContext(ctx).Preload("Group").Preload("PortForwards").First(&host, id).Error
	if err != nil {
		return nil, err
	}
	return &host, nil
}

func (s *SQLiteStorage) GetHosts(ctx context.Context) ([]models.Host, error) {
	var hosts []models.Host
	err := s.db.WithContext(ctx).Preload("Group").Find(&hosts).Error
	return hosts, err
}

func (s *SQLiteStorage) GetHostsByGroup(ctx context.Context, groupID uint) ([]models.Host, error) {
	var hosts []models.Host
	err := s.db.WithContext(ctx).Where("group_id = ?", groupID).Find(&hosts).Error
	return hosts, err
}

func (s *SQLiteStorage) UpdateHost(ctx context.Context, host *models.Host) error {
	return s.db.WithContext(ctx).Save(host).Error
}

func (s *SQLiteStorage) DeleteHost(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.Host{}, id).Error
}

func (s *SQLiteStorage) GetHostStats(ctx context.Context, hostID uint) (*models.HostStats, error) {
	var stats models.HostStats

	// Get host info
	var host models.Host
	if err := s.db.WithContext(ctx).First(&host, hostID).Error; err != nil {
		return nil, err
	}

	stats.TotalConnections = host.ConnectionCount

	// Count active tunnels
	var tunnelCount int64
	s.db.WithContext(ctx).Model(&models.TunnelSession{}).Where("host_id = ? AND status = ?", hostID, "active").Count(&tunnelCount)
	stats.ActiveTunnels = int(tunnelCount)

	// Calculate uptime percentage (simplified)
	stats.UptimePercentage = 95.0 // TODO: implement real calculation

	return &stats, nil
}

func (s *SQLiteStorage) SearchHosts(ctx context.Context, query string) ([]models.Host, error) {
	var hosts []models.Host
	searchPattern := "%" + query + "%"
	err := s.db.WithContext(ctx).Preload("Group").Where("name LIKE ? OR hostname LIKE ? OR description LIKE ?", searchPattern, searchPattern, searchPattern).Find(&hosts).Error
	return hosts, err
}
