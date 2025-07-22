package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/aqz236/port-fly/core/manager"
	"github.com/aqz236/port-fly/core/models"
	"github.com/aqz236/port-fly/core/utils"
	"github.com/aqz236/port-fly/server/storage"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	storage        storage.StorageInterface
	sessionManager *manager.SessionManager
	logger         utils.Logger
}

// NewHandlers creates a new handlers instance
func NewHandlers(storage storage.StorageInterface, sessionManager *manager.SessionManager, logger utils.Logger) *Handlers {
	return &Handlers{
		storage:        storage,
		sessionManager: sessionManager,
		logger:         logger,
	}
}

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// Health check endpoint
func (h *Handlers) Health(c *gin.Context) {
	// Check storage health
	if err := h.storage.Health(); err != nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Success: false,
			Error:   "Storage unhealthy: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: gin.H{
			"status": "healthy",
			"service": "portfly-api",
		},
	})
}

// Host Groups

func (h *Handlers) GetHostGroups(c *gin.Context) {
	groups, err := h.storage.GetHostGroups(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    groups,
	})
}

func (h *Handlers) CreateHostGroup(c *gin.Context) {
	var group models.HostGroup
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if err := h.storage.CreateHostGroup(c.Request.Context(), &group); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    group,
	})
}

func (h *Handlers) GetHostGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid group ID",
		})
		return
	}

	group, err := h.storage.GetHostGroup(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Host group not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    group,
	})
}

func (h *Handlers) UpdateHostGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid group ID",
		})
		return
	}

	var group models.HostGroup
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	group.ID = uint(id)
	if err := h.storage.UpdateHostGroup(c.Request.Context(), &group); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    group,
	})
}

func (h *Handlers) DeleteHostGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid group ID",
		})
		return
	}

	if err := h.storage.DeleteHostGroup(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Host group deleted successfully",
	})
}

func (h *Handlers) GetHostGroupStats(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid group ID",
		})
		return
	}

	stats, err := h.storage.GetHostGroupStats(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    stats,
	})
}

// Hosts

func (h *Handlers) GetHosts(c *gin.Context) {
	hosts, err := h.storage.GetHosts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    hosts,
	})
}

func (h *Handlers) CreateHost(c *gin.Context) {
	var host models.Host
	if err := c.ShouldBindJSON(&host); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if err := h.storage.CreateHost(c.Request.Context(), &host); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    host,
	})
}

func (h *Handlers) GetHost(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid host ID",
		})
		return
	}

	host, err := h.storage.GetHost(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Host not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    host,
	})
}

func (h *Handlers) UpdateHost(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid host ID",
		})
		return
	}

	var host models.Host
	if err := c.ShouldBindJSON(&host); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	host.ID = uint(id)
	if err := h.storage.UpdateHost(c.Request.Context(), &host); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    host,
	})
}

func (h *Handlers) DeleteHost(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid host ID",
		})
		return
	}

	if err := h.storage.DeleteHost(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Host deleted successfully",
	})
}

func (h *Handlers) SearchHosts(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Search query is required",
		})
		return
	}

	hosts, err := h.storage.SearchHosts(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    hosts,
	})
}

// Port Groups

func (h *Handlers) GetPortGroups(c *gin.Context) {
	groups, err := h.storage.GetPortGroups(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    groups,
	})
}

func (h *Handlers) CreatePortGroup(c *gin.Context) {
	var group models.PortGroup
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if err := h.storage.CreatePortGroup(c.Request.Context(), &group); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    group,
	})
}

func (h *Handlers) GetPortGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid group ID",
		})
		return
	}

	group, err := h.storage.GetPortGroup(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Port group not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    group,
	})
}

func (h *Handlers) UpdatePortGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid group ID",
		})
		return
	}

	var group models.PortGroup
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	group.ID = uint(id)
	if err := h.storage.UpdatePortGroup(c.Request.Context(), &group); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    group,
	})
}

func (h *Handlers) DeletePortGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid group ID",
		})
		return
	}

	if err := h.storage.DeletePortGroup(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Port group deleted successfully",
	})
}

func (h *Handlers) GetPortGroupStats(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid group ID",
		})
		return
	}

	stats, err := h.storage.GetPortGroupStats(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    stats,
	})
}

// Port Forwards

func (h *Handlers) GetPortForwards(c *gin.Context) {
	portForwards, err := h.storage.GetPortForwards(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    portForwards,
	})
}

func (h *Handlers) CreatePortForward(c *gin.Context) {
	var portForward models.PortForward
	if err := c.ShouldBindJSON(&portForward); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if err := h.storage.CreatePortForward(c.Request.Context(), &portForward); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    portForward,
	})
}

func (h *Handlers) GetPortForward(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid port forward ID",
		})
		return
	}

	portForward, err := h.storage.GetPortForward(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Port forward not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    portForward,
	})
}

func (h *Handlers) UpdatePortForward(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid port forward ID",
		})
		return
	}

	var portForward models.PortForward
	if err := c.ShouldBindJSON(&portForward); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	portForward.ID = uint(id)
	if err := h.storage.UpdatePortForward(c.Request.Context(), &portForward); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    portForward,
	})
}

func (h *Handlers) DeletePortForward(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid port forward ID",
		})
		return
	}

	if err := h.storage.DeletePortForward(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Port forward deleted successfully",
	})
}

func (h *Handlers) SearchPortForwards(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Search query is required",
		})
		return
	}

	portForwards, err := h.storage.SearchPortForwards(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    portForwards,
	})
}

// Tunnel Sessions

func (h *Handlers) GetTunnelSessions(c *gin.Context) {
	sessions, err := h.storage.GetTunnelSessions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    sessions,
	})
}

func (h *Handlers) CreateTunnelSession(c *gin.Context) {
	var session models.TunnelSession
	if err := c.ShouldBindJSON(&session); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if err := h.storage.CreateTunnelSession(c.Request.Context(), &session); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    session,
	})
}

func (h *Handlers) GetTunnelSession(c *gin.Context) {
	id := c.Param("id")

	session, err := h.storage.GetTunnelSession(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Tunnel session not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    session,
	})
}

func (h *Handlers) UpdateTunnelSession(c *gin.Context) {
	id := c.Param("id")

	var session models.TunnelSession
	if err := c.ShouldBindJSON(&session); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	session.ID = id
	if err := h.storage.UpdateTunnelSession(c.Request.Context(), &session); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    session,
	})
}

func (h *Handlers) DeleteTunnelSession(c *gin.Context) {
	id := c.Param("id")

	if err := h.storage.DeleteTunnelSession(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Tunnel session deleted successfully",
	})
}

func (h *Handlers) GetActiveTunnelSessions(c *gin.Context) {
	sessions, err := h.storage.GetActiveTunnelSessions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    sessions,
	})
}

// Tunnel Control (simplified for now)

func (h *Handlers) StartTunnel(c *gin.Context) {
	// TODO: Implement tunnel start logic
	c.JSON(http.StatusNotImplemented, Response{
		Success: false,
		Error:   "Tunnel start not implemented yet",
	})
}

func (h *Handlers) StopTunnel(c *gin.Context) {
	// TODO: Implement tunnel stop logic
	c.JSON(http.StatusNotImplemented, Response{
		Success: false,
		Error:   "Tunnel stop not implemented yet",
	})
}

// WebSocket Handler (simplified for now)

func (h *Handlers) WebSocketHandler(upgrader websocket.Upgrader) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			h.logger.Error("WebSocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		// TODO: Implement WebSocket logic for real-time updates
		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				h.logger.Error("WebSocket read error: %v", err)
				break
			}
			
			// Echo the message back for now
			if err := conn.WriteMessage(messageType, p); err != nil {
				h.logger.Error("WebSocket write error: %v", err)
				break
			}
		}
	}
}
