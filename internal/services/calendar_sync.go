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

func (s *CalendarSyncService) getGoogleService(ctx context.Context, vendorID string) (*calendar.Service, error) {
	var access, refresh string
	var expiry time.Time
	err := db.Pool.QueryRow(ctx, "SELECT access_token, refresh_token, expires_at FROM vendor_oauth_connections WHERE vendor_id = $1 AND provider = 'google'", vendorID).Scan(&access, &refresh, &expiry)
	if err != nil {
		log.Printf("[CalendarSync] Error fetching OAuth connection for vendor %s: %v", vendorID, err)
		return nil, fmt.Errorf("no oauth connection for vendor %s: %v", vendorID, err)
	}

	if s.Config.GoogleClientID == "" || s.Config.GoogleClientSecret == "" {
		log.Printf("[CalendarSync] Missing Google client credentials in config")
		return nil, fmt.Errorf("google client credentials not configured")
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

	newToken, err := tokenSource.Token()
	if err != nil {
		log.Printf("[CalendarSync] Failed to refresh token for vendor %s: %v", vendorID, err)
		return nil, fmt.Errorf("failed to get valid token: %v", err)
	}

	if newToken.AccessToken != access {
		_, _ = db.Pool.Exec(ctx, "UPDATE vendor_oauth_connections SET access_token = $1, expires_at = $2, updated_at = NOW() WHERE vendor_id = $3", newToken.AccessToken, newToken.Expiry, vendorID)
	}

	return calendar.NewService(ctx, option.WithTokenSource(oauth2.StaticTokenSource(newToken)))
}

func (s *CalendarSyncService) SyncGoogleToBventy(vendorID string) error {
	ctx := context.Background()
	srv, err := s.getGoogleService(ctx, vendorID)
	if err != nil {
		return err
	}

	// 1. Get List of all calendars
	calendarList, err := srv.CalendarList.List().Do()
	if err != nil {
		log.Printf("[CalendarSync] Failed to list calendars for vendor %s: %v", vendorID, err)
		return fmt.Errorf("failed to list calendars: %v", err)
	}

	// 2. Define sync window (1 month back to 6 months ahead)
	tMin := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)
	tMax := time.Now().AddDate(0, 6, 0).Format(time.RFC3339)

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	totalSynced := 0
	for _, cal := range calendarList.Items {
		// Only sync calendars that are primary or selected by the user in their UI
		if !cal.Selected && cal.Id != "primary" {
			continue
		}

		log.Printf("[CalendarSync] Syncing calendar %s (%s) for vendor %s", cal.Summary, cal.Id, vendorID)
		
		events, err := srv.Events.List(cal.Id).ShowDeleted(false).SingleEvents(true).TimeMin(tMin).TimeMax(tMax).Do()
		if err != nil {
			log.Printf("[CalendarSync] Google API List events failed for calendar %s: %v", cal.Id, err)
			continue
		}

		for _, item := range events.Items {
			if item.Status == "confirmed" {
				title := item.Summary
				if title == "" {
					title = "Busy"
				}
				
				var start, end time.Time
				isAllDay := false

				if item.Start.DateTime != "" {
					start, _ = time.Parse(time.RFC3339, item.Start.DateTime)
					end, _ = time.Parse(time.RFC3339, item.End.DateTime)
				} else {
					isAllDay = true
					start, _ = time.Parse("2006-01-02", item.Start.Date)
					// Handle exclusive end dates for all-day events
					tempEnd, _ := time.Parse("2006-01-02", item.End.Date)
					end = tempEnd.Add(-1 * time.Second)
				}

				// Title Preservation Logic:
				// If we already have this event, and it's NOT a generic "Busy" title, 
				// we might want to keep the local title if it was modified.
				// However, for simplicity now, we overwrite unless the local title is "special".
				// A better way is checking if 'type' is 'manual_block' and it wasn't originally from Google.
				// But since we use google_event_id for conflict, we can assume Google is the source of truth for these.

				query := `
					INSERT INTO vendor_calendar_blocks (vendor_id, title, start_time, end_time, is_all_day, type, google_event_id)
					VALUES ($1, $2, $3, $4, $5, 'manual_block', $6)
					ON CONFLICT (vendor_id, google_event_id) WHERE google_event_id IS NOT NULL DO UPDATE SET
						title = EXCLUDED.title,
						start_time = EXCLUDED.start_time,
						end_time = EXCLUDED.end_time,
						is_all_day = EXCLUDED.is_all_day,
						updated_at = CURRENT_TIMESTAMP
				`
				_, err = tx.Exec(ctx, query, vendorID, title, start, end, isAllDay, item.Id)
				if err != nil {
					log.Printf("Failed to upsert google event %s: %v", item.Id, err)
				} else {
					totalSynced++
				}
			}
		}
	}

	log.Printf("[CalendarSync] Successfully synced %d events total for vendor %s", totalSynced, vendorID)
	return tx.Commit(ctx)
}

func (s *CalendarSyncService) PushBventyToGoogle(vendorID string) error {
	ctx := context.Background()
	srv, err := s.getGoogleService(ctx, vendorID)
	if err != nil {
		return err
	}

	// 1. Sync Manual Blocks (that are NOT from Google)
	rows, err := db.Pool.Query(ctx, `
		SELECT id, title, start_time, end_time, is_all_day, google_event_id 
		FROM vendor_calendar_blocks 
		WHERE vendor_id = $1 AND google_event_id IS NULL AND type = 'manual_block'
	`, vendorID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id, title string
		var start, end time.Time
		var isAllDay bool
		var gID *string
		if err := rows.Scan(&id, &title, &start, &end, &isAllDay, &gID); err == nil {
			event := &calendar.Event{
				Summary: title,
				Start: &calendar.EventDateTime{
					DateTime: start.Format(time.RFC3339),
				},
				End: &calendar.EventDateTime{
					DateTime: end.Format(time.RFC3339),
				},
			}
			if isAllDay {
				event.Start = &calendar.EventDateTime{Date: start.Format("2006-01-02")}
				event.End = &calendar.EventDateTime{Date: end.Format("2006-01-02")}
			}

			createdEvent, err := srv.Events.Insert("primary", event).Do()
			if err == nil {
				_, _ = db.Pool.Exec(ctx, "UPDATE vendor_calendar_blocks SET google_event_id = $1 WHERE id = $2", createdEvent.Id, id)
			}
		}
	}

	// 2. Sync Confirmed Bookings (Accepted Quote Requests)
	quoteRows, err := db.Pool.Query(ctx, `
		SELECT qr.id, e.title, e.event_date, qr.google_event_id
		FROM quote_requests qr
		JOIN events e ON qr.event_id = e.id
		WHERE qr.vendor_id = $1 AND qr.status = 'accepted' AND qr.google_event_id IS NULL
	`, vendorID)
	if err == nil {
		defer quoteRows.Close()
		for quoteRows.Next() {
			var id, title string
			var eventDate time.Time
			var gID *string
			if err := quoteRows.Scan(&id, &title, &eventDate, &gID); err == nil {
				event := &calendar.Event{
					Summary: "Bventy Booking: " + title,
					Description: "Confirmed booking via Bventy platform.",
					Start: &calendar.EventDateTime{
						Date: eventDate.Format("2006-01-02"),
					},
					End: &calendar.EventDateTime{
						Date: eventDate.AddDate(0, 0, 1).Format("2006-01-02"),
					},
				}
				createdEvent, err := srv.Events.Insert("primary", event).Do()
				if err == nil {
					_, _ = db.Pool.Exec(ctx, "UPDATE quote_requests SET google_event_id = $1 WHERE id = $2", createdEvent.Id, id)
				}
			}
		}
	}

	return nil
}

func (s *CalendarSyncService) DeleteGoogleEvent(ctx context.Context, vendorID string, googleEventID string) error {
	srv, err := s.getGoogleService(ctx, vendorID)
	if err != nil {
		return err
	}
	return srv.Events.Delete("primary", googleEventID).Do()
}
