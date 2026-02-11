package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/once-human/bventy-backend/internal/db"
)

type VendorHandler struct{}

func NewVendorHandler() *VendorHandler {
	return &VendorHandler{}
}

type OnboardVendorRequest struct {
	Name         string `json:"name" binding:"required"`
	Category     string `json:"category" binding:"required"`
	City         string `json:"city" binding:"required"`
	Bio          string `json:"bio"`
	WhatsappLink string `json:"whatsapp_link" binding:"required"`
}

func (h *VendorHandler) OnboardVendor(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req OnboardVendorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `
		INSERT INTO vendors (user_id, name, category, city, bio, whatsapp_link, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending')
		RETURNING id
	`

	var vendorID string
	err := db.Pool.QueryRow(context.Background(), query, userID, req.Name, req.Category, req.City, req.Bio, req.WhatsappLink).Scan(&vendorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to onboard vendor: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Vendor profile created successfully", "vendor_id": vendorID})
}
