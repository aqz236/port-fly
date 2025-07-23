package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"

	"github.com/aqz236/port-fly/core/models"
	sshpkg "github.com/aqz236/port-fly/core/ssh"
)

// TerminalMessage represents a terminal WebSocket message
type TerminalMessage struct {
	Type         string      `json:"type"`
	Data         interface{} `json:"data"`
	ConnectionID string      `json:"connectionId,omitempty"`
}

// TerminalConnectionParams represents terminal connection parameters
type TerminalConnectionParams struct {
	HostID int    `json:"hostId"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Shell  string `json:"shell"`
}

// TerminalResizeData represents terminal resize data
type TerminalResizeData struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

// TerminalSession represents an active terminal session
type TerminalSession struct {
	ID         string
	HostID     int
	WebSocket  *websocket.Conn
	WSMutex    sync.Mutex  // 添加WebSocket写入锁
	SSHClient  *sshpkg.SSHClient
	SSHSession *ssh.Session
	Stdin      io.WriteCloser
	Context    context.Context
	Cancel     context.CancelFunc
	CreatedAt  time.Time
}

// TerminalManager manages terminal sessions
type TerminalManager struct {
	sessions map[string]*TerminalSession
	mutex    sync.RWMutex
	handlers *Handlers
}

// NewTerminalManager creates a new terminal manager
func NewTerminalManager(handlers *Handlers) *TerminalManager {
	return &TerminalManager{
		sessions: make(map[string]*TerminalSession),
		handlers: handlers,
	}
}

// WebSocket upgrader for terminal connections
var terminalUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 生产环境中应该检查源
	},
}

// TerminalWebSocketHandler handles terminal WebSocket connections
func (h *Handlers) TerminalWebSocketHandler(terminalManager *TerminalManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取主机ID
		hostIDStr := c.Param("hostId")
		hostID, err := strconv.Atoi(hostIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Error:   "Invalid host ID",
			})
			return
		}

		// 升级到WebSocket连接
		conn, err := terminalUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			h.logger.Error("Failed to upgrade to WebSocket", "error", err)
			return
		}
		defer conn.Close()

		// 处理终端连接
		terminalManager.HandleTerminalConnection(hostID, conn)
	}
}

// HandleTerminalConnection handles a terminal WebSocket connection
func (tm *TerminalManager) HandleTerminalConnection(hostID int, ws *websocket.Conn) {
	sessionID := fmt.Sprintf("terminal_%d_%d", hostID, time.Now().UnixNano())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session := &TerminalSession{
		ID:        sessionID,
		HostID:    hostID,
		WebSocket: ws,
		Context:   ctx,
		Cancel:    cancel,
		CreatedAt: time.Now(),
	}

	tm.mutex.Lock()
	tm.sessions[sessionID] = session
	tm.mutex.Unlock()

	defer func() {
		tm.mutex.Lock()
		delete(tm.sessions, sessionID)
		tm.mutex.Unlock()

		if session.SSHSession != nil {
			session.SSHSession.Close()
		}
		if session.SSHClient != nil {
			session.SSHClient.Disconnect()
		}
	}()

	// 处理WebSocket消息
	for {
		var msg TerminalMessage
		err := ws.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				tm.handlers.logger.Error("WebSocket read error", "error", err)
			}
			break
		}

		switch msg.Type {
		case "terminal_connect":
			err := tm.handleTerminalConnect(session, msg.Data)
			if err != nil {
				tm.sendError(ws, fmt.Sprintf("连接失败: %v", err))
				return
			}

		case "terminal_data":
			if session.Stdin != nil {
				data, ok := msg.Data.(string)
				if ok {
					session.Stdin.Write([]byte(data))
				}
			}

		case "terminal_resize":
			if session.SSHSession != nil {
				var resizeData TerminalResizeData
				if dataBytes, err := json.Marshal(msg.Data); err == nil {
					if err := json.Unmarshal(dataBytes, &resizeData); err == nil {
						session.SSHSession.WindowChange(resizeData.Rows, resizeData.Cols)
					}
				}
			}

		case "terminal_disconnect":
			return
		}
	}
}

// handleTerminalConnect establishes SSH connection and starts terminal session
func (tm *TerminalManager) handleTerminalConnect(session *TerminalSession, data interface{}) error {
	// 获取主机信息
	host, err := tm.handlers.storage.GetHost(context.Background(), uint(session.HostID))
	if err != nil {
		return fmt.Errorf("failed to get host: %w", err)
	}

	// 创建SSH配置
	sshConfig := models.SSHConnectionConfig{
		Host:            host.Hostname,
		Port:            host.Port,
		Username:        host.Username,
		AuthMethod:      models.AuthMethod(host.AuthMethod),
		Password:        host.Password,
		PrivateKeyData:  []byte(host.PrivateKey),
		ConnectTimeout:  30 * time.Second,
		HostKeyCallback: "accept", // 终端连接接受所有主机密钥
	}

	// 创建SSH客户端
	sshClient := sshpkg.NewSSHClient(
		sshConfig,
		tm.handlers.logger.With("host_id", session.HostID),
	)

	// 连接SSH
	err = sshClient.Connect(session.Context)
	if err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}

	session.SSHClient = sshClient

	// 创建SSH会话
	client := sshClient.GetClient()
	if client == nil {
		return fmt.Errorf("SSH client not available")
	}

	sshSession, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}

	session.SSHSession = sshSession

	// 设置终端模式
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	// 解析连接参数
	var params TerminalConnectionParams
	if dataBytes, err := json.Marshal(data); err == nil {
		json.Unmarshal(dataBytes, &params)
	}

	if params.Width == 0 {
		params.Width = 80
	}
	if params.Height == 0 {
		params.Height = 24
	}
	if params.Shell == "" {
		params.Shell = "bash"
	}

	// 请求伪终端
	err = sshSession.RequestPty("xterm-256color", params.Height, params.Width, modes)
	if err != nil {
		return fmt.Errorf("failed to request pty: %w", err)
	}

	// 设置输入输出
	stdin, err := sshSession.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}
	session.Stdin = stdin

	stdout, err := sshSession.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := sshSession.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// 启动shell
	err = sshSession.Shell()
	if err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	// 启动输出读取goroutine
	go tm.handleTerminalOutput(session, stdout, "stdout")
	go tm.handleTerminalOutput(session, stderr, "stderr")

	// 发送连接成功消息
	tm.sendMessage(session.WebSocket, "terminal_connected", map[string]interface{}{
		"sessionId": session.ID,
		"hostId":    session.HostID,
	})

	// 更新主机状态为已连接
	host.Status = "connected"
	host.LastConnected = &session.CreatedAt
	tm.handlers.storage.UpdateHost(context.Background(), host)

	return nil
}

// handleTerminalOutput handles terminal output and sends to WebSocket
func (tm *TerminalManager) handleTerminalOutput(session *TerminalSession, reader io.Reader, outputType string) {
	buffer := make([]byte, 1024)

	for {
		select {
		case <-session.Context.Done():
			return
		default:
			n, err := reader.Read(buffer)
			if err != nil {
				if err != io.EOF {
					tm.handlers.logger.Error("Error reading terminal output", "error", err, "type", outputType)
				}
				return
			}

			if n > 0 {
				// 使用会话的WebSocket锁
				session.WSMutex.Lock()
				
				msg := TerminalMessage{
					Type: "terminal_data",
					Data: string(buffer[:n]),
				}

				err := session.WebSocket.WriteJSON(msg)
				session.WSMutex.Unlock()
				
				if err != nil {
					tm.handlers.logger.Error("Error sending terminal data", "error", err)
					return
				}
			}
		}
	}
}

// sendMessage sends a message to the WebSocket (with concurrency protection)
func (tm *TerminalManager) sendMessage(ws *websocket.Conn, msgType string, data interface{}) {
	// 获取会话以使用其锁
	var session *TerminalSession
	tm.mutex.RLock()
	for _, s := range tm.sessions {
		if s.WebSocket == ws {
			session = s
			break
		}
	}
	tm.mutex.RUnlock()

	if session == nil {
		tm.handlers.logger.Error("Session not found for WebSocket")
		return
	}

	// 使用会话的WebSocket锁
	session.WSMutex.Lock()
	defer session.WSMutex.Unlock()

	msg := TerminalMessage{
		Type: msgType,
		Data: data,
	}
	
	if err := ws.WriteJSON(msg); err != nil {
		tm.handlers.logger.Error("Failed to send WebSocket message", "error", err, "type", msgType)
	}
}

// sendError sends an error message to the WebSocket (with concurrency protection)
func (tm *TerminalManager) sendError(ws *websocket.Conn, message string) {
	// 获取会话以使用其锁
	var session *TerminalSession
	tm.mutex.RLock()
	for _, s := range tm.sessions {
		if s.WebSocket == ws {
			session = s
			break
		}
	}
	tm.mutex.RUnlock()

	if session == nil {
		tm.handlers.logger.Error("Session not found for WebSocket")
		return
	}

	// 使用会话的WebSocket锁
	session.WSMutex.Lock()
	defer session.WSMutex.Unlock()

	msg := TerminalMessage{
		Type: "terminal_error",
		Data: message,
	}
	
	if err := ws.WriteJSON(msg); err != nil {
		tm.handlers.logger.Error("Failed to send WebSocket error", "error", err, "message", message)
	}
}

// GetActiveSessions returns active terminal sessions
func (tm *TerminalManager) GetActiveSessions() map[string]*TerminalSession {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	sessions := make(map[string]*TerminalSession)
	for id, session := range tm.sessions {
		sessions[id] = session
	}
	return sessions
}

// GetSession returns a specific terminal session
func (tm *TerminalManager) GetSession(sessionID string) (*TerminalSession, bool) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	session, exists := tm.sessions[sessionID]
	return session, exists
}

// CloseSession closes a terminal session
func (tm *TerminalManager) CloseSession(sessionID string) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	session, exists := tm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.Cancel()
	delete(tm.sessions, sessionID)

	return nil
}

// CloseAllSessions closes all terminal sessions for a host
func (tm *TerminalManager) CloseAllSessions(hostID int) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	for sessionID, session := range tm.sessions {
		if session.HostID == hostID {
			session.Cancel()
			delete(tm.sessions, sessionID)
		}
	}
}

// GetSessionCount returns the number of active sessions
func (tm *TerminalManager) GetSessionCount() int {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	return len(tm.sessions)
}

// GetHostSessions returns all sessions for a specific host
func (tm *TerminalManager) GetHostSessions(hostID int) []*TerminalSession {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	var sessions []*TerminalSession
	for _, session := range tm.sessions {
		if session.HostID == hostID {
			sessions = append(sessions, session)
		}
	}
	return sessions
}
