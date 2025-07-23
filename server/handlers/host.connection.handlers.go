package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/aqz236/port-fly/core/models"
	sshpkg "github.com/aqz236/port-fly/core/ssh"
)

// SSHExecRequest represents SSH command execution request
type SSHExecRequest struct {
	Command string `json:"command" binding:"required"`
	Timeout int    `json:"timeout"` // in milliseconds
}

// SSHExecResponse represents SSH command execution response
type SSHExecResponse struct {
	Success  bool   `json:"success"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
	Duration int64  `json:"duration"` // in milliseconds
}

// ConnectHost connects to a host via SSH
func (h *Handlers) ConnectHost(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid host ID",
		})
		return
	}

	// 获取主机信息
	host, err := h.storage.GetHost(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Host not found",
		})
		return
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
		HostKeyCallback: "accept", // 对于API连接，接受所有主机密钥
	}

	// 创建SSH客户端
	sshClient := sshpkg.NewSSHClient(
		sshConfig,
		h.logger.With("host_id", id),
	)

	// 更新主机状态为连接中
	host.Status = "connecting"
	if err := h.storage.UpdateHost(c.Request.Context(), host); err != nil {
		h.logger.Error("Failed to update host status", "error", err)
	}

	// 尝试连接
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	err = sshClient.Connect(ctx)
	if err != nil {
		// 连接失败，更新状态
		host.Status = "error"
		h.storage.UpdateHost(c.Request.Context(), host)

		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 连接成功，更新状态
	now := time.Now()
	host.Status = "connected"
	host.LastConnected = &now
	host.ConnectionCount++
	
	if err := h.storage.UpdateHost(c.Request.Context(), host); err != nil {
		h.logger.Error("Failed to update host after successful connection", "error", err)
	}

	// 断开连接（这只是测试连接）
	sshClient.Disconnect()

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Successfully connected to host",
		Data:    host,
	})
}

// DisconnectHost disconnects from a host
func (h *Handlers) DisconnectHost(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid host ID",
		})
		return
	}

	// 获取主机信息
	host, err := h.storage.GetHost(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Host not found",
		})
		return
	}

	// 更新主机状态为断开
	host.Status = "disconnected"
	if err := h.storage.UpdateHost(c.Request.Context(), host); err != nil {
		h.logger.Error("Failed to update host status", "error", err)
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Host disconnected",
		Data:    host,
	})
}

// ExecuteSSHCommand executes a command on a host via SSH
func (h *Handlers) ExecuteSSHCommand(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid host ID",
		})
		return
	}

	var req SSHExecRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 获取主机信息
	host, err := h.storage.GetHost(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Host not found",
		})
		return
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
		HostKeyCallback: "accept",
	}

	// 创建SSH客户端
	sshClient := sshpkg.NewSSHClient(
		sshConfig,
		h.logger.With("host_id", id),
	)

	// 设置超时
	timeout := time.Duration(req.Timeout) * time.Millisecond
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	// 连接SSH
	err = sshClient.Connect(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "SSH connection failed: " + err.Error(),
		})
		return
	}
	defer sshClient.Disconnect()

	// 执行命令
	startTime := time.Now()
	
	client := sshClient.GetClient()
	if client == nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "SSH client not available",
		})
		return
	}

	session, err := client.NewSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to create SSH session: " + err.Error(),
		})
		return
	}
	defer session.Close()

	// 执行命令并获取输出
	output, err := session.CombinedOutput(req.Command)
	duration := time.Since(startTime)

	response := SSHExecResponse{
		Success:  err == nil,
		Duration: duration.Milliseconds(),
	}

	if err != nil {
		// 命令执行失败
		response.Stderr = err.Error()
		response.ExitCode = 1
		if len(output) > 0 {
			response.Stdout = string(output)
		}
	} else {
		// 命令执行成功
		response.Stdout = string(output)
		response.ExitCode = 0
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    response,
	})
}

// TestHostConnection tests connection to a host
func (h *Handlers) TestHostConnection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid host ID",
		})
		return
	}

	// 获取主机信息
	host, err := h.storage.GetHost(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Host not found",
		})
		return
	}

	// 创建SSH配置
	sshConfig := models.SSHConnectionConfig{
		Host:            host.Hostname,
		Port:            host.Port,
		Username:        host.Username,
		AuthMethod:      models.AuthMethod(host.AuthMethod),
		Password:        host.Password,
		PrivateKeyData:  []byte(host.PrivateKey),
		ConnectTimeout:  10 * time.Second,
		HostKeyCallback: "accept",
	}

	// 创建SSH客户端
	sshClient := sshpkg.NewSSHClient(
		sshConfig,
		h.logger.With("host_id", id),
	)

	// 测试连接
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	err = sshClient.Connect(ctx)
	if err != nil {
		c.JSON(http.StatusOK, Response{
			Success: false,
			Message: "Connection failed: " + err.Error(),
		})
		return
	}

	// 连接成功，立即断开
	sshClient.Disconnect()

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Connection test successful",
	})
}
