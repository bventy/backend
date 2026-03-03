package db

import (
	"context"
	"fmt"
	"log"
	"net/url"

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
	InitSchema() // Trigger one-time migration
}

func InitSchema() {
	query := `
    CREATE TABLE IF NOT EXISTS "email_logs" (
        "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        "to_email" TEXT NOT NULL,
        "subject" TEXT NOT NULL,
        "body_html" TEXT NOT NULL,
        "template_key" VARCHAR(50),
        "sent_at" TIMESTAMP WITH TIME ZONE DEFAULT now()
    );
    CREATE INDEX IF NOT EXISTS idx_email_logs_sent_at ON email_logs(sent_at);
    CREATE INDEX IF NOT EXISTS idx_email_logs_to_email ON email_logs(to_email);
    `
	_, err := Pool.Exec(context.Background(), query)
	if err != nil {
		log.Printf("⚠️ Temp migration failed: %v", err)
	} else {
		fmt.Println("✅ Temp migration (email_logs) successful!")
	}
}
