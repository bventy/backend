package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/bventy/backend/internal/config"
	"github.com/bventy/backend/internal/db"
	"github.com/bventy/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type VendorHandler struct {
	Config       *config.Config
	MediaService *services.MediaService
}

func NewVendorHandler(cfg *config.Config) *VendorHandler {
	svc, _ := services.NewMediaService(cfg)
	return &VendorHandler{
		Config:       cfg,
		MediaService: svc,
	}
}

type OnboardVendorRequest struct {
	BusinessName string `json:"business_name" binding:"required"`
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

	slug := generateSlug(req.BusinessName, req.City)

	// Insert into vendor_profiles
	query := `
		INSERT INTO vendor_profiles (owner_user_id, business_name, slug, category, city, bio, whatsapp_link, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending')
		RETURNING id
	`

	var vendorID string
	err := db.Pool.QueryRow(context.Background(), query, userID, req.BusinessName, slug, req.Category, req.City, req.Bio, req.WhatsappLink).Scan(&vendorID)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "Vendor profile already exists for this user or slug conflict"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to onboard vendor: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Vendor profile created successfully", "vendor_id": vendorID, "slug": slug})
}

func (h *VendorHandler) GetMyProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Use COALESCE for nullable text fields to avoid Scan errors
	// Use 'status' column instead of non-existent 'verified' column
	query := `
		SELECT business_name, slug, category, city, COALESCE(bio, ''), whatsapp_link, portfolio_image_url, gallery_images, portfolio_files, status
		FROM vendor_profiles 
		WHERE owner_user_id = $1
	`

	var name, slug, category, city, bio, whatsappLink, status string
	var portfolioImageURL *string
	var galleryImages []string
	var portfolioFiles []interface{}

	err := db.Pool.QueryRow(context.Background(), query, userID).Scan(
		&name, &slug, &category, &city, &bio, &whatsappLink,
		&portfolioImageURL, &galleryImages, &portfolioFiles, &status,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor profile not found"})
		return
	}

	// Map status to verified boolean
	verified := (status == "verified")

	c.JSON(http.StatusOK, gin.H{
		"business_name":       name,
		"slug":                slug,
		"category":            category,
		"city":                city,
		"bio":                 bio,
		"whatsapp_link":       whatsappLink,
		"portfolio_image_url": portfolioImageURL,
		"gallery_images":      galleryImages,
		"portfolio_files":     portfolioFiles,
		"verified":            verified,
	})
}

func (h *VendorHandler) ListVerifiedVendors(c *gin.Context) {
	query := `
		SELECT business_name, slug, category, city, bio, whatsapp_link, portfolio_image_url 
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
		var portfolioImageURL *string
		if err := rows.Scan(&name, &slug, &category, &city, &bio, &whatsappLink, &portfolioImageURL); err != nil {
			continue
		}
		vendors = append(vendors, gin.H{
			"business_name":       name,
			"slug":                slug,
			"category":            category,
			"city":                city,
			"bio":                 bio,
			"whatsapp_link":       whatsappLink,
			"portfolio_image_url": portfolioImageURL,
		})
	}

	c.JSON(http.StatusOK, vendors)
}

func (h *VendorHandler) GetVendorBySlug(c *gin.Context) {
	slug := c.Param("slug")
	query := `
		SELECT id, business_name, slug, category, city, bio, whatsapp_link, portfolio_image_url, gallery_images, portfolio_files
		FROM vendor_profiles 
		WHERE slug = $1 AND status = 'verified'
	`

	var id, name, s, category, city, bio, whatsappLink string
	var portfolioImageURL *string
	var galleryImages []string
	var portfolioFiles []interface{} // Changed to []interface{} to handle JSONB properly

	// We need to handle potential NULLs for array/jsonb if they weren't set with defaults correctly in old rows
	// But our alteration set defaults.
	err := db.Pool.QueryRow(context.Background(), query, slug).Scan(
		&id, &name, &s, &category, &city, &bio, &whatsappLink,
		&portfolioImageURL, &galleryImages, &portfolioFiles,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                  id,
		"business_name":       name,
		"slug":                s,
		"category":            category,
		"city":                city,
		"bio":                 bio,
		"whatsapp_link":       whatsappLink,
		"portfolio_image_url": portfolioImageURL,
		"gallery_images":      galleryImages,
		"portfolio_files":     portfolioFiles,
	})
}

type UpdateVendorRequest struct {
	BusinessName      string        `json:"business_name"`
	Category          string        `json:"category"`
	City              string        `json:"city"`
	Bio               string        `json:"bio"`
	WhatsappLink      string        `json:"whatsapp_link"`
	PortfolioImageURL string        `json:"portfolio_image_url"`
	GalleryImages     []string      `json:"gallery_images"`
	PortfolioFiles    []interface{} `json:"portfolio_files"` // Array of objects {name, url}
}

func (h *VendorHandler) UpdateVendor(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req UpdateVendorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Calculate new slug if business name changes?
	// For simplicity, let's keep slug persistent or only update if explicitly needed.
	// The requirement doesn't specify slug updates, so we'll skip slug updates to avoid breaking links.

	// Handling PortfolioFiles JSONB
	// We might need to marshal it to string or byte[] if the driver requires it,
	// but pgx usually handles []interface{} -> jsonb automatically if we pass it right.
	// However, it's safer to just pass it directly if the driver supports it.

	// Handle Updates
	// We need to verify ownership first or just update where owner_user_id = userID
	query := `
		UPDATE vendor_profiles 
		SET business_name = COALESCE(NULLIF($2, ''), business_name),
		    category = COALESCE(NULLIF($3, ''), category),
		    city = COALESCE(NULLIF($4, ''), city),
		    bio = COALESCE(NULLIF($5, ''), bio),
		    whatsapp_link = COALESCE(NULLIF($6, ''), whatsapp_link),
		    portfolio_image_url = $7,
		    gallery_images = $8,
		    portfolio_files = $9
		WHERE owner_user_id = $1
		RETURNING id
	`

	// Note: For arrays and jsonb, if they are empty/nil in request, we might want to keep existing?
	// But usually a save replaces the list. So we will overwrite.
	// COALESCE logic above is for strings. For arrays, we probably want to allow clearing them (empty list).
	// So we pass them directly.

	var id string
	err := db.Pool.QueryRow(context.Background(), query,
		userID,
		req.BusinessName,
		req.Category,
		req.City,
		req.Bio,
		req.WhatsappLink,
		req.PortfolioImageURL,
		req.GalleryImages,
		req.PortfolioFiles,
	).Scan(&id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update vendor profile: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Vendor profile updated successfully"})
}

// UploadGalleryImage adds an image to the vendor's gallery
func (h *VendorHandler) UploadGalleryImage(c *gin.Context) {
	vendorID := c.Param("id")
	userID := c.MustGet("userID").(string)

	// Validate ownership
	var ownerID string
	err := db.Pool.QueryRow(context.TODO(), "SELECT owner_user_id FROM vendor_profiles WHERE id=$1", vendorID).Scan(&ownerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor not found"})
		return
	}
	if ownerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not own this vendor profile"})
		return
	}

	// Check limit (25)
	var count int
	err = db.Pool.QueryRow(context.TODO(), "SELECT COUNT(*) FROM vendor_gallery_images WHERE vendor_id=$1", vendorID).Scan(&count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if count >= 25 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Gallery limit reached (max 25 images)"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}

	// Size limit 5MB
	if fileHeader.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large (max 5MB)"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer file.Close()

	// Upload
	prefix := fmt.Sprintf("vendors/%s/gallery", vendorID)
	url, err := h.MediaService.CompressAndUploadImage(file, fileHeader.Filename, prefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image"})
		return
	}

	// Insert into DB
	_, err = db.Pool.Exec(context.TODO(),
		"INSERT INTO vendor_gallery_images (vendor_id, image_url, sort_order) VALUES ($1, $2, $3)",
		vendorID, url, count+1) // Simple sort order
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image metadata"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Image uploaded", "url": url})
}

// DeleteGalleryImage removes an image from the gallery
func (h *VendorHandler) DeleteGalleryImage(c *gin.Context) {
	vendorID := c.Param("id")
	imageID := c.Param("imageID")
	userID := c.MustGet("userID").(string)

	// Validate ownership
	var ownerID string
	err := db.Pool.QueryRow(context.TODO(), "SELECT owner_user_id FROM vendor_profiles WHERE id=$1", vendorID).Scan(&ownerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor not found"})
		return
	}
	if ownerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		return
	}

	// Get URL to delete from R2
	var url string
	err = db.Pool.QueryRow(context.TODO(), "SELECT image_url FROM vendor_gallery_images WHERE id=$1 AND vendor_id=$2", imageID, vendorID).Scan(&url)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	// Delete from R2
	_ = h.MediaService.DeleteFile(url)

	// Delete from DB
	_, err = db.Pool.Exec(context.TODO(), "DELETE FROM vendor_gallery_images WHERE id=$1", imageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete image record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Image deleted"})
}

// UploadPortfolioFile adds a PDF to the vendor's portfolio
func (h *VendorHandler) UploadPortfolioFile(c *gin.Context) {
	vendorID := c.Param("id")
	userID := c.MustGet("userID").(string)

	// Validate ownership
	var ownerID string
	err := db.Pool.QueryRow(context.TODO(), "SELECT owner_user_id FROM vendor_profiles WHERE id=$1", vendorID).Scan(&ownerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor not found"})
		return
	}
	if ownerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		return
	}

	// Check limit (20)
	var count int
	err = db.Pool.QueryRow(context.TODO(), "SELECT COUNT(*) FROM vendor_portfolio_files WHERE vendor_id=$1", vendorID).Scan(&count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if count >= 20 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Portfolio limit reached (max 20 files)"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}

	// Validate PDF
	if fileHeader.Header.Get("Content-Type") != "application/pdf" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only PDF files allowed"})
		return
	}

	// Size limit 5MB
	if fileHeader.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large (max 5MB)"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer file.Close()

	// Upload (Raw file, no compression for PDF)
	prefix := fmt.Sprintf("vendors/%s/portfolio", vendorID)
	url, err := h.MediaService.UploadFile(file, fileHeader.Filename, "application/pdf", prefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
		return
	}

	title := c.PostForm("title")
	if title == "" {
		title = fileHeader.Filename
	}

	// Insert into DB
	_, err = db.Pool.Exec(context.TODO(),
		"INSERT INTO vendor_portfolio_files (vendor_id, file_url, title, sort_order) VALUES ($1, $2, $3, $4)",
		vendorID, url, title, count+1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file metadata"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File uploaded", "url": url})
}

// DeletePortfolioFile removes a file from the portfolio
func (h *VendorHandler) DeletePortfolioFile(c *gin.Context) {
	vendorID := c.Param("id")
	fileID := c.Param("fileID")
	userID := c.MustGet("userID").(string)

	// Validate ownership
	var ownerID string
	err := db.Pool.QueryRow(context.TODO(), "SELECT owner_user_id FROM vendor_profiles WHERE id=$1", vendorID).Scan(&ownerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor not found"})
		return
	}
	if ownerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		return
	}

	// Get URL to delete from R2
	var url string
	err = db.Pool.QueryRow(context.TODO(), "SELECT file_url FROM vendor_portfolio_files WHERE id=$1 AND vendor_id=$2", fileID, vendorID).Scan(&url)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Delete from R2
	_ = h.MediaService.DeleteFile(url)

	// Delete from DB
	_, err = db.Pool.Exec(context.TODO(), "DELETE FROM vendor_portfolio_files WHERE id=$1", fileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File deleted"})
}
