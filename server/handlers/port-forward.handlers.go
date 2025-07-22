package handlers

import (
	"net/http"
	"strconv"

	"github.com/aqz236/port-fly/core/models"
	"github.com/gin-gonic/gin"
)

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
