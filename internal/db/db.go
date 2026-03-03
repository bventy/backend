package db

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"os"

	"github.com/bventy/backend/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func Connect(cfg *config.Config) {
	var dbURL string

	if cfg.DatabaseURL != "" {
		dbURL = cfg.DatabaseURL
	} else {
		dsn := url.URL{
			Scheme: "postgres",
			User:   url.UserPassword(cfg.DBUser, cfg.DBPassword),
			Host:   fmt.Sprintf("%s:%s", cfg.DBHost, cfg.DBPort),
			Path:   "/" + cfg.DBName,
		}
		dbURL = dsn.String()
	}

	var err error
	Pool, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatal("❌ DB connection failed:", err)
	}

	err = Pool.Ping(context.Background())
	if err != nil {
		log.Fatal("❌ DB ping failed:", err)
	}

	fmt.Println("✅ Connected to PostgreSQL successfully!")

	// Auto-run migration 015
	InitSchema()
}

func InitSchema() {
	migrations := []string{
		"internal/db/migrations/015_email_system.sql",
		"internal/db/migrations/016_email_sender_customization.sql",
	}

	for _, migrationFile := range migrations {
		content, err := os.ReadFile(migrationFile)
		if err != nil {
			log.Printf("⚠️ Warning: Could not read migration file %s: %v", migrationFile, err)
			continue
		}

		fmt.Println("Running auto-migration:", migrationFile)
		_, err = Pool.Exec(context.Background(), string(content))
		if err != nil {
			log.Printf("❌ Migration failed for %s: %v", migrationFile, err)
		} else {
			fmt.Println("✅ Migration applied successfully:", migrationFile)
		}
	}
}
