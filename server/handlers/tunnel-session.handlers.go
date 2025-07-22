package handlers

import (
	"net/http"
	"strconv"

	"github.com/aqz236/port-fly/core/models"
	"github.com/gin-gonic/gin"
)

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
