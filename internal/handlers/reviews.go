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
		INSERT INTO vendor_reviews (vendor_id, organizer_user_id, quote_id, rating, comment, is_public, helpful_count)
		VALUES ($1, $2, $3, $4, $5, true, 0)
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
	ratingFilter := c.Query("rating")
	hasReply := c.Query("has_reply")
	sort := c.Query("sort") // newest, oldest, highest, lowest

	ctx := c.Request.Context()

	query := `
		SELECT 
			r.id, r.rating, r.comment, r.created_at, 
			u.full_name as organizer_name, u.profile_image_url,
			r.reply_text, r.replied_at, 
			COALESCE(r.is_public, true), 
			COALESCE(r.helpful_count, 0)
		FROM vendor_reviews r
		JOIN users u ON r.organizer_user_id = u.id
		WHERE (r.vendor_id::text = $1 OR r.vendor_id = (SELECT id FROM vendor_profiles WHERE slug = $1))
	`

	args := []interface{}{vendorID}
	argCount := 1

	if ratingFilter != "" {
		argCount++
		query += fmt.Sprintf(" AND r.rating = $%d", argCount)
		args = append(args, ratingFilter)
	}

	if hasReply == "true" {
		query += " AND r.reply_text IS NOT NULL"
	} else if hasReply == "false" {
		query += " AND r.reply_text IS NULL"
	}

	// Always show public reviews. Private ones might be filtered later for vendor dashboard.
	query += " AND COALESCE(r.is_public, true) = true"

	// Sorting
	switch sort {
	case "oldest":
		query += " ORDER BY r.created_at ASC"
	case "highest":
		query += " ORDER BY r.rating DESC, r.created_at DESC"
	case "lowest":
		query += " ORDER BY r.rating ASC, r.created_at DESC"
	default: // newest
		query += " ORDER BY r.created_at DESC"
	}

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		fmt.Printf("ERROR: Failed to fetch reviews for %s: %v\n", vendorID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reviews"})
		return
	}
	defer rows.Close()

	var reviews []gin.H
	for rows.Next() {
		var id, organizerName string
		var rating, helpfulCount int
		var comment, replyText, profileImage *string
		var createdAt, repliedAt *time.Time
		var isPublic bool

		if err := rows.Scan(&id, &rating, &comment, &createdAt, &organizerName, &profileImage, &replyText, &repliedAt, &isPublic, &helpfulCount); err != nil {
			fmt.Printf("ERROR: Scan failed: %v\n", err)
			continue
		}

		reviews = append(reviews, gin.H{
			"id":             id,
			"rating":         rating,
			"comment":        comment,
			"created_at":     createdAt,
			"organizer_name": organizerName,
			"profile_image":  profileImage,
			"reply_text":     replyText,
			"replied_at":     repliedAt,
			"helpful_count":  helpfulCount,
			"is_public":      isPublic,
		})
	}

	if reviews == nil {
		reviews = []gin.H{}
	}

	c.JSON(http.StatusOK, reviews)
}
func (h *ReviewHandler) LikeReview(c *gin.Context) {
	reviewID := c.Param("id")
	ctx := c.Request.Context()

	_, err := db.Pool.Exec(ctx, "UPDATE vendor_reviews SET helpful_count = helpful_count + 1 WHERE id::text = $1", reviewID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to like review"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Liked review"})
}

func (h *ReviewHandler) ReplyToReview(c *gin.Context) {
	userID, _ := c.Get("userID")
	reviewID := c.Param("id")

	var req struct {
		ReplyText string `json:"reply_text" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Verify the user owns the vendor associated with this review
	var vendorID string
	verifyQuery := `
		SELECT r.vendor_id 
		FROM vendor_reviews r
		JOIN vendor_profiles vp ON r.vendor_id = vp.id
		WHERE r.id::text = $1 AND vp.owner_user_id::text = $2
	`
	err := db.Pool.QueryRow(ctx, verifyQuery, reviewID, userID).Scan(&vendorID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to reply to this review."})
		return
	}

	// Update the review with the reply
	updateQuery := `
		UPDATE vendor_reviews 
		SET reply_text = $1, replied_at = NOW(), updated_at = NOW()
		WHERE id::text = $2
	`
	_, err = db.Pool.Exec(ctx, updateQuery, req.ReplyText, reviewID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save reply"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reply saved successfully"})
}
