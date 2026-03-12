package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bventy/backend/internal/config"
	"github.com/bventy/backend/internal/db"
	"github.com/bventy/backend/internal/middleware"
	"github.com/bventy/backend/internal/routes"
	"github.com/bventy/backend/internal/services"
	"github.com/bventy/backend/internal/tracking"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func main() {
	// ... config and DB setup
	cfg := config.LoadConfig()
	db.Connect(cfg)
	db.RunMigrations()
	tracking.Init(cfg)
	defer tracking.Flush()

	// Initialize Media & Backup Services
	mediaService, err := services.NewMediaService(cfg)
	if err != nil {
		log.Printf("⚠️  Warning: Failed to initialize MediaService for backups: %v", err)
	} else if mediaService != nil {
		backupService := services.NewBackupService(cfg, mediaService)
		go backupService.Start()
	}

	r := gin.Default()
	r.Use(middleware.ActivityTracker())

	// Step 2.1: CORS Middleware (Must be before security headers to handle OPTIONS)
	allowedOrigins := []string{
		"https://bventy.in",
		"https://www.bventy.in",
		"https://auth.bventy.in",
		"https://vendor.bventy.in",
		"https://partner.bventy.in",
		"https://admin.bventy.in",
		"https://app.bventy.in",
		"https://status.bventy.in",
		"http://localhost:3000",
		"http://localhost:3001",
		"http://localhost:3002",
		"http://localhost:3003",
		"http://localhost:3004",
		"http://www.lvh.me:3000",
		"http://auth.lvh.me:3001",
		"http://app.lvh.me:3002",
		"http://vendor.lvh.me:3003",
		"http://partner.lvh.me:3003",
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

	// Step 2.2: Gzip Compression
	r.Use(gzip.Gzip(gzip.DefaultCompression))

	// Step 2.3: Security Headers Middleware
	r.Use(func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' https://va.vercel-scripts.com https://us-assets.i.posthog.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data: https://media.bventy.in; connect-src 'self' https://api.bventy.in https://status.bventy.in https://us.i.posthog.com https://cloud.umami.is;")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

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

	ticker := time.NewTicker(30 * time.Minute)
	log.Printf("Heartbeat started: pinging %s every 30 minutes", url)

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
