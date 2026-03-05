package handlers

import (
	"context"
	"math/rand"
	"net/http"

	"github.com/bventy/backend/internal/db"
	"github.com/gin-gonic/gin"
)

type TrackHandler struct{}

func NewTrackHandler() *TrackHandler {
	return &TrackHandler{}
}

type TrackActivityPayload struct {
	EntityType string      `json:"entity_type" binding:"required"`
	EntityID   string      `json:"entity_id" binding:"required"`
	ActionType string      `json:"action_type" binding:"required"`
	Metadata   interface{} `json:"metadata"`
}

// POST /track/activity
// Unified fire-and-forget tracking endpoint
func (h *TrackHandler) TrackActivity(c *gin.Context) {
	var payload TrackActivityPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var actorUserID *string
	userID, exists := c.Get("userID")
	if exists {
		idStr := userID.(string)
		actorUserID = &idStr
	}

	ctx := context.Background()

	insertLogQuery := `
		INSERT INTO platform_activity_log (entity_type, entity_id, action_type, actor_user_id, metadata)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, _ = db.Pool.Exec(ctx, insertLogQuery, payload.EntityType, payload.EntityID, payload.ActionType, actorUserID, payload.Metadata)

	// Smart View Tracking for Vendors
	if payload.EntityType == "vendor" && payload.ActionType == "view" {
		// Only count views if the actor is not the owner
		go h.handleVendorView(payload.EntityID, actorUserID)
	}

	// Fire-and-forget: we don't care about the error returning to the client
	// Just return success immediately
	c.JSON(http.StatusOK, gin.H{"status": "tracked"})
}

func (h *TrackHandler) handleVendorView(vendorID string, actorUserID *string) {
	ctx := context.Background()

	// 1. Check if actor is the owner
	if actorUserID != nil {
		var ownerID string
		err := db.Pool.QueryRow(ctx, "SELECT owner_user_id::text FROM vendor_profiles WHERE id::text = $1", vendorID).Scan(&ownerID)
		if err == nil && ownerID == *actorUserID {
			// Owner viewed their own profile - don't count
			return
		}
	}

	// 2. Proceeed with view count increment (Smart View logic)
	var currentViews int64
	err := db.Pool.QueryRow(ctx, "SELECT views_count FROM vendor_profiles WHERE id = $1", vendorID).Scan(&currentViews)
	if err != nil {
		return
	}

	increment := int64(0)
	probability := 1.0

	if currentViews < 100 {
		increment = 1
		probability = 1.0
	} else if currentViews < 1000 {
		increment = 10
		probability = 0.1
	} else {
		increment = 100
		probability = 0.01
	}

	if rand.Float64() <= probability {
		_, _ = db.Pool.Exec(ctx, "UPDATE vendor_profiles SET views_count = views_count + $1 WHERE id = $2", increment, vendorID)
	}
}
