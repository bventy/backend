package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bventy/backend/internal/db"
	"github.com/bventy/backend/internal/services"
	"github.com/bventy/backend/internal/websocket"
	"github.com/gin-gonic/gin"
)

type QuotesHandler struct {
	MediaService *services.MediaService
	EmailService *services.EmailService
	Hub          *websocket.Hub
}

func NewQuotesHandler(emailService *services.EmailService, hub *websocket.Hub) *QuotesHandler {
	return &QuotesHandler{EmailService: emailService, Hub: hub}
}

type CreateQuoteRequestPayload struct {
	EventID             string  `json:"event_id" binding:"required"`
	VendorID            string  `json:"vendor_id" binding:"required"`
	Message             string  `json:"message" binding:"required"`
	BudgetRange         *string `json:"budget_range"`
	SpecialRequirements *string `json:"special_requirements"`
	Deadline            *string `json:"deadline"` // ISO string
}

type RevisionPayload struct {
	Message string `json:"message"`
}

type CreateManualLeadPayload struct {
	EventTitle  string `json:"event_title" binding:"required"`
	EventDate   string `json:"event_date" binding:"required"`
	EventType   string `json:"event_type"`
	Organizer   string `json:"organizer" binding:"required"` // Name of the organizer
	Budget      string `json:"budget"`
	Notes       string `json:"notes"`
	ContactInfo string `json:"contact_info"`
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
	err := db.Pool.QueryRow(ctx, "SELECT organizer_user_id FROM events WHERE id::text = $1", payload.EventID).Scan(&eventOrganizerID)
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
	err = db.Pool.QueryRow(ctx, "SELECT 1 FROM vendor_profiles WHERE id::text = $1", payload.VendorID).Scan(&vendorExists)
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

	// 5. Create Thread & Initial Quote System Card
	go func(qID, vID, oID string, payload CreateQuoteRequestPayload) {
		bgCtx := context.Background()

		// Insert Conversation
		var convID string
		err := db.Pool.QueryRow(bgCtx, `
			INSERT INTO conversations (quote_id, vendor_id, organizer_user_id, chat_locked) 
			VALUES ($1, $2, $3, true) RETURNING id
		`, qID, vID, oID).Scan(&convID)

		if err == nil {
			// Insert System Message
			sysPayload := map[string]interface{}{
				"event_id":             payload.EventID,
				"budget_range":         payload.BudgetRange,
				"special_requirements": payload.SpecialRequirements,
				"deadline":             payload.Deadline,
			}
			_, _ = db.Pool.Exec(bgCtx, `
				INSERT INTO messages (conversation_id, sender_user_id, message_type, system_payload)
				VALUES ($1, $2, 'quote_card', $3)
			`, convID, oID, sysPayload)
		} else {
			log.Printf("ERROR: Failed to create conversation for quote %s: %v", qID, err)
		}
	}(quoteID, payload.VendorID, organizerID, payload)

	// 6. Notifications
	go func() {
		var vendorEmail string
		_ = db.Pool.QueryRow(ctx, "SELECT u.email FROM users u JOIN vendor_profiles vp ON vp.owner_user_id = u.id WHERE vp.id::text = $1", payload.VendorID).Scan(&vendorEmail)
		if vendorEmail != "" {
			vars := map[string]string{
				"vendor_name": "Vendor",
				"event_title": "the requested event",
			}
			_ = h.EmailService.SendQuoteNotification(vendorEmail, "quote_requested", vars)
		}
	}()

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
	h.lazyUpdateQuotesAndEvents(ctx, userID.(string))

	// Get vendor ID from userID
	var vendorID string
	err := db.Pool.QueryRow(ctx, "SELECT id FROM vendor_profiles WHERE owner_user_id::text = $1", userID.(string)).Scan(&vendorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor profile not found for this user"})
		return
	}

	query := `
		SELECT qr.id, qr.event_id, e.title as event_title, qr.organizer_user_id, u.full_name as organizer_name, 
		       qr.message, qr.quoted_price, qr.vendor_response, qr.status, qr.responded_at, qr.created_at, qr.budget_range,
		       qr.special_requirements, qr.deadline, qr.attachment_url, qr.accepted_at, qr.rejected_at, qr.revision_requested_at, qr.contact_unlocked_at,
		       qr.contact_expires_at, qr.archived_at, qr.revision_message, e.event_date, e.event_type
		FROM quote_requests qr
		JOIN events e ON qr.event_id = e.id
		JOIN users u ON qr.organizer_user_id = u.id
		WHERE qr.vendor_id::text = $1
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
		var message, vendorResponse, budgetRange, specialReq, attachmentURL, revisionMsg, eventType *string
		var quotedPrice *float64
		var respondedAt, createdAt, acceptedAt, rejectedAt, revisionAt, unlockedAt, expiresAt, archivedAt, deadline, eventDate interface{}

		err := rows.Scan(
			&id, &eventID, &eventTitle, &organizerID, &organizerName, &message, &quotedPrice, &vendorResponse, &status,
			&respondedAt, &createdAt, &budgetRange, &specialReq, &deadline, &attachmentURL, &acceptedAt, &rejectedAt, &revisionAt, &unlockedAt,
			&expiresAt, &archivedAt, &revisionMsg, &eventDate, &eventType,
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
			"vendor_response":       vendorResponse,
			"status":                status,
			"created_at":            createdAt,
			"responded_at":          respondedAt,
			"accepted_at":           acceptedAt,
			"rejected_at":           rejectedAt,
			"revision_requested_at": revisionAt,
			"contact_unlocked_at":   unlockedAt,
			"special_requirements":  specialReq,
			"budget_range":          budgetRange,
			"deadline":              deadline,
			"attachment_url":        attachmentURL,
			"contact_expires_at":    expiresAt,
			"archived_at":           archivedAt,
			"revision_message":      revisionMsg,
			"event_date":            eventDate,
			"event_type":            eventType,
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
	h.lazyUpdateQuotesAndEvents(ctx, userID.(string))

	query := `
		SELECT qr.id, qr.event_id, e.title as event_title, qr.vendor_id, v.business_name as vendor_name, 
		       qr.message, qr.quoted_price, qr.vendor_response, qr.status, qr.responded_at, qr.created_at, qr.budget_range,
		       qr.special_requirements, qr.deadline, qr.attachment_url, qr.accepted_at, qr.rejected_at, qr.revision_requested_at, qr.contact_unlocked_at,
		       qr.contact_expires_at, qr.archived_at, qr.revision_message
		FROM quote_requests qr
		JOIN events e ON qr.event_id = e.id
		JOIN vendor_profiles v ON qr.vendor_id = v.id
		WHERE qr.organizer_user_id::text = $1
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
		var message, vendorResponse, budgetRange, specialReq, attachmentURL, revisionMsg *string
		var quotedPrice *float64
		var respondedAt, createdAt, acceptedAt, rejectedAt, revisionAt, unlockedAt, expiresAt, archivedAt, deadline interface{}

		err := rows.Scan(
			&id, &eventID, &eventTitle, &vendorID, &vendorName, &message, &quotedPrice, &vendorResponse, &status,
			&respondedAt, &createdAt, &budgetRange, &specialReq, &deadline, &attachmentURL, &acceptedAt, &rejectedAt, &revisionAt, &unlockedAt,
			&expiresAt, &archivedAt, &revisionMsg,
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
			"vendor_response":       vendorResponse,
			"status":                status,
			"created_at":            createdAt,
			"responded_at":          respondedAt,
			"accepted_at":           acceptedAt,
			"rejected_at":           rejectedAt,
			"revision_requested_at": revisionAt,
			"contact_unlocked_at":   unlockedAt,
			"special_requirements":  specialReq,
			"budget_range":          budgetRange,
			"deadline":              deadline,
			"attachment_url":        attachmentURL,
			"contact_expires_at":    expiresAt,
			"archived_at":           archivedAt,
			"revision_message":      revisionMsg,
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
	err := db.Pool.QueryRow(ctx, "SELECT id FROM vendor_profiles WHERE owner_user_id = $1::uuid", userID.(string)).Scan(&vendorID)
	if err != nil {
		log.Printf("DEBUG: respondToQuote - owner_user_id mismatch or no profile for user %s: %v", userID, err)
		c.JSON(http.StatusForbidden, gin.H{"error": "Only vendors with a profile can respond to quotes"})
		return
	}

	var quoteVendorID string
	err = db.Pool.QueryRow(ctx, "SELECT vendor_id FROM quote_requests WHERE id = $1::uuid", quoteID).Scan(&quoteVendorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quote not found or invalid ID format", "details": err.Error()})
		return
	}
	if quoteVendorID != vendorID {
		log.Printf("AUTHORIZATION FAILURE: Vendor profile ID (%s) does not match Quote Request vendor_id (%s) for quote %s", vendorID, quoteVendorID, quoteID)
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "You are not authorized to respond to this quote",
			"details": fmt.Sprintf("Vendor ID mismatch. Profile: %s, Quote: %s", vendorID, quoteVendorID),
		})
		return
	}

	// Update quote
	updateQuery := `
		UPDATE quote_requests
		SET quoted_price = $1, vendor_response = $2, attachment_url = $3, status = 'responded', responded_at = NOW(), updated_at = NOW()
		WHERE id = $4::uuid
	`
	_, err = db.Pool.Exec(ctx, updateQuery, payload.QuotedPrice, payload.VendorResponse, payload.AttachmentURL, quoteID)
	if err != nil {
		log.Printf("ERROR: Failed to update quote response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update quote", "details": err.Error()})
		return
	}

	// 3. Update Conversation & Insert System Message
	// We do this synchronously or at least log errors properly
	var convID string
	err = db.Pool.QueryRow(ctx, "SELECT id FROM conversations WHERE quote_id = $1::uuid", quoteID).Scan(&convID)
	if err == nil {
		sysPayload := map[string]interface{}{
			"quoted_price":    payload.QuotedPrice,
			"vendor_response": payload.VendorResponse,
			"attachment_url":  payload.AttachmentURL,
		}
		var msgID string
		var createdAt time.Time
		// Added explicit body as fallback
		fallbackBody := fmt.Sprintf("Quote Response: ₹%.2f", payload.QuotedPrice)
		err = db.Pool.QueryRow(ctx, `
			INSERT INTO messages (conversation_id, sender_user_id, message_type, body, system_payload)
			VALUES ($1::uuid, $2::uuid, 'quote_response', $3, $4)
			RETURNING id, created_at
		`, convID, userID.(string), fallbackBody, sysPayload).Scan(&msgID, &createdAt)

		if err != nil {
			log.Printf("ERROR: Failed to insert quote_response message for quote %s into conv %s: %v", quoteID, convID, err)
		} else {
			log.Printf("SUCCESS: Inserted quote_response message %s for quote %s", msgID, quoteID)
			// Broadcast (can be background)
			go func(cID string, mID string, uID string, sp map[string]interface{}, ca time.Time, body string) {
				h.Hub.Broadcast <- websocket.MessageEvent{
					Type:           "new_message",
					ConversationID: cID,
					Payload: map[string]interface{}{
						"id":             mID,
						"sender_user_id": uID,
						"message_type":   "quote_response",
						"body":           body,
						"system_payload": sp,
						"created_at":     ca,
					},
				}
			}(convID, msgID, userID.(string), sysPayload, createdAt, fallbackBody)
		}
	} else {
		log.Printf("CRITICAL WARNING: No conversation found for quote %s. User %s attempted to respond. DB error: %v", quoteID, userID, err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Quote responded successfully"})
}

// PATCH /quotes/accept/:id
func (h *QuotesHandler) AcceptQuote(c *gin.Context) {
	h.updateQuoteStatusByOrganizer(c, "accepted", "")
}

// PATCH /quotes/reject/:id
func (h *QuotesHandler) RejectQuote(c *gin.Context) {
	h.updateQuoteStatusByOrganizer(c, "rejected", "")
}

// PATCH /quotes/revision/:id
func (h *QuotesHandler) GetQuoteById(c *gin.Context) {
	quoteID := c.Param("id")
	userID, _ := c.Get("userID")
	ctx := c.Request.Context()

	type QuoteDetail struct {
		ID                  string     `json:"id"`
		Status              *string    `json:"status"`
		Message             *string    `json:"message"`
		Deadline            *time.Time `json:"deadline"`
		BudgetRange         *string    `json:"budget_range"`
		SpecialRequirements *string    `json:"special_requirements"`
		QuotedPrice         *float64   `json:"quoted_price"`
		VendorResponse      *string    `json:"vendor_response"`
		AttachmentURL       *string    `json:"attachment_url"`
		CreatedBy           string     `json:"organizer_user_id"`
		CreatedAt           time.Time  `json:"created_at"`
		RespondedAt         *time.Time `json:"responded_at"`
		AcceptedAt          *time.Time `json:"accepted_at"`
		RejectedAt          *time.Time `json:"rejected_at"`
		RevisionRequestedAt *time.Time `json:"revision_requested_at"`
		RevisionMessage     *string    `json:"revision_message"`
		InternalNotes       *string    `json:"internal_notes"`
		ConversationID      *string    `json:"conversation_id"`

		Event struct {
			ID        string     `json:"id"`
			Title     string     `json:"title"`
			EventDate *time.Time `json:"event_date"`
			City      *string    `json:"city"`
			EventType *string    `json:"event_type"`
			BudgetMin *int       `json:"budget_min"`
			BudgetMax *int       `json:"budget_max"`
		} `json:"event"`

		Organizer struct {
			FullName string `json:"full_name"`
		} `json:"organizer"`

		Vendor struct {
			ID           string `json:"id"`
			BusinessName string `json:"business_name"`
		} `json:"vendor"`
	}

	var q QuoteDetail
	var vendorOwnerID, organizerID string
	err := db.Pool.QueryRow(ctx, `
		SELECT 
			qr.id, qr.status, qr.message, qr.deadline, qr.budget_range, qr.special_requirements,
			qr.quoted_price, qr.vendor_response, qr.attachment_url, qr.created_at,
			qr.responded_at, qr.accepted_at, qr.rejected_at, qr.revision_requested_at, qr.revision_message, qr.internal_notes,
			e.id, e.title, e.event_date, e.city, e.event_type, e.budget_min, e.budget_max,
			u.full_name,
			vp.id, vp.business_name,
			vp.owner_user_id, e.organizer_user_id,
			c.id as conversation_id
		FROM quote_requests qr
		JOIN events e ON qr.event_id = e.id
		JOIN users u ON e.organizer_user_id = u.id
		JOIN vendor_profiles vp ON qr.vendor_id = vp.id
		LEFT JOIN conversations c ON qr.id = c.quote_id
		WHERE qr.id = $1
	`, quoteID).Scan(
		&q.ID, &q.Status, &q.Message, &q.Deadline, &q.BudgetRange, &q.SpecialRequirements,
		&q.QuotedPrice, &q.VendorResponse, &q.AttachmentURL, &q.CreatedAt,
		&q.RespondedAt, &q.AcceptedAt, &q.RejectedAt, &q.RevisionRequestedAt, &q.RevisionMessage, &q.InternalNotes,
		&q.Event.ID, &q.Event.Title, &q.Event.EventDate, &q.Event.City, &q.Event.EventType, &q.Event.BudgetMin, &q.Event.BudgetMax,
		&q.Organizer.FullName,
		&q.Vendor.ID, &q.Vendor.BusinessName,
		&vendorOwnerID, &organizerID,
		&q.ConversationID,
	)

	if err != nil {
		log.Printf("ERROR: Failed to scan quote %s: %v", quoteID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load lead details due to scanning error"})
		return
	}

	if userID.(string) != vendorOwnerID && userID.(string) != organizerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have access to this quote"})
		return
	}

	c.JSON(http.StatusOK, q)
}

func (h *QuotesHandler) UpdateInternalNotes(c *gin.Context) {
	quoteID := c.Param("id")
	userID, _ := c.Get("userID")
	ctx := c.Request.Context()

	var payload struct {
		Notes string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	// Verify ownership
	var vendorOwnerID string
	err := db.Pool.QueryRow(ctx, `
		SELECT vp.owner_user_id
		FROM quote_requests qr
		JOIN vendor_profiles vp ON qr.vendor_id = vp.id
		WHERE qr.id = $1
	`, quoteID).Scan(&vendorOwnerID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quote not found"})
		return
	}

	if userID.(string) != vendorOwnerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to add notes to this quote"})
		return
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE quote_requests SET internal_notes = $1 WHERE id = $2
	`, payload.Notes, quoteID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update notes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notes updated successfully"})
}

func (h *QuotesHandler) RequestRevision(c *gin.Context) {
	var payload RevisionPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message is required for revision requests"})
		return
	}
	h.updateQuoteStatusByOrganizer(c, "revision_requested", payload.Message)
}

func (h *QuotesHandler) GetQuoteContact(c *gin.Context) {
	quoteID := c.Param("id")
	userID, _ := c.Get("userID")
	ctx := c.Request.Context()

	var qr struct {
		OrganizerUserID string
		VendorUserID    string
		Status          string
		UnlockedAt      *time.Time
	}

	err := db.Pool.QueryRow(ctx, `
		SELECT qr.organizer_user_id, vp.owner_user_id, qr.status, qr.contact_unlocked_at
		FROM quote_requests qr
		JOIN vendor_profiles vp ON qr.vendor_id = vp.id
		WHERE qr.id = $1
	`, quoteID).Scan(&qr.OrganizerUserID, &qr.VendorUserID, &qr.Status, &qr.UnlockedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quote not found"})
		return
	}

	if qr.OrganizerUserID != userID.(string) && qr.VendorUserID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	if qr.UnlockedAt == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Contact not yet unlocked"})
		return
	}

	var organizer struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	}
	var vendor struct {
		WhatsApp string `json:"whatsapp"`
		Phone    string `json:"phone"`
		Email    string `json:"email"`
	}

	// Fetch Organizer Info
	err = db.Pool.QueryRow(ctx, `
		SELECT full_name, email, phone FROM users WHERE id = $1
	`, qr.OrganizerUserID).Scan(&organizer.Name, &organizer.Email, &organizer.Phone)
	if err != nil {
		log.Printf("Error fetching organizer info: %v", err)
	}

	// Fetch Vendor Info
	err = db.Pool.QueryRow(ctx, `
		SELECT u.email, u.phone, vp.whatsapp_link 
		FROM users u
		JOIN vendor_profiles vp ON u.id = vp.owner_user_id
		WHERE u.id = $1
	`, qr.VendorUserID).Scan(&vendor.Email, &vendor.Phone, &vendor.WhatsApp)
	if err != nil {
		log.Printf("Error fetching vendor info: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"organizer": organizer,
		"vendor":    vendor,
	})
}

func (h *QuotesHandler) updateQuoteStatusByOrganizer(c *gin.Context, newStatus string, revisionMessage string) {
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
	if newStatus == "accepted" {
		// Policy: Earlier of (event_date + 15 days) OR (NOW + 30 days)
		var eventDate *time.Time
		err := db.Pool.QueryRow(ctx, "SELECT event_date FROM events WHERE id = (SELECT event_id FROM quote_requests WHERE id = $1)", quoteID).Scan(&eventDate)

		approvalExpiry := time.Now().AddDate(0, 0, 30)
		expiry := approvalExpiry

		if err == nil && eventDate != nil {
			eventCompletionExpiry := eventDate.AddDate(0, 0, 15)
			// Choose the EARLIER (lower value) of the two as requested
			if eventCompletionExpiry.Before(approvalExpiry) {
				expiry = eventCompletionExpiry
			}
		}

		updateQuery := `UPDATE quote_requests SET accepted_at = NOW(), contact_unlocked_at = NOW(), contact_expires_at = $1, status = $2, updated_at = NOW() WHERE id = $3`
		_, err = db.Pool.Exec(ctx, updateQuery, expiry, newStatus, quoteID)
	} else {
		switch newStatus {
		case "rejected":
			timestampColumn = "rejected_at = NOW(),"
		case "revision_requested":
			timestampColumn = "revision_requested_at = NOW(), revision_message = $1,"
		}
		if newStatus == "revision_requested" {
			updateQuery := `UPDATE quote_requests SET ` + timestampColumn + ` status = $2, updated_at = NOW() WHERE id = $3`
			_, err = db.Pool.Exec(ctx, updateQuery, revisionMessage, newStatus, quoteID)
		} else {
			updateQuery := `UPDATE quote_requests SET ` + timestampColumn + ` status = $1, updated_at = NOW() WHERE id = $2`
			_, err = db.Pool.Exec(ctx, updateQuery, newStatus, quoteID)
		}
	}

	if err != nil {
		log.Printf("ERROR: Failed to update quote status (%s): %v", newStatus, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update quote status"})
		return
	}

	// Activity Log & Status Change Message
	// Perform message insertion synchronously to catch errors
	var convID string
	err = db.Pool.QueryRow(ctx, "SELECT id FROM conversations WHERE quote_id = $1::uuid", quoteID).Scan(&convID)
	if err == nil {
		// Activity Log
		actionType := "quote_" + newStatus
		_, _ = db.Pool.Exec(ctx, `INSERT INTO platform_activity_log (entity_type, entity_id, action_type, actor_user_id) VALUES ('quote', $1::uuid, $2, $3::uuid)`, quoteID, actionType, organizerID)

		msgType := "quote_" + newStatus
		sysPayload := map[string]interface{}{
			"message": revisionMessage,
		}
		var messageID string
		var createdAt time.Time
		fallbackBody := "Quote " + newStatus
		err = db.Pool.QueryRow(ctx, `
			INSERT INTO messages (conversation_id, sender_user_id, message_type, body, system_payload)
			VALUES ($1::uuid, $2::uuid, $3, $4, $5)
			RETURNING id, created_at
		`, convID, organizerID, msgType, fallbackBody, sysPayload).Scan(&messageID, &createdAt)

		if err != nil {
			log.Printf("ERROR: Failed to insert status message (%s) for quote %s: %v", msgType, quoteID, err)
		} else {
			log.Printf("SUCCESS: Inserted status message %s of type %s", messageID, msgType)
			// Broadcast
			h.Hub.Broadcast <- websocket.MessageEvent{
				Type:           "new_message",
				ConversationID: convID,
				Payload: map[string]interface{}{
					"id":             messageID,
					"sender_user_id": organizerID,
					"message_type":   msgType,
					"body":           fallbackBody,
					"system_payload": sysPayload,
					"created_at":     createdAt,
				},
			}

			if newStatus == "accepted" {
				// 1. Unlock Chat in DB
				_, err = db.Pool.Exec(ctx, `UPDATE conversations SET chat_locked = false WHERE id = $1::uuid`, convID)
				if err != nil {
					log.Printf("ERROR: Failed to unlock chat for conv %s: %v", convID, err)
				}

				// 2. Also insert/broadcast "Chat unlocked"
				var unlockID string
				var unlockCreated time.Time
				err = db.Pool.QueryRow(ctx, `
					INSERT INTO messages (conversation_id, message_type, body) 
					VALUES ($1::uuid, 'system', 'Chat unlocked. You can now communicate directly.')
					RETURNING id, created_at
				`, convID).Scan(&unlockID, &unlockCreated)

				if err == nil {
					h.Hub.Broadcast <- websocket.MessageEvent{
						Type:           "new_message",
						ConversationID: convID,
						Payload: map[string]interface{}{
							"id":           unlockID,
							"message_type": "system",
							"body":         "Chat unlocked. You can now communicate directly.",
							"created_at":   unlockCreated,
						},
					}
				}
			}
		}
	} else {
		log.Printf("CRITICAL WARNING: No conversation found for quote %s during status update (%s). DB error: %v", quoteID, newStatus, err)
	}

	// Notifications - USE context.Background() in goroutines
	go func() {
		bgCtx := context.Background()
		var recipientEmail string
		templateKey := "quote_" + newStatus
		if newStatus == "accepted" || newStatus == "rejected" || newStatus == "revision_requested" {
			// Notify Vendor
			_ = db.Pool.QueryRow(bgCtx, "SELECT u.email FROM users u JOIN vendor_profiles vp ON vp.owner_user_id = u.id WHERE vp.id = (SELECT vendor_id FROM quote_requests WHERE id = $1)", quoteID).Scan(&recipientEmail)
		} else if newStatus == "responded" {
			// Notify Organizer
			_ = db.Pool.QueryRow(bgCtx, "SELECT u.email FROM users u JOIN quote_requests qr ON qr.organizer_user_id = u.id WHERE qr.id = $1", quoteID).Scan(&recipientEmail)
		}

		if recipientEmail != "" {
			vars := map[string]string{
				"quote_id":    quoteID,
				"event_title": "the event",
			}
			_ = h.EmailService.SendQuoteNotification(recipientEmail, templateKey, vars)
		}
	}()
	c.JSON(http.StatusOK, gin.H{"message": "Quote " + newStatus + " successfully"})
}

func (h *QuotesHandler) lazyUpdateQuotesAndEvents(ctx context.Context, userID string) {
	// 1. Auto-complete events: event_date < CURRENT_DATE
	updateEventsQuery := `
		UPDATE events 
		SET status = 'completed', completed_at = NOW() 
		WHERE organizer_user_id = $1 AND event_date < CURRENT_DATE AND status != 'completed'
	`
	_, _ = db.Pool.Exec(ctx, updateEventsQuery, userID)

	// 2. Auto-archive quotes based on the multi-stage policy
	// Check if user is vendor first
	var vendorID string
	_ = db.Pool.QueryRow(ctx, "SELECT id FROM vendor_profiles WHERE owner_user_id = $1", userID).Scan(&vendorID)

	archiveQueries := []struct {
		Query string
		Args  []interface{}
	}{
		{
			// Accepted Quotes: Archive 5 days after contact access expires
			Query: `UPDATE quote_requests SET status = 'archived', archived_at = NOW() 
					WHERE (organizer_user_id = $1 OR vendor_id = $2) AND status = 'accepted' 
					AND contact_expires_at + INTERVAL '5 days' < NOW() AND archived_at IS NULL RETURNING id`,
			Args: []interface{}{userID, vendorID},
		},
		{
			// Pending Quotes: Archive 30 days after creation (Vendor response window)
			Query: `UPDATE quote_requests SET status = 'archived', archived_at = NOW() 
					WHERE (organizer_user_id = $1 OR vendor_id = $2) AND status = 'pending' 
					AND created_at + INTERVAL '30 days' < NOW() AND archived_at IS NULL RETURNING id`,
			Args: []interface{}{userID, vendorID},
		},
		{
			// Other Non-Approved (responded, revision_requested, rejected): Archive 20 days after last update
			Query: `UPDATE quote_requests SET status = 'archived', archived_at = NOW() 
					WHERE (organizer_user_id = $1 OR vendor_id = $2) AND status IN ('responded', 'revision_requested', 'rejected') 
					AND updated_at + INTERVAL '20 days' < NOW() AND archived_at IS NULL RETURNING id`,
			Args: []interface{}{userID, vendorID},
		},
	}

	for _, aq := range archiveQueries {
		rows, err := db.Pool.Query(ctx, aq.Query, aq.Args...)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var quoteID string
				if err := rows.Scan(&quoteID); err == nil {
					queryLog := `INSERT INTO platform_activity_log (entity_type, entity_id, action_type, actor_user_id, metadata) VALUES ('quote', $1, 'quote_archived', $2, $3)`
					metadata := fmt.Sprintf(`{"triggered_by": "lazy_cleanup", "user_context": "%s"}`, userID)
					_, _ = db.Pool.Exec(ctx, queryLog, quoteID, userID, metadata)

					// Lock Chat & Notify Expiry
					go func(qID string) {
						bgCtx := context.Background()
						var convID string
						err := db.Pool.QueryRow(bgCtx, `UPDATE conversations SET chat_locked = true WHERE quote_id = $1 RETURNING id`, qID).Scan(&convID)
						if err == nil {
							_, _ = db.Pool.Exec(bgCtx, `INSERT INTO messages (conversation_id, message_type, body) VALUES ($1, 'system', 'Chat access expired.')`, convID)
						}
					}(quoteID)
				}
			}
		}
	}
}

// PATCH /quotes/vendor/reject/:id
func (h *QuotesHandler) RejectQuoteByVendor(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	quoteID := c.Param("id")
	ctx := c.Request.Context()

	// Verify vendor ownership
	var vendorID string
	err := db.Pool.QueryRow(ctx, "SELECT id FROM vendor_profiles WHERE owner_user_id = $1", userID.(string)).Scan(&vendorID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Vendor profile not found"})
		return
	}

	res, err := db.Pool.Exec(ctx, `
		UPDATE quote_requests 
		SET status = 'rejected', rejected_at = NOW(), updated_at = NOW() 
		WHERE id = $1 AND vendor_id = $2
	`, quoteID, vendorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject quote"})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quote not found or unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Quote rejected successfully"})
}

// PATCH /quotes/vendor/confirm/:id
func (h *QuotesHandler) ConfirmQuoteByVendor(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	quoteID := c.Param("id")
	ctx := c.Request.Context()

	// Verify vendor ownership
	var vendorID string
	err := db.Pool.QueryRow(ctx, "SELECT id FROM vendor_profiles WHERE owner_user_id = $1", userID.(string)).Scan(&vendorID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Vendor profile not found"})
		return
	}

	// Move to 'responded' status
	res, err := db.Pool.Exec(ctx, `
		UPDATE quote_requests 
		SET status = 'responded', responded_at = NOW(), updated_at = NOW() 
		WHERE id = $1 AND vendor_id = $2 AND status = 'pending'
	`, quoteID, vendorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm hold"})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quote not found, already responded, or unauthorized"})
		return
	}

	// Add a system message to the conversation
	var convID string
	err = db.Pool.QueryRow(ctx, "SELECT id FROM conversations WHERE quote_id = $1", quoteID).Scan(&convID)
	if err == nil {
		_, _ = db.Pool.Exec(ctx, `
			INSERT INTO messages (conversation_id, sender_user_id, message_type, body)
			VALUES ($1, $2, 'text', 'I have confirmed my availability for this date. Let’s discuss further!')
		`, convID, userID.(string))
	}

	c.JSON(http.StatusOK, gin.H{"message": "Hold confirmed successfully"})
}

// POST /quotes/manual (Vendors only)
func (h *QuotesHandler) CreateManualQuote(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var payload CreateManualLeadPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Get vendor ID
	var vendorID string
	err := db.Pool.QueryRow(ctx, "SELECT id FROM vendor_profiles WHERE owner_user_id::text = $1", userID.(string)).Scan(&vendorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor profile not found"})
		return
	}

	// 1. Create a "Ghost" Event for this manual lead
	var eventID string
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO events (title, event_type, event_date, organizer_user_id, status)
		VALUES ($1, $2, $3, $4, 'draft')
		RETURNING id
	`, payload.EventTitle, payload.EventType, payload.EventDate, userID.(string)).Scan(&eventID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ghost event: " + err.Error()})
		return
	}

	// 2. Create the Quote Request
	var quoteID string
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO quote_requests (
			event_id, vendor_id, organizer_user_id, message, budget_range, status, internal_notes
		)
		VALUES ($1, $2, $3, $4, $5, 'accepted', $6)
		RETURNING id
	`, eventID, vendorID, userID.(string), "Manual Entry: "+payload.Organizer, payload.Budget, payload.Notes).Scan(&quoteID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create manual lead: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Manual lead created successfully",
		"quote_id": quoteID,
	})
}
