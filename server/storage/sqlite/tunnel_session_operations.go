package sqlite

import (
	"context"

	"github.com/aqz236/port-fly/core/models"
)

// ===== Tunnel Session Operations =====

func (s *SQLiteStorage) CreateTunnelSession(ctx context.Context, session *models.TunnelSession) error {
	return s.db.WithContext(ctx).Create(session).Error
}

func (s *SQLiteStorage) GetTunnelSession(ctx context.Context, id uint) (*models.TunnelSession, error) {
	var session models.TunnelSession
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortForward").First(&session, id).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *SQLiteStorage) GetTunnelSessions(ctx context.Context) ([]models.TunnelSession, error) {
	var sessions []models.TunnelSession
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortForward").Find(&sessions).Error
	return sessions, err
}

func (s *SQLiteStorage) GetActiveTunnelSessions(ctx context.Context) ([]models.TunnelSession, error) {
	var sessions []models.TunnelSession
	err := s.db.WithContext(ctx).Preload("Host").Preload("PortForward").Where("status = ?", "active").Find(&sessions).Error
	return sessions, err
}

func (s *SQLiteStorage) UpdateTunnelSession(ctx context.Context, session *models.TunnelSession) error {
	return s.db.WithContext(ctx).Save(session).Error
}

func (s *SQLiteStorage) DeleteTunnelSession(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.TunnelSession{}, id).Error
}

func (s *SQLiteStorage) GetSessionStats(ctx context.Context) (*models.SessionStats, error) {
	var stats models.SessionStats

	// Count total connections
	var totalCount int64
	s.db.WithContext(ctx).Model(&models.TunnelSession{}).Count(&totalCount)
	stats.TotalConnections = totalCount

	// Count active connections
	var activeCount int64
	s.db.WithContext(ctx).Model(&models.TunnelSession{}).Where("status = ?", "active").Count(&activeCount)
	stats.ActiveConnections = activeCount

	// Count failed connections
	var failedCount int64
	s.db.WithContext(ctx).Model(&models.TunnelSession{}).Where("status = ?", "error").Count(&failedCount)
	stats.FailedConnections = failedCount

	// Sum bytes sent and received (using data_transferred as total)
	var totalTransferred int64
	s.db.WithContext(ctx).Model(&models.TunnelSession{}).Select("COALESCE(SUM(data_transferred), 0)").Scan(&totalTransferred)
	stats.BytesSent = totalTransferred / 2 // Simplified: assume half sent, half received
	stats.BytesReceived = totalTransferred / 2

	return &stats, nil
}
