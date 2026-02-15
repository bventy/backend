package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/once-human/bventy-backend/internal/config"
	"github.com/once-human/bventy-backend/internal/db"
)

type AuthHandler struct {
	Config *config.Config
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{Config: cfg}
}

// AuthMe handles Firebase authentication (Login/Signup in one step)
func (h *AuthHandler) AuthMe(c *gin.Context) {
	// 1. Get Firebase UID from context (set by middleware)
	firebaseUID, exists := c.Get("firebase_uid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	email, _ := c.Get("email") // Optional
	emailStr, _ := email.(string)

	var userID, role, fullName string
	var dbEmail string

	// 2. Check if user exists
	query := `SELECT id, role, full_name, email FROM users WHERE firebase_uid = $1`
	err := db.Pool.QueryRow(context.Background(), query, firebaseUID).Scan(&userID, &role, &fullName, &dbEmail)

	if err == nil {
		// User exists -> Return user
		c.JSON(http.StatusOK, gin.H{
			"message": "User authenticated",
			"user": gin.H{
				"id":           userID,
				"email":        dbEmail,
				"full_name":    fullName,
				"role":         role,
				"firebase_uid": firebaseUID,
			},
		})
		return
	}

	// 3. User does not exist -> Create new user
	// Default full_name if not provided (Firebase token might not have it, or client didn't send it)
	// We use "New User" as fallback
	newFullName := "New User"

	insertQuery := `
		INSERT INTO users (email, firebase_uid, full_name, role)
		VALUES ($1, $2, $3, 'user')
		RETURNING id
	`

	err = db.Pool.QueryRow(context.Background(), insertQuery,
		emailStr,
		firebaseUID,
		newFullName,
	).Scan(&userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully",
		"user": gin.H{
			"id":           userID,
			"email":        emailStr,
			"full_name":    newFullName,
			"role":         "user",
			"firebase_uid": firebaseUID,
		},
	})
}
