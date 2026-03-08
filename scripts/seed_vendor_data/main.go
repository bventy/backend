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

	// Target Vendor: admin@bventy.in's profile
	adminVendorID := "5886a640-02ae-4d6c-b0af-0e70df99a315"

	organizerIDs := []string{
		"a52c153e-614f-4515-a846-04891f0b2091",
		"d97e6450-2192-4584-9ffd-ac467e928f60",
		"285109f1-83be-4039-80ee-708ba6642d45",
		"a8c804ec-813b-4305-93a3-392650727072",
		"106634e3-fd7b-437a-881f-7f73678fec26",
		"29627bd6-0991-4a36-affe-71363d630aec",
		"5ec9ff27-34d8-49e2-9664-41042f5b3e70",
		"4c34c81b-b283-4ef8-bdc2-0e96ed97e4c2",
		"677681f7-174e-401a-8a9e-8234af97f4cd",
		"189e5ad4-6bd5-4bc7-925b-6632136d5e75",
	}

	// 1. Create Events for these organizers if they don't have enough
	fmt.Println("Ensuring events exist for organizers...")
	var eventIDs []string
	for i, oid := range organizerIDs {
		title := fmt.Sprintf("Event by Organizer %d", i)
		city := "Pune"
		date := time.Now().AddDate(0, 0, rand.Intn(60)-30)
		var eid string
		err := pool.QueryRow(ctx, "INSERT INTO events (title, city, event_date, organizer_user_id, event_type) VALUES ($1, $2, $3, $4, $5) RETURNING id", 
			title, city, date, oid, "party").Scan(&eid)
		if err == nil {
			eventIDs = append(eventIDs, eid)
		}
	}

	// 2. Create 15 Leads (Quote Requests) for our Admin Vendor
	fmt.Println("Creating Leads (Quote Requests)...")
	leadStatuses := []string{"pending", "responded", "accepted", "rejected", "revision_requested"}
	messages := []string{
		"Love your portfolio! Available for a corporate lunch?",
		"Initial inquiry for a small private gathering.",
		"Looking for premium catering for 50 pax.",
		"Can you accommodate dietary restrictions?",
		"Need a quote for a weekend event in October.",
		"Follow up on our previous conversation about the gala.",
		"Budget is firm but looking for the best quality.",
		"Heard great things from FOSS Pune organizers.",
	}

	for i := 0; i < 15; i++ {
		eid := eventIDs[rand.Intn(len(eventIDs))]
		oid := organizerIDs[rand.Intn(len(organizerIDs))]
		status := leadStatuses[rand.Intn(len(leadStatuses))]
		msg := messages[rand.Intn(len(messages))]
		price := rand.Intn(8000) + 2000
		
		query := `INSERT INTO quote_requests (event_id, vendor_id, organizer_user_id, message, status, quoted_price, created_at) 
                  VALUES ($1, $2, $3, $4, $5, $6, $7)`
		_, err := pool.Exec(ctx, query, eid, adminVendorID, oid, msg, status, price, time.Now().AddDate(0, 0, -i))
		if err != nil {
			fmt.Printf("Error creating lead %d: %v\n", i, err)
		}
	}

	// 3. Create 8 Reviews
	fmt.Println("Creating Reviews...")
	revTexts := []string{
		"Absolutely stunning service! Highly recommended.",
		"Professional and timely. Made our event special.",
		"The quality exceeded our expectations.",
		"Good, but could improve coordination.",
		"Our guests loved everything. Thank you!",
		"Consistent quality and great communication.",
		"Will definitely book again for our next retreat.",
		"A bit pricey but worth every penny for the peace of mind.",
	}

	for i := 0; i < 8; i++ {
		oid := organizerIDs[rand.Intn(len(organizerIDs))]
		rating := rand.Intn(2) + 4 // 4 or 5 stars
		comment := revTexts[i]
		
		query := `INSERT INTO vendor_reviews (vendor_id, organizer_user_id, rating, comment, created_at) VALUES ($1, $2, $3, $4, $5)`
		_, err := pool.Exec(ctx, query, adminVendorID, oid, rating, comment, time.Now().AddDate(0, 0, -rand.Intn(60)))
		if err != nil {
			fmt.Printf("Error creating review %d: %v\n", i, err)
		}
	}

	// 4. Create Platform Activity Logs
	fmt.Println("Creating Activity Logs (Profile Views)...")
	for i := 0; i < 50; i++ {
		oid := organizerIDs[rand.Intn(len(organizerIDs))]
		query := `INSERT INTO platform_activity_log (entity_type, entity_id, action_type, actor_user_id, created_at) 
                  VALUES ('vendor_profile', $1, 'view', $2, $3)`
		_, err := pool.Exec(ctx, query, adminVendorID, oid, time.Now().AddDate(0, 0, -rand.Intn(30)))
		if err != nil {
			fmt.Printf("Error creating activity log %d: %v\n", i, err)
		}
	}

	fmt.Println("✅ Batch seeding for Vendor (admin@bventy.in) completed successfully!")
}
