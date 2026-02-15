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

	// Public Routes
	r.GET("/health", handlers.HealthCheck)
	r.GET("/vendors", vendorHandler.ListVerifiedVendors)
	r.GET("/vendors/slug/:slug", vendorHandler.GetVendorBySlug)

	// Protected Routes (Require Firebase Auth)
	protected := r.Group("/")
	protected.Use(middleware.FirebaseAuthMiddleware())
	{

		// Auth
		protected.POST("/auth/firebase-login", authHandler.FirebaseLogin)
		protected.GET("/auth/me", authHandler.GetMe)
		protected.POST("/auth/complete-profile", authHandler.CompleteProfile)

		// User & Dashboard
		protected.GET("/me", userHandler.GetMe)

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
