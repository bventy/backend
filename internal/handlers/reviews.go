package handlers

import (
	"fmt"
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

	ctx := c.Request.Context()

	// Strict Gating: Verify user has an accepted quote for a completed/past event
	// If req.QuoteID is provided, we check that specific one.
	// If not, we check if they have ANY such quote.

	eligibilityQuery := `
		SELECT qr.id
		FROM quote_requests qr
		JOIN events e ON qr.event_id = e.id
		WHERE qr.organizer_user_id::text = $1 
		  AND qr.vendor_id::text = $2 
		  AND qr.status = 'accepted'
		  AND (e.status = 'completed' OR e.event_date < NOW())
	`

	var validQuoteID string
	var err error
	if req.QuoteID != "" {
		eligibilityQuery += " AND qr.id = $3"
		err = db.Pool.QueryRow(ctx, eligibilityQuery, organizerID, vendorID, req.QuoteID).Scan(&validQuoteID)
	} else {
		err = db.Pool.QueryRow(ctx, eligibilityQuery, organizerID, vendorID).Scan(&validQuoteID)
	}

	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "You are not eligible to review this vendor. You must have an accepted quote for a completed event.",
		})
		return
	}

	// Insert review
	query := `
		INSERT INTO vendor_reviews (vendor_id, organizer_user_id, quote_id, rating, comment)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, created_at
	`

	var reviewID string
	var createdAt time.Time
	err = db.Pool.QueryRow(ctx, query, vendorID, organizerID, validQuoteID, req.Rating, req.Comment).Scan(&reviewID, &createdAt)
	if err != nil {
		fmt.Printf("ERROR: Failed to submit review for vendor %s by user %s: %v\n", vendorID, organizerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit review"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Review submitted successfully",
		"id":         reviewID,
		"created_at": createdAt,
	})
}

func (h *ReviewHandler) CheckEligibility(c *gin.Context) {
	userID, _ := c.Get("userID")
	organizerID := userID.(string)
	vendorID := c.Param("id")

	ctx := c.Request.Context()

	query := `
		SELECT EXISTS (
			SELECT 1
			FROM quote_requests qr
			JOIN events e ON qr.event_id = e.id
			WHERE qr.organizer_user_id::text = $1 
			  AND qr.vendor_id::text = $2 
			  AND qr.status = 'accepted'
			  AND (e.status = 'completed' OR e.event_date < NOW())
		)
	`

	var isEligible bool
	err := db.Pool.QueryRow(ctx, query, organizerID, vendorID).Scan(&isEligible)
	if err != nil {
		fmt.Printf("ERROR: Failed to check review eligibility for vendor %s by user %s: %v\n", vendorID, organizerID, err)
		c.JSON(http.StatusOK, gin.H{"eligible": false, "message": "Could not verify eligibility"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"eligible": isEligible,
		"message":  "Check eligibility completed",
	})
}

func (h *ReviewHandler) GetVendorReviews(c *gin.Context) {
	vendorID := c.Param("id")

	// We can also support slug if needed, but let's stick to ID for the specific route for now.
	// Actually, the frontend often has slug, so we should support looking up by slug too.

	ctx := c.Request.Context()

	query := `
		SELECT r.id, r.rating, r.comment, r.created_at, u.full_name as organizer_name, u.profile_image_url
		FROM vendor_reviews r
		JOIN users u ON r.organizer_user_id = u.id
		WHERE r.vendor_id::text = $1 OR r.vendor_id = (SELECT id FROM vendor_profiles WHERE slug = $1)
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
