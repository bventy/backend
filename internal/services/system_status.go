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
	CheckedAt time.Time     `json:"checked_at"`
}

type Monitor struct {
	Name        string        `json:"name"`
	Display     string        `json:"display"`
	Status      MonitorStatus `json:"status"`
	Category    string        `json:"category"`
	LastChecked time.Time     `json:"last_checked"`
	History     []StatusPoint `json:"history,omitempty"`
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
				{Name: "bventy.in", Display: "Main Website", Category: "Web", Status: StatusOffline},
				{Name: "app.bventy.in", Display: "User Portal", Category: "Web", Status: StatusOffline},
				
				// Frontend
				{Name: "auth.bventy.in", Display: "Auth Service", Category: "Frontend", Status: StatusOffline},
				{Name: "partner.bventy.in", Display: "Vendor Dashboard", Category: "Frontend", Status: StatusOffline},
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
			},
		}
		go instance.startMonitoring()
	})
	return instance
}

func (s *SystemStatusService) GetStatus() []Monitor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Copy monitors but enrich with history from DB
	monitors := make([]Monitor, len(s.monitors))
	copy(monitors, s.monitors)

	for i := range monitors {
		monitors[i].History = s.getHistory(monitors[i].Name)
	}

	return monitors
}

func (s *SystemStatusService) getHistory(name string) []StatusPoint {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT status, checked_at 
		FROM system_status_history 
		WHERE monitor_name = $1 
		ORDER BY checked_at DESC 
		LIMIT 90
	`, name)
	if err != nil {
		fmt.Printf("⚠️ Failed to fetch history for %s: %v\n", name, err)
		return nil
	}
	defer rows.Close()

	var history []StatusPoint
	for rows.Next() {
		var p StatusPoint
		if err := rows.Scan(&p.Status, &p.CheckedAt); err == nil {
			history = append(history, p)
		}
	}
	return history
}

func (s *SystemStatusService) startMonitoring() {
	// Ping every 10 minutes as requested
	ticker := time.NewTicker(10 * time.Minute)
	
	// Initial check
	s.checkAll()

	for range ticker.C {
		s.checkAll()
	}
}

func (s *SystemStatusService) checkAll() {
	for i := range s.monitors {
		// Staggered pings: wait 5 seconds between each monitor to be extra safe on free tiers
		if i > 0 {
			time.Sleep(5 * time.Second)
		}

		status := s.checkMonitor(s.monitors[i])
		now := time.Now()

		s.mu.Lock()
		s.monitors[i].Status = status
		s.monitors[i].LastChecked = now
		s.mu.Unlock()

		// Persist to DB
		s.persistStatus(s.monitors[i].Name, string(status), now)
	}
}

func (s *SystemStatusService) persistStatus(name, status string, checkedAt time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO system_status_history (monitor_name, status, checked_at) 
		VALUES ($1, $2, $3)
	`, name, status, checkedAt)
	if err != nil {
		fmt.Printf("⚠️ Failed to persist status for %s: %v\n", name, err)
	}
}

func (s *SystemStatusService) checkMonitor(m Monitor) MonitorStatus {
	client := http.Client{
		Timeout: 5 * time.Second,
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
		url = "https://status.neon.tech" 
	case "Render":
		url = "https://status.render.com"
	case "Cloudflare R2":
		url = "https://www.cloudflarestatus.com"
	case "PostHog":
		url = "https://status.posthog.com"
	case "Umami":
		url = "https://status.umami.is"
	default:
		return StatusOperational
	}

	resp, err := client.Get(url)
	if err != nil {
		return StatusDown
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return StatusOperational
	}

	return StatusDown
}
