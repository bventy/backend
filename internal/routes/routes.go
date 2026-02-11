package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/once-human/bventy-backend/internal/config"
	"github.com/once-human/bventy-backend/internal/handlers"
	"github.com/once-human/bventy-backend/internal/middleware"
)

func RegisterRoutes(r *gin.Engine) {

	cfg := config.LoadConfig()

	// Handlers
	authHandler := handlers.NewAuthHandler(cfg)
	vendorHandler := handlers.NewVendorHandler()
	organizerHandler := handlers.NewOrganizerHandler()
	adminHandler := handlers.NewAdminHandler()

	// Public Routes
	r.GET("/health", handlers.HealthCheck)
	
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/signup", authHandler.Signup)
		authGroup.POST("/login", authHandler.Login)
	}

	// Protected Routes (Require Auth)
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware(cfg))
	{
		// Vendor
		vendorRoutes := protected.Group("/vendor")
		vendorRoutes.Use(middleware.RoleMiddleware("vendor"))
		{
			vendorRoutes.POST("/onboard", vendorHandler.OnboardVendor)
		}

		// Organizer
		organizerRoutes := protected.Group("/organizer")
		organizerRoutes.Use(middleware.RoleMiddleware("organizer"))
		{
			organizerRoutes.POST("/onboard", organizerHandler.OnboardOrganizer)
		}

		// Admin
		adminRoutes := protected.Group("/admin")
		adminRoutes.Use(middleware.RoleMiddleware("admin"))
		{
			adminRoutes.GET("/vendors/pending", adminHandler.GetPendingVendors)
			adminRoutes.POST("/vendors/:id/verify", adminHandler.VerifyVendor)
			adminRoutes.POST("/vendors/:id/reject", adminHandler.RejectVendor)
		}
	}
}
