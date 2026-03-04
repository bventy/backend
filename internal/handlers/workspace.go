package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

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

	// Upcoming Bookings: Detailed list of top 3 future accepted quotes
	type UpcomingBooking struct {
		ID        string    `json:"id"`
		Title     string    `json:"title"`
		EventDate time.Time `json:"event_date"`
		Status    string    `json:"status"`
	}
	var upcomingBookingsList []UpcomingBooking
	bookingRows, err := db.Pool.Query(ctx, `
		SELECT qr.id, e.title, e.event_date, qr.status
		FROM quote_requests qr
		JOIN events e ON qr.event_id = e.id
		WHERE qr.vendor_id = $1 AND qr.status = 'accepted' AND e.event_date >= CURRENT_DATE
		ORDER BY e.event_date ASC
		LIMIT 3
	`, vendorID)
	if err == nil {
		defer bookingRows.Close()
		for bookingRows.Next() {
			var b UpcomingBooking
			if err := bookingRows.Scan(&b.ID, &b.Title, &b.EventDate, &b.Status); err == nil {
				upcomingBookingsList = append(upcomingBookingsList, b)
			}
		}
	}
	if upcomingBookingsList == nil {
		upcomingBookingsList = []UpcomingBooking{}
	}

	// Profile Views: Last 30 days
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM vendor_profile_views
		WHERE vendor_id = $1 AND created_at > NOW() - INTERVAL '30 days'
	`, vendorID).Scan(&profileViews)
	if err != nil {
		log.Printf("ERROR getting profile views: %v", err)
	}

	// 5. Tentative Holds: Top 3 pending quote requests
	type TentativeHold struct {
		ID        string    `json:"id"`
		Title     string    `json:"title"`
		ExpiresIn string    `json:"expires_in"` // Placeholder or derived from deadline
		CreatedAt time.Time `json:"created_at"`
	}
	var tentativeHolds []TentativeHold
	holdRows, err := db.Pool.Query(ctx, `
		SELECT qr.id, e.title, qr.created_at
		FROM quote_requests qr
		JOIN events e ON qr.event_id = e.id
		WHERE qr.vendor_id = $1 AND qr.status IN ('pending', 'responded', 'revision_requested')
		ORDER BY qr.created_at DESC
		LIMIT 3
	`, vendorID)
	if err == nil {
		defer holdRows.Close()
		for holdRows.Next() {
			var h TentativeHold
			if err := holdRows.Scan(&h.ID, &h.Title, &h.CreatedAt); err == nil {
				// Simple logic for expires_in: distance from NOW to 48h after created_at
				expiry := h.CreatedAt.Add(48 * time.Hour)
				if time.Now().After(expiry) {
					h.ExpiresIn = "Expired"
				} else {
					hoursLeft := int(expiry.Sub(time.Now()).Hours())
					if hoursLeft > 24 {
						h.ExpiresIn = "Expires in 2 days"
					} else {
						h.ExpiresIn = fmt.Sprintf("Expires in %d hours", hoursLeft)
					}
				}
				tentativeHolds = append(tentativeHolds, h)
			}
		}
	}

	if tentativeHolds == nil {
		tentativeHolds = []TentativeHold{}
	}

	c.JSON(http.StatusOK, gin.H{
		"urgent_requests":   urgentRequests,
		"avg_response_time": avgResponseTimeHours, // in hours
		"upcoming_bookings": upcomingBookingsList,
		"profile_views":     profileViews,
		"tentative_holds":   tentativeHolds,
	})
}
