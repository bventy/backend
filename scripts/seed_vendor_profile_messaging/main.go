package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type QuotePair struct {
	ID              string
	OrganizerUserID string
}

func main() {
	connStr := "postgresql://neondb_owner:npg_ABuQl7cj5heW@ep-wispy-brook-a1ij8hbi-pooler.ap-southeast-1.aws.neon.tech/bventy_mv1?sslmode=require"
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	adminVendorID := "5886a640-02ae-4d6c-b0af-0e70df99a315"
	adminUserID := "600c1bf5-aee0-4c51-8d78-7da9144bab4d"

	quotePairs := []QuotePair{
		{"fc11cf04-3b57-44b8-af1a-002cc95d7a33", "d97e6450-2192-4584-9ffd-ac467e928f60"},
		{"ec0f1d29-076e-4995-a918-e18deb19ce7e", "4c34c81b-b283-4ef8-bdc2-0e96ed97e4c2"},
		{"1b006ec9-b635-4d0f-b5ba-dcd95325d5a5", "a8c804ec-813b-4305-93a3-392650727072"},
		{"c99dd0cd-260c-4bbf-ac53-3942a4a485ed", "29627bd6-0991-4a36-affe-71363d630aec"},
		{"628c4ead-59b8-4148-b76a-ca95bdcdca91", "d97e6450-2192-4584-9ffd-ac467e928f60"},
		{"3cf619c2-599a-44d4-9c2b-d4f05eeeb4d4", "285109f1-83be-4039-80ee-708ba6642d45"},
		{"39b5b8a2-f4e7-4e9e-94f7-f9f5b52d20c6", "189e5ad4-6bd5-4bc7-925b-6632136d5e75"},
		{"ca7e4b46-18fb-44aa-885f-60216503f3d9", "285109f1-83be-4039-80ee-708ba6642d45"},
		{"ae5dd7e6-2594-4d57-930f-2908e68c3e3c", "106634e3-fd7b-437a-881f-7f73678fec26"},
		{"31b0b542-7204-4131-b84d-7497e9c40190", "106634e3-fd7b-437a-881f-7f73678fec26"},
	}

	// 1. Seed Services
	fmt.Println("Seeding Services...")
	services := []struct {
		name      string
		price     float64
		unit      string
		desc      string
	}{
		{"Full Course Buffet", 1200, "/ Plate", "Premium international cuisine catering for large events."},
		{"Live Pasta Counter", 450, "/ Person", "Chef-prepared fresh pasta with choice of sauces."},
		{"Mocktail Bar", 15000, "/ Event", "Unlimited mocktails and soft drinks with professional bartenders."},
		{"Premium Decor Package", 50000, "/ Setup", "Elegant floral and lighting setup for high-end gatherings."},
		{"Audio-Visual Setup", 25000, "/ Day", "Complete sound system and LED screen setup for presentations."},
	}

	for _, s := range services {
		_, err := pool.Exec(ctx, "INSERT INTO vendor_services (vendor_id, name, base_price, price_unit, status, description) VALUES ($1, $2, $3, $4, 'active', $5)",
			adminVendorID, s.name, s.price, s.unit, s.desc)
		if err != nil {
			fmt.Printf("Error seeding service %s: %v\n", s.name, err)
		}
	}

	// 2. Seed Pricing Rules
	fmt.Println("Seeding Pricing Rules...")
	_, err = pool.Exec(ctx, `
		INSERT INTO vendor_pricing_rules (vendor_id, weekend_premium_enabled, weekend_premium_percentage, last_minute_booking_enabled, last_minute_booking_percentage, last_minute_days)
		VALUES ($1, true, 15, true, 20, 7)
		ON CONFLICT (vendor_id) DO UPDATE SET
			weekend_premium_enabled = EXCLUDED.weekend_premium_enabled,
			weekend_premium_percentage = EXCLUDED.weekend_premium_percentage,
			last_minute_booking_enabled = EXCLUDED.last_minute_booking_enabled,
			last_minute_booking_percentage = EXCLUDED.last_minute_booking_percentage,
			last_minute_days = EXCLUDED.last_minute_days
	`, adminVendorID)
	if err != nil {
		fmt.Printf("Error seeding pricing rules: %v\n", err)
	}

	// 3. Seed Messaging
	fmt.Println("Seeding Conversations & Messages...")
	organizerMessages := []string{
		"Hi, I saw your profile and I'm interested in your services. Can we discuss further?",
		"Do you have a menu I can look at?",
		"What is included in the premium decor package?",
		"Is there a discount for groups larger than 100?",
		"Thank you for the quick response!",
	}
	vendorMessages := []string{
		"Hello! Thank you for reaching out. I'd be happy to help you with your event.",
		"Yes, I'll send over our latest brochure and pricing list right away.",
		"The premium package includes floral arrangements, ambient lighting, and stage setup.",
		"We can certainly discuss a volume discount based on your final numbers.",
		"You're welcome! Looking forward to working with you.",
	}

	for _, p := range quotePairs {
		var cid string
		err := pool.QueryRow(ctx, "INSERT INTO conversations (vendor_id, organizer_user_id, quote_id, last_message_at) VALUES ($1, $2, $3, $4) ON CONFLICT (quote_id) DO UPDATE SET last_message_at = EXCLUDED.last_message_at RETURNING id",
			adminVendorID, p.OrganizerUserID, p.ID, time.Now()).Scan(&cid)
		if err != nil {
			fmt.Printf("Error creating conversation for quote %s: %v\n", p.ID, err)
			continue
		}

		// Create 4 messages per conversation
		for i := 0; i < 4; i++ {
			var senderID, body string
			if i%2 == 0 {
				senderID = p.OrganizerUserID
				body = organizerMessages[i/2%len(organizerMessages)]
			} else {
				senderID = adminUserID
				body = vendorMessages[i/2%len(vendorMessages)]
			}
			
			_, err := pool.Exec(ctx, "INSERT INTO messages (conversation_id, sender_user_id, body, created_at) VALUES ($1, $2, $3, $4)",
				cid, senderID, body, time.Now().Add(time.Duration(i)*time.Minute))
			if err != nil {
				fmt.Printf("Error seeding message %d for conversation %s: %v\n", i, cid, err)
			}
		}
	}

	fmt.Println("✅ Batch seeding for Vendor Profile & Messaging completed successfully!")
}
