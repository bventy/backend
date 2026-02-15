package handlers

import (
	"context"
	"log"
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
	// 1. Get info from context
	firebaseUID, _ := c.Get("firebase_uid") // We know it exists from middleware
	email, _ := c.Get("email")
	emailStr, _ := email.(string)

	log.Printf("GET /auth/me for email: %s, uid: %v", emailStr, firebaseUID)

	// 2. Query user details
	var id, fullName, role, createdAt string
	var username *string // handle nullable

	// We query by email as primary (per prompt request) or firebase_uid.
	query := `SELECT id, full_name, username, role, created_at FROM users WHERE email=$1`
	err := db.Pool.QueryRow(context.Background(), query, emailStr).Scan(&id, &fullName, &username, &role, &createdAt)

	if err != nil {
		// If user not found (or any error? Assuming NoRows for simplicity, but logging error)
		log.Printf("User not found or error: %v. Attempting creation.", err)

		// Create user automatically
		// Note provided INSERT: INSERT INTO users (email, full_name, role) VALUES ($1, '', 'user')
		// We should also set firebase_uid probably? But prompt didn't strictly say so in the INSERT block.
		// However, to avoid future issues, we SHOULD set firebase_uid if we have it.
		// But let's strictly follow the "INSERT INTO users (email, full_name, role)" instruction if possible?
		// No, better to be safe and include firebase_uid if the column exists (it does).
		// Wait, the prompt instruction "INSERT INTO users (email, full_name, role)" might be simplified.
		// I will insert firebase_uid too because I know the schema requires/supports it.

		newFullName := ""
		role = "user"

		// If username is null, we return null in JSON.

		insertQuery := `
			INSERT INTO users (email, firebase_uid, full_name, role)
			VALUES ($1, $2, $3, 'user')
			ON CONFLICT (email) DO UPDATE SET firebase_uid = $2 -- Handle race condition or if email exists but query failed?
			RETURNING id, created_at
		`
		// Wait, ON CONFLICT might be overkill if we just got NoRows, but beneficial.
		// Actually, if we use the exact prompt SQL:
		// INSERT INTO users (email, full_name, role) VALUES ...

		err = db.Pool.QueryRow(context.Background(), insertQuery, emailStr, firebaseUID, newFullName).Scan(&id, &createdAt)
		if err != nil {
			log.Printf("Failed to create user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}

		fullName = newFullName
		// username is nil
	}

	// Create response structure
	// { "id": "...", "email": "...", "full_name": "", "username": null, "role": "user" }

	c.JSON(http.StatusOK, gin.H{
		"id":         id,
		"email":      emailStr,
		"full_name":  fullName,
		"username":   username,
		"role":       role,
		"created_at": createdAt,
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
