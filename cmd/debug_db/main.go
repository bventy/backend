package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bventy/backend/internal/config"
	"github.com/bventy/backend/internal/db"
)

func main() {
	cfg := config.LoadConfig()
	db.Connect(cfg)
	defer db.Pool.Close()

	tables := []string{"events", "vendor_reviews", "quote_requests", "platform_activity_log", "users", "vendor_profiles"}

	for _, table := range tables {
		fmt.Printf("\n--- Table: %s ---\n", table)

		// Check if table exists
		var exists bool
		err := db.Pool.QueryRow(context.Background(),
			"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)", table).Scan(&exists)
		if err != nil {
			log.Printf("Error checking table %s: %v", table, err)
			continue
		}

		if !exists {
			fmt.Printf("❌ Table %s DOES NOT EXIST\n", table)
			continue
		}
		fmt.Printf("✅ Table %s exists\n", table)

		// List columns
		rows, err := db.Pool.Query(context.Background(),
			"SELECT column_name, data_type FROM information_schema.columns WHERE table_name = $1", table)
		if err != nil {
			log.Printf("Error listing columns for %s: %v", table, err)
			continue
		}
		defer rows.Close()

		fmt.Println("Columns:")
		for rows.Next() {
			var name, dtype string
			if err := rows.Scan(&name, &dtype); err == nil {
				fmt.Printf("  - %s (%s)\n", name, dtype)
			}
		}
	}
	// Simulation
	fmt.Println("\n--- Query Simulation: GetVendorReviews ---")
	testVendorID := "2be92968-1e58-4f08-bb3d-0b07314588c7"
	query1 := `
		SELECT r.id, r.rating, r.comment, r.created_at, u.full_name as organizer_name, u.profile_image_url
		FROM vendor_reviews r
		JOIN users u ON r.organizer_user_id = u.id
		WHERE r.vendor_id::text = $1 OR r.vendor_id = (SELECT id FROM vendor_profiles WHERE slug = $1)
		ORDER BY r.created_at DESC
	`
	rows1, err := db.Pool.Query(context.Background(), query1, testVendorID)
	if err != nil {
		fmt.Printf("❌ GetVendorReviews Query FAILED: %v\n", err)
	} else {
		fmt.Println("✅ GetVendorReviews Query Success")
		rows1.Close()
	}

	fmt.Println("\n--- Query Simulation: CheckEligibility ---")
	// testOrgID := " some-existing-user-id " // We need a real one for a true test, but let's try a format check
	query2 := `
		SELECT EXISTS (
			SELECT 1
			FROM quote_requests qr
			JOIN events e ON qr.event_id = e.id
			WHERE qr.organizer_user_id::text = $1 
			  AND qr.vendor_id::text = $2 
			  AND qr.status = 'accepted'
			  AND (e.status = 'completed' OR e.event_date < NOW())
		)
	`
	var exists bool
	err = db.Pool.QueryRow(context.Background(), query2, "00000000-0000-0000-0000-000000000000", testVendorID).Scan(&exists)
	if err != nil {
		fmt.Printf("❌ CheckEligibility Query FAILED: %v\n", err)
	} else {
		fmt.Println("✅ CheckEligibility Query Success")
	}
}
