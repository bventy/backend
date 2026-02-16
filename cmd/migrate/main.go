package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/once-human/bventy-backend/internal/config"
	"github.com/once-human/bventy-backend/internal/db"
)

func main() {
	// Load config
	cfg := config.LoadConfig()

	// Connect to DB
	db.Connect(cfg)
	defer db.Pool.Close()

	// Read migration file
	migrationFile := "internal/db/migrations/010_restore_password_auth.sql"
	content, err := os.ReadFile(migrationFile)
	if err != nil {
		log.Fatalf("Failed to read migration file: %v", err)
	}

	fmt.Println("Running migration:", migrationFile)

	// Execute SQL
	_, err = db.Pool.Exec(context.Background(), string(content))
	if err != nil {
		log.Fatalf("Failed to execute migration: %v", err)
	}

	fmt.Println("✅ Migration applied successfully!")
}
