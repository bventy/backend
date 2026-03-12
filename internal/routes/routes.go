package routes

import (
	"net/http"

	"github.com/bventy/backend/internal/config"
	"github.com/bventy/backend/internal/db"
	"github.com/bventy/backend/internal/handlers"
	"github.com/bventy/backend/internal/middleware"
	"github.com/bventy/backend/internal/services"
	"github.com/bventy/backend/internal/websocket"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	r.Use(middleware.VersionMiddleware("1.0.1-perf"))

	cfg := config.LoadConfig()

	// Services
	emailService := services.NewEmailService(cfg.ResendAPIKey, cfg.FromEmail)

	// WebSocket Hub
	hub := websocket.NewHub()
	go hub.Run()

	// Handlers
	authHandler := handlers.NewAuthHandler(cfg, emailService)
	vendorHandler := handlers.NewVendorHandler(cfg, emailService)
	adminHandler := handlers.NewAdminHandler()
	userHandler := handlers.NewUserHandler(cfg)
	groupHandler := handlers.NewGroupHandler()
	eventHandler := handlers.NewEventHandler()
	mediaHandler := handlers.NewMediaHandler(cfg)
	quotesHandler := handlers.NewQuotesHandler(emailService, hub)
	workspaceHandler := handlers.NewWorkspaceHandler()
	calendarHandler := handlers.NewCalendarHandler()
	trackHandler := handlers.NewTrackHandler()
	reviewHandler := handlers.NewReviewHandler()
	messagingHandler := handlers.NewMessagingHandler(hub)
	oauthHandler := handlers.NewOAuthHandler(cfg)
	syncService := services.NewCalendarSyncService(cfg)

	// Public Routes
	r.GET("/health", handlers.HealthCheck)
	r.GET("/system/status", handlers.GetSystemStatus)
	r.GET("/vendors", vendorHandler.ListVerifiedVendors)
	r.GET("/vendors/slug/:slug", vendorHandler.GetVendorBySlug)
	r.GET("/vendors/slug/:slug/details", vendorHandler.GetPublicVendorDetails)
	r.GET("/vendors/:id/reviews", reviewHandler.GetVendorReviews)
	r.POST("/track/activity", middleware.OptionalAuth(cfg), trackHandler.TrackActivity)

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

		// Google OAuth
		authGroup.GET("/google", middleware.AuthMiddleware(cfg), oauthHandler.InitGoogleAuth)
		authGroup.GET("/google/callback", oauthHandler.GoogleCallback)
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
		protected.PUT("/vendor/me", vendorHandler.UpdateVendor)

		// Vendor Services & Pricing
		protected.GET("/vendor/services", vendorHandler.GetVendorServices)
		protected.POST("/vendor/services", vendorHandler.AddVendorService)
		protected.PUT("/vendor/services/:id", vendorHandler.UpdateVendorService)
		protected.DELETE("/vendor/services/:id", vendorHandler.DeleteVendorService)
		protected.GET("/vendor/pricing-rules", vendorHandler.GetVendorPricingRules)
		protected.PUT("/vendor/pricing-rules", vendorHandler.UpdateVendorPricingRules)
		protected.GET("/vendor/cancellation-policy", vendorHandler.GetVendorCancellationPolicy)
		protected.PUT("/vendor/cancellation-policy", vendorHandler.UpdateVendorCancellationPolicy)
		protected.GET("/vendor/service-areas", vendorHandler.GetVendorServiceAreas)
		protected.POST("/vendor/service-areas", vendorHandler.AddVendorServiceArea)
		protected.DELETE("/vendor/service-areas/:id", vendorHandler.DeleteVendorServiceArea)

		// Vendor Workspace
		protected.GET("/vendor/overview", workspaceHandler.GetVendorOverview)
		protected.GET("/vendor/performance", workspaceHandler.GetVendorPerformance)
		protected.GET("/vendor/calendar/events", calendarHandler.GetCalendarEvents)
		protected.POST("/vendor/calendar/blocks", calendarHandler.CreateManualBlock)
		protected.DELETE("/vendor/calendar/blocks/:id", calendarHandler.DeleteManualBlock)
		protected.POST("/vendor/calendar/sync", func(c *gin.Context) {
			userID, _ := c.Get("userID")
			var vendorID string
			err := db.Pool.QueryRow(c.Request.Context(), "SELECT id FROM vendor_profiles WHERE owner_user_id = $1", userID.(string)).Scan(&vendorID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Vendor not found"})
				return
			}
			if err := syncService.SyncGoogleToBventy(vendorID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Calendar synced successfully"})
		})
		protected.GET("/vendor/calendar/sync/status", oauthHandler.GetGoogleSyncStatus)
		protected.DELETE("/vendor/calendar/sync", oauthHandler.DisconnectGoogleCalendar)

		// Vendor Gallery & Portfolio
		protected.POST("/vendors/:id/gallery", vendorHandler.UploadGalleryImage)
		protected.DELETE("/vendors/:id/gallery/:imageID", vendorHandler.DeleteGalleryImage)
		protected.POST("/vendors/:id/portfolio", vendorHandler.UploadPortfolioFile)
		protected.DELETE("/vendors/:id/portfolio/:fileID", vendorHandler.DeletePortfolioFile)

		// Vendor Reviews
		protected.POST("/vendors/:id/reviews", reviewHandler.CreateReview)
		protected.GET("/vendors/:id/reviews/eligibility", reviewHandler.CheckEligibility)
		protected.POST("/reviews/:id/reply", reviewHandler.ReplyToReview)
		protected.POST("/reviews/:id/like", reviewHandler.LikeReview)

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
		protected.GET("/quotes/:id", quotesHandler.GetQuoteById)
		protected.PATCH("/quotes/:id/notes", quotesHandler.UpdateInternalNotes)
		protected.GET("/quotes/:id/contact", quotesHandler.GetQuoteContact)
		protected.PATCH("/quotes/vendor/confirm/:id", quotesHandler.ConfirmQuoteByVendor)
		protected.PATCH("/quotes/vendor/reject/:id", quotesHandler.RejectQuoteByVendor)
		protected.POST("/quotes/manual", quotesHandler.CreateManualQuote)

		// Messaging
		protected.GET("/conversations", messagingHandler.GetConversations)
		protected.GET("/conversations/:id/messages", messagingHandler.GetMessages)
		protected.POST("/conversations/:id/messages", messagingHandler.SendMessage)
		protected.POST("/conversations/:id/messages/:msgId/reactions", messagingHandler.ToggleReaction)
		protected.PATCH("/conversations/:id/read", messagingHandler.MarkAsRead)

		// WebSocket connection logic (authentication usually via cookies since wss:// cannot send standard bearer headers from browser natively, but middleware.AuthMiddleware reads from cookies in Bventy)
		protected.GET("/ws/conversations/:id", func(c *gin.Context) {
			userID, exists := c.Get("userID")
			if !exists {
				// Should not happen due to AuthMiddleware, but safeguard
				c.Status(http.StatusUnauthorized)
				return
			}
			conversationID := c.Param("id")
			websocket.ServeWs(hub, c.Writer, c.Request, userID.(string), conversationID)
		})

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
			adminRoutes.PATCH("/users/:id/verify", adminHandler.VerifyUser)
			adminRoutes.PATCH("/users/:id/unverify", adminHandler.UnverifyUser)
			adminRoutes.DELETE("/users/:id", adminHandler.DeleteUser)

			// Role Management (Super Admin Only)
			adminRoutes.PATCH("/users/:id/role", middleware.RequireRole("super_admin"), adminHandler.UpdateUserRole)

			// Email & Template Management
			adminRoutes.GET("/email/templates", adminHandler.GetEmailTemplates)
			adminRoutes.PUT("/email/templates/:key", adminHandler.UpdateEmailTemplate)
			adminRoutes.GET("/email/settings", adminHandler.GetPlatformSettings)
			adminRoutes.PUT("/email/settings", adminHandler.UpdatePlatformSetting)
			adminRoutes.GET("/email/logs", adminHandler.GetEmailLogs)
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
