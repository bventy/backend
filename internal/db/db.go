package db

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

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
}

func RunMigrations() {
	ctx := context.Background()

	// 1. Create migrations tracking table if not exists
	_, err := Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			version TEXT UNIQUE NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatalf("❌ Failed to create migrations table: %v", err)
	}

	// 2. Read migrations directory
	migrationsDir := "internal/db/migrations"
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		log.Fatalf("❌ Failed to read migrations directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		version := file.Name()

		// Check if migration already applied
		var exists bool
		err := Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&exists)
		if err != nil {
			log.Fatalf("❌ Failed to check migration status for %s: %v", version, err)
		}

		if exists {
			continue
		}

		// Read and execute migration
		fmt.Printf("🚀 Applying migration: %s...\n", version)
		path := filepath.Join(migrationsDir, version)
		content, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("❌ Failed to read migration file %s: %v", version, err)
		}

		tx, err := Pool.Begin(ctx)
		if err != nil {
			log.Fatalf("❌ Failed to start transaction for %s: %v", version, err)
		}

		if _, err := tx.Exec(ctx, string(content)); err != nil {
			tx.Rollback(ctx)
			log.Fatalf("❌ Migration failed for %s: %v", version, err)
		}

		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			tx.Rollback(ctx)
			log.Fatalf("❌ Failed to record migration %s: %v", version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			log.Fatalf("❌ Failed to commit migration %s: %v", version, err)
		}
		fmt.Printf("✅ Applied %s\n", version)
	}

	fmt.Println("✨ All migrations are up to date.")
}
