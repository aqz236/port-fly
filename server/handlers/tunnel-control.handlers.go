package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

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
