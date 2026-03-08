package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/bventy/backend/internal/config"
	"github.com/bventy/backend/internal/db"
	"github.com/bventy/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type CalendarHandler struct{}

func NewCalendarHandler() *CalendarHandler {
	return &CalendarHandler{}
}

type CalendarEvent struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	IsAllDay       bool      `json:"is_all_day"`
	Type           string    `json:"type"` // manual_block, confirmed_booking, tentative_reserve
	Details        *string   `json:"details,omitempty"`
	GoogleEventID  *string   `json:"google_event_id,omitempty"`
}

type CreateManualBlockRequest struct {
	Title     string    `json:"title" binding:"required"`
	StartTime time.Time `json:"start_time" binding:"required"`
	EndTime   time.Time `json:"end_time" binding:"required"`
	IsAllDay  bool      `json:"is_all_day"`
}

// GET /vendor/calendar/events
// Fetch both manual blocks AND auto-derived quote event entries
func (h *CalendarHandler) GetCalendarEvents(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	startDate := c.Query("start_date") // e.g. "2024-01-01"
	endDate := c.Query("end_date")

	if startDate == "" || endDate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_date and end_date are required format YYYY-MM-DD"})
		return
	}

	ctx := c.Request.Context()
	var vendorID string
	err := db.Pool.QueryRow(ctx, "SELECT id FROM vendor_profiles WHERE owner_user_id = $1", userID.(string)).Scan(&vendorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor not found"})
		return
	}

	var events []CalendarEvent

	// First, fetch manual blocks
	manualBlocksQuery := `
		SELECT id, title, start_time, end_time, is_all_day, type, google_event_id
		FROM vendor_calendar_blocks
		WHERE vendor_id = $1 AND start_time >= $2 AND start_time <= $3
	`
	rows, err := db.Pool.Query(ctx, manualBlocksQuery, vendorID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch manual calendar blocks"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var ev CalendarEvent
		if err := rows.Scan(&ev.ID, &ev.Title, &ev.StartTime, &ev.EndTime, &ev.IsAllDay, &ev.Type, &ev.GoogleEventID); err == nil {
			events = append(events, ev)
		}
	}

	// Next, automatically derive events from Quote Requests
	// (Accepted = confirmed_booking, Pending/Responded = tentative_reserve)
	autoEventsQuery := `
		SELECT qr.id as quote_id, e.title as event_title, e.event_date, e.city, qr.status
		FROM quote_requests qr
		JOIN events e ON qr.event_id = e.id
		WHERE qr.vendor_id = $1 
		  AND e.event_date >= $2::DATE AND e.event_date <= $3::DATE
		  AND qr.status IN ('pending', 'responded', 'revision_requested', 'accepted')
	`
	quoteRows, err := db.Pool.Query(ctx, autoEventsQuery, vendorID, startDate, endDate)
	if err != nil {
		log.Printf("Error fetching auto calendar events: %v", err)
	} else {
		defer quoteRows.Close()
		for quoteRows.Next() {
			var quoteID, title, status string
			var city *string
			var eventDate time.Time

			err := quoteRows.Scan(&quoteID, &title, &eventDate, &city, &status)
			if err != nil {
				continue
			}

			eventType := "tentative_reserve"
			if status == "accepted" {
				eventType = "confirmed_booking"
			}

			details := ""
			if city != nil {
				details = *city
			}

			// For auto quotes, we block the entire day by default
			// (Future enhancement: specific start/end times inside `events` schema)
			events = append(events, CalendarEvent{
				ID:        quoteID, // Using quoteID as the calendar event ID for UI mapping
				Title:     title,
				StartTime: eventDate,
				EndTime:   eventDate.Add(24 * time.Hour).Add(-time.Second),
				IsAllDay:  true,
				Type:      eventType,
				Details:   &details,
			})
		}
	}

	if events == nil {
		events = []CalendarEvent{}
	}

	c.JSON(http.StatusOK, events)
}

// POST /vendor/calendar/blocks
func (h *CalendarHandler) CreateManualBlock(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var payload CreateManualBlockRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	var vendorID string
	err := db.Pool.QueryRow(ctx, "SELECT id FROM vendor_profiles WHERE owner_user_id = $1", userID.(string)).Scan(&vendorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor not found"})
		return
	}

	query := `
		INSERT INTO vendor_calendar_blocks (vendor_id, title, start_time, end_time, is_all_day, type)
		VALUES ($1, $2, $3, $4, $5, 'manual_block')
		RETURNING id
	`
	var insertedID string
	err = db.Pool.QueryRow(ctx, query, vendorID, payload.Title, payload.StartTime, payload.EndTime, payload.IsAllDay).Scan(&insertedID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create block"})
		return
	}

	// Trigger push to google
	syncService := services.NewCalendarSyncService(config.LoadConfig())
	go func() {
		_ = syncService.SyncGoogleToBventy(vendorID)
		_ = syncService.PushBventyToGoogle(vendorID)
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "Calendar block created",
		"id":      insertedID,
	})
}

// DELETE /vendor/calendar/blocks/:id
func (h *CalendarHandler) DeleteManualBlock(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	blockID := c.Param("id")

	ctx := c.Request.Context()
	var vendorID string
	err := db.Pool.QueryRow(ctx, "SELECT id FROM vendor_profiles WHERE owner_user_id = $1", userID.(string)).Scan(&vendorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor not found"})
		return
	}

	// Delete from Google if it was a synced event
	var gID *string
	_ = db.Pool.QueryRow(ctx, "SELECT google_event_id FROM vendor_calendar_blocks WHERE id = $1", blockID).Scan(&gID)

	query := `DELETE FROM vendor_calendar_blocks WHERE id = $1 AND vendor_id = $2`
	res, err := db.Pool.Exec(ctx, query, blockID, vendorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete block"})
		return
	}

	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Block not found or unauthorized"})
		return
	}

	if gID != nil && *gID != "" {
		syncService := services.NewCalendarSyncService(config.LoadConfig())
		go func() {
			_ = syncService.DeleteGoogleEvent(context.Background(), vendorID, *gID)
		}()
	}

	c.JSON(http.StatusOK, gin.H{"message": "Block deleted successfully"})
}
