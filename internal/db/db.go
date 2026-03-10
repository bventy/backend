package db

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatal("❌ DB config parsing failed:", err)
	}

	// Tune pool settings for performance and stability
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute

	Pool, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatal("❌ DB connection failed:", err)
	}

	err = Pool.Ping(context.Background())
	if err != nil {
		log.Fatal("❌ DB ping failed:", err)
	}

	fmt.Println("✅ Connected to PostgreSQL successfully (Pool sized)!")
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
	// Try a few common paths since CWD can vary between local/prod
	paths := []string{"internal/db/migrations", "../../internal/db/migrations", "./migrations"}
	var migrationsDir string
	var files []os.DirEntry

	for _, p := range paths {
		files, err = os.ReadDir(p)
		if err == nil {
			migrationsDir = p
			break
		}
	}

	if migrationsDir == "" {
		abs, _ := filepath.Abs(".")
		log.Printf("⚠️ WARNING: Could not find migrations directory in any of: %v (Current CWD: %s)", paths, abs)
		fmt.Println("⏭️ Skipping migrations due to missing directory.")
		return
	}

	// 3. Sort migrations (CRITICAL: os.ReadDir doesn't guarantee numerical order)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		version := file.Name()

		// SAFETY CHECK: If we are about to run 001_init.sql, ensure the DB is actually empty.
		// If it's not empty, we MUST NOT run init automatically as it might have destructive logic
		// (even though we just removed the DROPs, it's a good pattern).
		if version == "001_init.sql" {
			var tableCount int
			_ = Pool.QueryRow(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name != 'schema_migrations'").Scan(&tableCount)
			if tableCount > 0 {
				fmt.Printf("⚠️ SAFETY: Skipping %s because the database is not empty. Please mark it as applied manually if needed.\n", version)
				continue
			}
		}

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
			// If the error is about something already existing, we can log and continue
			// This is a safety net for legacy migrations that aren't fully idempotent.
			if strings.Contains(err.Error(), "already exists") {
				fmt.Printf("⚠️ Migration %s skipped/partially applied (already exists): %v\n", version, err)
				// We don't record it in schema_migrations here to stay safe,
				// but we also don't crash.
				continue
			}
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
