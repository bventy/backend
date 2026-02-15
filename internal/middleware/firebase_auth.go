package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/once-human/bventy-backend/internal/db"
	app "github.com/once-human/bventy-backend/internal/firebase"
)

// FirebaseAuthMiddleware verifies the Firebase ID token in the Authorization header.
func FirebaseAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
			return
		}

		idToken := parts[1]

		client, err := app.App.Auth(context.Background())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error initializing Firebase Auth client"})
			return
		}

		token, err := client.VerifyIDToken(context.Background(), idToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// Store Firebase UID in context for handlers to use
		c.Set("firebase_uid", token.UID)
		c.Set("email", token.Claims["email"])

		// Fetch user details from DB to support RBAC
		// We ignore error here because for /auth/me (signup), the user won't exist yet.
		// Handlers that require a user (RequireRole) will fail if userID/role are not set.
		var userID, role string
		query := `SELECT id, role FROM users WHERE firebase_uid = $1`
		err = db.Pool.QueryRow(context.Background(), query, token.UID).Scan(&userID, &role)

		if err == nil {
			c.Set("userID", userID)
			c.Set("role", role)
		}

		c.Next()
	}
}
