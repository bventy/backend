package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/once-human/bventy-backend/internal/db"
)

type AdminHandler struct{}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

func (h *AdminHandler) GetPendingVendors(c *gin.Context) {
	query := `
		SELECT id, name, slug, category, city, status, whatsapp_link 
		FROM vendor_profiles 
		WHERE status = 'pending'
	`
	rows, err := db.Pool.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch vendors"})
		return
	}
	defer rows.Close()

	var vendors []gin.H
	for rows.Next() {
		var id, name, slug, category, city, status, whatsappLink string
		if err := rows.Scan(&id, &name, &slug, &category, &city, &status, &whatsappLink); err != nil {
			continue
		}
		vendors = append(vendors, gin.H{
			"id":            id,
			"name":          name,
			"slug":          slug,
			"category":      category,
			"city":          city,
			"status":        status,
			"whatsapp_link": whatsappLink,
		})
	}

	c.JSON(http.StatusOK, vendors)
}

func (h *AdminHandler) VerifyVendor(c *gin.Context) {
	vendorID := c.Param("id")
	query := `UPDATE vendor_profiles SET status = 'verified' WHERE id = $1 RETURNING id`
	var id string
	err := db.Pool.QueryRow(context.Background(), query, vendorID).Scan(&id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor not found or already processed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Vendor verified successfully"})
}

func (h *AdminHandler) RejectVendor(c *gin.Context) {
	vendorID := c.Param("id")
	query := `UPDATE vendor_profiles SET status = 'rejected' WHERE id = $1 RETURNING id`
	var id string
	err := db.Pool.QueryRow(context.Background(), query, vendorID).Scan(&id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor not found or already processed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Vendor rejected successfully"})
}
