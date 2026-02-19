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
	// Load .env to get DATABASE_URL
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

	// Query key users columns mostly recently created
	rows, err := conn.Query(context.Background(), "SELECT id, email, username, profile_image_url FROM users ORDER BY created_at DESC LIMIT 5")
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	fmt.Println("Most recent updated users:")
	for rows.Next() {
		var id, email, username string
		var imageURL *string
		if err := rows.Scan(&id, &email, &username, &imageURL); err != nil {
			log.Fatalf("Scan failed: %v", err)
		}
		img := "<nil>"
		if imageURL != nil {
			img = *imageURL
		}
		fmt.Printf("User: %s (%s) - Image: %s\n", username, email, img)
	}
}
