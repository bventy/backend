package handlers

import (
	"context"
	"net/http"

	"github.com/bventy/backend/internal/db"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct{}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

// Vendor Moderation
func (h *AdminHandler) GetVendors(c *gin.Context) {
	status := c.Query("status")
	query := `
		SELECT 
			vp.id, 
			vp.business_name, 
			vp.owner_user_id, 
			vp.city,
			vp.category,
			u.profile_image_url
		FROM vendor_profiles vp
		JOIN users u ON vp.owner_user_id = u.id
	`

	args := []interface{}{}
	if status != "" {
		query += " WHERE vp.status = $1"
		args = append(args, status)
	}

	rows, err := db.Pool.Query(context.Background(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch vendors"})
		return
	}
	defer rows.Close()

	var vendors []gin.H
	for rows.Next() {
		var id, businessName, ownerID, city, category string
		var profileImageURL *string

		if err := rows.Scan(&id, &businessName, &ownerID, &city, &category, &profileImageURL); err != nil {
			continue
		}

		vendors = append(vendors, gin.H{
			"id":                        id,
			"business_name":             businessName,
			"user_id":                   ownerID,
			"city":                      city,
			"category":                  category,
			"primary_profile_image_url": profileImageURL,
		})
	}

	// Return empty list instead of null
	if vendors == nil {
		vendors = []gin.H{}
	}

	c.JSON(http.StatusOK, vendors)
}

func (h *AdminHandler) VerifyVendor(c *gin.Context) { // Mapped to Approve
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

// User Management
func (h *AdminHandler) GetUsers(c *gin.Context) {
	query := `SELECT id, email, full_name, role, created_at FROM users`
	rows, err := db.Pool.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	defer rows.Close()

	var users []gin.H
	for rows.Next() {
		var id, email, fullName, role string
		var createdAt interface{}
		if err := rows.Scan(&id, &email, &fullName, &role, &createdAt); err != nil {
			continue
		}
		users = append(users, gin.H{
			"id":         id,
			"email":      email,
			"full_name":  fullName,
			"role":       role,
			"created_at": createdAt,
		})
	}

	if users == nil {
		users = []gin.H{}
	}

	c.JSON(http.StatusOK, users)
}

func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	userID := c.Param("id")
	var input struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	validRoles := map[string]bool{"user": true, "staff": true, "admin": true, "super_admin": true}
	if !validRoles[input.Role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
		return
	}

	query := `UPDATE users SET role = $1 WHERE id = $2 RETURNING id`
	var id string
	err := db.Pool.QueryRow(context.Background(), query, input.Role, userID).Scan(&id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User role updated successfully"})
}

// Email & Template Management
func (h *AdminHandler) GetEmailTemplates(c *gin.Context) {
	query := `SELECT template_key, subject, body_html, is_enabled, from_name, from_email FROM email_templates ORDER BY template_key ASC`
	rows, err := db.Pool.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch email templates"})
		return
	}
	defer rows.Close()

	var templates []gin.H
	for rows.Next() {
		var key, subject, body string
		var isEnabled bool
		var fromName, fromEmail *string
		if err := rows.Scan(&key, &subject, &body, &isEnabled, &fromName, &fromEmail); err != nil {
			continue
		}
		templates = append(templates, gin.H{
			"template_key": key,
			"subject":      subject,
			"body_html":    body,
			"is_enabled":   isEnabled,
			"from_name":    fromName,
			"from_email":   fromEmail,
		})
	}
	c.JSON(http.StatusOK, templates)
}

func (h *AdminHandler) UpdateEmailTemplate(c *gin.Context) {
	key := c.Param("key")
	var input struct {
		Subject   string `json:"subject"`
		BodyHTML  string `json:"body_html"`
		IsEnabled *bool  `json:"is_enabled"`
		FromName  string `json:"from_name"`
		FromEmail string `json:"from_email"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	query := `
		UPDATE email_templates 
		SET subject = COALESCE(NULLIF($2, ''), subject),
		    body_html = COALESCE(NULLIF($3, ''), body_html),
		    is_enabled = COALESCE($4, is_enabled),
		    from_name = COALESCE(NULLIF($5, ''), from_name),
		    from_email = COALESCE(NULLIF($6, ''), from_email),
		    updated_at = NOW()
		WHERE template_key = $1
		RETURNING template_key
	`
	var updatedKey string
	err := db.Pool.QueryRow(context.Background(), query, key, input.Subject, input.BodyHTML, input.IsEnabled, input.FromName, input.FromEmail).Scan(&updatedKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Template updated successfully"})
}

func (h *AdminHandler) GetPlatformSettings(c *gin.Context) {
	rows, err := db.Pool.Query(context.Background(), "SELECT key, value FROM platform_settings")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch settings"})
		return
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, val string
		_ = rows.Scan(&key, &val)
		settings[key] = val
	}
	c.JSON(http.StatusOK, settings)
}

func (h *AdminHandler) UpdatePlatformSetting(c *gin.Context) {
	var input struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	query := `INSERT INTO platform_settings (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()`
	_, err := db.Pool.Exec(context.Background(), query, input.Key, input.Value)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update setting"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Setting updated successfully"})
}

// Stats (Legacy mapping for dashboard stats)
func (h *AdminHandler) GetStats(c *gin.Context) {
	// Re-route or reuse the overview logic
	metricsHandler := NewAdminMetricsHandler()
	metricsHandler.GetAdminMetricsOverview(c)
}
