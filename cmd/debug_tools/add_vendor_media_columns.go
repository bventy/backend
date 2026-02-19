package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer conn.Close(context.Background())

	// Add gallery_images column
	_, err = conn.Exec(context.Background(), `
		ALTER TABLE vendor_profiles 
		ADD COLUMN IF NOT EXISTS gallery_images TEXT[] DEFAULT '{}';
	`)
	if err != nil {
		log.Fatalf("Failed to add gallery_images: %v", err)
	}
	fmt.Println("✅ Added gallery_images column")

	// Add portfolio_files column
	_, err = conn.Exec(context.Background(), `
		ALTER TABLE vendor_profiles 
		ADD COLUMN IF NOT EXISTS portfolio_files JSONB DEFAULT '[]';
	`)
	if err != nil {
		log.Fatalf("Failed to add portfolio_files: %v", err)
	}
	fmt.Println("✅ Added portfolio_files column")
}
