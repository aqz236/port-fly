package handlers

import (
	"github.com/aqz236/port-fly/core/manager"
	"github.com/aqz236/port-fly/core/utils"
	"github.com/aqz236/port-fly/server/storage"
)

// Handlers contains all HTTP handlers for the new Project->Group->Resource architecture
type Handlers struct {
	storage        storage.StorageInterface
	sessionManager *manager.SessionManager
	portManager    *PortManager
	logger         utils.Logger
}

// NewHandlers creates a new handlers instance
func NewHandlers(storage storage.StorageInterface, sessionManager *manager.SessionManager, logger utils.Logger) *Handlers {
	portManager := NewPortManager(sessionManager, logger)
	return &Handlers{
		storage:        storage,
		sessionManager: sessionManager,
		portManager:    portManager,
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
