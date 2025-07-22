package main

import (
	"log"

	"github.com/aqz236/port-fly/server"
)

func main() {
	// Create default configuration
	config := server.DefaultConfig()
	
	// Create and start server
	srv, err := server.NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	
	// Start server (this blocks until shutdown)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
