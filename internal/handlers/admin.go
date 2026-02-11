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

type Vendor struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	City         string `json:"city"`
	Status       string `json:"status"`
	WhatsappLink string `json:"whatsapp_link"`
}

func (h *AdminHandler) GetPendingVendors(c *gin.Context) {
	query := `SELECT id, name, category, city, status, whatsapp_link FROM vendors WHERE status = 'pending'`
	rows, err := db.Pool.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch vendors"})
		return
	}
	defer rows.Close()

	var vendors []Vendor
	for rows.Next() {
		var v Vendor
		if err := rows.Scan(&v.ID, &v.Name, &v.Category, &v.City, &v.Status, &v.WhatsappLink); err != nil {
			continue
		}
		vendors = append(vendors, v)
	}

	c.JSON(http.StatusOK, vendors)
}

func (h *AdminHandler) VerifyVendor(c *gin.Context) {
	vendorID := c.Param("id")
	query := `UPDATE vendors SET status = 'verified' WHERE id = $1 RETURNING id`
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
	query := `UPDATE vendors SET status = 'rejected' WHERE id = $1 RETURNING id`
	var id string
	err := db.Pool.QueryRow(context.Background(), query, vendorID).Scan(&id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor not found or already processed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Vendor rejected successfully"})
}
