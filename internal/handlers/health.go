package handlers

import (
	"context"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/bventy/backend/internal/db"
)

func HealthCheck(c *gin.Context) {
	err := db.Pool.Ping(context.Background())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "Database connection failed",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"message":  "bventy backend operational",
		"database": "connected",
		"version":  "1.0.1-perf",
		"env":      os.Getenv("GIN_MODE"),
	})
}
