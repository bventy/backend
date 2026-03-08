package main

import (
	"context"
	"fmt"
	"log"
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

	userEmail := "onkaryaglewad@gmail.com"
	userID := "3623c481-1429-4323-9744-fbaab773c48e"
	adminVendorID := "5886a640-02ae-4d6c-b0af-0e70df99a315"
	adminUserID := "600c1bf5-aee0-4c51-8d78-7da9144bab4d"

	fmt.Printf("Seeding chat for %s...\n", userEmail)

	// 1. Create Event
	var eid string
	err = pool.QueryRow(ctx, "INSERT INTO events (title, city, event_date, organizer_user_id, event_type) VALUES ($1, $2, $3, $4, $5) RETURNING id", 
		"Private Celebration", "Pune", time.Now().AddDate(0, 1, 0), userID, "party").Scan(&eid)
	if err != nil {
		log.Fatalf("Error creating event: %v", err)
	}

	// 2. Create ACCEPTED Quote Request
	var qid string
	err = pool.QueryRow(ctx, `
		INSERT INTO quote_requests (event_id, vendor_id, organizer_user_id, message, status, quoted_price, accepted_at) 
		VALUES ($1, $2, $3, $4, 'accepted', $5, NOW()) RETURNING id`,
		eid, adminVendorID, userID, "Hi Admin, I'm interested in your services for my private celebration. Please share a quote.", 5000).Scan(&qid)
	if err != nil {
		log.Fatalf("Error creating quote: %v", err)
	}

	// 3. Create UNLOCKED Conversation
	var cid string
	err = pool.QueryRow(ctx, `
		INSERT INTO conversations (vendor_id, organizer_user_id, quote_id, last_message_at, chat_locked) 
		VALUES ($1, $2, $3, NOW(), false) RETURNING id`,
		adminVendorID, userID, qid).Scan(&cid)
	if err != nil {
		log.Fatalf("Error creating conversation: %v", err)
	}

	// 4. Seed Messages
	messages := []struct {
		sender string
		body   string
	}{
		{userID, "Hi! I just accepted your quote. Excited to work with you!"},
		{adminUserID, "That's great news! Thank you for choosing us for your private celebration."},
		{userID, "Can we discuss the menu options next?"},
		{adminUserID, "Absolutely! I've shared our standard menu via email, but we can customize it as well."},
		{userID, "Perfect, I'll take a look and get back to you."},
	}

	for i, m := range messages {
		_, err := pool.Exec(ctx, "INSERT INTO messages (conversation_id, sender_user_id, body, created_at) VALUES ($1, $2, $3, $4)",
			cid, m.sender, m.body, time.Now().Add(time.Duration(i)*time.Minute))
		if err != nil {
			fmt.Printf("Error seeding message %d: %v\n", i, err)
		}
	}

	fmt.Println("✅ Successfully seeded UNLOCKED chat between User and Admin Vendor!")
}
