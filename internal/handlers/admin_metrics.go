package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/bventy/backend/internal/db"
	"github.com/gin-gonic/gin"
)

type AdminMetricsHandler struct{}

func NewAdminMetricsHandler() *AdminMetricsHandler {
	return &AdminMetricsHandler{}
}

// 1. Overview Endpoint
func (h *AdminMetricsHandler) GetAdminMetricsOverview(c *gin.Context) {
	var totalUsers, totalGroups, totalEvents, publishedEvents, completedEvents int
	var totalVendors, verifiedVendors, pendingVendors int

	ctx := context.Background()

	// Users
	db.Pool.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&totalUsers)

	// Groups
	db.Pool.QueryRow(ctx, "SELECT count(*) FROM groups").Scan(&totalGroups)

	// Vendors
	db.Pool.QueryRow(ctx, "SELECT count(*) FROM vendor_profiles").Scan(&totalVendors)
	db.Pool.QueryRow(ctx, "SELECT count(*) FROM vendor_profiles WHERE status = 'verified'").Scan(&verifiedVendors)
	db.Pool.QueryRow(ctx, "SELECT count(*) FROM vendor_profiles WHERE status = 'pending'").Scan(&pendingVendors)

	// Events
	db.Pool.QueryRow(ctx, "SELECT count(*) FROM events").Scan(&totalEvents)
	// Completed events (date is in the past)
	db.Pool.QueryRow(ctx, "SELECT count(*) FROM events WHERE event_date < CURRENT_DATE").Scan(&completedEvents)
	// Published events (upcoming/today)
	db.Pool.QueryRow(ctx, "SELECT count(*) FROM events WHERE event_date >= CURRENT_DATE").Scan(&publishedEvents)

	c.JSON(http.StatusOK, gin.H{
		"total_users":      totalUsers,
		"total_vendors":    totalVendors,
		"verified_vendors": verifiedVendors,
		"pending_vendors":  pendingVendors,
		"total_events":     totalEvents,
		"published_events": publishedEvents,
		"completed_events": completedEvents,
		"total_groups":     totalGroups,
	})
}

// 2. Growth Endpoint
func (h *AdminMetricsHandler) GetAdminMetricsGrowth(c *gin.Context) {
	ctx := context.Background()

	// Get dates for the last 30 days
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	type dailyStat struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	}

	fetchGrowthData := func(query string, args ...interface{}) []dailyStat {
		rows, err := db.Pool.Query(ctx, query, args...)
		if err != nil {
			return []dailyStat{}
		}
		defer rows.Close()

		var stats []dailyStat
		for rows.Next() {
			var date time.Time
			var count int
			if err := rows.Scan(&date, &count); err == nil {
				stats = append(stats, dailyStat{Date: date.Format("2006-01-02"), Count: count})
			}
		}
		if stats == nil {
			stats = []dailyStat{}
		}
		return stats
	}

	userSignupsQuery := `
		SELECT DATE(created_at) as date, count(*) as count
		FROM users
		WHERE created_at >= $1
		GROUP BY DATE(created_at)
		ORDER BY DATE(created_at) ASC
	`
	userSignups := fetchGrowthData(userSignupsQuery, thirtyDaysAgo)

	vendorSignupsQuery := `
		SELECT DATE(created_at) as date, count(*) as count
		FROM vendor_profiles
		WHERE created_at >= $1
		GROUP BY DATE(created_at)
		ORDER BY DATE(created_at) ASC
	`
	vendorSignups := fetchGrowthData(vendorSignupsQuery, thirtyDaysAgo)

	eventsCreatedQuery := `
		SELECT DATE(created_at) as date, count(*) as count
		FROM events
		WHERE created_at >= $1
		GROUP BY DATE(created_at)
		ORDER BY DATE(created_at) ASC
	`
	eventsCreated := fetchGrowthData(eventsCreatedQuery, thirtyDaysAgo)

	c.JSON(http.StatusOK, gin.H{
		"user_signups_by_day":   userSignups,
		"vendor_signups_by_day": vendorSignups,
		"events_created_by_day": eventsCreated,
	})
}

// 3. Events Endpoint
func (h *AdminMetricsHandler) GetAdminMetricsEvents(c *gin.Context) {
	ctx := context.Background()

	// Events by status (Upcoming vs Completed)
	var eventsUpcoming, eventsCompleted int
	db.Pool.QueryRow(ctx, "SELECT count(*) FROM events WHERE event_date >= CURRENT_DATE").Scan(&eventsUpcoming)
	db.Pool.QueryRow(ctx, "SELECT count(*) FROM events WHERE event_date < CURRENT_DATE").Scan(&eventsCompleted)

	eventsByStatusList := []gin.H{
		{"status": "Upcoming", "count": eventsUpcoming},
		{"status": "Completed", "count": eventsCompleted},
	}

	// Events by city
	eventsByCityQuery := `
		SELECT city, count(*) as count
		FROM events
		GROUP BY city
		ORDER BY count DESC
	`
	rows, _ := db.Pool.Query(ctx, eventsByCityQuery)
	var eventsByCity []gin.H
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var city string
			var count int
			if err := rows.Scan(&city, &count); err == nil {
				eventsByCity = append(eventsByCity, gin.H{"city": city, "count": count})
			}
		}
	}
	if eventsByCity == nil {
		eventsByCity = []gin.H{}
	}

	// Average Budgets
	var avgBudgetMin, avgBudgetMax float64
	db.Pool.QueryRow(ctx, "SELECT COALESCE(AVG(budget_min), 0) FROM events").Scan(&avgBudgetMin)
	db.Pool.QueryRow(ctx, "SELECT COALESCE(AVG(budget_max), 0) FROM events").Scan(&avgBudgetMax)

	c.JSON(http.StatusOK, gin.H{
		"events_by_status":   eventsByStatusList,
		"events_by_city":     eventsByCity,
		"average_budget_min": avgBudgetMin,
		"average_budget_max": avgBudgetMax,
	})
}

// 4. Vendors Endpoint
func (h *AdminMetricsHandler) GetAdminMetricsVendors(c *gin.Context) {
	ctx := context.Background()

	// Most Shortlisted Vendors
	mostShortlistsQuery := `
		SELECT vp.id, vp.business_name, vp.city, vp.category, COUNT(esv.event_id) as shortlist_count
		FROM vendor_profiles vp
		JOIN event_shortlisted_vendors esv ON vp.id = esv.vendor_id
		GROUP BY vp.id, vp.business_name, vp.city, vp.category
		ORDER BY shortlist_count DESC
		LIMIT 10
	`

	rows, _ := db.Pool.Query(ctx, mostShortlistsQuery)
	var mostShortlisted []gin.H
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var id, name, city, category string
			var count int
			if err := rows.Scan(&id, &name, &city, &category, &count); err == nil {
				mostShortlisted = append(mostShortlisted, gin.H{
					"vendor_id":       id,
					"business_name":   name,
					"city":            city,
					"category":        category,
					"shortlist_count": count,
				})
			}
		}
	}
	if mostShortlisted == nil {
		mostShortlisted = []gin.H{}
	}

	// Inactive Vendors (Pending for > 30 days)
	inactiveVendorsQuery := `
		SELECT id, business_name, city, category, created_at
		FROM vendor_profiles
		WHERE status = 'pending' AND created_at < CURRENT_DATE - INTERVAL '30 days'
		ORDER BY created_at ASC
	`
	rowsInactive, _ := db.Pool.Query(ctx, inactiveVendorsQuery)
	var inactiveVendors []gin.H
	if rowsInactive != nil {
		defer rowsInactive.Close()
		for rowsInactive.Next() {
			var id, name, city, category string
			var createdAt time.Time
			if err := rowsInactive.Scan(&id, &name, &city, &category, &createdAt); err == nil {
				inactiveVendors = append(inactiveVendors, gin.H{
					"vendor_id":     id,
					"business_name": name,
					"city":          city,
					"category":      category,
					"created_at":    createdAt,
				})
			}
		}
	}
	if inactiveVendors == nil {
		inactiveVendors = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"vendors_with_most_shortlists": mostShortlisted,
		"inactive_vendors":             inactiveVendors,
		"top_viewed_vendors":           []gin.H{}, // Empty array since no view tracking exists
	})
}
