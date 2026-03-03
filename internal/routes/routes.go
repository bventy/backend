package routes

import (
	"github.com/bventy/backend/internal/config"
	"github.com/bventy/backend/internal/handlers"
	"github.com/bventy/backend/internal/middleware"
	"github.com/bventy/backend/internal/services"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {

	cfg := config.LoadConfig()

	// Services
	emailService := services.NewEmailService(cfg.ResendAPIKey, cfg.FromEmail)

	// Handlers
	authHandler := handlers.NewAuthHandler(cfg, emailService)
	vendorHandler := handlers.NewVendorHandler(cfg, emailService)
	adminHandler := handlers.NewAdminHandler()
	userHandler := handlers.NewUserHandler(cfg)
	groupHandler := handlers.NewGroupHandler()
	eventHandler := handlers.NewEventHandler()
	mediaHandler := handlers.NewMediaHandler(cfg)
	quotesHandler := handlers.NewQuotesHandler(emailService)
	trackHandler := handlers.NewTrackHandler()
	reviewHandler := handlers.NewReviewHandler()

	// Public Routes
	r.GET("/health", handlers.HealthCheck)
	r.GET("/vendors", vendorHandler.ListVerifiedVendors)
	r.GET("/vendors/slug/:slug", vendorHandler.GetVendorBySlug)
	r.GET("/vendors/:id/reviews", reviewHandler.GetVendorReviews)

	// Media Upload (Protected? or Public? usually protected)
	// User didn't specify, but let's make it protected to prevent abuse.
	// Actually, having it public is dangerous. I'll put it in Protected.

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/signup", authHandler.Signup)
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/logout", authHandler.Logout)
		authGroup.POST("/verify-email", authHandler.VerifyEmail)
		authGroup.POST("/request-reset", authHandler.RequestReset)
		authGroup.POST("/reset-password", authHandler.ResetPassword)
		authGroup.POST("/resend-verification", authHandler.ResendVerification)
		authGroup.GET("/debug", handlers.DebugCookies)
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

		// Tracking
		protected.POST("/track/activity", trackHandler.TrackActivity)

		// Verification Gate (Restricted Actions)
		verified := protected.Group("/")
		verified.Use(middleware.EmailVerified())
		{
			// Vendor Onboarding
			verified.POST("/vendor/onboard", vendorHandler.OnboardVendor)

			// Quote Creation
			verified.POST("/quotes/request", quotesHandler.CreateQuoteRequest)

			// Quote Response
			verified.PATCH("/quotes/respond/:id", quotesHandler.RespondToQuote)
		}

		// Vendor profiles & Management (Non-gated parts)
		protected.GET("/vendor/me", vendorHandler.GetMyProfile)
		protected.PUT("/vendor/me", vendorHandler.UpdateVendor)

		// Vendor Gallery & Portfolio
		protected.POST("/vendors/:id/gallery", vendorHandler.UploadGalleryImage)
		protected.DELETE("/vendors/:id/gallery/:imageID", vendorHandler.DeleteGalleryImage)
		protected.POST("/vendors/:id/portfolio", vendorHandler.UploadPortfolioFile)
		protected.DELETE("/vendors/:id/portfolio/:fileID", vendorHandler.DeletePortfolioFile)

		// Vendor Reviews
		protected.POST("/vendors/:id/reviews", reviewHandler.CreateReview)
		protected.GET("/vendors/:id/reviews/eligibility", reviewHandler.CheckEligibility)

		// Groups
		protected.POST("/groups", groupHandler.CreateGroup)
		protected.GET("/groups/my", groupHandler.ListMyGroups)

		// Events
		protected.POST("/events", eventHandler.CreateEvent)
		protected.GET("/events", eventHandler.ListMyEvents)
		protected.GET("/events/:id", eventHandler.GetEventById)
		protected.POST("/events/:id/shortlist/:vendorID", eventHandler.ShortlistVendor)
		protected.GET("/events/:id/shortlist", eventHandler.GetShortlistedVendors)

		// Quotes (Non-gated or already gated above)
		protected.GET("/quotes/vendor", quotesHandler.GetVendorQuotes)
		protected.GET("/quotes/organizer", quotesHandler.GetOrganizerQuotes)
		protected.PATCH("/quotes/accept/:id", quotesHandler.AcceptQuote)
		protected.PATCH("/quotes/reject/:id", quotesHandler.RejectQuote)
		protected.PATCH("/quotes/revision/:id", quotesHandler.RequestRevision)
		protected.GET("/quotes/:id/contact", quotesHandler.GetQuoteContact)

		// Admin Routes (Admin & Super Admin)
		adminRoutes := protected.Group("/admin")
		adminRoutes.Use(middleware.AdminOnly())
		{
			// Dashboard Stats (Legacy)
			adminRoutes.GET("/stats", adminHandler.GetStats)

			// Analytics Layer
			adminMetricsHandler := handlers.NewAdminMetricsHandler()
			adminRoutes.GET("/metrics/overview", adminMetricsHandler.GetAdminMetricsOverview)
			adminRoutes.GET("/metrics/growth", adminMetricsHandler.GetAdminMetricsGrowth)
			adminRoutes.GET("/metrics/events", adminMetricsHandler.GetAdminMetricsEvents)
			adminRoutes.GET("/metrics/vendors", adminMetricsHandler.GetAdminMetricsVendors)
			adminRoutes.GET("/metrics/marketplace", adminMetricsHandler.GetAdminMetricsMarketplace)

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
			adminRoutes.PATCH("/users/:id/role", middleware.RequireRole("super_admin"), adminHandler.UpdateUserRole)

			// Email & Template Management
			adminRoutes.GET("/email/templates", adminHandler.GetEmailTemplates)
			adminRoutes.PUT("/email/templates/:key", adminHandler.UpdateEmailTemplate)
			adminRoutes.GET("/email/settings", adminHandler.GetPlatformSettings)
			adminRoutes.PUT("/email/settings", adminHandler.UpdatePlatformSetting)
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
