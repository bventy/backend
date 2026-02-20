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
			vp.created_at, 
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
		var createdAt interface{}

		if err := rows.Scan(&id, &businessName, &ownerID, &createdAt, &city, &category, &profileImageURL); err != nil {
			continue
		}

		vendors = append(vendors, gin.H{
			"id":                        id,
			"business_name":             businessName,
			"user_id":                   ownerID,
			"created_at":                createdAt,
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
	// Only super_admin can access this (Middleware check should be applied in routes)
	// Additional check here just in case, or delegate to middleware.
	// Plan said "Only super_admin can change roles", so we'll rely on route middleware
	// BUT since this might be a generic route, let's enforce double check or assume middleware handles it.
	// The requirement "Include: ... Validate role"

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

// Stats
func (h *AdminHandler) GetStats(c *gin.Context) {
	var totalUsers, totalVendors, pendingVendors, totalEvents int

	// Run queries concurrently or sequentially. Sequential is fine for now.

	// Total Users
	err := db.Pool.QueryRow(context.Background(), "SELECT count(*) FROM users").Scan(&totalUsers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}

	// Total Vendors
	err = db.Pool.QueryRow(context.Background(), "SELECT count(*) FROM vendor_profiles").Scan(&totalVendors)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}

	// Pending Vendors
	err = db.Pool.QueryRow(context.Background(), "SELECT count(*) FROM vendor_profiles WHERE status = 'pending'").Scan(&pendingVendors)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}

	// Total Events
	err = db.Pool.QueryRow(context.Background(), "SELECT count(*) FROM events").Scan(&totalEvents)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_users":     totalUsers,
		"total_vendors":   totalVendors,
		"pending_vendors": pendingVendors,
		"total_events":    totalEvents,
	})
}
