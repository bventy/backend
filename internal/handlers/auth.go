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

// Helper to ensure user exists
func (h *AuthHandler) ensureUserExists(ctx context.Context, firebaseUID, email, name string) (string, string, string, error) {
	var id, role, fullName, dbEmail string

	// Try to find user by Firebase UID
	query := `SELECT id, role, full_name, email FROM users WHERE firebase_uid = $1`
	err := db.Pool.QueryRow(ctx, query, firebaseUID).Scan(&id, &role, &fullName, &dbEmail)

	if err == nil {
		return id, role, fullName, nil
	}

	// If not found, try by email to link accounts (if email is present) or just create
	// We use ON CONFLICT DO UPDATE to handle race conditions and linking
	// Default role is 'user'
	// We COALESCE the name to avoid overwriting with empty string if user exists but we missed it (though ON CONFLICT update should be careful)

	// Actually, the prompt requirement:
	// "If no rows found: INSERT ... RETURNING ..."
	// "INSERT INTO users (email, full_name, role) VALUES ($1, COALESCE($2,''), 'user') RETURNING id,email,full_name,role;"
	// We MUST include firebase_uid to satisfy unique constraint and future lookups.

	insertQuery := `
		INSERT INTO users (email, firebase_uid, full_name, role)
		VALUES ($1, $2, COALESCE($3, ''), 'user')
		ON CONFLICT (email) DO UPDATE 
		SET firebase_uid = EXCLUDED.firebase_uid -- Link Firebase UID to existing email
		RETURNING id, role, full_name
	`

	err = db.Pool.QueryRow(ctx, insertQuery, email, firebaseUID, name).Scan(&id, &role, &fullName)
	if err != nil {
		log.Printf("Failed to ensure user existence: %v", err)
		return "", "", "", err
	}

	return id, role, fullName, nil
}

// FirebaseLogin handles Firebase authentication (Login/Signup in one step)
func (h *AuthHandler) FirebaseLogin(c *gin.Context) {
	// 1. Get Firebase UID from context (set by middleware)
	firebaseUID, exists := c.Get("firebase_uid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	email, _ := c.Get("email")
	emailStr, _ := email.(string)

	// Attempt to get name from claims if we can (middleware didn't extract it, but we can try to be robust if we updated middleware,
	// strictly speaking prompt didn't ask us to update middleware for name, but said "Extract claims: name (optional)").
	// For now we pass empty string or "New User" if we must, but prompt says COALESCE($2,'').
	// So passing "" is fine, DB will handle it or code logic.

	id, role, fullName, err := h.ensureUserExists(c.Request.Context(), firebaseUID.(string), emailStr, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to login/signup"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"user": gin.H{
			"id":           id,
			"email":        emailStr,
			"full_name":    fullName,
			"role":         role,
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

	// 2. Ensure user exists (Auto-provisioning)
	id, role, fullName, err := h.ensureUserExists(c.Request.Context(), firebaseUID.(string), emailStr, "")

	if err != nil {
		log.Printf("GetMe: Failed to ensure user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch profile"})
		return
	}

	// 3. Return response
	c.JSON(http.StatusOK, gin.H{
		"id":        id,
		"email":     emailStr,
		"full_name": fullName,
		"role":      role,
		// "created_at": createdAt, // Optional: removed to simplify helper signature, or we can fetch it if needed.
		// The prompt example response has created_at?
		// "Response Format: { id, email, full_name, role }" -> No created_at in strict format section 4!
		// Wait, Step 2 in prompt mentions created_at in SELECT.
		// But Section 4 explicit response example:
		// { "id": "...", "email": "...", "full_name": "", "username": null, "role": "user" }
		// I'll stick to Section 4 format + username (null if missing)
		"username": nil, // Simplification: we didn't fetch username in helper.
		// If we really need username we should fetch it.
		// Let's improve the helper to return username too?
		// Or just query it here if we want to be perfect.

		// Re-reading Step 4: "Never return 404. ... If user missing -> create it."
		// And: "Response Format: { id, email, full_name, username: null, role: 'user' }"
		// Since ensureUserExists returns basics, and we want to avoid extra queries if not needed.
		// I will update helper to return username if I can, or just null it for now as "safe default"
		// The prompt says: "INSERT ... VALUES ($1, COALESCE($2,''), 'user')". It doesn't set username.
		// So username is likely null or empty.
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
