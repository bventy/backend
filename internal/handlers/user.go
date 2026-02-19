package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bventy/backend/internal/db"
	"github.com/gin-gonic/gin"
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
	var email, role, fullName string
	var username, profileImageURL *string // Use pointer for nullable string

	query := `SELECT email, role, full_name, username, profile_image_url FROM users WHERE id=$1`
	err := db.Pool.QueryRow(context.Background(), query, userID).Scan(&email, &role, &fullName, &username, &profileImageURL)
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
		"id":                    userID, // Added ID to response as it's useful
		"email":                 email,
		"full_name":             fullName,
		"username":              username,        // Returns string or null
		"profile_image_url":     profileImageURL, // Returns string or null
		"role":                  role,
		"vendor_profile_exists": vendorExists,
		"groups":                groups,
	})
}

type UpdateUserRequest struct {
	FullName        string `json:"full_name"`
	Username        string `json:"username"`
	Phone           string `json:"phone"`
	City            string `json:"city"`
	Bio             string `json:"bio"`
	ProfileImageURL string `json:"profile_image_url"`
}

func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Explicit Uniqueness Check (as requested)
	// Only check if username is provided/non-empty, assuming we allow setting it to NULL (empty) without uniqueness check
	// EXCEPT if we treat empty as NULL, we don't need to check uniqueness for empty/NULL (Postgres unique allows multiple nulls).
	// So we only check if req.Username != ""

	if req.Username != "" {
		var count int
		checkQuery := `SELECT count(*) FROM users WHERE username = $1 AND id != $2`
		err := db.Pool.QueryRow(context.Background(), checkQuery, req.Username, userID).Scan(&count)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate username"})
			return
		}
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Username is already taken"})
			return
		}
	}

	// 2. Prepare Update Data
	// Convert empty strings to nil for DB to avoid unique constraint violations or bad data
	var usernameArg interface{} = req.Username
	if req.Username == "" {
		usernameArg = nil
	}

	var phoneArg interface{} = req.Phone
	if req.Phone == "" {
		phoneArg = nil
	}

	var cityArg interface{} = req.City
	if req.City == "" {
		cityArg = nil
	}

	var bioArg interface{} = req.Bio
	if req.Bio == "" {
		bioArg = nil
	}

	var imageArg interface{} = req.ProfileImageURL
	if req.ProfileImageURL == "" {
		imageArg = nil
	}

	query := `
		UPDATE users 
		SET full_name = $2, username = $3, phone = $4, city = $5, bio = $6, profile_image_url = $7
		WHERE id = $1
		RETURNING id, email, full_name, username, role
	`

	var id, email, fullName, role string
	var username *string // Scan into pointer for potential NULL

	err := db.Pool.QueryRow(context.Background(), query,
		userID,
		req.FullName,
		usernameArg,
		phoneArg,
		cityArg,
		bioArg,
		imageArg,
	).Scan(&id, &email, &fullName, &username, &role)

	if err != nil {
		// Log the actual error for debugging
		fmt.Printf("❌ Profile Update Error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        id,
		"email":     email,
		"full_name": fullName,
		"username":  username,
		"role":      role,
		"message":   "Profile updated successfully",
	})
}
