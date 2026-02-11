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
	userHandler := handlers.NewUserHandler()

	// Public Routes
	r.GET("/health", handlers.HealthCheck)
	r.GET("/vendors", vendorHandler.ListVerifiedVendors)
	r.GET("/vendors/slug/:slug", vendorHandler.GetVendorBySlug)
	
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/signup", authHandler.Signup)
		authGroup.POST("/login", authHandler.Login)
	}

	// Protected Routes (Require Auth)
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware(cfg))
	{
		// Dashboard
		protected.GET("/me", userHandler.GetMe)

		// Vendor Onboarding
		protected.POST("/vendor/onboard", vendorHandler.OnboardVendor)

		// Organizer Onboarding
		protected.POST("/organizer/onboard", organizerHandler.OnboardOrganizer)

		// Admin Routes (Role: Staff/Admin/SuperAdmin + Permissions)
		adminRoutes := protected.Group("/admin")
		// Require at least 'staff' role to access admin routes base, though specific endpoints check permissions
		adminRoutes.Use(middleware.RequireRole("staff")) 
		{
			// Vendor Management
			adminRoutes.GET("/vendors/pending", middleware.RequirePermission("vendor.verify"), adminHandler.GetPendingVendors)
			adminRoutes.POST("/vendors/:id/verify", middleware.RequirePermission("vendor.verify"), adminHandler.VerifyVendor)
			adminRoutes.POST("/vendors/:id/reject", middleware.RequirePermission("vendor.verify"), adminHandler.RejectVendor)
			
			// User Management
			// Promote Staff: Admin or SuperAdmin
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
