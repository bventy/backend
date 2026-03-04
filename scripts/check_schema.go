package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	connStr := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	if os.Getenv("DATABASE_URL") != "" {
		connStr = os.Getenv("DATABASE_URL")
	}

	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(context.Background())

	tables := []string{"events"}
	for _, table := range tables {
		fmt.Printf("\n--- Table: %s ---\n", table)
		rows, err := conn.Query(context.Background(), `
			SELECT column_name, data_type, is_nullable 
			FROM information_schema.columns 
			WHERE table_name = $1
			ORDER BY ordinal_position
		`, table)
		if err != nil {
			log.Printf("Error querying table %s: %v\n", table, err)
			continue
		}
		for rows.Next() {
			var name, dtype, nullable string
			rows.Scan(&name, &dtype, &nullable)
			fmt.Printf("%s (%s) Nullable: %s\n", name, dtype, nullable)
		}
		rows.Close()
	}
}
