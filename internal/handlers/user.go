package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/once-human/bventy-backend/internal/db"
)

type UserHandler struct{}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

func (h *UserHandler) PromoteToAdmin(c *gin.Context) {
	targetUserID := c.Param("id")
	// Logic remains same
	var currentRole string
	err := db.Pool.QueryRow(context.Background(), "SELECT role FROM users WHERE id=$1", targetUserID).Scan(&currentRole)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if currentRole == "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot change role of super_admin"})
		return
	}
	_, err = db.Pool.Exec(context.Background(), "UPDATE users SET role='admin' WHERE id=$1", targetUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to promote user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User promoted to admin"})
}

func (h *UserHandler) PromoteToStaff(c *gin.Context) {
	targetUserID := c.Param("id")
	// Logic remains same
	var currentRole string
	err := db.Pool.QueryRow(context.Background(), "SELECT role FROM users WHERE id=$1", targetUserID).Scan(&currentRole)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if currentRole == "admin" || currentRole == "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot demote/change admin users via this endpoint"})
		return
	}
	_, err = db.Pool.Exec(context.Background(), "UPDATE users SET role='staff' WHERE id=$1", targetUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to promote user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User promoted to staff"})
}

func (h *UserHandler) GetMe(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Fetch user details
	var email, role, fullName, username string
	// Check if username is NULL in DB (it is optional), need to handle potential NULL in scan.
	// Actually `username` is text nullable. `Scan` to *string works if value is not null. 
	// If it is null, we need NullString or *string.
	// Let's use simple logic: COALESCE(username, '')
	query := `SELECT email, role, full_name, COALESCE(username, '') FROM users WHERE id=$1`
	err := db.Pool.QueryRow(context.Background(), query, userID).Scan(&email, &role, &fullName, &username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check profiles
	var vendorExists bool
	var dummy int
	err = db.Pool.QueryRow(context.Background(), "SELECT 1 FROM vendor_profiles WHERE owner_user_id=$1", userID).Scan(&dummy)
	vendorExists = err == nil

	// Fetch groups
	var groups []gin.H
	rows, err := db.Pool.Query(context.Background(), `
		SELECT g.id, g.name, g.slug, gm.role 
		FROM groups g
		JOIN group_members gm ON g.id = gm.group_id
		WHERE gm.user_id = $1
	`, userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var gid, gname, gslug, grole string
			if err := rows.Scan(&gid, &gname, &gslug, &grole); err == nil {
				groups = append(groups, gin.H{"id": gid, "name": gname, "slug": gslug, "role": grole})
			}
		}
	} else {
		groups = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"email":                 email,
		"full_name":             fullName,
		"username":              username,
		"role":                  role,
		"vendor_profile_exists": vendorExists,
		"groups":                groups,
	})
}
