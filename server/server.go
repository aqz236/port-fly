package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/aqz236/port-fly/core/manager"
	"github.com/aqz236/port-fly/core/utils"
	"github.com/aqz236/port-fly/server/handlers"
	"github.com/aqz236/port-fly/server/middleware"
	"github.com/aqz236/port-fly/server/storage"
)

// Server represents the HTTP API server
type Server struct {
	config          *Config
	router          *gin.Engine
	storage         storage.StorageInterface
	sessionManager  *manager.SessionManager
	terminalManager *handlers.TerminalManager
	logger          utils.Logger
	upgrader        websocket.Upgrader
}

// Config holds server configuration
type Config struct {
	Host            string                `json:"host"`
	Port            int                   `json:"port"`
	Mode            string                `json:"mode"` // debug, release, test
	EnableCORS      bool                  `json:"enable_cors"`
	CORSOrigins     []string              `json:"cors_origins"`
	EnableWebSocket bool                  `json:"enable_websocket"`
	JWTSecret       string                `json:"jwt_secret"`
	StorageConfig   storage.StorageConfig `json:"storage"`
}

// NewServer creates a new server instance
func NewServer(config *Config) (*Server, error) {
	// Initialize logger
	loggerConfig := utils.LoggerConfig{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	}
	logger, err := utils.NewLogger(loggerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize storage
	store, err := storage.NewStorage(config.StorageConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Run migrations
	if err := store.Migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize session manager - we'll create a simple version for now
	sessionManager := &manager.SessionManager{}

	// Configure Gin mode
	if config.Mode != "" {
		gin.SetMode(config.Mode)
	}

	// Create server
	server := &Server{
		config:         config,
		storage:        store,
		sessionManager: sessionManager,
		logger:         logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for WebSocket connections
				// In production, you might want to be more restrictive
				return true
			},
		},
	}

	// Initialize handlers
	h := handlers.NewHandlers(server.storage, server.sessionManager, server.logger)
	
	// Initialize terminal manager
	server.terminalManager = handlers.NewTerminalManager(h)

	// Setup routes
	server.setupRoutes()

	return server, nil
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes() {
	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(s.logger))

	// CORS middleware
	if s.config.EnableCORS {
		corsConfig := cors.DefaultConfig()
		if len(s.config.CORSOrigins) > 0 {
			corsConfig.AllowOrigins = s.config.CORSOrigins
		} else {
			corsConfig.AllowAllOrigins = true
		}
		corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
		corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
		router.Use(cors.New(corsConfig))
	}

	// Initialize handlers
	h := handlers.NewHandlers(s.storage, s.sessionManager, s.logger)

	// Health check
	router.GET("/health", h.Health)

	// API routes
	api := router.Group("/api/v1")
	{
		// Projects
		projects := api.Group("/projects")
		{
			projects.GET("", h.GetProjects)
			projects.POST("", h.CreateProject)
			projects.GET("/:id", h.GetProject)
			projects.PUT("/:id", h.UpdateProject)
			projects.DELETE("/:id", h.DeleteProject)
			projects.GET("/:id/stats", h.GetProjectStats)
			projects.GET("/:id/children", h.GetProjectChildren)
			projects.POST("/move", h.MoveProject)
		}

		// Groups
		groups := api.Group("/groups")
		{
			groups.GET("", h.GetGroups)
			groups.POST("", h.CreateGroup)
			groups.GET("/:id", h.GetGroup)
			groups.PUT("/:id", h.UpdateGroup)
			groups.DELETE("/:id", h.DeleteGroup)
			groups.GET("/:id/stats", h.GetGroupStats)
		}

		// Hosts
		hosts := api.Group("/hosts")
		{
			hosts.GET("", h.GetHosts)
			hosts.POST("", h.CreateHost)
			hosts.GET("/:id", h.GetHost)
			hosts.PUT("/:id", h.UpdateHost)
			hosts.DELETE("/:id", h.DeleteHost)
			hosts.GET("/:id/stats", h.GetHostStats)
			hosts.GET("/search", h.SearchHosts)
			// Host connection endpoints
			hosts.POST("/:id/connect", h.ConnectHost)
			hosts.POST("/:id/disconnect", h.DisconnectHost)
			hosts.POST("/:id/test", h.TestHostConnection)
			hosts.POST("/:id/execute", h.ExecuteSSHCommand)
		}

		// Port Forwards
		portForwards := api.Group("/port-forwards")
		{
			portForwards.GET("", h.GetPortForwards)
			portForwards.POST("", h.CreatePortForward)
			portForwards.GET("/:id", h.GetPortForward)
			portForwards.PUT("/:id", h.UpdatePortForward)
			portForwards.DELETE("/:id", h.DeletePortForward)
			portForwards.GET("/:id/stats", h.GetPortForwardStats)
			portForwards.GET("/search", h.SearchPortForwards)
		}

		// Tunnel Sessions
		sessions := api.Group("/sessions")
		{
			sessions.GET("", h.GetTunnelSessions)
			sessions.POST("", h.CreateTunnelSession)
			sessions.GET("/:id", h.GetTunnelSession)
			sessions.PUT("/:id", h.UpdateTunnelSession)
			sessions.DELETE("/:id", h.DeleteTunnelSession)
			sessions.GET("/active", h.GetActiveTunnelSessions)
			sessions.POST("/:id/start", h.StartTunnel)
			sessions.POST("/:id/stop", h.StopTunnel)
		}
	}

	// WebSocket endpoint
	if s.config.EnableWebSocket {
		router.GET("/ws", h.WebSocketHandler(s.upgrader))
		// Terminal WebSocket endpoint
		router.GET("/ws/terminal/:hostId", h.TerminalWebSocketHandler(s.terminalManager))
	}

	s.router = router
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.logger.Info("Starting server on %s", addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Failed to start server: %v", err)
		}
	}()

	s.logger.Info("Server started successfully on %s", addr)

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		s.logger.Error("Server forced to shutdown: %v", err)
		return err
	}

	s.logger.Info("Server exited")
	return nil
}

// Stop stops the server and cleans up resources
func (s *Server) Stop() error {
	// TODO: Stop all active sessions when SessionManager is properly implemented

	// Close storage connection
	if s.storage != nil {
		return s.storage.Close()
	}

	return nil
}

// DefaultConfig returns a default server configuration
func DefaultConfig() *Config {
	return &Config{
		Host:       "localhost",
		Port:       8080,
		Mode:       "release",
		EnableCORS: true,
		CORSOrigins: []string{
			"http://localhost:3000", // For Remix frontend
			"http://localhost:5173", // For Vite dev server
			"http://localhost:4173", // For Vite preview
		},
		EnableWebSocket: true,
		JWTSecret:       "your-secret-key-change-in-production",
		StorageConfig:   storage.DefaultSQLiteConfig(),
	}
}
