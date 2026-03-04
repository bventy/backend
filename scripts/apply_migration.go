package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL must be set")
	}

	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(context.Background(), "ALTER TABLE quote_requests ADD COLUMN IF NOT EXISTS internal_notes TEXT")
	if err != nil {
		log.Fatalf("Migration failed: %v\n", err)
	}

	log.Println("✅ Migration applied: internal_notes added to quote_requests")
}
