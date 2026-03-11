package handlers

import (
	"net/http"

	"github.com/bventy/backend/internal/services"
	"github.com/gin-gonic/gin"
)

func GetSystemStatus(c *gin.Context) {
	service := services.GetSystemStatusService()
	status := service.GetStatus()

	c.JSON(http.StatusOK, gin.H{
		"monitors": status,
	})
}
