package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bventy/backend/internal/config"
	"github.com/bventy/backend/internal/db"
	"github.com/bventy/backend/internal/routes"
	"github.com/bventy/backend/internal/tracking"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {

	// Step 0: Load config
	cfg := config.LoadConfig()

	// Step 1: Connect DB
	db.Connect(cfg)
	db.RunMigrations()

	// Step 1.5: Initialize Tracking
	tracking.Init(cfg)
	defer tracking.Flush()

	// Step 2: Start Gin server
	r := gin.Default()

	// Step 2.5: CORS Middleware
	allowedOrigins := []string{
		"https://bventy.in",
		"https://www.bventy.in",
		"https://auth.bventy.in",
		"https://vendor.bventy.in",
		"https://admin.bventy.in",
		"https://app.bventy.in",
		"http://localhost:3000",
		"http://localhost:3001",
		"http://localhost:3002",
		"http://localhost:3003",
		"http://localhost:3004",
		"http://www.lvh.me:3000",
		"http://auth.lvh.me:3001",
		"http://app.lvh.me:3002",
		"http://vendor.lvh.me:3003",
		"http://admin.lvh.me:3004",
	}

	// Allow overrides via environment variable
	if customOrigins := os.Getenv("ALLOWED_ORIGINS"); customOrigins != "" {
		origins := strings.Split(customOrigins, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
		allowedOrigins = append(allowedOrigins, origins...)
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept", "X-Requested-With", "Credentials"},
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

	// Step 5: Start Self-Ping Heartbeat (Keep Render Alive)
	go startSelfPing(port)

	log.Printf("Starting server on port %s...", port)
	r.Run("0.0.0.0:" + port)
}

func startSelfPing(port string) {
	// Wait for server to start
	time.Sleep(5 * time.Second)

	url := "http://localhost:" + port + "/health"
	// Use external URL if available for better keep-alive
	if externalURL := os.Getenv("RENDER_EXTERNAL_URL"); externalURL != "" {
		url = externalURL + "/health"
	}

	ticker := time.NewTicker(5 * time.Minute)
	log.Printf("Heartbeat started: pinging %s every 5 minutes", url)

	for range ticker.C {
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Heartbeat ERROR: %v", err)
			continue
		}
		resp.Body.Close()
		log.Printf("Heartbeat: Self-ping successful (%s)", resp.Status)
	}
}
