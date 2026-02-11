package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/once-human/bventy-backend/internal/db"
)

type OrganizerHandler struct{}

func NewOrganizerHandler() *OrganizerHandler {
	return &OrganizerHandler{}
}

type OnboardOrganizerRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
	City        string `json:"city" binding:"required"`
}

func (h *OrganizerHandler) OnboardOrganizer(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req OnboardOrganizerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `
		INSERT INTO organizer_profiles (user_id, display_name, city)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	var organizerID string
	err := db.Pool.QueryRow(context.Background(), query, userID, req.DisplayName, req.City).Scan(&organizerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to onboard organizer: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Organizer profile created successfully", "organizer_id": organizerID})
}
