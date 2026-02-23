package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/bventy/backend/internal/db"
	"github.com/gin-gonic/gin"
)

type ReviewHandler struct{}

func NewReviewHandler() *ReviewHandler {
	return &ReviewHandler{}
}

type CreateReviewRequest struct {
	Rating  int    `json:"rating" binding:"required,min=1,max=5"`
	Comment string `json:"comment"`
	QuoteID string `json:"quote_id"` // Optional: link to a quote/hire
}

func (h *ReviewHandler) CreateReview(c *gin.Context) {
	userID, _ := c.Get("userID")
	organizerID := userID.(string)
	vendorID := c.Param("id")

	var req CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Optional: Check if the user is an organizer?
	// Our middleware usually handles role checks if we apply handlers.OrganizerOnly()
	// but let's just ensure they aren't reviewing themselves if they are also a vendor.

	ctx := context.Background()

	// Verify vendor exists
	var exists bool
	err := db.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM vendor_profiles WHERE id = $1)", vendorID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vendor not found"})
		return
	}

	// Insert review
	query := `
		INSERT INTO vendor_reviews (vendor_id, organizer_user_id, quote_id, rating, comment)
		VALUES ($1, $2, NULLIF($3, '')::UUID, $4, $5)
		RETURNING id, created_at
	`

	var reviewID string
	var createdAt time.Time
	err = db.Pool.QueryRow(ctx, query, vendorID, organizerID, req.QuoteID, req.Rating, req.Comment).Scan(&reviewID, &createdAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit review: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Review submitted successfully",
		"id":         reviewID,
		"created_at": createdAt,
	})
}

func (h *ReviewHandler) GetVendorReviews(c *gin.Context) {
	vendorID := c.Param("id")

	// We can also support slug if needed, but let's stick to ID for the specific route for now.
	// Actually, the frontend often has slug, so we should support looking up by slug too.

	ctx := context.Background()

	query := `
		SELECT r.id, r.rating, r.comment, r.created_at, u.full_name as organizer_name, u.profile_image_url
		FROM vendor_reviews r
		JOIN users u ON r.organizer_user_id = u.id
		WHERE r.vendor_id = $1 OR r.vendor_id = (SELECT id FROM vendor_profiles WHERE slug = $1)
		ORDER BY r.created_at DESC
	`

	rows, err := db.Pool.Query(ctx, query, vendorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reviews"})
		return
	}
	defer rows.Close()

	var reviews []gin.H
	for rows.Next() {
		var id, organizerName string
		var rating int
		var comment *string
		var createdAt time.Time
		var profileImage *string

		if err := rows.Scan(&id, &rating, &comment, &createdAt, &organizerName, &profileImage); err != nil {
			continue
		}

		reviews = append(reviews, gin.H{
			"id":             id,
			"rating":         rating,
			"comment":        comment,
			"created_at":     createdAt,
			"organizer_name": organizerName,
			"profile_image":  profileImage,
		})
	}

	if reviews == nil {
		reviews = []gin.H{}
	}

	c.JSON(http.StatusOK, reviews)
}
