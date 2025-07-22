package manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/aqz236/port-fly/core/models"
	"github.com/aqz236/port-fly/core/ssh"
	"github.com/aqz236/port-fly/core/utils"
)

// SessionManager manages SSH tunnel sessions
type SessionManager struct {
	sessions    map[string]*ManagedSession
	mu          sync.RWMutex
	logger      utils.Logger
	connPool    *ssh.ConnectionPool
	config      models.SSHConfig
}

// ManagedSession wraps a session with management functionality
type ManagedSession struct {
	session      *models.Session
	sshClient    *ssh.SSHClient
	tunnelMgr    *ssh.TunnelManager
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
}

// SessionManagerInterface defines the contract for session management
type SessionManagerInterface interface {
	CreateSession(config models.SSHConnectionConfig, tunnelConfig models.TunnelConfig) (*models.Session, error)
	StartSession(sessionID string) error
	StopSession(sessionID string) error
	GetSession(sessionID string) (*models.Session, error)
	ListSessions() ([]*models.Session, error)
	DeleteSession(sessionID string) error
	GetSessionStats(sessionID string) (models.SessionStats, error)
}

// NewSessionManager creates a new session manager
func NewSessionManager(config models.SSHConfig, logger utils.Logger) *SessionManager {
	// Create connection pool
	connPool := ssh.NewConnectionPool(
		config.MaxConnections,
		config.IdleTimeout,
		logger.WithGroup("connection_pool"),
	)

	return &SessionManager{
		sessions: make(map[string]*ManagedSession),
		logger:   logger.WithGroup("session_manager"),
		connPool: connPool,
		config:   config,
	}
}

// CreateSession creates a new SSH tunnel session
func (sm *SessionManager) CreateSession(sshConfig models.SSHConnectionConfig, tunnelConfig models.TunnelConfig) (*models.Session, error) {
	// Generate unique session ID
	sessionID := uuid.New().String()
	
	// Apply default SSH configuration
	sm.applyDefaultSSHConfig(&sshConfig)
	
	// Validate configurations
	if err := sm.validateConfigs(sshConfig, tunnelConfig); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	
	// Create session model
	session := &models.Session{
		ID:           sessionID,
		Name:         fmt.Sprintf("session-%s", sessionID[:8]),
		Description:  tunnelConfig.GetTunnelDescription(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Status:       models.StatusCreated,
		SSHConfig:    sshConfig,
		TunnelConfig: tunnelConfig,
		Stats:        models.SessionStats{},
		StopChan:     make(chan bool, 1),
	}
	
	// Create SSH client
	sshClient := ssh.NewSSHClientWithPool(
		sshConfig,
		sm.connPool,
		sm.logger.With("session_id", sessionID),
	)
	
	// Create tunnel manager
	tunnelMgr := ssh.NewTunnelManager(
		sshClient,
		tunnelConfig,
		sm.logger.With("session_id", sessionID),
	)
	
	// Create context for session lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create managed session
	managedSession := &ManagedSession{
		session:   session,
		sshClient: sshClient,
		tunnelMgr: tunnelMgr,
		ctx:       ctx,
		cancel:    cancel,
	}
	
	// Store session
	sm.mu.Lock()
	sm.sessions[sessionID] = managedSession
	sm.mu.Unlock()
	
	sm.logger.Info("session created", 
		"session_id", sessionID,
		"tunnel_type", tunnelConfig.Type,
		"description", tunnelConfig.GetTunnelDescription())
	
	return session, nil
}

// StartSession starts a session
func (sm *SessionManager) StartSession(sessionID string) error {
	sm.mu.RLock()
	managedSession, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	
	managedSession.mu.Lock()
	defer managedSession.mu.Unlock()
	
	if managedSession.session.Status != models.StatusCreated && 
	   managedSession.session.Status != models.StatusStopped {
		return fmt.Errorf("session cannot be started in current status: %s", managedSession.session.Status)
	}
	
	// Update status
	managedSession.session.Status = models.StatusConnecting
	managedSession.session.UpdatedAt = time.Now()
	
	// Start the session in a goroutine
	go sm.runSession(managedSession)
	
	sm.logger.Info("session start initiated", "session_id", sessionID)
	return nil
}

// runSession runs the session lifecycle
func (sm *SessionManager) runSession(ms *ManagedSession) {
	sessionID := ms.session.ID
	logger := sm.logger.With("session_id", sessionID)
	
	defer func() {
		if r := recover(); r != nil {
			logger.Error("session panic recovered", "panic", r)
			ms.mu.Lock()
			ms.session.Status = models.StatusError
			ms.session.LastError = fmt.Sprintf("panic: %v", r)
			ms.session.UpdatedAt = time.Now()
			ms.mu.Unlock()
		}
	}()
	
	// Establish SSH connection
	logger.Info("establishing SSH connection")
	ms.mu.Lock()
	ms.session.Status = models.StatusConnecting
	ms.session.UpdatedAt = time.Now()
	ms.mu.Unlock()
	
	if err := ms.sshClient.Connect(ms.ctx); err != nil {
		logger.Error("failed to establish SSH connection", "error", err)
		ms.mu.Lock()
		ms.session.Status = models.StatusError
		ms.session.LastError = err.Error()
		ms.session.UpdatedAt = time.Now()
		ms.mu.Unlock()
		return
	}
	
	// Update status to connected
	now := time.Now()
	ms.mu.Lock()
	ms.session.Status = models.StatusConnected
	ms.session.ConnectedAt = &now
	ms.session.UpdatedAt = now
	ms.mu.Unlock()
	
	logger.Info("SSH connection established")
	
	// Start tunnel
	logger.Info("starting tunnel")
	if err := ms.tunnelMgr.Start(ms.ctx); err != nil {
		logger.Error("failed to start tunnel", "error", err)
		ms.mu.Lock()
		ms.session.Status = models.StatusError
		ms.session.LastError = err.Error()
		ms.session.UpdatedAt = time.Now()
		ms.mu.Unlock()
		
		// Disconnect SSH client
		ms.sshClient.Disconnect()
		return
	}
	
	// Update status to active
	ms.mu.Lock()
	ms.session.Status = models.StatusActive
	ms.session.UpdatedAt = time.Now()
	ms.mu.Unlock()
	
	logger.Info("session is now active")
	
	// Monitor session
	sm.monitorSession(ms)
}

// monitorSession monitors a running session
func (sm *SessionManager) monitorSession(ms *ManagedSession) {
	sessionID := ms.session.ID
	logger := sm.logger.With("session_id", sessionID)
	
	ticker := time.NewTicker(30 * time.Second) // Update stats every 30 seconds
	defer ticker.Stop()
	
	for {
		select {
		case <-ms.ctx.Done():
			logger.Info("session context cancelled")
			return
			
		case <-ms.session.StopChan:
			logger.Info("session stop requested")
			sm.stopSession(ms)
			return
			
		case <-ticker.C:
			// Update session statistics
			sm.updateSessionStats(ms)
			
			// Check SSH connection health
			if !ms.sshClient.IsConnected() {
				logger.Warn("SSH connection lost, attempting reconnection")
				ms.mu.Lock()
				ms.session.Status = models.StatusConnecting
				ms.session.UpdatedAt = time.Now()
				reconnectCount := ms.session.Stats.ReconnectCount + 1
				ms.session.Stats.ReconnectCount = reconnectCount
				now := time.Now()
				ms.session.Stats.LastReconnectAt = &now
				ms.mu.Unlock()
				
				// Attempt reconnection
				if err := ms.sshClient.Reconnect(ms.ctx); err != nil {
					logger.Error("reconnection failed", "error", err)
					ms.mu.Lock()
					ms.session.Status = models.StatusError
					ms.session.LastError = err.Error()
					ms.session.UpdatedAt = time.Now()
					ms.mu.Unlock()
					sm.stopSession(ms)
					return
				}
				
				logger.Info("SSH reconnection successful")
				ms.mu.Lock()
				ms.session.Status = models.StatusActive
				ms.session.UpdatedAt = time.Now()
				ms.mu.Unlock()
			}
		}
	}
}

// stopSession stops a managed session
func (sm *SessionManager) stopSession(ms *ManagedSession) {
	logger := sm.logger.With("session_id", ms.session.ID)
	
	ms.mu.Lock()
	ms.session.Status = models.StatusStopping
	ms.session.UpdatedAt = time.Now()
	ms.mu.Unlock()
	
	// Stop tunnel
	if ms.tunnelMgr.IsRunning() {
		logger.Info("stopping tunnel")
		if err := ms.tunnelMgr.Stop(); err != nil {
			logger.Error("error stopping tunnel", "error", err)
		}
	}
	
	// Disconnect SSH client
	logger.Info("disconnecting SSH client")
	if err := ms.sshClient.Disconnect(); err != nil {
		logger.Error("error disconnecting SSH client", "error", err)
	}
	
	// Update final status
	now := time.Now()
	ms.mu.Lock()
	ms.session.Status = models.StatusStopped
	ms.session.DisconnectedAt = &now
	ms.session.UpdatedAt = now
	ms.mu.Unlock()
	
	logger.Info("session stopped")
}

// StopSession stops a session
func (sm *SessionManager) StopSession(sessionID string) error {
	sm.mu.RLock()
	managedSession, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	
	managedSession.mu.RLock()
	status := managedSession.session.Status
	managedSession.mu.RUnlock()
	
	if status == models.StatusStopped || status == models.StatusStopping {
		return fmt.Errorf("session is already stopped or stopping")
	}
	
	// Signal stop
	select {
	case managedSession.session.StopChan <- true:
	default:
		// Channel might be full or closed, use context cancellation
		managedSession.cancel()
	}
	
	sm.logger.Info("session stop requested", "session_id", sessionID)
	return nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*models.Session, error) {
	sm.mu.RLock()
	managedSession, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	
	managedSession.mu.RLock()
	defer managedSession.mu.RUnlock()
	
	// Create a copy of the session to avoid race conditions
	sessionCopy := *managedSession.session
	return &sessionCopy, nil
}

// ListSessions returns all sessions
func (sm *SessionManager) ListSessions() ([]*models.Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	sessions := make([]*models.Session, 0, len(sm.sessions))
	
	for _, managedSession := range sm.sessions {
		managedSession.mu.RLock()
		sessionCopy := *managedSession.session
		sessions = append(sessions, &sessionCopy)
		managedSession.mu.RUnlock()
	}
	
	return sessions, nil
}

// DeleteSession deletes a session
func (sm *SessionManager) DeleteSession(sessionID string) error {
	sm.mu.Lock()
	managedSession, exists := sm.sessions[sessionID]
	if exists {
		delete(sm.sessions, sessionID)
	}
	sm.mu.Unlock()
	
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	
	// Stop the session if it's running
	managedSession.mu.RLock()
	status := managedSession.session.Status
	managedSession.mu.RUnlock()
	
	if status != models.StatusStopped {
		sm.stopSession(managedSession)
	}
	
	// Cancel context
	managedSession.cancel()
	
	sm.logger.Info("session deleted", "session_id", sessionID)
	return nil
}

// GetSessionStats retrieves session statistics
func (sm *SessionManager) GetSessionStats(sessionID string) (models.SessionStats, error) {
	sm.mu.RLock()
	managedSession, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()
	
	if !exists {
		return models.SessionStats{}, fmt.Errorf("session not found: %s", sessionID)
	}
	
	// Update stats from tunnel manager
	sm.updateSessionStats(managedSession)
	
	managedSession.mu.RLock()
	defer managedSession.mu.RUnlock()
	
	return managedSession.session.Stats, nil
}

// updateSessionStats updates session statistics
func (sm *SessionManager) updateSessionStats(ms *ManagedSession) {
	if ms.tunnelMgr.IsRunning() {
		tunnelStats := ms.tunnelMgr.GetStats()
		
		ms.mu.Lock()
		ms.session.Stats = tunnelStats
		ms.session.UpdatedAt = time.Now()
		ms.mu.Unlock()
	}
}

// applyDefaultSSHConfig applies default SSH configuration
func (sm *SessionManager) applyDefaultSSHConfig(config *models.SSHConnectionConfig) {
	if config.ConnectTimeout == 0 {
		config.ConnectTimeout = sm.config.ConnectTimeout
	}
	if config.KeepAliveTimeout == 0 {
		config.KeepAliveTimeout = sm.config.KeepAliveTimeout
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = sm.config.MaxRetries
	}
	if config.RetryInterval == 0 {
		config.RetryInterval = sm.config.RetryInterval
	}
	if config.HostKeyCallback == "" {
		config.HostKeyCallback = sm.config.HostKeyCallback
	}
}

// validateConfigs validates SSH and tunnel configurations
func (sm *SessionManager) validateConfigs(sshConfig models.SSHConnectionConfig, tunnelConfig models.TunnelConfig) error {
	// Validate SSH configuration
	authManager := ssh.NewAuthManager()
	if err := authManager.ValidateConfig(sshConfig); err != nil {
		return fmt.Errorf("invalid SSH configuration: %w", err)
	}
	
	// Validate tunnel configuration
	if err := tunnelConfig.Validate(); err != nil {
		return fmt.Errorf("invalid tunnel configuration: %w", err)
	}
	
	return nil
}

// Close shuts down the session manager
func (sm *SessionManager) Close() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// Stop all sessions
	for sessionID := range sm.sessions {
		sm.StopSession(sessionID)
	}
	
	// Close connection pool
	sm.connPool.Close()
	
	sm.logger.Info("session manager closed")
	return nil
}
