package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bventy/backend/internal/config"
	"github.com/bventy/backend/internal/middleware"
)

type BackupService struct {
	MediaService *MediaService
	Config       *config.Config
	
	lastFullBackup   time.Time
	lastSchemaBackup time.Time
	mu              sync.Mutex
}

func NewBackupService(cfg *config.Config, media *MediaService) *BackupService {
	return &BackupService{
		Config:       cfg,
		MediaService: media,
	}
}

func (s *BackupService) Start() {
	log.Println("🛡️  Smart Backup Service started (Idle-aware)")
	
	// Check every 15 minutes
	ticker := time.NewTicker(15 * time.Minute)
	
	// Startup delay: don't check immediately to avoid interfering with deployment health checks
	log.Println("⏳ Backup Service: Delaying initial check for 10 minutes to protect startup integrity")
	time.Sleep(10 * time.Minute)
	
	// Initial check after delay
	go s.checkAndPerformBackups()

	for range ticker.C {
		s.checkAndPerformBackups()
	}
}

func (s *BackupService) alreadyDoneToday(last time.Time, now time.Time) bool {
	return last.Year() == now.Year() && last.Month() == now.Month() && last.Day() == now.Day()
}

func (s *BackupService) checkAndPerformBackups() {
	now := time.Now().UTC()
	lastActivity := middleware.GetLastActivity()
	inactivityDuration := time.Since(lastActivity)
	isIdle := inactivityDuration >= 10*time.Minute
	
	s.mu.Lock()
	defer s.mu.Unlock()

	// --- Full Backup Logic ---
	// Window: 00:00 - 06:00 UTC
	if now.Hour() >= 0 && now.Hour() < 6 && !s.alreadyDoneToday(s.lastFullBackup, now) {
		// Force if: 1. Idle OR 2. Last 30 mins of the window (safety catch)
		force := isIdle || (now.Hour() == 5 && now.Minute() >= 30)
		
		if force {
			reason := "idle state detected"
			if !isIdle {
				reason = "closing window safety (forced)"
			}
			log.Printf("📦 Triggering full database backup (%s)...", reason)
			
			if err := s.PerformBackup(false); err != nil {
				log.Printf("❌ Full backup failed: %v", err)
			} else {
				s.lastFullBackup = now
				log.Println("✅ Full backup completed")
			}
		} else {
			log.Printf("⏳ Full backup window open, but server is busy (%s since last activity). Waiting for idle...", inactivityDuration.Round(time.Second))
		}
	}

	// --- Schema Backup Logic ---
	// Window: 12:00 - 18:00 UTC
	if now.Hour() >= 12 && now.Hour() < 18 && !s.alreadyDoneToday(s.lastSchemaBackup, now) {
		force := isIdle || (now.Hour() == 17 && now.Minute() >= 30)
		
		if force {
			reason := "idle state detected"
			if !isIdle {
				reason = "closing window safety (forced)"
			}
			log.Printf("📦 Triggering schema-only backup (%s)...", reason)
			
			if err := s.PerformBackup(true); err != nil {
				log.Printf("❌ Schema backup failed: %v", err)
			} else {
				s.lastSchemaBackup = now
				log.Println("✅ Schema backup completed")
			}
		} else {
			log.Printf("⏳ Schema backup window open, but server is busy (%s since last activity). Waiting for idle...", inactivityDuration.Round(time.Second))
		}
	}
}

func (s *BackupService) PerformBackup(schemaOnly bool) error {
	timestamp := time.Now().Format("20060102_1504")
	prefix := "full"
	if schemaOnly {
		prefix = "schema"
	}
	
	filename := fmt.Sprintf("backup_%s_%s.sql", prefix, timestamp)
	tmpPath := filepath.Join(os.TempDir(), filename)
	
	// pg_dump command
	args := []string{"-d", s.Config.DatabaseURL, "-f", tmpPath}
	if schemaOnly {
		args = append(args, "--schema-only")
	}

	cmd := exec.Command("pg_dump", args...)
	
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pg_dump failed: %w (output: %s)", err, string(output))
	}
	defer os.Remove(tmpPath)

	file, err := os.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	r2Key := fmt.Sprintf("internal_backups/%s", filename)
	
	_, err = s.MediaService.Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(s.MediaService.Bucket),
		Key:         aws.String(r2Key),
		Body:        file,
		ContentType: aws.String("application/sql"),
	})
	
	if err != nil {
		return fmt.Errorf("failed to upload backup to R2: %w", err)
	}

	return nil
}
