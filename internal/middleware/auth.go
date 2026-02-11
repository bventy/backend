package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/once-human/bventy-backend/internal/auth"
	"github.com/once-human/bventy-backend/internal/config"
	"github.com/once-human/bventy-backend/internal/db"
)

func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := auth.ValidateToken(tokenString, cfg)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// Role Hierarchy: super_admin > admin > staff > user
func getRoleLevel(role string) int {
	switch role {
	case "super_admin":
		return 4
	case "admin":
		return 3
	case "staff":
		return 2
	case "user":
		return 1
	default:
		return 0
	}
}

func RequireRole(minRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		roleStr := userRole.(string)
		if getRoleLevel(roleStr) >= getRoleLevel(minRole) {
			c.Next()
			return
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Insufficient role"})
		c.Abort()
	}
}

func RequirePermission(requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Check database for permission
		query := `
			SELECT 1 FROM user_permissions up
			JOIN permissions p ON up.permission_id = p.id
			WHERE up.user_id = $1 AND p.code = $2
		`
		var existsFlag int
		err := db.Pool.QueryRow(context.Background(), query, userID, requiredPermission).Scan(&existsFlag)

		if err == nil {
			c.Next()
			return
		}

		// Allow super_admin to bypass permission check
		userRole, _ := c.Get("role")
		if userRole == "super_admin" {
			c.Next()
			return
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Missing permission '" + requiredPermission + "'"})
		c.Abort()
	}
}
