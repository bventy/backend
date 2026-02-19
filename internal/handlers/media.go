package handlers

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/bventy/backend/internal/config"
	"github.com/bventy/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type MediaHandler struct {
	Service *services.MediaService
}

func NewMediaHandler(cfg *config.Config) *MediaHandler {
	svc, err := services.NewMediaService(cfg)
	if err != nil {
		// Log error but don't panic? Or panic if media service is critical?
		// For now, we'll just print/log it and maybe return nil, but `NewMediaService` error means config issue.
		// Let's panic to fail fast if config is bad, or just log.
		// Ideally routes.go handles the error. But here we are in NewMediaHandler.
		// Let's assume valid config for MVP or panic.
		panic("Failed to initialize MediaService: " + err.Error())
	}
	return &MediaHandler{Service: svc}
}

func (h *MediaHandler) Upload(c *gin.Context) {
	// Parse multipart form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request: No file provided"})
		return
	}
	defer file.Close()

	// Validate file type (simple check)
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	if !allowedExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type. Only JPG, PNG, WEBP allowed."})
		return
	}

	// Upload
	url, err := h.Service.UploadFile(file, header.Filename, header.Header.Get("Content-Type"), "uploads")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url": url,
	})
}
