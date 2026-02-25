package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func DebugCookies(c *gin.Context) {
	cookies := c.Request.Cookies()
	cookieMap := make(map[string]string)
	for _, cookie := range cookies {
		cookieMap[cookie.Name] = cookie.Value
	}

	headers := make(map[string]string)
	headers["Origin"] = c.GetHeader("Origin")
	headers["Cookie"] = c.GetHeader("Cookie")

	c.JSON(http.StatusOK, gin.H{
		"cookies": cookieMap,
		"headers": headers,
		"message": "Auth debug endpoint",
	})
}
