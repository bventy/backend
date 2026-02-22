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

	// Check all existing tables
	fmt.Println("--- Existing Tables ---")
	tableRows, err := db.Pool.Query(context.Background(), "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'")
	if err == nil {
		for tableRows.Next() {
			var tableName string
			tableRows.Scan(&tableName)
			fmt.Println("-", tableName)
		}
		tableRows.Close()
	}

	tables := []string{"quote_requests", "events", "users", "platform_activity_log"}

	for _, table := range tables {
		fmt.Printf("\n--- Inspection of table: %s ---\n", table)
		query := fmt.Sprintf(`
			SELECT column_name, data_type, is_nullable, column_default
			FROM information_schema.columns
			WHERE table_name = '%s'
			ORDER BY ordinal_position;
		`, table)

		rows, err := db.Pool.Query(context.Background(), query)
		if err != nil {
			fmt.Printf("Error inspecting table %s: %v\n", table, err)
			continue
		}

		for rows.Next() {
			var name, dtype, nullable string
			var def *string
			if err := rows.Scan(&name, &dtype, &nullable, &def); err != nil {
				log.Fatal(err)
			}
			defaultVal := "NULL"
			if def != nil {
				defaultVal = *def
			}
			fmt.Printf("Column: %-25s | Type: %-20s | Null: %-5s | Default: %s\n", name, dtype, nullable, defaultVal)
		}
		rows.Close()
	}
}
