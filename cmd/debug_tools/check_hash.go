package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	var hash string
	err = pool.QueryRow(context.Background(), "SELECT password_hash FROM users WHERE email = 'admin@gmail.com'").Scan(&hash)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("User: admin@gmail.com\n")
	fmt.Printf("Hash Length: %d\n", len(hash))
	fmt.Printf("Hash Prefix: %s\n", hash[:5])
}
