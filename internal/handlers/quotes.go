package handlers

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/bventy/backend/internal/config"
	"github.com/bventy/backend/internal/db"
	"github.com/bventy/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type QuotesHandler struct {
	MediaService *services.MediaService
}

func NewQuotesHandler(cfg *config.Config) *QuotesHandler {
	svc, _ := services.NewMediaService(cfg)
	return &QuotesHandler{MediaService: svc}
}

type CreateQuoteRequestPayload struct {
	EventID             string  `json:"event_id"` // Not required if inline event creation is used
	VendorID            string  `json:"vendor_id" binding:"required"`
	Message             string  `json:"message" binding:"required"`
	BudgetRange         *string `json:"budget_range"`         // Added
	SpecialRequirements *string `json:"special_requirements"` // Existing
	Deadline            *string `json:"deadline"`             // Existing (ISO string)
	// Inline event creation fields
	EventTitle     string `json:"event_title"`
	EventType      string `json:"event_type"`
	EventCity      string `json:"event_city"`
	EventDate      string `json:"event_date"`
	EventBudgetMin *int   `json:"event_budget_min"`
	EventBudgetMax *int   `json:"event_budget_max"`
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
	eventID := payload.EventID

	// 1. Inline Event Creation if no EventID
	if eventID == "" {
		if payload.EventTitle == "" || payload.EventCity == "" || payload.EventDate == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "event_id or event details (title, city, date) required"})
			return
		}

		parsedDate, err := time.Parse("2006-01-02", payload.EventDate)
		if err != nil {
			parsedDate, err = time.Parse(time.RFC3339, payload.EventDate)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event_date format (YYYY-MM-DD)"})
				return
			}
		}

		query := `
			INSERT INTO events (title, city, event_type, event_date, budget_min, budget_max, organizer_user_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id
		`
		err = db.Pool.QueryRow(ctx, query,
			payload.EventTitle, payload.EventCity, payload.EventType, parsedDate,
			payload.EventBudgetMin, payload.EventBudgetMax, organizerID,
		).Scan(&eventID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create inline event: " + err.Error()})
			return
		}
	} else {
		// Validate event belongs to user
		var eventOrganizerID string
		err := db.Pool.QueryRow(ctx, "SELECT organizer_user_id FROM events WHERE id = $1", eventID).Scan(&eventOrganizerID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}
		if eventOrganizerID != organizerID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not own this event"})
			return
		}
	}

	// 2. Validate vendor exists
	var vendorExists int
	err := db.Pool.QueryRow(ctx, "SELECT 1 FROM vendor_profiles WHERE id = $1", payload.VendorID).Scan(&vendorExists)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor not found"})
		return
	}

	// 3. Insert quote request
	var quoteID string
	var deadlineAt interface{} = nil
	if payload.Deadline != nil && *payload.Deadline != "" {
		d, err := time.Parse(time.RFC3339, *payload.Deadline)
		if err == nil {
			deadlineAt = d
		} else {
			d, err = time.Parse("2006-01-02", *payload.Deadline)
			if err == nil {
				deadlineAt = d
			}
		}
	}

	insertQuoteQuery := `
		INSERT INTO quote_requests (event_id, vendor_id, organizer_user_id, message, budget_range, special_requirements, deadline, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending')
		RETURNING id
	`
	err = db.Pool.QueryRow(ctx, insertQuoteQuery, eventID, payload.VendorID, organizerID, payload.Message, payload.BudgetRange, payload.SpecialRequirements, deadlineAt).Scan(&quoteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create quote request: " + err.Error()})
		return
	}

	// 4. Activity Log
	insertLogQuery := `
		INSERT INTO platform_activity_log (entity_type, entity_id, action_type, actor_user_id)
		VALUES ('quote', $1, 'quote_created', $2)
	`
	_, _ = db.Pool.Exec(ctx, insertLogQuery, quoteID, organizerID)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Quote requested successfully",
		"quote_id": quoteID,
		"event_id": eventID,
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
	var vendorID string
	err := db.Pool.QueryRow(ctx, "SELECT id FROM vendor_profiles WHERE owner_user_id = $1", userID.(string)).Scan(&vendorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor profile not found"})
		return
	}

	query := `
		SELECT qr.id, qr.event_id, e.title as event_title, qr.organizer_user_id, u.full_name as organizer_name, 
		       qr.message, qr.quoted_price, qr.vendor_response, qr.status, qr.responded_at, qr.created_at, qr.budget_range,
		       qr.special_requirements, qr.deadline, qr.attachment_url, qr.revision_requested_at
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
		var message, vendorResponse, budgetRange, specialRequirements, deadline, attachmentURL *string
		var quotedPrice *float64 // Changed from *int to *float64
		var respondedAt, createdAt, revisionRequestedAt interface{}

		err := rows.Scan(&id, &eventID, &eventTitle, &organizerID, &organizerName, &message, &quotedPrice, &vendorResponse, &status, &respondedAt, &createdAt, &budgetRange, &specialRequirements, &deadline, &attachmentURL, &revisionRequestedAt)
		if err != nil {
			log.Printf("ERROR: Error scanning vendor quote row: %v", err)
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
			"responded_at":          respondedAt,
			"created_at":            createdAt,
			"budget_range":          budgetRange,
			"special_requirements":  specialRequirements,
			"deadline":              deadline,
			"attachment_url":        attachmentURL,
			"revision_requested_at": revisionRequestedAt,
		})
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
		       qr.special_requirements, qr.deadline, qr.attachment_url, qr.revision_requested_at
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
		var message, vendorResponse, budgetRange, specialRequirements, deadline, attachmentURL *string
		var quotedPrice *float64 // Changed from *int to *float64
		var respondedAt, createdAt, revisionRequestedAt interface{}

		err := rows.Scan(&id, &eventID, &eventTitle, &vendorID, &vendorName, &message, &quotedPrice, &vendorResponse, &status, &respondedAt, &createdAt, &budgetRange, &specialRequirements, &deadline, &attachmentURL, &revisionRequestedAt)
		if err != nil {
			log.Printf("ERROR: Error scanning organizer quote row: %v", err)
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
			"responded_at":          respondedAt,
			"created_at":            createdAt,
			"budget_range":          budgetRange,
			"special_requirements":  specialRequirements,
			"deadline":              deadline,
			"attachment_url":        attachmentURL,
			"revision_requested_at": revisionRequestedAt,
		})
	}

	c.JSON(http.StatusOK, quotes)
}

// PATCH /quotes/respond/:id
func (h *QuotesHandler) RespondToQuote(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	quoteID := c.Param("id")
	ctx := c.Request.Context()

	// 1. Verify vendor owns this quote & state
	var vendorID string
	err := db.Pool.QueryRow(ctx, "SELECT id FROM vendor_profiles WHERE owner_user_id = $1", userID.(string)).Scan(&vendorID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only vendors can respond"})
		return
	}

	var quoteVendorID, status string
	err = db.Pool.QueryRow(ctx, "SELECT vendor_id, status FROM quote_requests WHERE id = $1", quoteID).Scan(&quoteVendorID, &status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quote not found"})
		return
	}
	if quoteVendorID != vendorID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not your quote"})
		return
	}
	if status != "pending" && status != "revision_requested" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quote cannot be responded to in current status: " + status})
		return
	}

	// 2. Parse payload and file
	// Using c.PostForm for quoted_price and vendor_response to handle multipart/form-data
	quotedPriceStr := c.PostForm("quoted_price")
	vendorResponse := c.PostForm("vendor_response")

	var quotedPrice *float64
	if quotedPriceStr != "" {
		price, parseErr := services.ParsePrice(quotedPriceStr)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quoted_price format"})
			return
		}
		quotedPrice = &price
	}

	var attachmentURL *string
	file, header, err := c.Request.FormFile("attachment")
	if err == nil {
		defer file.Close()
		// Validate
		if header.Size > 5*1024*1024 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Attachment too large (max 5MB)"})
			return
		}
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext != ".pdf" && ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Only PDF or Image attachments allowed"})
			return
		}

		// Upload to R2 (Private/Structured path)
		prefix := fmt.Sprintf("quotes/%s", quoteID)
		url, err := h.MediaService.UploadFile(file, header.Filename, header.Header.Get("Content-Type"), prefix)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload attachment"})
			return
		}
		attachmentURL = &url
	}

	// 3. Update DB
	updateQuery := `
		UPDATE quote_requests
		SET quoted_price = $1, vendor_response = $2, attachment_url = COALESCE($3, attachment_url), 
		    status = 'responded', responded_at = NOW(), updated_at = NOW()
		WHERE id = $4
	`
	_, err = db.Pool.Exec(ctx, updateQuery, quotedPrice, vendorResponse, attachmentURL, quoteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update quote"})
		return
	}

	// 4. Log
	_, _ = db.Pool.Exec(ctx, "INSERT INTO platform_activity_log (entity_type, entity_id, action_type, actor_user_id) VALUES ('quote', $1, 'quote_responded', $2)", quoteID, userID)

	c.JSON(http.StatusOK, gin.H{"message": "Quote responded successfully"})
}

// PATCH /quotes/accept/:id
func (h *QuotesHandler) AcceptQuote(c *gin.Context) {
	h.updateQuoteStatusByOrganizer(c, "accepted", "quote_accepted")
}

// PATCH /quotes/reject/:id
func (h *QuotesHandler) RejectQuote(c *gin.Context) {
	h.updateQuoteStatusByOrganizer(c, "rejected", "quote_rejected")
}

// PATCH /quotes/request-revision/:id
func (h *QuotesHandler) RequestRevision(c *gin.Context) {
	quoteID := c.Param("id")
	userID, _ := c.Get("userID")
	ctx := c.Request.Context()

	// Verify ownership
	var organizerID string
	err := db.Pool.QueryRow(ctx, "SELECT organizer_user_id FROM quote_requests WHERE id = $1", quoteID).Scan(&organizerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quote not found"})
		return
	}
	if organizerID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not your quote"})
		return
	}

	query := `UPDATE quote_requests SET status = 'revision_requested', revision_requested_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err = db.Pool.Exec(ctx, query, quoteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to request revision"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Revision requested"})
}

func (h *QuotesHandler) updateQuoteStatusByOrganizer(c *gin.Context, newStatus string, actionType string) {
	userID, _ := c.Get("userID")
	quoteID := c.Param("id")
	ctx := c.Request.Context()

	var organizerID string
	err := db.Pool.QueryRow(ctx, "SELECT organizer_user_id FROM quote_requests WHERE id = $1", quoteID).Scan(&organizerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quote not found"})
		return
	}
	if organizerID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		return
	}

	_, err = db.Pool.Exec(ctx, "UPDATE quote_requests SET status = $1, updated_at = NOW() WHERE id = $2", newStatus, quoteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
		return
	}

	_, _ = db.Pool.Exec(ctx, "INSERT INTO platform_activity_log (entity_type, entity_id, action_type, actor_user_id) VALUES ('quote', $1, $2, $3)", quoteID, actionType, userID)

	c.JSON(http.StatusOK, gin.H{"message": "Quote " + newStatus})
}

// GET /quotes/:id/attachment
func (h *QuotesHandler) GetAttachment(c *gin.Context) {
	userID, _ := c.Get("userID")
	quoteID := c.Param("id")
	ctx := c.Request.Context()

	var attachmentURL *string
	var organizerID, vendorID string
	query := `
		SELECT qr.attachment_url, qr.organizer_user_id, vp.owner_user_id
		FROM quote_requests qr
		JOIN vendor_profiles vp ON qr.vendor_id = vp.id
		WHERE qr.id = $1
	`
	err := db.Pool.QueryRow(ctx, query, quoteID).Scan(&attachmentURL, &organizerID, &vendorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quote or attachment not found"})
		return
	}
	if attachmentURL == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No attachment for this quote"})
		return
	}

	// Permission check
	currUserID := userID.(string)
	if currUserID != organizerID && currUserID != vendorID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to view this attachment"})
		return
	}

	signedURL, err := h.MediaService.GetSignedURL(*attachmentURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate signed URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": signedURL})
}
