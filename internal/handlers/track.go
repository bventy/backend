package handlers

import (
	"context"
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

	// Fire-and-forget: we don't care about the error returning to the client
	// Just return success immediately
	c.JSON(http.StatusOK, gin.H{"status": "tracked"})
}
