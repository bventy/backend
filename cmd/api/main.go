package main

import (
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

	// Step 4: Run server
	r.Run(":8080")
}
