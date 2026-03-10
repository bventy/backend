package middleware

import (
	"github.com/gin-gonic/gin"
)

func VersionMiddleware(version string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Bventy-API-Version", version)
		c.Next()
	}
}
