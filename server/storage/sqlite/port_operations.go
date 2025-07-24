package sqlite

import (
	"context"
	"fmt"

	"github.com/aqz236/port-fly/core/models"
	"gorm.io/gorm"
)

// ===== Port Operations =====

// CreatePort creates a new port
func (s *SQLiteStorage) CreatePort(ctx context.Context, port *models.Port) error {
	if err := port.Validate(); err != nil {
		return fmt.Errorf("invalid port data: %w", err)
	}

	if err := s.db.WithContext(ctx).Create(port).Error; err != nil {
		return fmt.Errorf("failed to create port: %w", err)
	}

	return nil
}

// GetPort retrieves a port by ID
func (s *SQLiteStorage) GetPort(ctx context.Context, id uint) (*models.Port, error) {
	var port models.Port
	err := s.db.WithContext(ctx).
		Preload("Group").
		Preload("Host").
		Preload("TargetPort").
		Preload("SourcePorts").
		Preload("TunnelSessions").
		First(&port, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("port not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get port: %w", err)
	}

	return &port, nil
}

// GetPorts retrieves all ports
func (s *SQLiteStorage) GetPorts(ctx context.Context) ([]models.Port, error) {
	var ports []models.Port
	err := s.db.WithContext(ctx).
		Preload("Group").
		Preload("Host").
		Preload("TargetPort").
		Find(&ports).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get ports: %w", err)
	}

	return ports, nil
}

// GetPortsByGroup retrieves all ports in a group
func (s *SQLiteStorage) GetPortsByGroup(ctx context.Context, groupID uint) ([]models.Port, error) {
	var ports []models.Port
	err := s.db.WithContext(ctx).
		Preload("Group").
		Preload("Host").
		Preload("TargetPort").
		Where("group_id = ?", groupID).
		Find(&ports).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get ports by group: %w", err)
	}

	return ports, nil
}

// GetPortsByHost retrieves all ports associated with a host
func (s *SQLiteStorage) GetPortsByHost(ctx context.Context, hostID uint) ([]models.Port, error) {
	var ports []models.Port
	err := s.db.WithContext(ctx).
		Preload("Group").
		Preload("Host").
		Preload("TargetPort").
		Where("host_id = ?", hostID).
		Find(&ports).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get ports by host: %w", err)
	}

	return ports, nil
}

// UpdatePort updates an existing port
func (s *SQLiteStorage) UpdatePort(ctx context.Context, port *models.Port) error {
	if err := port.Validate(); err != nil {
		return fmt.Errorf("invalid port data: %w", err)
	}

	result := s.db.WithContext(ctx).Save(port)
	if result.Error != nil {
		return fmt.Errorf("failed to update port: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("port not found: %d", port.ID)
	}

	return nil
}

// DeletePort deletes a port by ID
func (s *SQLiteStorage) DeletePort(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Delete(&models.Port{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete port: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("port not found: %d", id)
	}

	return nil
}

// GetPortStats retrieves statistics for a port
func (s *SQLiteStorage) GetPortStats(ctx context.Context, portID uint) (*models.PortStats, error) {
	var stats models.PortStats

	// Get active connections count
	var activeConnections int64
	err := s.db.WithContext(ctx).
		Model(&models.PortConnection{}).
		Where("(remote_port_id = ? OR local_port_id = ?) AND status = ?", 
			portID, portID, models.PortStatusActive).
		Count(&activeConnections).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count active connections: %w", err)
	}

	// Get total connections count
	var totalConnections int64
	err = s.db.WithContext(ctx).
		Model(&models.PortConnection{}).
		Where("remote_port_id = ? OR local_port_id = ?", portID, portID).
		Count(&totalConnections).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count total connections: %w", err)
	}

	// Calculate success rate
	var successfulConnections int64
	err = s.db.WithContext(ctx).
		Model(&models.PortConnection{}).
		Where("(remote_port_id = ? OR local_port_id = ?) AND status IN ?", 
			portID, portID, []models.PortStatus{models.PortStatusActive, models.PortStatusAvailable}).
		Count(&successfulConnections).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count successful connections: %w", err)
	}

	stats.ActiveConnections = int(activeConnections)
	stats.TotalConnections = int(totalConnections)
	
	if totalConnections > 0 {
		stats.SuccessRate = float64(successfulConnections) / float64(totalConnections) * 100
	}

	return &stats, nil
}

// SearchPorts searches for ports by name or description
func (s *SQLiteStorage) SearchPorts(ctx context.Context, query string) ([]models.Port, error) {
	var ports []models.Port
	searchQuery := "%" + query + "%"
	
	err := s.db.WithContext(ctx).
		Preload("Group").
		Preload("Host").
		Where("name LIKE ? OR description LIKE ?", searchQuery, searchQuery).
		Find(&ports).Error

	if err != nil {
		return nil, fmt.Errorf("failed to search ports: %w", err)
	}

	return ports, nil
}

// UpdatePortStatus updates the status of a port
func (s *SQLiteStorage) UpdatePortStatus(ctx context.Context, portID uint, status models.PortStatus) error {
	result := s.db.WithContext(ctx).
		Model(&models.Port{}).
		Where("id = ?", portID).
		Update("status", status)

	if result.Error != nil {
		return fmt.Errorf("failed to update port status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("port not found: %d", portID)
	}

	return nil
}

// ===== Port Connection Operations =====

// CreatePortConnection creates a new port connection
func (s *SQLiteStorage) CreatePortConnection(ctx context.Context, connection *models.PortConnection) error {
	if err := s.db.WithContext(ctx).Create(connection).Error; err != nil {
		return fmt.Errorf("failed to create port connection: %w", err)
	}

	return nil
}

// GetPortConnection retrieves a port connection by ID
func (s *SQLiteStorage) GetPortConnection(ctx context.Context, id uint) (*models.PortConnection, error) {
	var connection models.PortConnection
	err := s.db.WithContext(ctx).
		Preload("RemotePort").
		Preload("LocalPort").
		Preload("TunnelSession").
		First(&connection, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("port connection not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get port connection: %w", err)
	}

	return &connection, nil
}

// GetPortConnections retrieves all port connections
func (s *SQLiteStorage) GetPortConnections(ctx context.Context) ([]models.PortConnection, error) {
	var connections []models.PortConnection
	err := s.db.WithContext(ctx).
		Preload("RemotePort").
		Preload("LocalPort").
		Preload("TunnelSession").
		Find(&connections).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get port connections: %w", err)
	}

	return connections, nil
}

// GetActivePortConnections retrieves all active port connections
func (s *SQLiteStorage) GetActivePortConnections(ctx context.Context) ([]models.PortConnection, error) {
	var connections []models.PortConnection
	err := s.db.WithContext(ctx).
		Preload("RemotePort").
		Preload("LocalPort").
		Preload("TunnelSession").
		Where("status = ?", models.PortStatusActive).
		Find(&connections).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get active port connections: %w", err)
	}

	return connections, nil
}

// UpdatePortConnection updates an existing port connection
func (s *SQLiteStorage) UpdatePortConnection(ctx context.Context, connection *models.PortConnection) error {
	result := s.db.WithContext(ctx).Save(connection)
	if result.Error != nil {
		return fmt.Errorf("failed to update port connection: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("port connection not found: %d", connection.ID)
	}

	return nil
}

// DeletePortConnection deletes a port connection by ID
func (s *SQLiteStorage) DeletePortConnection(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Delete(&models.PortConnection{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete port connection: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("port connection not found: %d", id)
	}

	return nil
}

// GetPortConnectionByPorts retrieves a port connection by remote and local port IDs
func (s *SQLiteStorage) GetPortConnectionByPorts(ctx context.Context, remotePortID, localPortID uint) (*models.PortConnection, error) {
	var connection models.PortConnection
	err := s.db.WithContext(ctx).
		Preload("RemotePort").
		Preload("LocalPort").
		Preload("TunnelSession").
		Where("remote_port_id = ? AND local_port_id = ?", remotePortID, localPortID).
		First(&connection).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("port connection not found for ports: %d -> %d", remotePortID, localPortID)
		}
		return nil, fmt.Errorf("failed to get port connection: %w", err)
	}

	return &connection, nil
}
