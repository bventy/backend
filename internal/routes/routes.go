package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/once-human/bventy-backend/internal/handlers"
)

func RegisterRoutes(r *gin.Engine) {

	// Week 1 only
	r.GET("/health", handlers.HealthCheck)
}
