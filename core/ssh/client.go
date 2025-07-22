package ssh

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/aqz236/port-fly/core/models"
	"github.com/aqz236/port-fly/core/utils"
)

// ConnectionPool manages SSH connections with pooling and reuse
type ConnectionPool struct {
	connections map[string]*PooledConnection
	mu          sync.RWMutex
	maxSize     int
	maxIdleTime time.Duration
	logger      utils.Logger
}

// PooledConnection represents a pooled SSH connection
type PooledConnection struct {
	client     *ssh.Client
	config     models.SSHConnectionConfig
	createdAt  time.Time
	lastUsed   time.Time
	inUse      bool
	mu         sync.RWMutex
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxSize int, maxIdleTime time.Duration, logger utils.Logger) *ConnectionPool {
	pool := &ConnectionPool{
		connections: make(map[string]*PooledConnection),
		maxSize:     maxSize,
		maxIdleTime: maxIdleTime,
		logger:      logger,
	}
	
	// Start cleanup goroutine
	go pool.cleanup()
	
	return pool
}

// SSHClient wraps SSH client functionality
type SSHClient struct {
	config      models.SSHConnectionConfig
	client      *ssh.Client
	authManager *AuthManager
	pool        *ConnectionPool
	logger      utils.Logger
	connected   bool
	mu          sync.RWMutex
}

// NewSSHClient creates a new SSH client
func NewSSHClient(config models.SSHConnectionConfig, logger utils.Logger) *SSHClient {
	return &SSHClient{
		config:      config,
		authManager: NewAuthManager(),
		logger:      logger,
	}
}

// NewSSHClientWithPool creates a new SSH client with connection pooling
func NewSSHClientWithPool(config models.SSHConnectionConfig, pool *ConnectionPool, logger utils.Logger) *SSHClient {
	return &SSHClient{
		config:      config,
		authManager: NewAuthManager(),
		pool:        pool,
		logger:      logger,
	}
}

// Connect establishes an SSH connection
func (c *SSHClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.connected && c.client != nil {
		return nil
	}
	
	// Validate configuration
	if err := c.authManager.ValidateConfig(c.config); err != nil {
		return fmt.Errorf("invalid SSH configuration: %w", err)
	}
	
	// Try to get connection from pool first
	if c.pool != nil {
		if conn, err := c.pool.Get(c.config); err == nil {
			c.client = conn
			c.connected = true
			c.logger.Info("reused pooled SSH connection", 
				"host", c.config.Host, 
				"user", c.config.Username)
			return nil
		}
	}
	
	// Create new connection
	client, err := c.createConnection(ctx)
	if err != nil {
		return err
	}
	
	c.client = client
	c.connected = true
	
	c.logger.Info("established SSH connection", 
		"host", c.config.Host, 
		"user", c.config.Username)
	
	return nil
}

// createConnection creates a new SSH connection
func (c *SSHClient) createConnection(ctx context.Context) (*ssh.Client, error) {
	// Get authentication methods
	authMethods, err := c.authManager.GetAuthMethods(c.config)
	if err != nil {
		// Try all available methods if specific method fails
		authMethods = c.authManager.GetAllAuthMethods(c.config)
		if len(authMethods) == 0 {
			return nil, fmt.Errorf("no authentication methods available: %w", err)
		}
	}
	
	// Get host key callback
	hostKeyCallback, err := c.authManager.HostKeyCallback(
		c.config.HostKeyCallback, 
		c.config.KnownHostsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create host key callback: %w", err)
	}
	
	// Create SSH client configuration
	clientVersion := c.config.ClientVersion
	if clientVersion == "" {
		clientVersion = "SSH-2.0-PortFly"
	}
	
	sshConfig := &ssh.ClientConfig{
		User:            c.config.Username,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		ClientVersion:   clientVersion,
		Timeout:         c.config.ConnectTimeout,
	}
	
	// Set cipher preferences if specified
	if len(c.config.Extensions) > 0 {
		// Handle SSH extensions/algorithms configuration
		// This would require additional implementation
	}
	
	// Create connection with context
	address := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	
	dialer := &net.Dialer{
		Timeout: c.config.ConnectTimeout,
	}
	
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", address, err)
	}
	
	// Perform SSH handshake
	sshConn, channels, requests, err := ssh.NewClientConn(conn, address, sshConfig)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("SSH handshake failed: %w", err)
	}
	
	client := ssh.NewClient(sshConn, channels, requests)
	
	// Add to pool if available
	if c.pool != nil {
		c.pool.Put(c.config, client)
	}
	
	return client, nil
}

// Disconnect closes the SSH connection
func (c *SSHClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.connected || c.client == nil {
		return nil
	}
	
	// Return to pool instead of closing if pool is available
	if c.pool != nil {
		c.pool.Return(c.config, c.client)
		c.client = nil
		c.connected = false
		return nil
	}
	
	err := c.client.Close()
	c.client = nil
	c.connected = false
	
	if err != nil {
		c.logger.Error("error closing SSH connection", "error", err)
		return err
	}
	
	c.logger.Info("closed SSH connection", 
		"host", c.config.Host, 
		"user", c.config.Username)
	
	return nil
}

// IsConnected returns whether the client is connected
func (c *SSHClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected && c.client != nil
}

// GetClient returns the underlying SSH client
func (c *SSHClient) GetClient() *ssh.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client
}

// Reconnect reconnects the SSH client
func (c *SSHClient) Reconnect(ctx context.Context) error {
	c.logger.Info("reconnecting SSH client", 
		"host", c.config.Host, 
		"user", c.config.Username)
	
	// Disconnect first
	if err := c.Disconnect(); err != nil {
		c.logger.Warn("error during disconnect for reconnect", "error", err)
	}
	
	// Reconnect with retry logic
	var lastErr error
	for i := 0; i < c.config.MaxRetries; i++ {
		if err := c.Connect(ctx); err == nil {
			c.logger.Info("reconnected successfully", 
				"host", c.config.Host, 
				"attempt", i+1)
			return nil
		} else {
			lastErr = err
			c.logger.Warn("reconnect attempt failed", 
				"host", c.config.Host, 
				"attempt", i+1, 
				"error", err)
			
			if i < c.config.MaxRetries-1 {
				select {
				case <-time.After(c.config.RetryInterval):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
	
	return fmt.Errorf("failed to reconnect after %d attempts: %w", c.config.MaxRetries, lastErr)
}

// Connection Pool Methods

// getConnectionKey returns a unique key for the connection
func (cp *ConnectionPool) getConnectionKey(config models.SSHConnectionConfig) string {
	return fmt.Sprintf("%s@%s:%d", config.Username, config.Host, config.Port)
}

// Get retrieves a connection from the pool
func (cp *ConnectionPool) Get(config models.SSHConnectionConfig) (*ssh.Client, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	
	key := cp.getConnectionKey(config)
	conn, exists := cp.connections[key]
	
	if !exists {
		return nil, fmt.Errorf("connection not found in pool")
	}
	
	conn.mu.Lock()
	defer conn.mu.Unlock()
	
	if conn.inUse {
		return nil, fmt.Errorf("connection is already in use")
	}
	
	// Check if connection is still alive
	if conn.client == nil {
		delete(cp.connections, key)
		return nil, fmt.Errorf("connection is dead")
	}
	
	conn.inUse = true
	conn.lastUsed = time.Now()
	
	return conn.client, nil
}

// Put adds a connection to the pool
func (cp *ConnectionPool) Put(config models.SSHConnectionConfig, client *ssh.Client) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	
	if len(cp.connections) >= cp.maxSize {
		// Pool is full, close the connection
		client.Close()
		return
	}
	
	key := cp.getConnectionKey(config)
	
	cp.connections[key] = &PooledConnection{
		client:    client,
		config:    config,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
		inUse:     true, // Initially in use by the creator
	}
	
	cp.logger.Debug("added connection to pool", "key", key)
}

// Return returns a connection to the pool
func (cp *ConnectionPool) Return(config models.SSHConnectionConfig, client *ssh.Client) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	
	key := cp.getConnectionKey(config)
	conn, exists := cp.connections[key]
	
	if !exists || conn.client != client {
		// Connection not in pool, close it
		client.Close()
		return
	}
	
	conn.mu.Lock()
	conn.inUse = false
	conn.lastUsed = time.Now()
	conn.mu.Unlock()
	
	cp.logger.Debug("returned connection to pool", "key", key)
}

// cleanup removes idle and dead connections from the pool
func (cp *ConnectionPool) cleanup() {
	ticker := time.NewTicker(cp.maxIdleTime / 2)
	defer ticker.Stop()
	
	for range ticker.C {
		cp.mu.Lock()
		now := time.Now()
		
		for key, conn := range cp.connections {
			conn.mu.RLock()
			shouldRemove := !conn.inUse && now.Sub(conn.lastUsed) > cp.maxIdleTime
			conn.mu.RUnlock()
			
			if shouldRemove {
				if conn.client != nil {
					conn.client.Close()
				}
				delete(cp.connections, key)
				cp.logger.Debug("removed idle connection from pool", "key", key)
			}
		}
		
		cp.mu.Unlock()
	}
}

// Close closes all connections in the pool
func (cp *ConnectionPool) Close() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	
	for key, conn := range cp.connections {
		if conn.client != nil {
			conn.client.Close()
		}
		delete(cp.connections, key)
	}
	
	cp.logger.Info("closed connection pool")
}

// Stats returns pool statistics
func (cp *ConnectionPool) Stats() map[string]interface{} {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	
	total := len(cp.connections)
	inUse := 0
	
	for _, conn := range cp.connections {
		conn.mu.RLock()
		if conn.inUse {
			inUse++
		}
		conn.mu.RUnlock()
	}
	
	return map[string]interface{}{
		"total":     total,
		"in_use":    inUse,
		"available": total - inUse,
		"max_size":  cp.maxSize,
	}
}
