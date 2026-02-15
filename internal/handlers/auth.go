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

// FirebaseLogin handles Firebase authentication (Login/Signup in one step)
func (h *AuthHandler) FirebaseLogin(c *gin.Context) {
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
			"status": "ok",
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
		"status": "ok",
		"user": gin.H{
			"id":           userID,
			"email":        emailStr,
			"full_name":    newFullName,
			"role":         "user",
			"firebase_uid": firebaseUID,
		},
	})
}

// GetMe fetches the current user's profile
func (h *AuthHandler) GetMe(c *gin.Context) {
	// 1. Get Firebase UID from context (set by middleware)
	firebaseUID, exists := c.Get("firebase_uid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 2. Query user details
	var id, email, fullName, role, createdAt, dbFirebaseUID string
	// Handle potential NULLs if necessary, but here we expect basic fields to be present
	query := `SELECT id, email, full_name, role, created_at, firebase_uid FROM users WHERE firebase_uid = $1`
	err := db.Pool.QueryRow(context.Background(), query, firebaseUID).Scan(&id, &email, &fullName, &role, &createdAt, &dbFirebaseUID)

	if err != nil {
		// If user not found, we could create it or return error.
		// The prompt says: "If user does not exist: Create user automatically (email only)"
		// But in FirebaseLogin we already create it.
		// Let's reuse the logic or just return 404 if we want strictness,
		// BUT prompt says: "If user does not exist: - Create user automatically"

		// Fallback creation
		emailFromToken, _ := c.Get("email")
		emailStr, _ := emailFromToken.(string)
		newFullName := "New User"

		insertQuery := `
			INSERT INTO users (email, firebase_uid, full_name, role)
			VALUES ($1, $2, $3, 'user')
			RETURNING id, created_at
		`
		err = db.Pool.QueryRow(context.Background(), insertQuery, emailStr, firebaseUID, newFullName).Scan(&id, &createdAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}

		email = emailStr
		fullName = newFullName
		role = "user"
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         id,
		"email":      email,
		"full_name":  fullName,
		"role":       role,
		"created_at": createdAt,
		// "username": "", // Optional, currently not strictly enforced in prompt JSON but in prompt description
	})
}

type CompleteProfileRequest struct {
	FullName string `json:"full_name" binding:"required"`
	Username string `json:"username" binding:"required"`
}

// CompleteProfile updates the user's profile (name, username)
func (h *AuthHandler) CompleteProfile(c *gin.Context) {
	// 1. Get Firebase UID from context (set by middleware)
	firebaseUID, exists := c.Get("firebase_uid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CompleteProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 2. Update user details
	var id, email, role, createdAt string

	query := `
		UPDATE users 
		SET full_name = $1, username = $2 
		WHERE firebase_uid = $3 
		RETURNING id, email, role, created_at
	`

	err := db.Pool.QueryRow(context.Background(), query, req.FullName, req.Username, firebaseUID).Scan(&id, &email, &role, &createdAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	// 3. Return updated profile
	c.JSON(http.StatusOK, gin.H{
		"id":         id,
		"email":      email,
		"full_name":  req.FullName,
		"username":   req.Username,
		"role":       role,
		"created_at": createdAt,
	})
}
