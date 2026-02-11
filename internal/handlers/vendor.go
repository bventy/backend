package handlers

import (
	"context"
	"net/http"
	"strings"

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

func generateSlug(name, city string) string {
	slug := strings.ToLower(name + "-" + city)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "--", "-") // cleanup double dashes
	return slug
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

	slug := generateSlug(req.Name, req.City)

	// Insert into vendor_profiles
	query := `
		INSERT INTO vendor_profiles (user_id, name, slug, category, city, bio, whatsapp_link, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending')
		RETURNING id
	`

	var vendorID string
	err := db.Pool.QueryRow(context.Background(), query, userID, req.Name, slug, req.Category, req.City, req.Bio, req.WhatsappLink).Scan(&vendorID)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "Vendor profile already exists or slug conflict"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to onboard vendor: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Vendor profile created successfully", "vendor_id": vendorID, "slug": slug})
}

// Public Endpoint
func (h *VendorHandler) ListVerifiedVendors(c *gin.Context) {
	query := `
		SELECT name, slug, category, city, bio, whatsapp_link 
		FROM vendor_profiles 
		WHERE status = 'verified'
	`
	rows, err := db.Pool.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch vendors"})
		return
	}
	defer rows.Close()

	var vendors []gin.H
	for rows.Next() {
		var name, slug, category, city, bio, whatsappLink string
		if err := rows.Scan(&name, &slug, &category, &city, &bio, &whatsappLink); err != nil {
			continue
		}
		vendors = append(vendors, gin.H{
			"name":          name,
			"slug":          slug,
			"category":      category,
			"city":          city,
			"bio":           bio,
			"whatsapp_link": whatsappLink,
		})
	}

	c.JSON(http.StatusOK, vendors)
}

// Public Endpoint
func (h *VendorHandler) GetVendorBySlug(c *gin.Context) {
	slug := c.Param("slug")
	query := `
		SELECT name, slug, category, city, bio, whatsapp_link 
		FROM vendor_profiles 
		WHERE slug = $1 AND status = 'verified'
	`

	var name, s, category, city, bio, whatsappLink string
	err := db.Pool.QueryRow(context.Background(), query, slug).Scan(&name, &s, &category, &city, &bio, &whatsappLink)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"name":          name,
		"slug":          s,
		"category":      category,
		"city":          city,
		"bio":           bio,
		"whatsapp_link": whatsappLink,
	})
}
