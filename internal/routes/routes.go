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
	adminHandler := handlers.NewAdminHandler()
	userHandler := handlers.NewUserHandler()
	groupHandler := handlers.NewGroupHandler()
	eventHandler := handlers.NewEventHandler()
	mediaHandler := handlers.NewMediaHandler(cfg)

	// Public Routes
	r.GET("/health", handlers.HealthCheck)
	r.GET("/vendors", vendorHandler.ListVerifiedVendors)
	r.GET("/vendors/slug/:slug", vendorHandler.GetVendorBySlug)

	// Media Upload (Protected? or Public? usually protected)
	// User didn't specify, but let's make it protected to prevent abuse.
	// Actually, having it public is dangerous. I'll put it in Protected.

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/signup", authHandler.Signup)
		authGroup.POST("/login", authHandler.Login)
	}

	// Protected Routes (Require Auth)
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware(cfg))
	{
		// User & Dashboard
		protected.GET("/me", userHandler.GetMe)
		protected.PUT("/me", userHandler.UpdateMe)

		// Media
		protected.POST("/media/upload", mediaHandler.Upload)

		// Vendor Onboarding
		protected.POST("/vendor/onboard", vendorHandler.OnboardVendor)

		// Groups
		protected.POST("/groups", groupHandler.CreateGroup)
		protected.GET("/groups/my", groupHandler.ListMyGroups)

		// Events
		protected.POST("/events", eventHandler.CreateEvent)
		protected.GET("/events", eventHandler.ListMyEvents)
		protected.POST("/events/:id/shortlist/:vendorID", eventHandler.ShortlistVendor)
		protected.GET("/events/:id/shortlist", eventHandler.GetShortlistedVendors)

		// Admin & Staff Routes
		adminRoutes := protected.Group("/admin")
		adminRoutes.Use(middleware.RequireRole("staff"))
		{
			// Vendor Management (Permission: vendor.verify)
			adminRoutes.GET("/vendors/pending", middleware.RequirePermission("vendor.verify"), adminHandler.GetPendingVendors)
			adminRoutes.POST("/vendors/:id/verify", middleware.RequirePermission("vendor.verify"), adminHandler.VerifyVendor)
			adminRoutes.POST("/vendors/:id/reject", middleware.RequirePermission("vendor.verify"), adminHandler.RejectVendor)

			// User Management
			adminRoutes.POST("/users/:id/promote-staff", middleware.RequireRole("admin"), userHandler.PromoteToStaff)
		}

		// Super Admin Routes
		superAdminRoutes := protected.Group("/superadmin")
		superAdminRoutes.Use(middleware.RequireRole("super_admin"))
		{
			superAdminRoutes.POST("/users/:id/promote-admin", userHandler.PromoteToAdmin)
		}
	}
}
