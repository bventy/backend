package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/bventy/backend/internal/auth"
	"github.com/bventy/backend/internal/config"
	"github.com/bventy/backend/internal/db"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	Config *config.Config
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{Config: cfg}
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
	err = db.Pool.QueryRow(context.Background(), query, 
		req.Email, 
		string(hashedPassword), 
		req.FullName, 
		usernameArg, 
		req.Phone,
	).Scan(&userID)

	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User already exists or valid constraint failed"})
		return
	}

	token, err := auth.GenerateToken(userID, "user", h.Config)
	if err != nil {
		c.JSON(http.StatusCreated, gin.H{"message": "User created, please login", "user_id": userID})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully",
		"token":   token,
		"user": gin.H{
			"id":        userID,
			"email":     req.Email,
			"full_name": req.FullName,
			"role":      "user",
		},
	})
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
	query := `SELECT id, role, password_hash, full_name FROM users WHERE email = $1`
	err := db.Pool.QueryRow(context.Background(), query, req.Email).Scan(&userID, &role, &passwordHash, &fullName)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := auth.GenerateToken(userID, role, h.Config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":     token,
		"role":      role,
		"user_id":   userID,
		"full_name": fullName,
	})
}
