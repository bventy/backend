package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/once-human/bventy-backend/internal/db"
)

type GroupHandler struct{}

func NewGroupHandler() *GroupHandler {
	return &GroupHandler{}
}

type CreateGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	City        string `json:"city"`
	Description string `json:"description"`
}

func (h *GroupHandler) CreateGroup(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slug := generateSlug(req.Name, req.City)

	// Transaction to create group AND add owner as member
	tx, err := db.Pool.Begin(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(context.Background())

	var groupID string
	queryGroup := `
		INSERT INTO groups (name, slug, city, description, owner_user_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	err = tx.QueryRow(context.Background(), queryGroup, req.Name, slug, req.City, req.Description, userID).Scan(&groupID)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "Group name/slug unavailable"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group"})
		return
	}

	// Add owner as member with role 'owner'
	queryMember := `
		INSERT INTO group_members (group_id, user_id, role)
		VALUES ($1, $2, 'owner')
	`
	_, err = tx.Exec(context.Background(), queryMember, groupID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add owner member"})
		return
	}

	if err := tx.Commit(context.Background()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Group created successfully", "group_id": groupID, "slug": slug})
}

func (h *GroupHandler) ListMyGroups(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	query := `
		SELECT g.id, g.name, g.slug, g.city, gm.role
		FROM groups g
		JOIN group_members gm ON g.id = gm.group_id
		WHERE gm.user_id = $1
	`
	rows, err := db.Pool.Query(context.Background(), query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups"})
		return
	}
	defer rows.Close()

	var groups []gin.H
	for rows.Next() {
		var id, name, slug, city, role string
		if err := rows.Scan(&id, &name, &slug, &city, &role); err != nil {
			continue
		}
		groups = append(groups, gin.H{
			"id":   id,
			"name": name,
			"slug": slug,
			"city": city,
			"role": role,
		})
	}

	c.JSON(http.StatusOK, groups)
}
