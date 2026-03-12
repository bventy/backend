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
	
	// Check more frequently for idle state
	ticker := time.NewTicker(15 * time.Minute)
	
	// Initial check
	go s.checkAndPerformBackups()

	for range ticker.C {
		s.checkAndPerformBackups()
	}
}

func (s *BackupService) checkAndPerformBackups() {
	now := time.Now().UTC()
	lastActivity := middleware.GetLastActivity()
	isIdle := time.Since(lastActivity) > 10*time.Minute
	
	s.mu.Lock()
	defer s.mu.Unlock()

	// --- Full Backup Logic ---
	// Window: 00:00 - 06:00 UTC
	inFullWindow := now.Hour() >= 0 && now.Hour() < 6
	alreadyBackupFullToday := s.lastFullBackup.Year() == now.Year() && 
		s.lastFullBackup.Month() == now.Month() && 
		s.lastFullBackup.Day() == now.Day()

	if inFullWindow && !alreadyBackupFullToday {
		// Trigger if idle OR if we're in the last hour of the window (safety catch)
		if isIdle || now.Hour() == 5 {
			reason := "idle state detected"
			if !isIdle {
				reason = "closing window safety catch"
			}
			log.Printf("📦 Triggering smart full backup (%s)...", reason)
			
			if err := s.PerformBackup(false); err != nil {
				log.Printf("❌ Smart full backup failed: %v", err)
			} else {
				s.lastFullBackup = now
				log.Println("✅ Smart full backup completed")
			}
		}
	}

	// --- Schema Backup Logic ---
	// Window: 12:00 - 18:00 UTC
	inSchemaWindow := now.Hour() >= 12 && now.Hour() < 18
	alreadyBackupSchemaToday := s.lastSchemaBackup.Year() == now.Year() && 
		s.lastSchemaBackup.Month() == now.Month() && 
		s.lastSchemaBackup.Day() == now.Day()

	if inSchemaWindow && !alreadyBackupSchemaToday {
		// Trigger if idle OR if we're in the last hour of the window
		if isIdle || now.Hour() == 17 {
			reason := "idle state detected"
			if !isIdle {
				reason = "closing window safety catch"
			}
			log.Printf("📦 Triggering smart schema backup (%s)...", reason)
			
			if err := s.PerformBackup(true); err != nil {
				log.Printf("❌ Smart schema backup failed: %v", err)
			} else {
				s.lastSchemaBackup = now
				log.Println("✅ Smart schema backup completed")
			}
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
