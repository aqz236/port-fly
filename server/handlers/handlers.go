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

// Handlers contains all HTTP handlers for the new Project->Group->Resource architecture
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
			"status":  "healthy",
			"service": "portfly-api",
		},
	})
}

// ===== Project Operations =====

func (h *Handlers) GetProjects(c *gin.Context) {
	projects, err := h.storage.GetProjects(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    projects,
	})
}

func (h *Handlers) CreateProject(c *gin.Context) {
	var project models.Project
	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if err := h.storage.CreateProject(c.Request.Context(), &project); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    project,
	})
}

func (h *Handlers) GetProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid project ID",
		})
		return
	}

	project, err := h.storage.GetProject(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Project not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    project,
	})
}

func (h *Handlers) UpdateProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid project ID",
		})
		return
	}

	var project models.Project
	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	project.ID = uint(id)
	if err := h.storage.UpdateProject(c.Request.Context(), &project); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    project,
	})
}

func (h *Handlers) DeleteProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid project ID",
		})
		return
	}

	if err := h.storage.DeleteProject(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Project deleted successfully",
	})
}

func (h *Handlers) GetProjectStats(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid project ID",
		})
		return
	}

	stats, err := h.storage.GetProjectStats(c.Request.Context(), uint(id))
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

// ===== Group Operations =====

func (h *Handlers) GetGroups(c *gin.Context) {
	groups, err := h.storage.GetGroups(c.Request.Context())
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

func (h *Handlers) GetGroupsByProject(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("projectId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid project ID",
		})
		return
	}

	groups, err := h.storage.GetGroupsByProject(c.Request.Context(), uint(projectID))
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

func (h *Handlers) CreateGroup(c *gin.Context) {
	var group models.Group
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if err := h.storage.CreateGroup(c.Request.Context(), &group); err != nil {
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

func (h *Handlers) GetGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid group ID",
		})
		return
	}

	group, err := h.storage.GetGroup(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Group not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    group,
	})
}

func (h *Handlers) UpdateGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid group ID",
		})
		return
	}

	var group models.Group
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	group.ID = uint(id)
	if err := h.storage.UpdateGroup(c.Request.Context(), &group); err != nil {
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

func (h *Handlers) DeleteGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid group ID",
		})
		return
	}

	if err := h.storage.DeleteGroup(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Group deleted successfully",
	})
}

func (h *Handlers) GetGroupStats(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid group ID",
		})
		return
	}

	stats, err := h.storage.GetGroupStats(c.Request.Context(), uint(id))
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

// ===== Host Operations =====

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

func (h *Handlers) GetHostsByGroup(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid group ID",
		})
		return
	}

	hosts, err := h.storage.GetHostsByGroup(c.Request.Context(), uint(groupID))
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

func (h *Handlers) GetHostStats(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid host ID",
		})
		return
	}

	stats, err := h.storage.GetHostStats(c.Request.Context(), uint(id))
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

// ===== Port Forward Operations =====

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

func (h *Handlers) GetPortForwardsByGroup(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("groupId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid group ID",
		})
		return
	}

	portForwards, err := h.storage.GetPortForwardsByGroup(c.Request.Context(), uint(groupID))
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

func (h *Handlers) GetPortForwardsByHost(c *gin.Context) {
	hostID, err := strconv.ParseUint(c.Param("hostId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid host ID",
		})
		return
	}

	portForwards, err := h.storage.GetPortForwardsByHost(c.Request.Context(), uint(hostID))
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

func (h *Handlers) GetPortForwardStats(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid port forward ID",
		})
		return
	}

	stats, err := h.storage.GetPortForwardStats(c.Request.Context(), uint(id))
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

// ===== Tunnel Session Operations =====

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
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid session ID",
		})
		return
	}

	session, err := h.storage.GetTunnelSession(c.Request.Context(), uint(id))
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
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid session ID",
		})
		return
	}

	var session models.TunnelSession
	if err := c.ShouldBindJSON(&session); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	session.ID = uint(id)
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
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid session ID",
		})
		return
	}

	if err := h.storage.DeleteTunnelSession(c.Request.Context(), uint(id)); err != nil {
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

// ===== Tunnel Control =====

func (h *Handlers) StartTunnel(c *gin.Context) {
	// TODO: Implement tunnel start logic using sessionManager
	c.JSON(http.StatusNotImplemented, Response{
		Success: false,
		Error:   "Tunnel start not implemented yet",
	})
}

func (h *Handlers) StopTunnel(c *gin.Context) {
	// TODO: Implement tunnel stop logic using sessionManager
	c.JSON(http.StatusNotImplemented, Response{
		Success: false,
		Error:   "Tunnel stop not implemented yet",
	})
}

// ===== WebSocket Handler =====

func (h *Handlers) WebSocketHandler(upgrader websocket.Upgrader) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			h.logger.Error("WebSocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		// TODO: Implement WebSocket logic for real-time updates
		// This should include:
		// - Session status updates
		// - Host connection status
		// - Port forward status
		// - Real-time logs

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
