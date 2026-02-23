package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/bventy/backend/internal/db"
	"github.com/gin-gonic/gin"
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
	tx, err := db.Pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(c.Request.Context())

	var groupID string
	queryGroup := `
		INSERT INTO groups (name, slug, city, description, owner_user_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	err = tx.QueryRow(c.Request.Context(), queryGroup, req.Name, slug, req.City, req.Description, userID).Scan(&groupID)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "Group name/slug unavailable"})
			return
		}
		fmt.Printf("ERROR: Failed to create group by user %s: %v\n", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group"})
		return
	}

	// Add owner as member with role 'owner'
	queryMember := `
		INSERT INTO group_members (group_id, user_id, role)
		VALUES ($1, $2, 'owner')
	`
	_, err = tx.Exec(c.Request.Context(), queryMember, groupID, userID)
	if err != nil {
		fmt.Printf("ERROR: Failed to add owner member to group %s: %v\n", groupID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add owner member"})
		return
	}

	if err := tx.Commit(c.Request.Context()); err != nil {
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
	rows, err := db.Pool.Query(c.Request.Context(), query, userID)
	if err != nil {
		fmt.Printf("ERROR: Failed to list groups for user %s: %v\n", userID, err)
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
