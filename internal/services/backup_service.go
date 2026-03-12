package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bventy/backend/internal/config"
)

type BackupService struct {
	MediaService *MediaService
	Config       *config.Config
}

func NewBackupService(cfg *config.Config, media *MediaService) *BackupService {
	return &BackupService{
		Config:       cfg,
		MediaService: media,
	}
}

func (s *BackupService) Start() {
	log.Println("🛡️  Backup Service started")
	
	// Check for backups every hour
	ticker := time.NewTicker(1 * time.Hour)
	
	// Run initial backup check
	go s.checkAndPerformBackups()

	for range ticker.C {
		s.checkAndPerformBackups()
	}
}

func (s *BackupService) checkAndPerformBackups() {
	now := time.Now().UTC()
	
	// Daily Full Backup (3:00 AM UTC)
	if now.Hour() == 3 {
		log.Println("📦 Starting daily full database backup...")
		if err := s.PerformBackup(false); err != nil {
			log.Printf("❌ Full backup failed: %v", err)
		} else {
			log.Println("✅ Full backup completed successfully")
		}
	}

	// Daily Schema Backup (3:00 PM UTC)
	if now.Hour() == 15 {
		log.Println("📦 Starting daily schema-only database backup...")
		if err := s.PerformBackup(true); err != nil {
			log.Printf("❌ Schema backup failed: %v", err)
		} else {
			log.Println("✅ Schema backup completed successfully")
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
	
	// Run pg_dump
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pg_dump failed: %w (output: %s)", err, string(output))
	}
	defer os.Remove(tmpPath)

	// Open the file for uploading
	file, err := os.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	// Upload to R2 in a secure directory
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
