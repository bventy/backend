package services

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/bventy/backend/internal/db"
)

type MonitorStatus string

const (
	StatusOperational MonitorStatus = "operational"
	StatusDown        MonitorStatus = "down"
	StatusOffline     MonitorStatus = "offline" // Untracked
)

type StatusPoint struct {
	Status    MonitorStatus `json:"status"`
	LatencyMS int           `json:"latency_ms"`
	ErrorMsg  string        `json:"error_msg,omitempty"`
	CheckedAt time.Time     `json:"checked_at"`
}

type DailyStat struct {
	Date             string  `json:"date"`
	UptimePercentage float64 `json:"uptime_percentage"`
	AvgLatencyMS     int     `json:"avg_latency_ms"`
}

type Incident struct {
	ID          string     `json:"id"`
	MonitorName string     `json:"monitor_name"`
	IssueType   string     `json:"issue_type"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at"`
}

type Monitor struct {
	Name             string        `json:"name"`
	Display          string        `json:"display"`
	Status           MonitorStatus `json:"status"`
	Category         string        `json:"category"`
	LastChecked      time.Time     `json:"last_checked"`
	UptimePercentage float64       `json:"uptime_percentage"`
	AvgLatencyMS     int           `json:"avg_latency_ms"`
	History          []StatusPoint `json:"history,omitempty"`
	DailyStats       []DailyStat   `json:"daily_stats,omitempty"`
}

type SystemStatusService struct {
	monitors []Monitor
	mu       sync.RWMutex
}

var (
	instance *SystemStatusService
	once     sync.Once
)

func GetSystemStatusService() *SystemStatusService {
	once.Do(func() {
		instance = &SystemStatusService{
			monitors: []Monitor{
				// Web
				{Name: "bventy.in", Display: "Website", Category: "Web", Status: StatusOffline},
				{Name: "app.bventy.in", Display: "User Portal", Category: "Web", Status: StatusOffline},
				
				// Frontend
				{Name: "auth.bventy.in", Display: "Auth Service", Category: "Frontend", Status: StatusOffline},
				{Name: "partner.bventy.in", Display: "Partner Portal", Category: "Frontend", Status: StatusOffline},
				{Name: "admin.bventy.in", Display: "Admin Panel", Category: "Frontend", Status: StatusOffline},

				// API
				{Name: "api.bventy.in", Display: "Core API", Category: "API", Status: StatusOffline},

				// Backend / Infra
				{Name: "Neon", Display: "PostgreSQL Database", Category: "Backend", Status: StatusOffline},
				{Name: "Render", Display: "Compute Engine", Category: "Backend", Status: StatusOffline},
				{Name: "Cloudflare R2", Display: "Object Storage", Category: "Backend", Status: StatusOffline},
				
				// Analytics
				{Name: "PostHog", Display: "User Analytics", Category: "Analytics", Status: StatusOffline},
				{Name: "Umami", Display: "Web Analytics", Category: "Analytics", Status: StatusOffline},

				// Communications
				{Name: "Resend", Display: "Email Delivery", Category: "Communications", Status: StatusOffline},
			},
		}
		go instance.startMonitoring()
	})
	return instance
}

func (s *SystemStatusService) GetStatus() ([]Monitor, []Incident, float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	monitors := make([]Monitor, len(s.monitors))
	copy(monitors, s.monitors)

	var totalUptime float64
	for i := range monitors {
		monitors[i].DailyStats = s.getDailyStats(monitors[i].Name)
		monitors[i].UptimePercentage = s.calculateUptime(monitors[i].Name)
		monitors[i].AvgLatencyMS = s.getAverageLatency(monitors[i].Name)
		totalUptime += monitors[i].UptimePercentage
	}

	overallUptime := 0.0
	if len(monitors) > 0 {
		overallUptime = totalUptime / float64(len(monitors))
	}

	return monitors, s.GetActiveIncidents(), overallUptime
}

func (s *SystemStatusService) getAverageLatency(name string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var avgLatency float64
	err := db.Pool.QueryRow(ctx, `
		SELECT COALESCE(AVG(latency_ms), 0)
		FROM system_status_history
		WHERE monitor_name = $1 AND checked_at > now() - interval '24 hours'
	`, name).Scan(&avgLatency)
	
	if err != nil {
		return 0
	}
	return int(avgLatency)
}

func (s *SystemStatusService) calculateUptime(name string) float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var uptime float64
	err := db.Pool.QueryRow(ctx, `
		SELECT 
			COALESCE(
				(COUNT(*) FILTER (WHERE status = 'operational')::float / COUNT(*)) * 100,
				100
			)
		FROM system_status_history
		WHERE monitor_name = $1 AND checked_at > now() - interval '90 days'
	`, name).Scan(&uptime)
	
	if err != nil {
		return 100.0
	}
	return uptime
}

func (s *SystemStatusService) getDailyStats(name string) []DailyStat {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Aggregate status by day for the last 90 days
	rows, err := db.Pool.Query(ctx, `
		WITH days AS (
			SELECT generate_series(
				date_trunc('day', now()) - interval '89 days',
				date_trunc('day', now()),
				interval '1 day'
			)::date as day
		)
		SELECT 
			d.day::text as date,
			COALESCE(
				(COUNT(h.id) FILTER (WHERE h.status = 'operational')::float / NULLIF(COUNT(h.id), 0)) * 100,
				-1 -- -1 means no data for that day
			) as uptime,
			COALESCE(AVG(h.latency_ms) FILTER (WHERE h.status = 'operational'), 0)::int as avg_latency
		FROM days d
		LEFT JOIN system_status_history h ON date_trunc('day', h.checked_at)::date = d.day AND h.monitor_name = $1
		GROUP BY d.day
		ORDER BY d.day DESC
	`, name)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var stats []DailyStat
	for rows.Next() {
		var st DailyStat
		if err := rows.Scan(&st.Date, &st.UptimePercentage, &st.AvgLatencyMS); err == nil {
			stats = append(stats, st)
		}
	}
	return stats
}

func (s *SystemStatusService) GetActiveIncidents() []Incident {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT id, monitor_name, issue_type, description, status, created_at, resolved_at 
		FROM system_incidents 
		WHERE resolved_at IS NULL OR created_at > now() - interval '7 days'
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var incidents []Incident
	for rows.Next() {
		var inc Incident
		if err := rows.Scan(&inc.ID, &inc.MonitorName, &inc.IssueType, &inc.Description, &inc.Status, &inc.CreatedAt, &inc.ResolvedAt); err == nil {
			incidents = append(incidents, inc)
		}
	}
	return incidents
}

func (s *SystemStatusService) startMonitoring() {
	// Ping every 30 minutes as requested for extreme efficiency
	ticker := time.NewTicker(30 * time.Minute)
	
	// Initial check
	s.checkAll()

	for range ticker.C {
		s.checkAll()
	}
}

func (s *SystemStatusService) persistStatus(name, status string, latency int, errMsg string, checkedAt time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO system_status_history (monitor_name, status, latency_ms, error_msg, checked_at) 
		VALUES ($1, $2, $3, $4, $5)
	`, name, status, latency, errMsg, checkedAt)
	if err != nil {
		fmt.Printf("⚠️ Failed to persist status for %s: %v\n", name, err)
	}
}

func (s *SystemStatusService) checkAll() {
	for i := range s.monitors {
		if i > 0 {
			time.Sleep(3 * time.Second) // Staggered for efficiency
		}

		oldStatus := s.monitors[i].Status
		status, latency, errMsg := s.checkMonitor(s.monitors[i])
		now := time.Now()

		s.mu.Lock()
		s.monitors[i].Status = status
		s.monitors[i].LastChecked = now
		s.mu.Unlock()

		// Automated Incident Logging
		if oldStatus == StatusOperational && status == StatusDown {
			s.createIncident(s.monitors[i].Name, "Service Down", fmt.Sprintf("Automated detection: service became unreachable. Diagnostic: %s", errMsg))
		} else if oldStatus == StatusDown && status == StatusOperational {
			s.resolveIncident(s.monitors[i].Name)
		}

		s.persistStatus(s.monitors[i].Name, string(status), latency, errMsg, now)
	}
}

func (s *SystemStatusService) createIncident(name, issueType, desc string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO system_incidents (monitor_name, issue_type, description, status, created_at) 
		VALUES ($1, $2, $3, 'investigating', now())
	`, name, issueType, desc)
	if err != nil {
		fmt.Printf("⚠️ Failed to create incident for %s: %v\n", name, err)
	}
}

func (s *SystemStatusService) resolveIncident(name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		UPDATE system_incidents 
		SET status = 'resolved', resolved_at = now() 
		WHERE monitor_name = $1 AND resolved_at IS NULL
	`, name)
	if err != nil {
		fmt.Printf("⚠️ Failed to resolve incident for %s: %v\n", name, err)
	}
}

func (s *SystemStatusService) checkMonitor(m Monitor) (MonitorStatus, int, string) {
	client := http.Client{
		Timeout: 4 * time.Second,
	}

	var url string
	switch m.Name {
	case "bventy.in":
		url = "https://bventy.in"
	case "app.bventy.in":
		url = "https://app.bventy.in"
	case "auth.bventy.in":
		url = "https://auth.bventy.in"
	case "partner.bventy.in":
		url = "https://partner.bventy.in"
	case "admin.bventy.in":
		url = "https://admin.bventy.in"
	case "api.bventy.in":
		url = "https://api.bventy.in/health"
	case "Neon":
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := db.Pool.Ping(ctx); err != nil {
			return StatusDown, int(time.Since(start).Milliseconds()), err.Error()
		}
		return StatusOperational, int(time.Since(start).Milliseconds()), ""
	case "Render":
		url = "https://api.bventy.in/health"
	case "Cloudflare R2":
		url = "https://www.cloudflarestatus.com"
	case "PostHog":
		url = "https://status.posthog.com"
	case "Umami":
		url = "https://umami.bventy.in/script.js" 
	case "Resend":
		url = "https://api.resend.com/health"
	default:
		return StatusOperational, 0, ""
	}

	start := time.Now()
	// Use HEAD request for efficiency
	req, _ := http.NewRequest(http.MethodHead, url, nil)
	req.Header.Set("User-Agent", "BventyMonitor/1.0")
	
	resp, err := client.Do(req)
	latency := int(time.Since(start).Milliseconds())

	if err != nil {
		// Fallback to GET if HEAD is not supported or fails
		req.Method = http.MethodGet
		resp, err = client.Do(req)
		latency = int(time.Since(start).Milliseconds())
		if err != nil {
			return StatusDown, latency, err.Error()
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return StatusOperational, latency, ""
	}

	return StatusDown, latency, fmt.Sprintf("HTTP %d", resp.StatusCode)
}
