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

	// Ensure target is not already super_admin
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

	// Prevent demoting admins/super_admins via this endpoint if desired, but spec says "Promote user to staff"
	// Let's safe-guard: admins cannot modify other admins/super_admins usually, but request says "Only role=admin OR super_admin"
	// "Admins cannot promote admins." <- This is for PromoteToAdmin.
	// We'll allow promoting 'user' to 'staff'.

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
	var email, role string
	err := db.Pool.QueryRow(context.Background(), "SELECT email, role FROM users WHERE id=$1", userID).Scan(&email, &role)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check profiles
	var vendorExists, organizerExists bool
	var dummy int
	err = db.Pool.QueryRow(context.Background(), "SELECT 1 FROM vendor_profiles WHERE user_id=$1", userID).Scan(&dummy)
	vendorExists = err == nil

	err = db.Pool.QueryRow(context.Background(), "SELECT 1 FROM organizer_profiles WHERE user_id=$1", userID).Scan(&dummy)
	organizerExists = err == nil

	// Fetch permissions
	rows, err := db.Pool.Query(context.Background(), `
		SELECT p.code FROM user_permissions up
		JOIN permissions p ON up.permission_id = p.id
		WHERE up.user_id = $1
	`, userID)
	
	var permissions []string
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var code string
			if err := rows.Scan(&code); err == nil {
				permissions = append(permissions, code)
			}
		}
	} else {
		permissions = []string{}
	}

	c.JSON(http.StatusOK, gin.H{
		"email":                    email,
		"role":                     role,
		"vendor_profile_exists":    vendorExists,
		"organizer_profile_exists": organizerExists,
		"permissions":              permissions,
	})
}
