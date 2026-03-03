package handlers

import (
	"log"
	"net/http"

	"github.com/bventy/backend/internal/db"
	"github.com/gin-gonic/gin"
)

type WorkspaceHandler struct{}

func NewWorkspaceHandler() *WorkspaceHandler {
	return &WorkspaceHandler{}
}

// GET /vendor/overview
func (h *WorkspaceHandler) GetVendorOverview(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	ctx := c.Request.Context()

	// 1. Get Vendor ID
	var vendorID string
	err := db.Pool.QueryRow(ctx, "SELECT id FROM vendor_profiles WHERE owner_user_id = $1", userID.(string)).Scan(&vendorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor profile not found for this user"})
		return
	}

	// 2. Fetch Aggregated Metrics
	var urgentRequests int
	var avgResponseTimeHours float64
	var upcomingBookings int
	var profileViews int

	// Urgent Requests: Pending quotes requested more than 24 hours ago OR Event is within 7 days
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM quote_requests qr
		JOIN events e ON qr.event_id = e.id
		WHERE qr.vendor_id = $1 AND qr.status = 'pending'
		AND (qr.created_at < NOW() - INTERVAL '24 hours' OR e.event_date < CURRENT_DATE + 7)
	`, vendorID).Scan(&urgentRequests)
	if err != nil {
		log.Printf("ERROR getting urgent requests count: %v", err)
	}

	// Avg Response Time (Hours): Avg time between created_at and responded_at
	err = db.Pool.QueryRow(ctx, `
		SELECT COALESCE(EXTRACT(EPOCH FROM AVG(responded_at - created_at))/3600, 0)
		FROM quote_requests
		WHERE vendor_id = $1 AND status != 'pending' AND responded_at IS NOT NULL
	`, vendorID).Scan(&avgResponseTimeHours)
	if err != nil {
		log.Printf("ERROR getting response time: %v", err)
	}

	// Upcoming Bookings: Accepted quotes for events in the future
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM quote_requests qr
		JOIN events e ON qr.event_id = e.id
		WHERE qr.vendor_id = $1 AND qr.status = 'accepted' AND e.event_date >= CURRENT_DATE
	`, vendorID).Scan(&upcomingBookings)
	if err != nil {
		log.Printf("ERROR getting upcoming bookings: %v", err)
	}

	// Profile Views: Last 30 days
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM vendor_profile_views
		WHERE vendor_id = $1 AND created_at > NOW() - INTERVAL '30 days'
	`, vendorID).Scan(&profileViews)
	if err != nil {
		log.Printf("ERROR getting profile views: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"urgent_requests":   urgentRequests,
		"avg_response_time": avgResponseTimeHours, // in hours
		"upcoming_bookings": upcomingBookings,
		"profile_views":     profileViews,
	})
}
