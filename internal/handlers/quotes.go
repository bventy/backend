package handlers

import (
	"context"
	"net/http"

	"github.com/bventy/backend/internal/db"
	"github.com/gin-gonic/gin"
)

type QuotesHandler struct{}

func NewQuotesHandler() *QuotesHandler {
	return &QuotesHandler{}
}

type CreateQuoteRequestPayload struct {
	EventID  string `json:"event_id" binding:"required"`
	VendorID string `json:"vendor_id" binding:"required"`
	Message  string `json:"message" binding:"required"`
}

// POST /quotes/request (Organizers only)
func (h *QuotesHandler) CreateQuoteRequest(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	organizerID := userID.(string)

	var payload CreateQuoteRequestPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()

	// 1. Validate event exists & belongs to the user
	var eventOrganizerID string
	err := db.Pool.QueryRow(ctx, "SELECT organizer_user_id FROM events WHERE id = $1", payload.EventID).Scan(&eventOrganizerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}
	if eventOrganizerID != organizerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not own this event"})
		return
	}

	// 2. Validate vendor exists
	var vendorExists int
	err = db.Pool.QueryRow(ctx, "SELECT 1 FROM vendor_profiles WHERE id = $1", payload.VendorID).Scan(&vendorExists)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor not found"})
		return
	}

	// 3. Insert quote request
	var quoteID string
	insertQuoteQuery := `
		INSERT INTO quote_requests (event_id, vendor_id, organizer_user_id, message, status)
		VALUES ($1, $2, $3, $4, 'pending')
		RETURNING id
	`
	err = db.Pool.QueryRow(ctx, insertQuoteQuery, payload.EventID, payload.VendorID, organizerID, payload.Message).Scan(&quoteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create quote request"})
		return
	}

	// 4. Activity Log: Fire-and-forget
	insertLogQuery := `
		INSERT INTO platform_activity_log (entity_type, entity_id, action_type, actor_user_id)
		VALUES ('quote', $1, 'quote_created', $2)
	`
	_, _ = db.Pool.Exec(ctx, insertLogQuery, quoteID, organizerID)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Quote requested successfully",
		"quote_id": quoteID,
	})
}

// GET /quotes/vendor
func (h *QuotesHandler) GetVendorQuotes(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx := context.Background()
	// Get vendor ID from userID
	var vendorID string
	err := db.Pool.QueryRow(ctx, "SELECT id FROM vendor_profiles WHERE user_id = $1", userID.(string)).Scan(&vendorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor profile not found"})
		return
	}

	query := `
		SELECT qr.id, qr.event_id, e.title as event_title, qr.organizer_user_id, u.full_name as organizer_name, 
		       qr.message, qr.quoted_price, qr.vendor_response, qr.status, qr.responded_at, qr.created_at
		FROM quote_requests qr
		JOIN events e ON qr.event_id = e.id
		JOIN users u ON qr.organizer_user_id = u.id
		WHERE qr.vendor_id = $1
		ORDER BY qr.created_at DESC
	`
	rows, err := db.Pool.Query(ctx, query, vendorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quotes"})
		return
	}
	defer rows.Close()

	var quotes []gin.H
	for rows.Next() {
		var id, eventID, eventTitle, organizerID, organizerName, message, status string
		var quotedPrice *float64
		var vendorResponse *string
		var respondedAt, createdAt interface{}

		if err := rows.Scan(&id, &eventID, &eventTitle, &organizerID, &organizerName, &message, &quotedPrice, &vendorResponse, &status, &respondedAt, &createdAt); err == nil {
			quotes = append(quotes, gin.H{
				"id":             id,
				"event_id":       eventID,
				"event_title":    eventTitle,
				"organizer_id":   organizerID,
				"organizer_name": organizerName,
				"message":        message,
				"quoted_price":   quotedPrice,
				"response":       vendorResponse,
				"status":         status,
				"responded_at":   respondedAt,
				"created_at":     createdAt,
			})
		}
	}
	if quotes == nil {
		quotes = []gin.H{}
	}

	c.JSON(http.StatusOK, quotes)
}

// GET /quotes/organizer
func (h *QuotesHandler) GetOrganizerQuotes(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx := context.Background()

	query := `
		SELECT qr.id, qr.event_id, e.title as event_title, qr.vendor_id, v.business_name as vendor_name, 
		       qr.message, qr.quoted_price, qr.vendor_response, qr.status, qr.responded_at, qr.created_at
		FROM quote_requests qr
		JOIN events e ON qr.event_id = e.id
		JOIN vendor_profiles v ON qr.vendor_id = v.id
		WHERE qr.organizer_user_id = $1
		ORDER BY qr.created_at DESC
	`
	rows, err := db.Pool.Query(ctx, query, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quotes"})
		return
	}
	defer rows.Close()

	var quotes []gin.H
	for rows.Next() {
		var id, eventID, eventTitle, vendorID, vendorName, message, status string
		var quotedPrice *float64
		var vendorResponse *string
		var respondedAt, createdAt interface{}

		if err := rows.Scan(&id, &eventID, &eventTitle, &vendorID, &vendorName, &message, &quotedPrice, &vendorResponse, &status, &respondedAt, &createdAt); err == nil {
			quotes = append(quotes, gin.H{
				"id":           id,
				"event_id":     eventID,
				"event_title":  eventTitle,
				"vendor_id":    vendorID,
				"vendor_name":  vendorName,
				"message":      message,
				"quoted_price": quotedPrice,
				"response":     vendorResponse,
				"status":       status,
				"responded_at": respondedAt,
				"created_at":   createdAt,
			})
		}
	}
	if quotes == nil {
		quotes = []gin.H{}
	}

	c.JSON(http.StatusOK, quotes)
}

type RespondQuotePayload struct {
	QuotedPrice    float64 `json:"quoted_price" binding:"required"`
	VendorResponse string  `json:"vendor_response" binding:"required"`
}

// PATCH /quotes/respond/:id
func (h *QuotesHandler) RespondToQuote(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	quoteID := c.Param("id")
	var payload RespondQuotePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()

	// Verify vendor owns this quote
	var vendorID string
	err := db.Pool.QueryRow(ctx, "SELECT id FROM vendor_profiles WHERE user_id = $1", userID.(string)).Scan(&vendorID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only vendors can respond"})
		return
	}

	var quoteVendorID string
	err = db.Pool.QueryRow(ctx, "SELECT vendor_id FROM quote_requests WHERE id = $1", quoteID).Scan(&quoteVendorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quote not found"})
		return
	}
	if quoteVendorID != vendorID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to respond to this quote"})
		return
	}

	// Update quote
	updateQuery := `
		UPDATE quote_requests
		SET quoted_price = $1, vendor_response = $2, status = 'responded', responded_at = NOW(), updated_at = NOW()
		WHERE id = $3
	`
	_, err = db.Pool.Exec(ctx, updateQuery, payload.QuotedPrice, payload.VendorResponse, quoteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update quote"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Quote responded successfully"})
}

// PATCH /quotes/accept/:id
func (h *QuotesHandler) AcceptQuote(c *gin.Context) {
	h.updateQuoteStatusByOrganizer(c, "accepted")
}

// PATCH /quotes/reject/:id
func (h *QuotesHandler) RejectQuote(c *gin.Context) {
	h.updateQuoteStatusByOrganizer(c, "rejected")
}

func (h *QuotesHandler) updateQuoteStatusByOrganizer(c *gin.Context, newStatus string) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	organizerID := userID.(string)
	quoteID := c.Param("id")

	ctx := context.Background()

	var quoteOrganizerID string
	err := db.Pool.QueryRow(ctx, "SELECT organizer_user_id FROM quote_requests WHERE id = $1", quoteID).Scan(&quoteOrganizerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quote not found"})
		return
	}
	if quoteOrganizerID != organizerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to update this quote"})
		return
	}

	updateQuery := `UPDATE quote_requests SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err = db.Pool.Exec(ctx, updateQuery, newStatus, quoteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update quote status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Quote " + newStatus + " successfully"})
}
