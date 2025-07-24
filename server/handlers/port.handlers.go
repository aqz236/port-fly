package handlers

import (
	"net/http"
	"strconv"

	"github.com/aqz236/port-fly/core/models"
	"github.com/gin-gonic/gin"
)

// ===== Port Operations =====

// GetPorts retrieves all ports with optional filtering
func (h *Handlers) GetPorts(c *gin.Context) {
	groupIDStr := c.Query("group_id")
	hostIDStr := c.Query("host_id")

	var ports []models.Port
	var err error

	if groupIDStr != "" {
		groupID, parseErr := strconv.ParseUint(groupIDStr, 10, 32)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Error:   "Invalid group_id parameter",
			})
			return
		}
		ports, err = h.storage.GetPortsByGroup(c.Request.Context(), uint(groupID))
	} else if hostIDStr != "" {
		hostID, parseErr := strconv.ParseUint(hostIDStr, 10, 32)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Error:   "Invalid host_id parameter",
			})
			return
		}
		ports, err = h.storage.GetPortsByHost(c.Request.Context(), uint(hostID))
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

// GetPort retrieves a single port by ID
func (h *Handlers) GetPort(c *gin.Context) {
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
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    port,
	})
}

// CreatePort creates a new port
func (h *Handlers) CreatePort(c *gin.Context) {
	var port models.Port
	if err := c.ShouldBindJSON(&port); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := h.storage.CreatePort(c.Request.Context(), &port); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    port,
	})
}

// UpdatePort updates an existing port
func (h *Handlers) UpdatePort(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid port ID",
		})
		return
	}

	// Get existing port
	existingPort, err := h.storage.GetPort(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Bind JSON to existing port
	if err := c.ShouldBindJSON(existingPort); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Ensure ID is not changed
	existingPort.ID = uint(id)

	if err := h.storage.UpdatePort(c.Request.Context(), existingPort); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    existingPort,
	})
}

// DeletePort deletes a port by ID
func (h *Handlers) DeletePort(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid port ID",
		})
		return
	}

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

// GetPortStats retrieves statistics for a port
func (h *Handlers) GetPortStats(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid port ID",
		})
		return
	}

	stats, err := h.storage.GetPortStats(c.Request.Context(), uint(id))
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

// SearchPorts searches for ports by name or description
func (h *Handlers) SearchPorts(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Search query parameter 'q' is required",
		})
		return
	}

	ports, err := h.storage.SearchPorts(c.Request.Context(), query)
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

// ===== Port Control Operations =====

// TestPortConnection tests the connection from a host to a port
func (h *Handlers) TestPortConnection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid port ID",
		})
		return
	}

	var request struct {
		HostID uint `json:"host_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Get port
	port, err := h.storage.GetPort(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Get host
	host, err := h.storage.GetHost(c.Request.Context(), request.HostID)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Host not found: " + err.Error(),
		})
		return
	}

	// TODO: Implement actual connection testing logic using SSH
	// For now, we'll simulate the test
	success := true // This should be replaced with actual testing logic

	// Update port status based on test result
	port.SetConnectionTestResult(success)
	port.HostID = &request.HostID

	if err := h.storage.UpdatePort(c.Request.Context(), port); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to update port status: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: gin.H{
			"port":            port,
			"host":            host,
			"connection_test": success,
			"message":         "Connection test completed",
		},
	})
}

// CreatePortForward creates a port forward connection between remote and local ports
func (h *Handlers) CreatePortForward(c *gin.Context) {
	var request struct {
		RemotePortID uint `json:"remote_port_id" binding:"required"`
		LocalPortID  uint `json:"local_port_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate ports exist and are correct types
	remotePort, err := h.storage.GetPort(c.Request.Context(), request.RemotePortID)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Remote port not found: " + err.Error(),
		})
		return
	}

	localPort, err := h.storage.GetPort(c.Request.Context(), request.LocalPortID)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Local port not found: " + err.Error(),
		})
		return
	}

	// Validate port types
	if !remotePort.IsRemotePort() {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Source port must be a remote_port",
		})
		return
	}

	if !localPort.IsLocalPort() {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Target port must be a local_port",
		})
		return
	}

	// Check if connection already exists
	existingConnection, _ := h.storage.GetPortConnectionByPorts(c.Request.Context(), request.RemotePortID, request.LocalPortID)
	if existingConnection != nil {
		c.JSON(http.StatusConflict, Response{
			Success: false,
			Error:   "Port connection already exists",
		})
		return
	}

	// Create port connection
	connection := &models.PortConnection{
		RemotePortID: request.RemotePortID,
		LocalPortID:  request.LocalPortID,
		Status:       models.PortStatusConnecting,
	}

	if err := h.storage.CreatePortConnection(c.Request.Context(), connection); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to create port connection: " + err.Error(),
		})
		return
	}

	// TODO: Start actual tunnel here using tunnel manager
	// For now, mark as active
	connection.Status = models.PortStatusActive

	if err := h.storage.UpdatePortConnection(c.Request.Context(), connection); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to update connection status: " + err.Error(),
		})
		return
	}

	// Update port statuses
	remotePort.UpdateStatus(models.PortStatusActive)
	localPort.UpdateStatus(models.PortStatusActive)

	h.storage.UpdatePort(c.Request.Context(), remotePort)
	h.storage.UpdatePort(c.Request.Context(), localPort)

	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    connection,
		Message: "Port forward connection created successfully",
	})
}

// RemovePortForward removes a port forward connection
func (h *Handlers) RemovePortForward(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid connection ID",
		})
		return
	}

	// Get connection
	connection, err := h.storage.GetPortConnection(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// TODO: Stop actual tunnel here using tunnel manager

	// Delete connection
	if err := h.storage.DeletePortConnection(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to delete port connection: " + err.Error(),
		})
		return
	}

	// Update port statuses
	if connection.RemotePort.IsActive() {
		connection.RemotePort.UpdateStatus(models.PortStatusAvailable)
		h.storage.UpdatePort(c.Request.Context(), &connection.RemotePort)
	}

	if connection.LocalPort.IsActive() {
		connection.LocalPort.UpdateStatus(models.PortStatusAvailable)
		h.storage.UpdatePort(c.Request.Context(), &connection.LocalPort)
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Port forward connection removed successfully",
	})
}

// UpdatePortStatus updates the status of a port
func (h *Handlers) UpdatePortStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid port ID",
		})
		return
	}

	var request struct {
		Status models.PortStatus `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := h.storage.UpdatePortStatus(c.Request.Context(), uint(id), request.Status); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Port status updated successfully",
	})
}
