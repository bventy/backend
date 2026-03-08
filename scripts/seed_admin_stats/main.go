package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	connStr := "postgresql://neondb_owner:npg_ABuQl7cj5heW@ep-wispy-brook-a1ij8hbi-pooler.ap-southeast-1.aws.neon.tech/bventy_mv1?sslmode=require"
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	adminUserID := "600c1bf5-aee0-4c51-8d78-7da9144bab4d"
	// adminVendorID := "5886a640-02ae-4d6c-b0af-0e70df99a315"

	vendorIDs := []string{
		"967fea30-a573-4b5f-a1b0-08a5e0b2618c",
		"c5f43207-23d1-4dfb-89ed-6a807e011dc1",
		"513b6849-01b5-4ae5-879e-937fc01aef34",
		"d6f47eca-ac33-46d8-93a6-fd5c29629345",
		"650cca00-5ec2-4f3e-9fd0-a8719a58d817",
		"a48257d4-9cf4-46b0-bcd2-0dfc0f9a02a4",
		"fe3681c6-3c27-40d6-b5d3-b69c1b798532",
		"ac31b068-511b-44b9-b1c2-56468bb4f9fc",
		"f04afa95-8076-4030-8760-4c7fad37169b",
		"e3d7c36d-20a7-4af1-9f63-440972bc359c",
	}

	// 1. Create 7 Groups
	fmt.Println("Creating groups...")
	groupNames := []string{
		"Tech Innovators Pune", "Pune Event Org", "Global SaaS Community",
		"Startup Networking", "Creative Arts Hub", "Wellness & Yoga Collective",
		"Product Design Leaders",
	}
	var groupIDs []string
	for _, name := range groupNames {
		slug := fmt.Sprintf("%s-%d", name, rand.Intn(10000))
		var gid string
		err := pool.QueryRow(ctx, "INSERT INTO groups (name, slug, owner_user_id) VALUES ($1, $2, $3) RETURNING id", name, slug, adminUserID).Scan(&gid)
		if err != nil {
			fmt.Printf("Error creating group %s: %v\n", name, err)
			continue
		}
		groupIDs = append(groupIDs, gid)
		// Add admin as owner
		_, _ = pool.Exec(ctx, "INSERT INTO group_members (group_id, user_id, role) VALUES ($1, $2, 'owner')", gid, adminUserID)
	}

	// 2. Create 10 Events
	fmt.Println("Creating events...")
	eventTitles := []string{
		"Pune FOSS Conf 2026", "Corporate Retreat Q2", "Wedding Anniversary Gala",
		"Product Launch Party", "DevOps Workshop", "Art Exhibition Pune",
		"Startup Demo Day", "Annual Charity Auction", "Summer Music Festival",
		"Tech Meetup #42",
	}

	var eventIDs []string
	for i, title := range eventTitles {
		status := "upcoming"
		var completedAt *time.Time
		if i < 5 { // First 5 are completed
			status = "completed"
			now := time.Now().AddDate(0, 0, -i-1)
			completedAt = &now
		}

		eventDate := time.Now().AddDate(0, 0, (i-5)*10)
		city := "Pune"
		eventType := "party"
		budgetMin := i * 500
		budgetMax := budgetMin + 1000

		var eid string
		query := `INSERT INTO events (title, city, event_type, event_date, budget_min, budget_max, organizer_user_id, status, completed_at) 
                  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
		err := pool.QueryRow(ctx, query, title, city, eventType, eventDate, budgetMin, budgetMax, adminUserID, status, completedAt).Scan(&eid)
		if err != nil {
			fmt.Printf("Error creating event %s: %v\n", title, err)
			continue
		}
		eventIDs = append(eventIDs, eid)
	}

	// 3. Create 15 Quote Requests
	fmt.Println("Creating quote requests...")
	statuses := []string{"pending", "responded", "accepted", "rejected"}
	for i := 0; i < 15; i++ {
		eid := eventIDs[rand.Intn(len(eventIDs))]
		vid := vendorIDs[rand.Intn(len(vendorIDs))]
		status := statuses[rand.Intn(len(statuses))]
		msg := fmt.Sprintf("Hi, we are interested in your services for our event %d. Please provide a quote.", i)
		
		query := `INSERT INTO quote_requests (event_id, vendor_id, organizer_user_id, message, status, quoted_price, created_at) 
                  VALUES ($1, $2, $3, $4, $5, $6, $7)`
		_, err := pool.Exec(ctx, query, eid, vid, adminUserID, msg, status, rand.Intn(5000)+1000, time.Now().AddDate(0, 0, -rand.Intn(30)))
		if err != nil {
			fmt.Printf("Error creating quote request %d: %v\n", i, err)
		}
	}

	fmt.Println("✅ Batch seeding for admin@bventy.in completed successfully!")
}
