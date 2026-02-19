package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/bventy/backend/internal/db"
	"github.com/gin-gonic/gin"
	pgx "github.com/jackc/pgx/v5"
)

type EventHandler struct{}

func NewEventHandler() *EventHandler {
	return &EventHandler{}
}

type CreateEventRequest struct {
	Title            string  `json:"title" binding:"required"`
	City             string  `json:"city" binding:"required"`
	EventType        string  `json:"event_type"`
	Date             string  `json:"event_date" binding:"required"` // ISO string
	BudgetMin        *int    `json:"budget_min"`
	BudgetMax        *int    `json:"budget_max"`
	OrganizerGroupID *string `json:"organizer_group_id"` // Optional
	CoverImageURL    *string `json:"cover_image_url"`    // Optional
}

func (h *EventHandler) CreateEvent(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	eventDate, err := time.Parse(time.RFC3339, req.Date)
	if err != nil {
		// Try generic date format if RFC3339 fails (simple YYYY-MM-DD)
		eventDate, err = time.Parse("2006-01-02", req.Date)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD or RFC3339"})
			return
		}
	}

	var organizerUserID interface{} = userID
	var organizerGroupID interface{} = nil

	// If group ID provided, verify membership
	if req.OrganizerGroupID != nil {
		organizerUserID = nil // Event is owned by group, not user directly (conceptually, though linked) -> Table check constraints say EITHER/OR.
		organizerGroupID = *req.OrganizerGroupID

		var isMember int
		queryCheck := `SELECT 1 FROM group_members WHERE group_id=$1 AND user_id=$2`
		err := db.Pool.QueryRow(context.Background(), queryCheck, organizerGroupID, userID).Scan(&isMember)

		if err == pgx.ErrNoRows {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this group"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking membership"})
			return
		}
	}

	// Updated query to use 'date' column (Neon/001_init schema) instead of 'event_date'
	query := `
		INSERT INTO events (title, city, event_type, date, budget_min, budget_max, organizer_user_id, organizer_group_id, cover_image_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	var eventID string
	err = db.Pool.QueryRow(context.Background(), query,
		req.Title, req.City, req.EventType, eventDate, req.BudgetMin, req.BudgetMax, organizerUserID, organizerGroupID, req.CoverImageURL,
	).Scan(&eventID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Event created successfully", "event_id": eventID})
}

func (h *EventHandler) ListMyEvents(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Use 'date' column
	query := `
		SELECT e.id, e.title, e.city, e.date, e.event_type, e.budget_min, e.budget_max, e.cover_image_url
		FROM events e
		LEFT JOIN group_members gm ON e.organizer_group_id = gm.group_id AND gm.user_id = $1
		WHERE e.organizer_user_id = $1 OR gm.user_id IS NOT NULL
	`

	rows, err := db.Pool.Query(context.Background(), query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events"})
		return
	}
	defer rows.Close()

	var events []gin.H
	for rows.Next() {
		var id, title, city, eventType string
		var date time.Time
		var budgetMin, budgetMax *int
		var coverImageURL *string

		if err := rows.Scan(&id, &title, &city, &date, &eventType, &budgetMin, &budgetMax, &coverImageURL); err != nil {
			continue
		}
		events = append(events, gin.H{
			"id":              id,
			"title":           title,
			"city":            city,
			"date":            date.Format("2006-01-02"),
			"event_date":      date.Format("2006-01-02"), // duplicated for safety
			"event_type":      eventType,
			"budget_min":      budgetMin,
			"budget_max":      budgetMax,
			"cover_image_url": coverImageURL,
		})
	}

	c.JSON(http.StatusOK, events)
}

func (h *EventHandler) GetEventById(c *gin.Context) {
	eventID := c.Param("id")

	// Use 'date' column
	query := `
		SELECT id, title, city, date, event_type, budget_min, budget_max, cover_image_url, organizer_user_id, organizer_group_id
		FROM events
		WHERE id = $1
	`

	var event gin.H
	var id, title, city, eventType string
	var date time.Time
	var budgetMin, budgetMax *int
	var coverImageURL, organizerUserID, organizerGroupID *string

	err := db.Pool.QueryRow(context.Background(), query, eventID).Scan(
		&id, &title, &city, &date, &eventType, &budgetMin, &budgetMax, &coverImageURL, &organizerUserID, &organizerGroupID,
	)

	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	event = gin.H{
		"id":                 id,
		"title":              title,
		"city":               city,
		"event_date":         date.Format("2006-01-02"),
		"event_type":         eventType,
		"budget_min":         budgetMin,
		"budget_max":         budgetMax,
		"cover_image_url":    coverImageURL,
		"organizer_user_id":  organizerUserID,
		"organizer_group_id": organizerGroupID,
	}

	c.JSON(http.StatusOK, event)
}

func (h *EventHandler) ShortlistVendor(c *gin.Context) {
	eventID := c.Param("id")
	vendorID := c.Param("vendorID")

	query := `INSERT INTO event_shortlisted_vendors (event_id, vendor_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := db.Pool.Exec(context.Background(), query, eventID, vendorID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to shortlist vendor"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Vendor shortlisted"})
}

func (h *EventHandler) GetShortlistedVendors(c *gin.Context) {
	eventID := c.Param("id")

	// Adjust for Neon Schema: display_name, no category
	query := `
		SELECT v.id, v.display_name
		FROM event_shortlisted_vendors esv
		JOIN vendor_profiles v ON esv.vendor_id = v.id
		WHERE esv.event_id = $1
	`
	rows, err := db.Pool.Query(context.Background(), query, eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch shortlisted vendors"})
		return
	}
	defer rows.Close()

	var vendors []gin.H
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}
		vendors = append(vendors, gin.H{
			"id":            id,
			"business_name": name,
			"category":      "General", // Default value as column missing
		})
	}

	c.JSON(http.StatusOK, vendors)
}
