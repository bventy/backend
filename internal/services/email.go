package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bventy/backend/internal/db"
	"github.com/resend/resend-go/v2"
)

type EmailService struct {
	client *resend.Client
}

func NewEmailService(apiKey string) *EmailService {
	client := resend.NewClient(apiKey)
	return &EmailService{client: client}
}

func (s *EmailService) sendEmail(to string, templateKey string, variables map[string]string) error {
	ctx := context.Background()

	// 1. Fetch template from DB
	var subject, bodyHTML string
	var isEnabled bool
	query := `SELECT subject, body_html, is_enabled FROM email_templates WHERE template_key = $1`
	err := db.Pool.QueryRow(ctx, query, templateKey).Scan(&subject, &bodyHTML, &isEnabled)
	if err != nil {
		return fmt.Errorf("failed to load template %s: %v", templateKey, err)
	}

	if !isEnabled {
		log.Printf("Template %s is disabled, skipping send", templateKey)
		return nil
	}

	// 2. Replace variables
	for k, v := range variables {
		placeholder := fmt.Sprintf("{{%s}}", k)
		subject = strings.ReplaceAll(subject, placeholder, v)
		bodyHTML = strings.ReplaceAll(bodyHTML, placeholder, v)
	}

	// 3. Send via Resend
	params := &resend.SendEmailRequest{
		From:    "Bventy <no-reply@bventy.in>", // User said domain is verified
		To:      []string{to},
		Subject: subject,
		Html:    bodyHTML,
	}

	_, err = s.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send email via Resend: %v", err)
	}

	return nil
}

func (s *EmailService) SendVerificationEmail(to string, code string) error {
	return s.sendEmail(to, "verify_email", map[string]string{"code": code})
}

func (s *EmailService) SendResetEmail(to string, code string) error {
	return s.sendEmail(to, "reset_password", map[string]string{"code": code})
}

func (s *EmailService) SendQuoteNotification(to string, templateKey string, vars map[string]string) error {
	// 1. Check global setting
	var notificationsEnabled string
	err := db.Pool.QueryRow(context.Background(), "SELECT value FROM platform_settings WHERE key = 'quote_email_notifications_enabled'").Scan(&notificationsEnabled)
	if err != nil {
		log.Printf("Warning: failed to check notification settings: %v", err)
		notificationsEnabled = "true" // fallback
	}

	if notificationsEnabled != "true" {
		log.Printf("Quote email notifications are globally disabled")
		return nil
	}

	return s.sendEmail(to, templateKey, vars)
}
