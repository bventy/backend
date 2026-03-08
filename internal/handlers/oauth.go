package handlers

import (
	"fmt"
	"net/http"

	"github.com/bventy/backend/internal/config"
	"github.com/bventy/backend/internal/db"
	"github.com/bventy/backend/internal/services"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

type OAuthHandler struct {
	Config      *config.Config
	OAuthConfig *oauth2.Config
}

func NewOAuthHandler(cfg *config.Config) *OAuthHandler {
	return &OAuthHandler{
		Config: cfg,
		OAuthConfig: &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  cfg.GoogleRedirectURI,
			Endpoint:     google.Endpoint,
			Scopes: []string{
				calendar.CalendarEventsScope,
				calendar.CalendarReadonlyScope,
				"https://www.googleapis.com/auth/userinfo.email",
				"openid",
			},
		},
	}
}

// GET /auth/google
func (h *OAuthHandler) InitGoogleAuth(c *gin.Context) {
	// Securely pass the vendorID or userID in the state, or use a session cookie
	// For now, we'll rely on the user being logged in during the flow
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// We'll store the state in a cookie or use it to verify the request
	state := userID.(string)
	url := h.OAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GET /auth/google/callback
func (h *OAuthHandler) GoogleCallback(c *gin.Context) {
	state := c.Query("state") // This is our userID
	code := c.Query("code")

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Code missing"})
		return
	}

	token, err := h.OAuthConfig.Exchange(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token: " + err.Error()})
		return
	}

	// Get vendor_id for this user
	var vendorID string
	err = db.Pool.QueryRow(c.Request.Context(), "SELECT id FROM vendor_profiles WHERE owner_user_id = $1", state).Scan(&vendorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor profile not found"})
		return
	}

	// Save or Update Connection
	query := `
		INSERT INTO vendor_oauth_connections (vendor_id, provider, access_token, refresh_token, expires_at)
		VALUES ($1, 'google', $2, $3, $4)
		ON CONFLICT (vendor_id, provider) DO UPDATE SET
			access_token = EXCLUDED.access_token,
			refresh_token = COALESCE(NULLIF(EXCLUDED.refresh_token, ''), vendor_oauth_connections.refresh_token),
			expires_at = EXCLUDED.expires_at,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err = db.Pool.Exec(c.Request.Context(), query,
		vendorID,
		token.AccessToken,
		token.RefreshToken,
		token.Expiry,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save connection: " + err.Error()})
		return
	}

	// Trigger initial sync in background
	syncService := services.NewCalendarSyncService(h.Config)
	go func() {
		if err := syncService.SyncGoogleToBventy(vendorID); err != nil {
			fmt.Printf("Initial sync failed for vendor %s: %v\n", vendorID, err)
		}
	}()

	// Redirect back to the frontend calendar page
	frontendURL := "https://partner.bventy.in/calendar"
	if h.Config.CookieDomain == ".lvh.me" {
		frontendURL = "http://partner.lvh.me:3003/calendar"
	}
	c.Redirect(http.StatusTemporaryRedirect, frontendURL)
}
// GET /vendor/calendar/sync/status
func (h *OAuthHandler) GetGoogleSyncStatus(c *gin.Context) {
	userID, _ := c.Get("userID")
	var vendorID string
	err := db.Pool.QueryRow(c.Request.Context(), "SELECT id FROM vendor_profiles WHERE owner_user_id = $1", userID.(string)).Scan(&vendorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor not found"})
		return
	}

	var exists bool
	err = db.Pool.QueryRow(c.Request.Context(), "SELECT EXISTS(SELECT 1 FROM vendor_oauth_connections WHERE vendor_id = $1 AND provider = 'google')", vendorID).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"connected": exists})
}

// DELETE /vendor/calendar/sync
func (h *OAuthHandler) DisconnectGoogleCalendar(c *gin.Context) {
	userID, _ := c.Get("userID")
	var vendorID string
	err := db.Pool.QueryRow(c.Request.Context(), "SELECT id FROM vendor_profiles WHERE owner_user_id = $1", userID.(string)).Scan(&vendorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor not found"})
		return
	}

	syncService := services.NewCalendarSyncService(h.Config)
	if err := syncService.DisconnectGoogle(vendorID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect and clean up calendar data: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Google Calendar disconnected and data cleaned up successfully (industry-standard bidirectional cleanup)"})
}
