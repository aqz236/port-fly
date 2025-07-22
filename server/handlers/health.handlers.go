package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

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
