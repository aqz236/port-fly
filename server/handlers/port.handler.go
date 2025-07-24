package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/aqz236/port-fly/core/manager"
	"github.com/aqz236/port-fly/core/models"
	"github.com/aqz236/port-fly/core/utils"
	"github.com/gin-gonic/gin"
)

// PortManager manages active port forwarding sessions
type PortManager struct {
	activePorts    map[uint]*ActivePort
	mu             sync.RWMutex
	sessionManager *manager.SessionManager
	logger         utils.Logger
}

// ActivePort represents an active port forwarding session
type ActivePort struct {
	Port      *models.Port
	SessionID string
	StartTime time.Time
	Status    string
	cancel    context.CancelFunc
}

// NewPortManager creates a new port manager
func NewPortManager(sessionManager *manager.SessionManager, logger utils.Logger) *PortManager {
	return &PortManager{
		activePorts:    make(map[uint]*ActivePort),
		sessionManager: sessionManager,
		logger:         logger,
	}
}

// ===== Port V3 Operations with Real SSH Tunneling =====

func (h *Handlers) CreatePortV3(c *gin.Context) {
	var req models.CreatePortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 创建端口对象
	port := &models.Port{
		Name:        req.Name,
		Type:        req.Type,
		Port:        req.Port,
		TargetHost:  req.TargetHost,
		TargetPort:  req.TargetPort,
		Description: req.Description,
		AutoStart:   req.AutoStart,
		Status:      "inactive",
		Color:       req.Color,
		Icon:        req.Icon,
		IsVisible:   true,
		Tags:        req.Tags,
		GroupID:     req.GroupID,
		HostID:      req.HostID,
	}

	// 验证端口对象
	if err := port.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 使用 storage 创建端口
	if err := h.storage.CreatePort(c.Request.Context(), port); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 如果设置了自动启动，尝试启动端口转发
	if port.AutoStart {
		if err := h.startPortForwarding(c.Request.Context(), port); err != nil {
			h.logger.Error("Failed to auto-start port forwarding", "port_id", port.ID, "error", err)
			// 不返回错误，只记录日志
		}
	}

	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    port,
	})
}

func (h *Handlers) GetPortsV3(c *gin.Context) {
	groupID := c.Query("group_id")
	hostID := c.Query("host_id")

	var ports []models.Port
	var err error

	if groupID != "" {
		gid, parseErr := strconv.ParseUint(groupID, 10, 32)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Error:   "Invalid group_id",
			})
			return
		}
		ports, err = h.storage.GetPortsByGroup(c.Request.Context(), uint(gid))
	} else if hostID != "" {
		hid, parseErr := strconv.ParseUint(hostID, 10, 32)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Error:   "Invalid host_id",
			})
			return
		}
		ports, err = h.storage.GetPortsByHost(c.Request.Context(), uint(hid))
	} else {
		ports, err = h.storage.GetPorts(c.Request.Context())
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    ports,
	})
}

func (h *Handlers) GetPortV3(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid port ID",
		})
		return
	}

	port, err := h.storage.GetPort(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Port not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    port,
	})
}

func (h *Handlers) UpdatePortV3(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid port ID",
		})
		return
	}

	// 获取现有端口
	port, err := h.storage.GetPort(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Port not found",
		})
		return
	}

	// 绑定更新数据
	var req models.UpdatePortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 检查是否有活跃的转发
	wasActive := port.Status == "active"

	// 更新字段
	if req.Name != "" {
		port.Name = req.Name
	}
	if req.Description != "" {
		port.Description = req.Description
	}
	port.AutoStart = req.AutoStart
	if req.Color != "" {
		port.Color = req.Color
	}
	if req.Icon != "" {
		port.Icon = req.Icon
	}
	port.IsVisible = req.IsVisible
	if len(req.Tags) > 0 {
		port.Tags = req.Tags
	}

	// 验证
	if err := port.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 如果端口转发配置发生变化且当前是活跃状态，需要重启
	if wasActive && (req.Port != 0 || req.TargetHost != "" || req.TargetPort != 0) {
		// 停止现有转发
		if err := h.stopPortForwarding(uint(id)); err != nil {
			h.logger.Error("Failed to stop port forwarding for update", "port_id", id, "error", err)
		}

		// 更新端口配置
		if req.Port != 0 {
			port.Port = req.Port
		}
		if req.TargetHost != "" {
			port.TargetHost = req.TargetHost
		}
		if req.TargetPort != 0 {
			port.TargetPort = req.TargetPort
		}

		// 保存更新
		if err := h.storage.UpdatePort(c.Request.Context(), port); err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Error:   err.Error(),
			})
			return
		}

		// 重新启动转发
		if err := h.startPortForwarding(c.Request.Context(), port); err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Error:   fmt.Sprintf("Failed to restart port forwarding: %v", err),
			})
			return
		}
	} else {
		// 保存更新
		if err := h.storage.UpdatePort(c.Request.Context(), port); err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Error:   err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    port,
	})
}

func (h *Handlers) DeletePortV3(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid port ID",
		})
		return
	}

	// 检查端口是否存在
	port, err := h.storage.GetPort(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Port not found",
		})
		return
	}

	// 如果端口转发是活跃的，先停止它
	if port.Status == "active" {
		if err := h.stopPortForwarding(uint(id)); err != nil {
			h.logger.Error("Failed to stop port forwarding before deletion", "port_id", id, "error", err)
		}
	}

	// 删除端口
	if err := h.storage.DeletePort(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Port deleted successfully",
	})
}

func (h *Handlers) ControlPortV3(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid port ID",
		})
		return
	}

	var req models.PortControlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 获取端口
	port, err := h.storage.GetPort(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Port not found",
		})
		return
	}

	// 执行控制操作
	switch req.Action {
	case "start":
		if err := h.startPortForwarding(c.Request.Context(), port); err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Error:   err.Error(),
			})
			return
		}
	case "stop":
		if err := h.stopPortForwarding(uint(id)); err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Error:   err.Error(),
			})
			return
		}
	case "restart":
		// 先停止
		if port.Status == "active" {
			if err := h.stopPortForwarding(uint(id)); err != nil {
				c.JSON(http.StatusInternalServerError, Response{
					Success: false,
					Error:   fmt.Sprintf("Failed to stop port forwarding: %v", err),
				})
				return
			}
		}
		// 再启动
		if err := h.startPortForwarding(c.Request.Context(), port); err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Error:   fmt.Sprintf("Failed to start port forwarding: %v", err),
			})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid action. Must be 'start', 'stop', or 'restart'",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    port,
	})
}

// GetPortStatsV3 获取端口统计信息
func (h *Handlers) GetPortStatsV3(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid port ID",
		})
		return
	}

	// 获取端口
	port, err := h.storage.GetPort(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Port not found",
		})
		return
	}

	// 构建响应数据
	stats := port.GetStats()

	// 如果有活跃的转发会话，获取实时统计
	if h.portManager != nil {
		if activePort := h.portManager.getActivePort(uint(id)); activePort != nil && activePort.SessionID != "" {
			if sessionStats, err := h.sessionManager.GetSessionStats(activePort.SessionID); err == nil {
				stats["connections"] = sessionStats.ActiveConnections
				stats["bytes_sent"] = sessionStats.BytesSent
				stats["bytes_received"] = sessionStats.BytesReceived
				stats["uptime"] = time.Since(activePort.StartTime).Seconds()
			}
		}
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    stats,
	})
}

// ===== 内部辅助方法 =====

// startPortForwarding 启动端口转发
func (h *Handlers) startPortForwarding(ctx context.Context, port *models.Port) error {
	// 检查是否已经在运行
	if port.Status == "active" {
		return fmt.Errorf("port forwarding is already active")
	}

	// 获取主机信息
	host, err := h.storage.GetHost(ctx, port.HostID)
	if err != nil {
		return fmt.Errorf("failed to get host: %v", err)
	}

	// 更新状态为连接中
	port.Status = "connecting"
	if err := h.storage.UpdatePort(ctx, port); err != nil {
		h.logger.Error("Failed to update port status to connecting", "port_id", port.ID, "error", err)
	}

	// 构建 SSH 连接配置
	sshConfig := models.SSHConnectionConfig{
		Host:            host.Hostname,
		Port:            host.Port,
		Username:        host.Username,
		Password:        host.Password,
		PrivateKeyPath:  host.PrivateKey,
		ConnectTimeout:  30 * time.Second,
	}

	// 构建隧道配置
	var tunnelConfig models.TunnelConfig
	switch port.Type {
	case "local":
		tunnelConfig = models.TunnelConfig{
			Type:             "local",
			LocalBindAddress: "127.0.0.1",
			LocalPort:        port.Port,
			RemoteHost:       port.TargetHost,
			RemotePort:       port.TargetPort,
		}
	case "remote":
		tunnelConfig = models.TunnelConfig{
			Type:              "remote",
			RemoteBindAddress: "0.0.0.0",
			RemotePort:        port.Port,
			RemoteHost:        port.TargetHost,
			LocalPort:         port.TargetPort,
		}
	default:
		port.Status = "error"
		if err := h.storage.UpdatePort(ctx, port); err != nil {
			h.logger.Error("Failed to update port status to error", "port_id", port.ID, "error", err)
		}
		return fmt.Errorf("unsupported port type: %s", port.Type)
	}

	// 创建 SSH 会话
	session, err := h.sessionManager.CreateSession(sshConfig, tunnelConfig)
	if err != nil {
		port.Status = "error"
		if updateErr := h.storage.UpdatePort(ctx, port); updateErr != nil {
			h.logger.Error("Failed to update port status to error", "port_id", port.ID, "error", updateErr)
		}
		return fmt.Errorf("failed to create SSH session: %v", err)
	}

	// 启动会话
	if err := h.sessionManager.StartSession(session.ID); err != nil {
		port.Status = "error"
		if updateErr := h.storage.UpdatePort(ctx, port); updateErr != nil {
			h.logger.Error("Failed to update port status to error", "port_id", port.ID, "error", updateErr)
		}
		// 清理会话
		if delErr := h.sessionManager.DeleteSession(session.ID); delErr != nil {
			h.logger.Error("Failed to cleanup session after start failure", "session_id", session.ID, "error", delErr)
		}
		return fmt.Errorf("failed to start SSH session: %v", err)
	}

	// 更新端口状态为活跃
	port.Status = "active"
	if err := h.storage.UpdatePort(ctx, port); err != nil {
		h.logger.Error("Failed to update port status to active", "port_id", port.ID, "error", err)
	}

	// 记录活跃端口（如果有端口管理器）
	if h.portManager != nil {
		h.portManager.addActivePort(port.ID, &ActivePort{
			Port:      port,
			SessionID: session.ID,
			StartTime: time.Now(),
			Status:    "active",
		})
	}

	h.logger.Info("Port forwarding started successfully", 
		"port_id", port.ID, 
		"type", port.Type, 
		"local_port", port.Port,
		"target", fmt.Sprintf("%s:%d", port.TargetHost, port.TargetPort))

	return nil
}

// stopPortForwarding 停止端口转发
func (h *Handlers) stopPortForwarding(portID uint) error {
	// 获取端口信息
	port, err := h.storage.GetPort(context.Background(), portID)
	if err != nil {
		return fmt.Errorf("failed to get port: %v", err)
	}

	// 更新状态为非活跃
	port.Status = "inactive"
	if err := h.storage.UpdatePort(context.Background(), port); err != nil {
		h.logger.Error("Failed to update port status to inactive", "port_id", portID, "error", err)
	}

	// 查找并停止相关的 SSH 会话
	if h.portManager != nil {
		if activePort := h.portManager.getActivePort(portID); activePort != nil {
			if activePort.SessionID != "" {
				if err := h.sessionManager.StopSession(activePort.SessionID); err != nil {
					h.logger.Error("Failed to stop SSH session", "session_id", activePort.SessionID, "error", err)
				}
				if err := h.sessionManager.DeleteSession(activePort.SessionID); err != nil {
					h.logger.Error("Failed to delete SSH session", "session_id", activePort.SessionID, "error", err)
				}
			}
			h.portManager.removeActivePort(portID)
		}
	}

	h.logger.Info("Port forwarding stopped successfully", "port_id", portID)
	return nil
}

// ===== PortManager 方法 =====

func (pm *PortManager) addActivePort(portID uint, activePort *ActivePort) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.activePorts[portID] = activePort
}

func (pm *PortManager) removeActivePort(portID uint) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if activePort, exists := pm.activePorts[portID]; exists {
		if activePort.cancel != nil {
			activePort.cancel()
		}
		delete(pm.activePorts, portID)
	}
}

func (pm *PortManager) getActivePort(portID uint) *ActivePort {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.activePorts[portID]
}

func (pm *PortManager) getAllActivePorts() map[uint]*ActivePort {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	result := make(map[uint]*ActivePort)
	for k, v := range pm.activePorts {
		result[k] = v
	}
	return result
}
