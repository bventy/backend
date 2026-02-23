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
}
