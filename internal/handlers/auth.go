package handlers

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"time"

	"github.com/bventy/backend/internal/auth"
	"github.com/bventy/backend/internal/config"
	"github.com/bventy/backend/internal/db"
	"github.com/bventy/backend/internal/services"
	"github.com/gin-gonic/gin"
	pgx "github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	Config       *config.Config
	EmailService *services.EmailService
}

func NewAuthHandler(cfg *config.Config, emailService *services.EmailService) *AuthHandler {
	return &AuthHandler{
		Config:       cfg,
		EmailService: emailService,
	}
}

type SignupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"full_name" binding:"required"`
	Username string `json:"username"`
	Phone    string `json:"phone"`
}

func (h *AuthHandler) Signup(c *gin.Context) {
	var req SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	var userID string
	// Handle empty username as NULL
	var usernameArg interface{} = req.Username
	if req.Username == "" {
		usernameArg = nil
	}

	// role defaults to 'user' in DB
	query := `
		INSERT INTO users (email, password_hash, full_name, username, phone)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	err = db.Pool.QueryRow(c.Request.Context(), query,
		req.Email,
		string(hashedPassword),
		req.FullName,
		usernameArg,
		req.Phone,
	).Scan(&userID)

	if err != nil {
		fmt.Printf("ERROR: Failed to signup user %s: %v\n", req.Email, err)
		c.JSON(http.StatusConflict, gin.H{"error": "User already exists or valid constraint failed"})
		return
	}

	// Step 2: Generate OTP
	otpCode := generateOTP()
	expiresAt := time.Now().Add(10 * time.Minute)

	// Step 3: Rate limit check (1 OTP per 60s)
	var latestCreatedAt time.Time
	err = db.Pool.QueryRow(c.Request.Context(), "SELECT created_at FROM email_otps WHERE email = $1 AND purpose = 'verify' ORDER BY created_at DESC LIMIT 1", req.Email).Scan(&latestCreatedAt)
	if err == nil && time.Since(latestCreatedAt) < 60*time.Second {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Please wait 60 seconds before requesting a new code."})
		return
	}

	otpQuery := `INSERT INTO email_otps (user_id, email, code, purpose, expires_at) VALUES ($1, $2, $3, 'verify', $4)`
	_, err = db.Pool.Exec(c.Request.Context(), otpQuery, userID, req.Email, otpCode, expiresAt)
	if err != nil {
		fmt.Printf("ERROR: Failed to store OTP for user %s: %v\n", req.Email, err)
		// We don't fail signup if OTP storage fails, but it's not ideal.
	} else {
		// Step 4: Send Verification Email
		go func() {
			err := h.EmailService.SendVerificationEmail(req.Email, otpCode)
			if err != nil {
				fmt.Printf("ERROR: Failed to send verification email to %s: %v\n", req.Email, err)
			}
		}()
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully. Please verify your email.",
		"user": gin.H{
			"id":        userID,
			"email":     req.Email,
			"full_name": req.FullName,
			"role":      "user",
		},
	})
}

func generateOTP() string {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		// Fallback to a less secure but functional method if crypto/rand fails
		return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	return fmt.Sprintf("%06d", n.Int64())
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userID, role, passwordHash, fullName string
	var emailVerified bool
	query := "SELECT id, role, password_hash, full_name, email_verified FROM users WHERE email = $1"
	err := db.Pool.QueryRow(c.Request.Context(), query, req.Email).Scan(&userID, &role, &passwordHash, &fullName, &emailVerified)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "No account found with this email address"})
			return
		}
		fmt.Printf("ERROR: Failed to login user %s: %v\n", req.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error. Please try again later"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect password. Please check and try again"})
		return
	}

	token, err := auth.GenerateToken(userID, role, h.Config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"bventy_session",
		token,
		3600*24*7, // 7 days
		"/",
		h.Config.CookieDomain,
		h.Config.CookieSecure,
		true,
	)

	c.JSON(http.StatusOK, gin.H{
		"token":          token,
		"role":           role,
		"user_id":        userID,
		"full_name":      fullName,
		"email_verified": emailVerified,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"bventy_session",
		"",
		-1, // Expire immediately
		"/",
		h.Config.CookieDomain,
		h.Config.CookieSecure,
		true,
	)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

type VerifyEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required,len=6"`
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var otpID, userID, role string
	var attempts int
	var expiresAt time.Time

	otpQuery := `
		SELECT o.id, o.user_id, o.attempts, o.expires_at, u.role 
		FROM email_otps o 
		JOIN users u ON o.user_id = u.id 
		WHERE o.email = $1 AND o.code = $2 AND o.purpose = 'verify'
	`
	err := db.Pool.QueryRow(c.Request.Context(), otpQuery, req.Email, req.OTP).Scan(&otpID, &userID, &attempts, &expiresAt, &role)
	if err != nil {
		// Increment attempts for that email if it exists
		_, _ = db.Pool.Exec(c.Request.Context(), "UPDATE email_otps SET attempts = attempts + 1 WHERE email = $1 AND purpose = 'verify'", req.Email)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid verification code."})
		return
	}

	if attempts >= 5 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Too many failed attempts. Please request a new code."})
		return
	}

	if time.Now().After(expiresAt) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Verification code has expired."})
		return
	}

	// Update user and delete OTP
	tx, err := db.Pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback(c.Request.Context())

	_, err = tx.Exec(c.Request.Context(), "UPDATE users SET email_verified = true, email_verified_at = NOW(), email_verification_attempts = 0 WHERE id = $1", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user status"})
		return
	}

	_, err = tx.Exec(c.Request.Context(), "DELETE FROM email_otps WHERE email = $1 AND purpose = 'verify'", req.Email)
	if err != nil {
		log.Printf("Warning: failed to delete used OTP: %v", err)
	}

	if err := tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize verification"})
		return
	}

	// NEW: Auto-login after successful verification
	token, err := auth.GenerateToken(userID, role, h.Config)
	if err == nil {
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(
			"bventy_session",
			token,
			3600*24*7, // 7 days
			"/",
			h.Config.CookieDomain,
			h.Config.CookieSecure,
			true,
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Email verified successfully!",
		"token":   token,
		"role":    role,
	})
}

type RequestResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func (h *AuthHandler) RequestReset(c *gin.Context) {
	var req RequestResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userID string
	err := db.Pool.QueryRow(c.Request.Context(), "SELECT id FROM users WHERE email = $1", req.Email).Scan(&userID)
	if err != nil {
		// We return success even if email not found to prevent user enumeration
		c.JSON(http.StatusOK, gin.H{"message": "If an account exists with this email, a reset code has been sent."})
		return
	}

	// Rate limit check (1 OTP per 60s)
	var latestCreatedAt time.Time
	err = db.Pool.QueryRow(c.Request.Context(), "SELECT created_at FROM email_otps WHERE email = $1 AND purpose = 'reset' ORDER BY created_at DESC LIMIT 1", req.Email).Scan(&latestCreatedAt)
	if err == nil && time.Since(latestCreatedAt) < 60*time.Second {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Please wait 60 seconds before requesting a new code."})
		return
	}

	otpCode := generateOTP()
	expiresAt := time.Now().Add(15 * time.Minute)

	_, err = db.Pool.Exec(c.Request.Context(), "INSERT INTO email_otps (user_id, email, code, purpose, expires_at) VALUES ($1, $2, $3, 'reset', $4)", userID, req.Email, otpCode, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request"})
		return
	}

	go func() {
		_ = h.EmailService.SendResetEmail(req.Email, otpCode)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "If an account exists with this email, a reset code has been sent."})
}

type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	OTP         string `json:"otp" binding:"required,len=6"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userID string
	var expiresAt time.Time
	err := db.Pool.QueryRow(c.Request.Context(), "SELECT user_id, expires_at FROM email_otps WHERE email = $1 AND code = $2 AND purpose = 'reset'", req.Email, req.OTP).Scan(&userID, &expiresAt)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired reset code."})
		return
	}

	if time.Now().After(expiresAt) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Reset code has expired."})
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)

	tx, _ := db.Pool.Begin(c.Request.Context())
	defer tx.Rollback(c.Request.Context())

	_, err = tx.Exec(c.Request.Context(), "UPDATE users SET password_hash = $1 WHERE id = $2", string(hashedPassword), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	_, _ = tx.Exec(c.Request.Context(), "DELETE FROM email_otps WHERE email = $1 AND purpose = 'reset'", req.Email)

	_ = tx.Commit(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully!"})
}

func (h *AuthHandler) ResendVerification(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userID string
	var verified bool
	var attempts int
	var latestCreatedAt *time.Time

	// Fetch user status and latest OTP in one query
	query := `
		SELECT u.id, u.email_verified, u.email_verification_attempts,
		(SELECT created_at FROM email_otps WHERE email = $1 AND purpose = 'verify' ORDER BY created_at DESC LIMIT 1)
		FROM users u WHERE u.email = $1
	`
	err := db.Pool.QueryRow(c.Request.Context(), query, req.Email).Scan(&userID, &verified, &attempts, &latestCreatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if verified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email already verified"})
		return
	}

	// Smart Rate Limiting (Exponential Backoff)
	// 0 resends (after signup): 2m wait
	// 1 resend: 5m wait
	// 2 resends: 15m wait
	// 3+ resends: 60m wait
	var waitTime time.Duration
	switch attempts {
	case 0:
		waitTime = 2 * time.Minute
	case 1:
		waitTime = 5 * time.Minute
	case 2:
		waitTime = 15 * time.Minute
	default:
		waitTime = 60 * time.Minute
	}

	// Daily Limit Check (Max 10 per 24h)
	if attempts >= 10 {
		// If last attempt was > 24h ago, we can reset the counter
		if latestCreatedAt != nil && time.Since(*latestCreatedAt) > 24*time.Hour {
			_, _ = db.Pool.Exec(c.Request.Context(), "UPDATE users SET email_verification_attempts = 0 WHERE id = $1", userID)
			attempts = 0
			waitTime = 2 * time.Minute // Reset wait time too
		} else {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Daily verification limit reached. Please try again tomorrow."})
			return
		}
	}

	if latestCreatedAt != nil && time.Since(*latestCreatedAt) < waitTime {
		remaining := waitTime - time.Since(*latestCreatedAt)
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":       fmt.Sprintf("Please wait %d seconds before requesting a new code.", int(remaining.Seconds())),
			"retry_after": int(remaining.Seconds()),
		})
		return
	}

	otpCode := generateOTP()
	expiresAt := time.Now().Add(10 * time.Minute)

	// Update OTP and increment user's attempt counter
	tx, err := db.Pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request"})
		return
	}
	defer tx.Rollback(c.Request.Context())

	// Delete old OTPs for this purpose
	_, _ = tx.Exec(c.Request.Context(), "DELETE FROM email_otps WHERE email = $1 AND purpose = 'verify'", req.Email)

	// Insert new OTP
	_, err = tx.Exec(c.Request.Context(), "INSERT INTO email_otps (user_id, email, code, purpose, expires_at) VALUES ($1, $2, $3, 'verify', $4)", userID, req.Email, otpCode, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate code"})
		return
	}

	// Increment user's verification attempts
	_, err = tx.Exec(c.Request.Context(), "UPDATE users SET email_verification_attempts = email_verification_attempts + 1 WHERE id = $1", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update attempt counter"})
		return
	}

	if err := tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize request"})
		return
	}

	go func() {
		_ = h.EmailService.SendVerificationEmail(req.Email, otpCode)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "Verification code resent successfully!"})
}
