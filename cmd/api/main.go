package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/once-human/bventy-backend/internal/config"
	"github.com/once-human/bventy-backend/internal/db"
	"github.com/once-human/bventy-backend/internal/routes"
)

func main() {

	// Step 0: Load config
	cfg := config.LoadConfig()

	// Step 1: Connect DB
	db.Connect(cfg)

	// Step 2: Start Gin server
	r := gin.Default()

	// Step 3: Register routes
	routes.RegisterRoutes(r)

	// DEBUG: Print all registered routes
	for _, route := range r.Routes() {
		log.Printf("Route: %s %s", route.Method, route.Path)
	}

	// Step 4: Run server
	log.Printf("Starting server on port %s...", cfg.ServerPort)
	r.Run(":" + cfg.ServerPort)
}
