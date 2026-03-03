package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Load configuration
	godotenv.Load()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	if len(os.Args) < 2 {
		log.Fatal("Usage: go run cmd/migrate/main.go <path_to_migration_sql>")
	}
	migrationPath := os.Args[1]

	// 2. Read migration file
	content, err := os.ReadFile(migrationPath)
	if err != nil {
		log.Fatalf("Failed to read migration file %s: %v", migrationPath, err)
	}

	// 3. Connect to DB
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer pool.Close()

	// 4. Execute migration
	fmt.Printf("Executing migration: %s...\n", filepath.Base(migrationPath))
	_, err = pool.Exec(context.Background(), string(content))
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("✅ Migration completed successfully!")
}
