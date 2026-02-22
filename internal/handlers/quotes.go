package handlers

import (
	"log"
	"net/http"

	"github.com/bventy/backend/internal/db"
	"github.com/bventy/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type QuotesHandler struct {
	MediaService *services.MediaService
}

func NewQuotesHandler() *QuotesHandler {
	// We might need media service here for some logic, but usually it's used in MediaHandler.
	// For RespondToQuote, we might want to handle attachment verification if needed.
	return &QuotesHandler{}
}

type CreateQuoteRequestPayload struct {
	EventID             string  `json:"event_id" binding:"required"`
	VendorID            string  `json:"vendor_id" binding:"required"`
	Message             string  `json:"message" binding:"required"`
	BudgetRange         *string `json:"budget_range"`
	SpecialRequirements *string `json:"special_requirements"`
	Deadline            *string `json:"deadline"` // ISO string
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

	ctx := c.Request.Context()

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
		INSERT INTO quote_requests (
			event_id, vendor_id, organizer_user_id, message, budget_range, 
			special_requirements, deadline, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending')
		RETURNING id
	`
	err = db.Pool.QueryRow(ctx, insertQuoteQuery,
		payload.EventID, payload.VendorID, organizerID, payload.Message, payload.BudgetRange,
		payload.SpecialRequirements, payload.Deadline,
	).Scan(&quoteID)
	if err != nil {
		log.Printf("ERROR: Failed to create quote request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create quote request: " + err.Error()})
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

	ctx := c.Request.Context()
	// Get vendor ID from userID
	var vendorID string
	err := db.Pool.QueryRow(ctx, "SELECT id FROM vendor_profiles WHERE owner_user_id = $1", userID.(string)).Scan(&vendorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor profile not found for this user"})
		return
	}

	query := `
		SELECT qr.id, qr.event_id, e.title as event_title, qr.organizer_user_id, u.full_name as organizer_name, 
		       qr.message, qr.quoted_price, qr.vendor_response, qr.status, qr.responded_at, qr.created_at, qr.budget_range,
		       qr.special_requirements, qr.deadline, qr.attachment_url, qr.accepted_at, qr.rejected_at, qr.revision_requested_at, qr.contact_unlocked_at
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
		var id, eventID, eventTitle, organizerID, organizerName, status string
		var message, vendorResponse, budgetRange, specialReq, deadline, attachmentURL *string
		var quotedPrice *float64
		var respondedAt, createdAt, acceptedAt, rejectedAt, revisionAt, unlockedAt interface{}

		err := rows.Scan(
			&id, &eventID, &eventTitle, &organizerID, &organizerName, &message, &quotedPrice, &vendorResponse, &status,
			&respondedAt, &createdAt, &budgetRange, &specialReq, &deadline, &attachmentURL, &acceptedAt, &rejectedAt, &revisionAt, &unlockedAt,
		)
		if err != nil {
			log.Printf("Error scanning vendor quote row: %v", err)
			continue
		}

		quotes = append(quotes, gin.H{
			"id":                    id,
			"event_id":              eventID,
			"event_title":           eventTitle,
			"organizer_id":          organizerID,
			"organizer_name":        organizerName,
			"message":               message,
			"quoted_price":          quotedPrice,
			"response":              vendorResponse,
			"status":                status,
			"accepted_at":           acceptedAt,
			"rejected_at":           rejectedAt,
			"revision_requested_at": revisionAt,
			"contact_unlocked_at":   unlockedAt,
			"special_requirements":  specialReq,
			"deadline":              deadline,
			"attachment_url":        attachmentURL,
		})
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

	ctx := c.Request.Context()

	query := `
		SELECT qr.id, qr.event_id, e.title as event_title, qr.vendor_id, v.business_name as vendor_name, 
		       qr.message, qr.quoted_price, qr.vendor_response, qr.status, qr.responded_at, qr.created_at, qr.budget_range,
		       qr.special_requirements, qr.deadline, qr.attachment_url, qr.accepted_at, qr.rejected_at, qr.revision_requested_at, qr.contact_unlocked_at
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
		var id, eventID, eventTitle, vendorID, vendorName, status string
		var message, vendorResponse, budgetRange, specialReq, deadline, attachmentURL *string
		var quotedPrice *float64
		var respondedAt, createdAt, acceptedAt, rejectedAt, revisionAt, unlockedAt interface{}

		err := rows.Scan(
			&id, &eventID, &eventTitle, &vendorID, &vendorName, &message, &quotedPrice, &vendorResponse, &status,
			&respondedAt, &createdAt, &budgetRange, &specialReq, &deadline, &attachmentURL, &acceptedAt, &rejectedAt, &revisionAt, &unlockedAt,
		)
		if err != nil {
			log.Printf("Error scanning organizer quote row: %v", err)
			continue
		}

		quotes = append(quotes, gin.H{
			"id":                    id,
			"event_id":              eventID,
			"event_title":           eventTitle,
			"vendor_id":             vendorID,
			"vendor_name":           vendorName,
			"message":               message,
			"quoted_price":          quotedPrice,
			"response":              vendorResponse,
			"status":                status,
			"accepted_at":           acceptedAt,
			"rejected_at":           rejectedAt,
			"revision_requested_at": revisionAt,
			"contact_unlocked_at":   unlockedAt,
			"special_requirements":  specialReq,
			"deadline":              deadline,
			"attachment_url":        attachmentURL,
		})
	}
	if quotes == nil {
		quotes = []gin.H{}
	}

	c.JSON(http.StatusOK, quotes)
}

type RespondQuotePayload struct {
	QuotedPrice    float64 `json:"quoted_price" binding:"required"`
	VendorResponse *string `json:"vendor_response"`
	AttachmentURL  *string `json:"attachment_url"`
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

	ctx := c.Request.Context()

	// Verify vendor owns this quote
	var vendorID string
	err := db.Pool.QueryRow(ctx, "SELECT id FROM vendor_profiles WHERE owner_user_id = $1", userID.(string)).Scan(&vendorID)
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
		SET quoted_price = $1, vendor_response = $2, attachment_url = $3, status = 'responded', responded_at = NOW(), updated_at = NOW()
		WHERE id = $4
	`
	_, err = db.Pool.Exec(ctx, updateQuery, payload.QuotedPrice, payload.VendorResponse, payload.AttachmentURL, quoteID)
	if err != nil {
		log.Printf("ERROR: Failed to update quote response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update quote: " + err.Error()})
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

// PATCH /quotes/revision/:id
func (h *QuotesHandler) RequestRevision(c *gin.Context) {
	h.updateQuoteStatusByOrganizer(c, "revision_requested")
}

// GET /quotes/:id/contact
func (h *QuotesHandler) GetQuoteContact(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	quoteID := c.Param("id")
	ctx := c.Request.Context()

	// 1. Get quote details and verify authorization
	var status, organizerID, vendorID, eventID string
	query := `SELECT status, organizer_user_id, vendor_id, event_id FROM quote_requests WHERE id = $1`
	err := db.Pool.QueryRow(ctx, query, quoteID).Scan(&status, &organizerID, &vendorID, &eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quote not found"})
		return
	}

	// 2. Check if event is completed
	var eventStatus string
	err = db.Pool.QueryRow(ctx, "SELECT status FROM events WHERE id = $1", eventID).Scan(&eventStatus)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}
	if eventStatus == "completed" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Event is completed. Contact access revoked."})
		return
	}

	// 3. Authorization: Only the involved organizer or the vendor can access this
	isOrganizer := organizerID == userID.(string)

	// Check if user is the vendor
	var isVendor bool
	var actualVendorID string
	_ = db.Pool.QueryRow(ctx, "SELECT id FROM vendor_profiles WHERE owner_user_id = $1", userID.(string)).Scan(&actualVendorID)
	if actualVendorID == vendorID {
		isVendor = true
	}

	if !isOrganizer && !isVendor {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to view contact information for this quote"})
		return
	}

	// 4. Strict Gating: Only allowed if status is 'accepted'
	if status != "accepted" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Contact information is only available for accepted quotes"})
		return
	}

	// 5. Fetch contact details
	var vendorWhatsApp, vendorPhone, vendorEmail string
	var organizerName, organizerPhone, organizerEmail string

	// Vendor contacts (from vendor_profiles and users)
	vendorQuery := `
		SELECT vp.whatsapp_link, u.phone, u.email 
		FROM vendor_profiles vp
		JOIN users u ON vp.owner_user_id = u.id
		WHERE vp.id = $1
	`
	err = db.Pool.QueryRow(ctx, vendorQuery, vendorID).Scan(&vendorWhatsApp, &vendorPhone, &vendorEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch vendor contacts"})
		return
	}

	// Organizer contacts (from users)
	organizerQuery := `SELECT full_name, phone, email FROM users WHERE id = $1`
	err = db.Pool.QueryRow(ctx, organizerQuery, organizerID).Scan(&organizerName, &organizerPhone, &organizerEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch organizer contacts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"vendor": gin.H{
			"whatsapp": vendorWhatsApp,
			"phone":    vendorPhone,
			"email":    vendorEmail,
		},
		"organizer": gin.H{
			"name":  organizerName,
			"phone": organizerPhone,
			"email": organizerEmail,
		},
	})
}

func (h *QuotesHandler) updateQuoteStatusByOrganizer(c *gin.Context, newStatus string) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	organizerID := userID.(string)
	quoteID := c.Param("id")

	ctx := c.Request.Context()

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

	timestampColumn := ""
	switch newStatus {
	case "accepted":
		timestampColumn = "accepted_at = NOW(), contact_unlocked_at = NOW(),"
	case "rejected":
		timestampColumn = "rejected_at = NOW(),"
	case "revision_requested":
		timestampColumn = "revision_requested_at = NOW(),"
	}

	updateQuery := `UPDATE quote_requests SET ` + timestampColumn + ` status = $1, updated_at = NOW() WHERE id = $2`
	_, err = db.Pool.Exec(ctx, updateQuery, newStatus, quoteID)
	if err != nil {
		log.Printf("ERROR: Failed to update quote status (%s): %v", newStatus, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update quote status"})
		return
	}

	// Activity Log
	actionType := "quote_" + newStatus
	_, _ = db.Pool.Exec(ctx, `INSERT INTO platform_activity_log (entity_type, entity_id, action_type, actor_user_id) VALUES ('quote', $1, $2, $3)`, quoteID, actionType, organizerID)

	if newStatus == "accepted" {
		_, _ = db.Pool.Exec(ctx, `INSERT INTO platform_activity_log (entity_type, entity_id, action_type, actor_user_id) VALUES ('quote', $1, 'contact_unlocked', $2)`, quoteID, organizerID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Quote " + newStatus + " successfully"})
}
