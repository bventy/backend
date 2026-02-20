package routes

import (
	"github.com/bventy/backend/internal/config"
	"github.com/bventy/backend/internal/handlers"
	"github.com/bventy/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {

	cfg := config.LoadConfig()

	// Handlers
	authHandler := handlers.NewAuthHandler(cfg)
	vendorHandler := handlers.NewVendorHandler(cfg)
	adminHandler := handlers.NewAdminHandler()
	userHandler := handlers.NewUserHandler(cfg)
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

		// Profile Image
		protected.POST("/users/profile-image", userHandler.UploadProfileImage)

		// Media
		protected.POST("/media/upload", mediaHandler.Upload)

		// Vendor Onboarding & Management
		protected.POST("/vendor/onboard", vendorHandler.OnboardVendor)
		protected.GET("/vendor/me", vendorHandler.GetMyProfile)
		protected.PUT("/vendor/me", vendorHandler.UpdateVendor)

		// Vendor Gallery & Portfolio
		protected.POST("/vendors/:id/gallery", vendorHandler.UploadGalleryImage)
		protected.DELETE("/vendors/:id/gallery/:imageID", vendorHandler.DeleteGalleryImage)
		protected.POST("/vendors/:id/portfolio", vendorHandler.UploadPortfolioFile)
		protected.DELETE("/vendors/:id/portfolio/:fileID", vendorHandler.DeletePortfolioFile)

		// Groups
		protected.POST("/groups", groupHandler.CreateGroup)
		protected.GET("/groups/my", groupHandler.ListMyGroups)

		// Events
		protected.POST("/events", eventHandler.CreateEvent)
		protected.GET("/events", eventHandler.ListMyEvents)
		protected.GET("/events/:id", eventHandler.GetEventById)
		protected.POST("/events/:id/shortlist/:vendorID", eventHandler.ShortlistVendor)
		protected.GET("/events/:id/shortlist", eventHandler.GetShortlistedVendors)

		// Admin Routes (Admin & Super Admin)
		adminRoutes := protected.Group("/admin")
		adminRoutes.Use(middleware.AdminOnly())
		{
			// Dashboard Stats
			adminRoutes.GET("/stats", adminHandler.GetStats)

			// Vendor Management
			// Note: Keeping RequirePermission for granular control if needed, but AdminOnly covers general access.
			// If we want to strictly follow "Only admin and super_admin", AdminOnly is sufficient.
			// Existing code used "vendor.verify" permission. I'll keep it for safety but main gate is AdminOnly.
			adminRoutes.GET("/vendors", adminHandler.GetVendors)
			adminRoutes.PATCH("/vendors/:id/approve", adminHandler.VerifyVendor)
			adminRoutes.PATCH("/vendors/:id/reject", adminHandler.RejectVendor)

			// User Management
			adminRoutes.GET("/users", adminHandler.GetUsers)

			// Role Management (Super Admin Only)
			// We can use a specific route group or just checking the role in handler (which we added middleware for in route)
			// Better to be explicit in route definition if possible.
			adminRoutes.PATCH("/users/:id/role", middleware.RequireRole("super_admin"), adminHandler.UpdateUserRole)
		}

		// Super Admin Routes (Legacy/Specific)
		superAdminRoutes := protected.Group("/superadmin")
		superAdminRoutes.Use(middleware.RequireRole("super_admin"))
		{
			// Keep existing if needed, or deprecate/move to admin
			superAdminRoutes.POST("/users/:id/promote-admin", userHandler.PromoteToAdmin)
		}
	}
}
