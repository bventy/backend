package handlers

import (
	"context"
	"net/http"

	"github.com/bventy/backend/internal/db"
	"github.com/gin-gonic/gin"
)

// Helper to get vendor ID for the logged in user
func (h *VendorHandler) getVendorID(ctx context.Context, userID string) (string, error) {
	var vendorID string
	err := db.Pool.QueryRow(ctx, "SELECT id FROM vendor_profiles WHERE owner_user_id = $1", userID).Scan(&vendorID)
	return vendorID, err
}

// --- Services ---

func (h *VendorHandler) GetVendorServices(c *gin.Context) {
	userID, _ := c.Get("userID")
	vendorID, err := h.getVendorID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor profile not found"})
		return
	}

	rows, err := db.Pool.Query(c.Request.Context(),
		"SELECT id, name, base_price, price_unit, status, COALESCE(description, '') FROM vendor_services WHERE vendor_id = $1 ORDER BY created_at DESC",
		vendorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch services"})
		return
	}
	defer rows.Close()

	var services []gin.H
	for rows.Next() {
		var id, name, priceUnit, status, description string
		var basePrice float64
		if err := rows.Scan(&id, &name, &basePrice, &priceUnit, &status, &description); err != nil {
			continue
		}
		services = append(services, gin.H{
			"id":          id,
			"name":        name,
			"base_price":  basePrice,
			"price_unit":  priceUnit,
			"status":      status,
			"description": description,
		})
	}

	c.JSON(http.StatusOK, services)
}

func (h *VendorHandler) AddVendorService(c *gin.Context) {
	userID, _ := c.Get("userID")
	vendorID, err := h.getVendorID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor profile not found"})
		return
	}

	var req struct {
		Name        string  `json:"name" binding:"required"`
		BasePrice   float64 `json:"base_price" binding:"required"`
		PriceUnit   string  `json:"price_unit" binding:"required"`
		Status      string  `json:"status"`
		Description string  `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Status == "" {
		req.Status = "active"
	}

	var id string
	err = db.Pool.QueryRow(c.Request.Context(),
		"INSERT INTO vendor_services (vendor_id, name, base_price, price_unit, status, description) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
		vendorID, req.Name, req.BasePrice, req.PriceUnit, req.Status, req.Description).Scan(&id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add service: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Service added successfully"})
}

func (h *VendorHandler) UpdateVendorService(c *gin.Context) {
	userID, _ := c.Get("userID")
	vendorID, err := h.getVendorID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor profile not found"})
		return
	}

	serviceID := c.Param("id")
	var req struct {
		Name        string  `json:"name"`
		BasePrice   float64 `json:"base_price"`
		PriceUnit   string  `json:"price_unit"`
		Status      string  `json:"status"`
		Description string  `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err = db.Pool.Exec(c.Request.Context(),
		`UPDATE vendor_services SET name = COALESCE(NULLIF($1, ''), name), base_price = CASE WHEN $2 > 0 THEN $2 ELSE base_price END, 
		 price_unit = COALESCE(NULLIF($3, ''), price_unit), status = COALESCE(NULLIF($4, ''), status), 
		 description = COALESCE(NULLIF($5, ''), description), updated_at = now() 
		 WHERE id = $6 AND vendor_id = $7`,
		req.Name, req.BasePrice, req.PriceUnit, req.Status, req.Description, serviceID, vendorID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update service"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Service updated successfully"})
}

func (h *VendorHandler) DeleteVendorService(c *gin.Context) {
	userID, _ := c.Get("userID")
	vendorID, err := h.getVendorID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor profile not found"})
		return
	}

	serviceID := c.Param("id")
	_, err = db.Pool.Exec(c.Request.Context(), "DELETE FROM vendor_services WHERE id = $1 AND vendor_id = $2", serviceID, vendorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete service"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Service deleted successfully"})
}

// --- Pricing Rules ---

func (h *VendorHandler) GetVendorPricingRules(c *gin.Context) {
	userID, _ := c.Get("userID")
	vendorID, err := h.getVendorID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor profile not found"})
		return
	}

	var rules struct {
		WeekendPremiumEnabled       bool    `json:"weekend_premium_enabled"`
		WeekendPremiumPercentage    float64 `json:"weekend_premium_percentage"`
		LastMinuteBookingEnabled    bool    `json:"last_minute_booking_enabled"`
		LastMinuteBookingPercentage float64 `json:"last_minute_booking_percentage"`
		LastMinuteDays              int     `json:"last_minute_days"`
	}

	err = db.Pool.QueryRow(c.Request.Context(),
		"SELECT weekend_premium_enabled, weekend_premium_percentage, last_minute_booking_enabled, last_minute_booking_percentage, last_minute_days FROM vendor_pricing_rules WHERE vendor_id = $1",
		vendorID).Scan(&rules.WeekendPremiumEnabled, &rules.WeekendPremiumPercentage, &rules.LastMinuteBookingEnabled, &rules.LastMinuteBookingPercentage, &rules.LastMinuteDays)

	if err != nil {
		// If no rules found, return defaults
		c.JSON(http.StatusOK, gin.H{
			"weekend_premium_enabled":        false,
			"weekend_premium_percentage":     15,
			"last_minute_booking_enabled":    false,
			"last_minute_booking_percentage": 20,
			"last_minute_days":               7,
		})
		return
	}

	c.JSON(http.StatusOK, rules)
}

func (h *VendorHandler) UpdateVendorPricingRules(c *gin.Context) {
	userID, _ := c.Get("userID")
	vendorID, err := h.getVendorID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor profile not found"})
		return
	}

	var req struct {
		WeekendPremiumEnabled       *bool    `json:"weekend_premium_enabled"`
		WeekendPremiumPercentage    *float64 `json:"weekend_premium_percentage"`
		LastMinuteBookingEnabled    *bool    `json:"last_minute_booking_enabled"`
		LastMinuteBookingPercentage *float64 `json:"last_minute_booking_percentage"`
		LastMinuteDays              *int     `json:"last_minute_days"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `
		INSERT INTO vendor_pricing_rules (vendor_id, weekend_premium_enabled, weekend_premium_percentage, last_minute_booking_enabled, last_minute_booking_percentage, last_minute_days)
		VALUES ($1, COALESCE($2, false), COALESCE($3, 15), COALESCE($4, false), COALESCE($5, 20), COALESCE($6, 7))
		ON CONFLICT (vendor_id) DO UPDATE SET
			weekend_premium_enabled = COALESCE($2, vendor_pricing_rules.weekend_premium_enabled),
			weekend_premium_percentage = COALESCE($3, vendor_pricing_rules.weekend_premium_percentage),
			last_minute_booking_enabled = COALESCE($4, vendor_pricing_rules.last_minute_booking_enabled),
			last_minute_booking_percentage = COALESCE($5, vendor_pricing_rules.last_minute_booking_percentage),
			last_minute_days = COALESCE($6, vendor_pricing_rules.last_minute_days),
			updated_at = now()
	`

	_, err = db.Pool.Exec(c.Request.Context(), query, vendorID, req.WeekendPremiumEnabled, req.WeekendPremiumPercentage, req.LastMinuteBookingEnabled, req.LastMinuteBookingPercentage, req.LastMinuteDays)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pricing rules: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pricing rules updated successfully"})
}

// --- Cancellation Policy ---

func (h *VendorHandler) GetVendorCancellationPolicy(c *gin.Context) {
	userID, _ := c.Get("userID")
	vendorID, err := h.getVendorID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor profile not found"})
		return
	}

	var policy struct {
		PolicyType string `json:"policy_type"`
		CustomText string `json:"custom_text"`
	}

	err = db.Pool.QueryRow(c.Request.Context(),
		"SELECT policy_type, COALESCE(custom_text, '') FROM vendor_cancellation_policies WHERE vendor_id = $1",
		vendorID).Scan(&policy.PolicyType, &policy.CustomText)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{"policy_type": "moderate", "custom_text": ""})
		return
	}

	c.JSON(http.StatusOK, policy)
}

func (h *VendorHandler) UpdateVendorCancellationPolicy(c *gin.Context) {
	userID, _ := c.Get("userID")
	vendorID, err := h.getVendorID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor profile not found"})
		return
	}

	var req struct {
		PolicyType string `json:"policy_type" binding:"required"`
		CustomText string `json:"custom_text"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `
		INSERT INTO vendor_cancellation_policies (vendor_id, policy_type, custom_text)
		VALUES ($1, $2, $3)
		ON CONFLICT (vendor_id) DO UPDATE SET
			policy_type = EXCLUDED.policy_type,
			custom_text = EXCLUDED.custom_text,
			updated_at = now()
	`

	_, err = db.Pool.Exec(c.Request.Context(), query, vendorID, req.PolicyType, req.CustomText)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update cancellation policy"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cancellation policy updated successfully"})
}

// --- Service Areas ---

func (h *VendorHandler) GetVendorServiceAreas(c *gin.Context) {
	userID, _ := c.Get("userID")
	vendorID, err := h.getVendorID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor profile not found"})
		return
	}

	rows, err := db.Pool.Query(c.Request.Context(), "SELECT id, area_name FROM vendor_service_areas WHERE vendor_id = $1 ORDER BY created_at ASC", vendorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch service areas"})
		return
	}
	defer rows.Close()

	var areas []gin.H
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}
		areas = append(areas, gin.H{"id": id, "name": name})
	}

	c.JSON(http.StatusOK, areas)
}

func (h *VendorHandler) AddVendorServiceArea(c *gin.Context) {
	userID, _ := c.Get("userID")
	vendorID, err := h.getVendorID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor profile not found"})
		return
	}

	var req struct {
		AreaName string `json:"area_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var id string
	err = db.Pool.QueryRow(c.Request.Context(), "INSERT INTO vendor_service_areas (vendor_id, area_name) VALUES ($1, $2) RETURNING id", vendorID, req.AreaName).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add service area"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Service area added successfully"})
}

func (h *VendorHandler) DeleteVendorServiceArea(c *gin.Context) {
	userID, _ := c.Get("userID")
	vendorID, err := h.getVendorID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor profile not found"})
		return
	}

	areaID := c.Param("id")
	_, err = db.Pool.Exec(c.Request.Context(), "DELETE FROM vendor_service_areas WHERE id = $1 AND vendor_id = $2", areaID, vendorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete service area"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Service area deleted successfully"})
}

// Public access for dynamic profile page
func (h *VendorHandler) GetPublicVendorDetails(c *gin.Context) {
	slug := c.Param("slug")

	// Get vendor ID first
	var vendorID string
	err := db.Pool.QueryRow(c.Request.Context(), "SELECT id FROM vendor_profiles WHERE slug = $1 AND status = 'verified'", slug).Scan(&vendorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor not found"})
		return
	}

	// Fetch Services
	servicesRows, _ := db.Pool.Query(c.Request.Context(), "SELECT name, base_price, price_unit, COALESCE(description, '') FROM vendor_services WHERE vendor_id = $1 AND status = 'active'", vendorID)
	var services []gin.H
	if servicesRows != nil {
		for servicesRows.Next() {
			var name, unit, desc string
			var price float64
			servicesRows.Scan(&name, &price, &unit, &desc)
			services = append(services, gin.H{"name": name, "price": price, "unit": unit, "description": desc})
		}
		servicesRows.Close()
	}

	// Fetch Pricing Rules
	var rules gin.H
	var wpe, lbe bool
	var wpp, lbp float64
	var lbd int
	err = db.Pool.QueryRow(c.Request.Context(), "SELECT weekend_premium_enabled, weekend_premium_percentage, last_minute_booking_enabled, last_minute_booking_percentage, last_minute_days FROM vendor_pricing_rules WHERE vendor_id = $1", vendorID).
		Scan(&wpe, &wpp, &lbe, &lbp, &lbd)
	if err == nil {
		rules = gin.H{
			"weekend_premium_enabled":        wpe,
			"weekend_premium_percentage":     wpp,
			"last_minute_booking_enabled":    lbe,
			"last_minute_booking_percentage": lbp,
			"last_minute_days":               lbd,
		}
	}

	// Fetch Policy
	var policy gin.H
	var ptype, ptext string
	err = db.Pool.QueryRow(c.Request.Context(), "SELECT policy_type, COALESCE(custom_text, '') FROM vendor_cancellation_policies WHERE vendor_id = $1", vendorID).Scan(&ptype, &ptext)
	if err == nil {
		policy = gin.H{"type": ptype, "text": ptext}
	}

	// Fetch Areas
	areasRows, _ := db.Pool.Query(c.Request.Context(), "SELECT area_name FROM vendor_service_areas WHERE vendor_id = $1", vendorID)
	var areas []string
	if areasRows != nil {
		for areasRows.Next() {
			var name string
			areasRows.Scan(&name)
			areas = append(areas, name)
		}
		areasRows.Close()
	}

	c.JSON(http.StatusOK, gin.H{
		"services":            services,
		"pricing_rules":       rules,
		"cancellation_policy": policy,
		"service_areas":       areas,
	})
}
