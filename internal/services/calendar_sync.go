package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bventy/backend/internal/config"
	"github.com/bventy/backend/internal/db"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type CalendarSyncService struct {
	Config *config.Config
}

func NewCalendarSyncService(cfg *config.Config) *CalendarSyncService {
	return &CalendarSyncService{Config: cfg}
}

func (s *CalendarSyncService) getOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.Config.GoogleClientID,
		ClientSecret: s.Config.GoogleClientSecret,
		RedirectURL:  s.Config.GoogleRedirectURI,
		Endpoint:     google.Endpoint,
		Scopes:       []string{calendar.CalendarEventsScope, calendar.CalendarReadonlyScope},
	}
}

func (s *CalendarSyncService) SyncGoogleToBventy(vendorID string) error {
	ctx := context.Background()
	
	// 1. Get OAuth Connection
	var access, refresh string
	var expiry time.Time
	err := db.Pool.QueryRow(ctx, "SELECT access_token, refresh_token, expires_at FROM vendor_oauth_connections WHERE vendor_id = $1 AND provider = 'google'", vendorID).Scan(&access, &refresh, &expiry)
	if err != nil {
		return fmt.Errorf("no oauth connection for vendor %s: %v", vendorID, err)
	}

	tokenSource := (&oauth2.Config{
		ClientID:     s.Config.GoogleClientID,
		ClientSecret: s.Config.GoogleClientSecret,
		Endpoint:     google.Endpoint,
	}).TokenSource(ctx, &oauth2.Token{
		AccessToken:  access,
		RefreshToken: refresh,
		Expiry:       expiry,
	})

	// 2. Refresh token if needed and save
	newToken, err := tokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to get valid token: %v", err)
	}
	if newToken.AccessToken != access {
		_, _ = db.Pool.Exec(ctx, "UPDATE vendor_oauth_connections SET access_token = $1, expires_at = $2, updated_at = NOW() WHERE vendor_id = $3", newToken.AccessToken, newToken.Expiry, vendorID)
	}

	srv, err := calendar.NewService(ctx, option.WithTokenSource(oauth2.StaticTokenSource(newToken)))
	if err != nil {
		return fmt.Errorf("unable to retrieve Calendar client: %v", err)
	}

	// 3. Fetch "Busy" events (Free/Busy API is better but for now let's use List)
	// We'll fetch events from now to 3 months ahead
	tMin := time.Now().Format(time.RFC3339)
	tMax := time.Now().AddDate(0, 3, 0).Format(time.RFC3339)
	
	events, err := srv.Events.List("primary").ShowDeleted(false).SingleEvents(true).TimeMin(tMin).TimeMax(tMax).Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve events: %v", err)
	}

	// 4. Map and Save to Bventy (Idempotent: we match by a 'external_id' column if we had one, or clear and rebuild)
	// For simplicity, let's add an 'external_id' column to vendor_calendar_blocks if it doesn't exist
	// Or we can just use a specific type 'google_sync'
	
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Clear old synced blocks
	_, _ = tx.Exec(ctx, "DELETE FROM vendor_calendar_blocks WHERE vendor_id = $1 AND type = 'manual_block' AND title LIKE '[Google] %'", vendorID)

	for _, item := range events.Items {
		if item.Status == "confirmed" {
			title := "[Google] Busy"
			if item.Summary != "" {
				// title = "[Google] " + item.Summary // User preference might be to hide details
			}
			
			start, _ := time.Parse(time.RFC3339, item.Start.DateTime)
			if item.Start.DateTime == "" {
				start, _ = time.Parse("2006-01-02", item.Start.Date)
			}
			end, _ := time.Parse(time.RFC3339, item.End.DateTime)
			if item.End.DateTime == "" {
				end, _ = time.Parse("2006-01-02", item.End.Date)
			}

			isAllDay := item.Start.DateTime == ""

			_, err = tx.Exec(ctx, `
				INSERT INTO vendor_calendar_blocks (vendor_id, title, start_time, end_time, is_all_day, type)
				VALUES ($1, $2, $3, $4, $5, 'manual_block')
			`, vendorID, title, start, end, isAllDay)
			if err != nil {
				log.Printf("Failed to insert google event: %v", err)
			}
		}
	}

	return tx.Commit(ctx)
}

func (s *CalendarSyncService) PushBventyToGoogle(vendorID string) error {
	// Future implementation: Push newly created Bventy bookings to Google
	return nil
}
