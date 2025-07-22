package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aqz236/port-fly/core/models"
	"github.com/aqz236/port-fly/core/utils"
)

// TunnelManager manages SSH tunnels
type TunnelManager struct {
	sshClient *SSHClient
	config    models.TunnelConfig
	logger    utils.Logger
	
	// State management
	running    int32
	listeners  []net.Listener
	connections sync.Map // map[net.Conn]bool
	stopChan   chan struct{}
	wg         sync.WaitGroup
	
	// Statistics
	stats models.SessionStats
	statsMu sync.RWMutex
}

// NewTunnelManager creates a new tunnel manager
func NewTunnelManager(sshClient *SSHClient, config models.TunnelConfig, logger utils.Logger) *TunnelManager {
	return &TunnelManager{
		sshClient: sshClient,
		config:    config,
		logger:    logger,
		stopChan:  make(chan struct{}),
	}
}

// Start starts the tunnel based on its type
func (tm *TunnelManager) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&tm.running, 0, 1) {
		return fmt.Errorf("tunnel is already running")
	}
	
	// Validate tunnel configuration
	if err := tm.config.Validate(); err != nil {
		atomic.StoreInt32(&tm.running, 0)
		return fmt.Errorf("invalid tunnel configuration: %w", err)
	}
	
	// Ensure SSH connection is established
	if !tm.sshClient.IsConnected() {
		if err := tm.sshClient.Connect(ctx); err != nil {
			atomic.StoreInt32(&tm.running, 0)
			return fmt.Errorf("failed to establish SSH connection: %w", err)
		}
	}
	
	var err error
	switch tm.config.Type {
	case models.TunnelTypeLocal:
		err = tm.startLocalForwarding(ctx)
	case models.TunnelTypeRemote:
		err = tm.startRemoteForwarding(ctx)
	case models.TunnelTypeDynamic:
		err = tm.startDynamicForwarding(ctx)
	default:
		err = fmt.Errorf("unsupported tunnel type: %s", tm.config.Type)
	}
	
	if err != nil {
		atomic.StoreInt32(&tm.running, 0)
		return err
	}
	
	tm.logger.Info("tunnel started successfully", 
		"type", tm.config.Type,
		"description", tm.config.GetTunnelDescription())
	
	return nil
}

// Stop stops the tunnel
func (tm *TunnelManager) Stop() error {
	if !atomic.CompareAndSwapInt32(&tm.running, 1, 0) {
		return fmt.Errorf("tunnel is not running")
	}
	
	tm.logger.Info("stopping tunnel")
	
	// Signal stop
	close(tm.stopChan)
	
	// Close all listeners
	for _, listener := range tm.listeners {
		if err := listener.Close(); err != nil {
			tm.logger.Warn("error closing listener", "error", err)
		}
	}
	
	// Close all active connections
	tm.connections.Range(func(key, value interface{}) bool {
		if conn, ok := key.(net.Conn); ok {
			conn.Close()
		}
		return true
	})
	
	// Wait for all goroutines to finish
	tm.wg.Wait()
	
	tm.logger.Info("tunnel stopped")
	return nil
}

// IsRunning returns whether the tunnel is running
func (tm *TunnelManager) IsRunning() bool {
	return atomic.LoadInt32(&tm.running) == 1
}

// GetStats returns tunnel statistics
func (tm *TunnelManager) GetStats() models.SessionStats {
	tm.statsMu.RLock()
	defer tm.statsMu.RUnlock()
	return tm.stats
}

// startLocalForwarding starts local port forwarding (-L)
func (tm *TunnelManager) startLocalForwarding(ctx context.Context) error {
	bindAddr := tm.config.LocalBindAddress
	if bindAddr == "" {
		bindAddr = "127.0.0.1"
	}
	
	localAddr := fmt.Sprintf("%s:%d", bindAddr, tm.config.LocalPort)
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", localAddr, err)
	}
	
	tm.listeners = append(tm.listeners, listener)
	
	tm.wg.Add(1)
	go tm.handleLocalConnections(ctx, listener)
	
	tm.logger.Info("local forwarding started", 
		"local_addr", localAddr,
		"remote_addr", fmt.Sprintf("%s:%d", tm.config.RemoteHost, tm.config.RemotePort))
	
	return nil
}

// handleLocalConnections handles incoming connections for local forwarding
func (tm *TunnelManager) handleLocalConnections(ctx context.Context, listener net.Listener) {
	defer tm.wg.Done()
	
	for {
		select {
		case <-tm.stopChan:
			return
		default:
		}
		
		// Set accept timeout to allow checking stop signal
		if tcpListener, ok := listener.(*net.TCPListener); ok {
			tcpListener.SetDeadline(time.Now().Add(1 * time.Second))
		}
		
		conn, err := listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // Timeout, check stop signal and continue
			}
			if atomic.LoadInt32(&tm.running) == 0 {
				return // Tunnel is stopping
			}
			tm.logger.Error("failed to accept connection", "error", err)
			continue
		}
		
		tm.wg.Add(1)
		go tm.handleLocalConnection(ctx, conn)
	}
}

// handleLocalConnection handles a single local connection
func (tm *TunnelManager) handleLocalConnection(ctx context.Context, localConn net.Conn) {
	defer tm.wg.Done()
	defer localConn.Close()
	
	// Track connection
	tm.connections.Store(localConn, true)
	defer tm.connections.Delete(localConn)
	
	// Update statistics
	tm.updateStats(func(stats *models.SessionStats) {
		stats.TotalConnections++
		stats.ActiveConnections++
	})
	defer tm.updateStats(func(stats *models.SessionStats) {
		stats.ActiveConnections--
	})
	
	// Establish SSH connection to remote host
	remoteAddr := fmt.Sprintf("%s:%d", tm.config.RemoteHost, tm.config.RemotePort)
	sshClient := tm.sshClient.GetClient()
	if sshClient == nil {
		tm.logger.Error("SSH client not available")
		tm.updateStats(func(stats *models.SessionStats) {
			stats.FailedConnections++
		})
		return
	}
	
	remoteConn, err := sshClient.Dial("tcp", remoteAddr)
	if err != nil {
		tm.logger.Error("failed to connect to remote host", 
			"remote_addr", remoteAddr, 
			"error", err)
		tm.updateStats(func(stats *models.SessionStats) {
			stats.FailedConnections++
		})
		return
	}
	defer remoteConn.Close()
	
	tm.logger.Debug("established connection", 
		"local_addr", localConn.RemoteAddr(),
		"remote_addr", remoteAddr)
	
	// Start bidirectional data transfer
	tm.transfer(localConn, remoteConn)
}

// startRemoteForwarding starts remote port forwarding (-R)
func (tm *TunnelManager) startRemoteForwarding(ctx context.Context) error {
	bindAddr := tm.config.RemoteBindAddress
	if bindAddr == "" {
		bindAddr = "127.0.0.1"
	}
	
	remoteAddr := fmt.Sprintf("%s:%d", bindAddr, tm.config.LocalPort)
	sshClient := tm.sshClient.GetClient()
	if sshClient == nil {
		return fmt.Errorf("SSH client not available")
	}
	
	listener, err := sshClient.Listen("tcp", remoteAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on remote %s: %w", remoteAddr, err)
	}
	
	tm.listeners = append(tm.listeners, listener)
	
	tm.wg.Add(1)
	go tm.handleRemoteConnections(ctx, listener)
	
	tm.logger.Info("remote forwarding started", 
		"remote_addr", remoteAddr,
		"local_addr", fmt.Sprintf("%s:%d", tm.config.RemoteHost, tm.config.RemotePort))
	
	return nil
}

// handleRemoteConnections handles incoming connections for remote forwarding
func (tm *TunnelManager) handleRemoteConnections(ctx context.Context, listener net.Listener) {
	defer tm.wg.Done()
	
	for {
		select {
		case <-tm.stopChan:
			return
		default:
		}
		
		conn, err := listener.Accept()
		if err != nil {
			if atomic.LoadInt32(&tm.running) == 0 {
				return // Tunnel is stopping
			}
			tm.logger.Error("failed to accept remote connection", "error", err)
			continue
		}
		
		tm.wg.Add(1)
		go tm.handleRemoteConnection(ctx, conn)
	}
}

// handleRemoteConnection handles a single remote connection
func (tm *TunnelManager) handleRemoteConnection(ctx context.Context, remoteConn net.Conn) {
	defer tm.wg.Done()
	defer remoteConn.Close()
	
	// Track connection
	tm.connections.Store(remoteConn, true)
	defer tm.connections.Delete(remoteConn)
	
	// Update statistics
	tm.updateStats(func(stats *models.SessionStats) {
		stats.TotalConnections++
		stats.ActiveConnections++
	})
	defer tm.updateStats(func(stats *models.SessionStats) {
		stats.ActiveConnections--
	})
	
	// Connect to local target
	localAddr := net.JoinHostPort(tm.config.RemoteHost, fmt.Sprintf("%d", tm.config.RemotePort))
	localConn, err := net.Dial("tcp", localAddr)
	if err != nil {
		tm.logger.Error("failed to connect to local target", 
			"local_addr", localAddr, 
			"error", err)
		tm.updateStats(func(stats *models.SessionStats) {
			stats.FailedConnections++
		})
		return
	}
	defer localConn.Close()
	
	tm.logger.Debug("established remote connection", 
		"remote_addr", remoteConn.RemoteAddr(),
		"local_addr", localAddr)
	
	// Start bidirectional data transfer
	tm.transfer(remoteConn, localConn)
}

// startDynamicForwarding starts dynamic port forwarding (SOCKS proxy)
func (tm *TunnelManager) startDynamicForwarding(ctx context.Context) error {
	bindAddr := tm.config.SOCKSBindAddress
	if bindAddr == "" {
		bindAddr = "127.0.0.1"
	}
	
	localAddr := fmt.Sprintf("%s:%d", bindAddr, tm.config.SOCKSPort)
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", localAddr, err)
	}
	
	tm.listeners = append(tm.listeners, listener)
	
	tm.wg.Add(1)
	go tm.handleSOCKSConnections(ctx, listener)
	
	tm.logger.Info("SOCKS proxy started", 
		"bind_addr", localAddr,
		"version", tm.config.SOCKSVersion)
	
	return nil
}

// handleSOCKSConnections handles incoming SOCKS connections
func (tm *TunnelManager) handleSOCKSConnections(ctx context.Context, listener net.Listener) {
	defer tm.wg.Done()
	
	for {
		select {
		case <-tm.stopChan:
			return
		default:
		}
		
		// Set accept timeout to allow checking stop signal
		if tcpListener, ok := listener.(*net.TCPListener); ok {
			tcpListener.SetDeadline(time.Now().Add(1 * time.Second))
		}
		
		conn, err := listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // Timeout, check stop signal and continue
			}
			if atomic.LoadInt32(&tm.running) == 0 {
				return // Tunnel is stopping
			}
			tm.logger.Error("failed to accept SOCKS connection", "error", err)
			continue
		}
		
		tm.wg.Add(1)
		go tm.handleSOCKSConnection(ctx, conn)
	}
}

// handleSOCKSConnection handles a single SOCKS connection
func (tm *TunnelManager) handleSOCKSConnection(ctx context.Context, conn net.Conn) {
	defer tm.wg.Done()
	defer conn.Close()
	
	// Track connection
	tm.connections.Store(conn, true)
	defer tm.connections.Delete(conn)
	
	// Update statistics
	tm.updateStats(func(stats *models.SessionStats) {
		stats.TotalConnections++
		stats.ActiveConnections++
	})
	defer tm.updateStats(func(stats *models.SessionStats) {
		stats.ActiveConnections--
	})
	
	// Handle SOCKS protocol
	var targetAddr string
	var err error
	
	switch tm.config.SOCKSVersion {
	case 4:
		targetAddr, err = tm.handleSOCKS4(conn)
	case 5:
		targetAddr, err = tm.handleSOCKS5(conn)
	default:
		tm.logger.Error("unsupported SOCKS version", "version", tm.config.SOCKSVersion)
		tm.updateStats(func(stats *models.SessionStats) {
			stats.FailedConnections++
		})
		return
	}
	
	if err != nil {
		tm.logger.Error("SOCKS protocol error", "error", err)
		tm.updateStats(func(stats *models.SessionStats) {
			stats.FailedConnections++
		})
		return
	}
	
	// Establish SSH connection to target
	sshClient := tm.sshClient.GetClient()
	if sshClient == nil {
		tm.logger.Error("SSH client not available")
		tm.updateStats(func(stats *models.SessionStats) {
			stats.FailedConnections++
		})
		return
	}
	
	targetConn, err := sshClient.Dial("tcp", targetAddr)
	if err != nil {
		tm.logger.Error("failed to connect to target", 
			"target_addr", targetAddr, 
			"error", err)
		tm.updateStats(func(stats *models.SessionStats) {
			stats.FailedConnections++
		})
		return
	}
	defer targetConn.Close()
	
	tm.logger.Debug("established SOCKS connection", 
		"client_addr", conn.RemoteAddr(),
		"target_addr", targetAddr)
	
	// Start bidirectional data transfer
	tm.transfer(conn, targetConn)
}

// transfer handles bidirectional data transfer between two connections
func (tm *TunnelManager) transfer(conn1, conn2 net.Conn) {
	var wg sync.WaitGroup
	
	// Transfer data from conn1 to conn2
	wg.Add(1)
	go func() {
		defer wg.Done()
		bytes, err := tm.copyData(conn2, conn1)
		tm.updateStats(func(stats *models.SessionStats) {
			stats.BytesSent += bytes
			now := time.Now()
			stats.LastActivityAt = &now
		})
		if err != nil && err != io.EOF {
			tm.logger.Debug("transfer error conn1->conn2", "error", err)
		}
		conn2.Close()
	}()
	
	// Transfer data from conn2 to conn1
	wg.Add(1)
	go func() {
		defer wg.Done()
		bytes, err := tm.copyData(conn1, conn2)
		tm.updateStats(func(stats *models.SessionStats) {
			stats.BytesReceived += bytes
			now := time.Now()
			stats.LastActivityAt = &now
		})
		if err != nil && err != io.EOF {
			tm.logger.Debug("transfer error conn2->conn1", "error", err)
		}
		conn1.Close()
	}()
	
	wg.Wait()
}

// copyData copies data from src to dst and returns bytes transferred
func (tm *TunnelManager) copyData(dst, src net.Conn) (int64, error) {
	// Set timeouts
	if tm.config.IdleTimeout > 0 {
		deadline := time.Now().Add(tm.config.IdleTimeout)
		src.SetReadDeadline(deadline)
		dst.SetWriteDeadline(deadline)
	}
	
	return io.Copy(dst, src)
}

// updateStats safely updates statistics
func (tm *TunnelManager) updateStats(fn func(*models.SessionStats)) {
	tm.statsMu.Lock()
	defer tm.statsMu.Unlock()
	fn(&tm.stats)
}

// SOCKS protocol handlers would be implemented here
// For brevity, I'll add placeholder implementations

// handleSOCKS4 handles SOCKS4 protocol
func (tm *TunnelManager) handleSOCKS4(conn net.Conn) (string, error) {
	// TODO: Implement SOCKS4 protocol
	return "", fmt.Errorf("SOCKS4 not implemented yet")
}

// handleSOCKS5 handles SOCKS5 protocol
func (tm *TunnelManager) handleSOCKS5(conn net.Conn) (string, error) {
	// TODO: Implement SOCKS5 protocol
	return "", fmt.Errorf("SOCKS5 not implemented yet")
}
