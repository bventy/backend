package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/once-human/bventy-backend/internal/config"
	"github.com/once-human/bventy-backend/internal/db"
	firebaseApp "github.com/once-human/bventy-backend/internal/firebase"
	"github.com/once-human/bventy-backend/internal/routes"
)

func main() {

	// Step 0: Load config
	cfg := config.LoadConfig()

	// Step 1: Connect DB
	db.Connect(cfg)

	// Step 1.5: Initialize Firebase
	firebaseApp.InitFirebase()

	// Step 2: Start Gin server
	r := gin.Default()

	// Step 2.5: CORS Middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"https://bventy-web.vercel.app",
			"https://bventy.in",
			"https://www.bventy.in",
			"http://localhost:3000",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Step 3: Register routes
	routes.RegisterRoutes(r)

	// DEBUG: Print all registered routes
	for _, route := range r.Routes() {
		log.Printf("Route: %s %s", route.Method, route.Path)
	}

	// Step 4: Run server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("Starting server on port %s...", port)
	r.Run("0.0.0.0:" + port)
}
