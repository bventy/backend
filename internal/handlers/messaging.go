package handlers

import (
	"log"
	"net/http"

	"github.com/bventy/backend/internal/db"
	"github.com/bventy/backend/internal/websocket"
	"github.com/gin-gonic/gin"
)

type MessagingHandler struct {
	Hub *websocket.Hub
}

func NewMessagingHandler(hub *websocket.Hub) *MessagingHandler {
	return &MessagingHandler{Hub: hub}
}

// GET /conversations
func (h *MessagingHandler) GetConversations(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	ctx := c.Request.Context()

	// Check if vendor
	var vendorID string
	_ = db.Pool.QueryRow(ctx, "SELECT id FROM vendor_profiles WHERE owner_user_id = $1", userID.(string)).Scan(&vendorID)

	query := `
		SELECT 
			c.id, c.quote_id, c.vendor_id, c.organizer_user_id, c.chat_locked, 
			c.last_message_at, c.created_at,
			e.title as event_title, 
			v.business_name as vendor_name,
			u.full_name as organizer_name,
			(SELECT COUNT(*) FROM messages m LEFT JOIN message_reads mr ON m.id = mr.message_id AND mr.user_id::text = $1 WHERE m.conversation_id = c.id AND mr.read_at IS NULL AND m.sender_user_id::text != $1) as unread_count,
            qr.status as quote_status
		FROM conversations c
		JOIN quote_requests qr ON c.quote_id = qr.id
		JOIN events e ON qr.event_id = e.id
		JOIN vendor_profiles v ON c.vendor_id = v.id
		LEFT JOIN users u ON c.organizer_user_id = u.id
		WHERE (c.organizer_user_id::text = $1 OR v.owner_user_id::text = $1)
		ORDER BY c.last_message_at DESC NULLS LAST, c.created_at DESC
	`

	rows, err := db.Pool.Query(ctx, query, userID.(string))
	if err != nil {
		log.Printf("ERROR FETCHING CONVERSATIONS for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch conversations", "details": err.Error()})
		return
	}
	defer rows.Close()

	var conversations []gin.H
	for rows.Next() {
		var id, quoteID, vID, eventTitle, vendorName, quoteStatus string
		var organizerName, oID *string
		var locked bool
		var lastMessageAt, createdAt interface{}
		var unreadCount int64

		if err := rows.Scan(&id, &quoteID, &vID, &oID, &locked, &lastMessageAt, &createdAt, &eventTitle, &vendorName, &organizerName, &unreadCount, &quoteStatus); err != nil {
			log.Printf("ERROR SCANNING CONVERSATION: %v", err)
			continue
		}

		conversations = append(conversations, gin.H{
			"id":                id,
			"quote_id":          quoteID,
			"vendor_id":         vID,
			"organizer_user_id": oID,
			"chat_locked":       locked,
			"last_message_at":   lastMessageAt,
			"created_at":        createdAt,
			"event_title":       eventTitle,
			"vendor_name":       vendorName,
			"organizer_name":    organizerName,
			"unread_count":      unreadCount,
			"quote_status":      quoteStatus,
		})
	}

	if conversations == nil {
		conversations = []gin.H{}
	}
	c.JSON(http.StatusOK, conversations)
}

// GET /conversations/:id/messages
func (h *MessagingHandler) GetMessages(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	conversationID := c.Param("id")
	ctx := c.Request.Context()

	// Validate access exists
	var vendorID string
	_ = db.Pool.QueryRow(ctx, "SELECT id FROM vendor_profiles WHERE owner_user_id = $1", userID.(string)).Scan(&vendorID)

	var hasAccess int
	authQuery := `SELECT 1 FROM conversations WHERE id::text = $1 AND (organizer_user_id::text = $2 OR vendor_id::text = $3)`
	if err := db.Pool.QueryRow(ctx, authQuery, conversationID, userID.(string), vendorID).Scan(&hasAccess); err != nil {
		log.Printf("ACCESS DENIED for user %s to conv %s: %v", userID, conversationID, err)
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have access to this conversation", "details": err.Error()})
		return
	}

	// Fetch messages
	query := `
		SELECT 
			m.id, m.sender_user_id, m.message_type, m.body, 
			m.attachment_url, m.attachment_type, m.system_payload,
			m.created_at, m.edited_at, m.deleted_at,
			u.full_name as sender_name,
			(SELECT COUNT(user_id) FROM message_reads WHERE message_id = m.id AND user_id != m.sender_user_id) > 0 as is_read
		FROM messages m
		LEFT JOIN users u ON m.sender_user_id = u.id
		WHERE m.conversation_id = $1
		ORDER BY m.created_at ASC
	`
	rows, err := db.Pool.Query(ctx, query, conversationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}
	defer rows.Close()

	var messages []gin.H
	for rows.Next() {
		var id, msgType string
		var senderID, body, attURL, attType, senderName *string
		var sysPayload interface{} // JSONB
		var createdAt, editedAt, deletedAt interface{}
		var isRead bool

		if err := rows.Scan(&id, &senderID, &msgType, &body, &attURL, &attType, &sysPayload, &createdAt, &editedAt, &deletedAt, &senderName, &isRead); err != nil {
			continue
		}

		messages = append(messages, gin.H{
			"id":              id,
			"sender_user_id":  senderID,
			"sender_name":     senderName,
			"message_type":    msgType,
			"body":            body,
			"attachment_url":  attURL,
			"attachment_type": attType,
			"system_payload":  sysPayload,
			"created_at":      createdAt,
			"edited_at":       editedAt,
			"deleted_at":      deletedAt,
			"is_read":         isRead,
		})
	}

	if messages == nil {
		messages = []gin.H{}
	}
	c.JSON(http.StatusOK, messages)
}

type SendMessagePayload struct {
	MessageType    string  `json:"message_type" binding:"required"`
	Body           *string `json:"body"`
	AttachmentURL  *string `json:"attachment_url"`
	AttachmentType *string `json:"attachment_type"`
}

// POST /conversations/:id/messages
func (h *MessagingHandler) SendMessage(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	conversationID := c.Param("id")

	var payload SendMessagePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// 1. Authorization & Locking check
	var vendorID string
	_ = db.Pool.QueryRow(ctx, "SELECT id FROM vendor_profiles WHERE owner_user_id = $1", userID.(string)).Scan(&vendorID)

	var chatLocked bool
	var quoteStatus string
	checkQuery := `
		SELECT c.chat_locked, qr.status 
		FROM conversations c 
		JOIN quote_requests qr ON c.quote_id = qr.id 
		WHERE c.id::text = $1 AND (c.organizer_user_id::text = $2 OR c.vendor_id::text = $3)
	`
	err := db.Pool.QueryRow(ctx, checkQuery, conversationID, userID.(string), vendorID).Scan(&chatLocked, &quoteStatus)
	if err != nil {
		log.Printf("SEND DENIED for user %s to conv %s: %v", userID, conversationID, err)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied or conversation not found", "details": err.Error()})
		return
	}

	if chatLocked || quoteStatus != "accepted" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chat is locked until quote is accepted."})
		return
	}

	// 2. Insert Message
	insertQuery := `
		INSERT INTO messages (conversation_id, sender_user_id, message_type, body, attachment_url, attachment_type)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	var messageID string
	var createdAt interface{}
	err = db.Pool.QueryRow(ctx, insertQuery, conversationID, userID.(string), payload.MessageType, payload.Body, payload.AttachmentURL, payload.AttachmentType).Scan(&messageID, &createdAt)
	if err != nil {
		log.Printf("ERROR: Failed to save message: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	// 3. Update Conversation last_message time
	_, _ = db.Pool.Exec(ctx, "UPDATE conversations SET last_message_at = NOW() WHERE id = $1", conversationID)

	// 4. Broadcast to WebSocket Room
	event := websocket.MessageEvent{
		Type:           "new_message",
		ConversationID: conversationID,
		Payload: map[string]interface{}{
			"id":              messageID,
			"sender_user_id":  userID.(string),
			"message_type":    payload.MessageType,
			"body":            payload.Body,
			"attachment_url":  payload.AttachmentURL,
			"attachment_type": payload.AttachmentType,
			"created_at":      createdAt,
		},
	}

	// Non-blocking dispatch to the Hub
	go func() {
		h.Hub.Broadcast <- event
	}()

	c.JSON(http.StatusOK, gin.H{
		"message_id": messageID,
		"created_at": createdAt,
		"status":     "sent",
	})
}

// PATCH /conversations/:id/read
func (h *MessagingHandler) MarkAsRead(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	conversationID := c.Param("id")
	ctx := c.Request.Context()

	// Insert into message_reads for all unread messages in this conversation not sent by me
	query := `
		INSERT INTO message_reads (message_id, user_id)
		SELECT m.id, $1 
		FROM messages m
		WHERE m.conversation_id::text = $2 
		AND m.sender_user_id::text != $1
		ON CONFLICT (message_id, user_id) DO NOTHING
	`
	_, err := db.Pool.Exec(ctx, query, userID.(string), conversationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark messages as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
