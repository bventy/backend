package services

import (
	"net/http"
	"sync"
	"time"
)

type MonitorStatus string

const (
	StatusOperational MonitorStatus = "operational"
	StatusDown        MonitorStatus = "down"
)

type Monitor struct {
	Name        string        `json:"name"`
	Display     string        `json:"display"`
	Status      MonitorStatus `json:"status"`
	Category    string        `json:"category"`
	LastChecked time.Time     `json:"last_checked"`
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
				{Name: "bventy.in", Display: "Main Website", Category: "Web", Status: StatusOperational},
				{Name: "app.bventy.in", Display: "User Portal", Category: "Web", Status: StatusOperational},
                
                // Frontend
				{Name: "auth.bventy.in", Display: "Auth Service", Category: "Frontend", Status: StatusOperational},
				{Name: "partner.bventy.in", Display: "Vendor Dashboard", Category: "Frontend", Status: StatusOperational},
				{Name: "admin.bventy.in", Display: "Admin Panel", Category: "Frontend", Status: StatusOperational},

				// API
				{Name: "api.bventy.in", Display: "Core API", Category: "API", Status: StatusOperational},

                // Backend / Infra
				{Name: "Neon", Display: "PostgreSQL Database", Category: "Backend", Status: StatusOperational},
				{Name: "Render", Display: "Compute Engine", Category: "Backend", Status: StatusOperational},
				{Name: "Cloudflare R2", Display: "Object Storage", Category: "Backend", Status: StatusOperational},
				
                // Analytics
                {Name: "PostHog", Display: "User Analytics", Category: "Analytics", Status: StatusOperational},
				{Name: "Umami", Display: "Web Analytics", Category: "Analytics", Status: StatusOperational},
			},
		}
		go instance.startMonitoring()
	})
	return instance
}

func (s *SystemStatusService) GetStatus() []Monitor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.monitors
}

func (s *SystemStatusService) startMonitoring() {
	ticker := time.NewTicker(5 * time.Minute)
	// Initial check
	s.checkAll()

	for range ticker.C {
		s.checkAll()
	}
}

func (s *SystemStatusService) checkAll() {
	var wg sync.WaitGroup
	for i := range s.monitors {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			status := s.checkMonitor(s.monitors[idx])
			s.mu.Lock()
			s.monitors[idx].Status = status
			s.monitors[idx].LastChecked = time.Now()
			s.mu.Unlock()
		}(i)
	}
	wg.Wait()
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
        // This is tricky, maybe a ping to the DB if possible, or just check the provider's status page
        url = "https://status.neon.tech" 
    case "Render":
        url = "https://status.render.com"
    case "Cloudflare R2":
        url = "https://www.cloudflarestatus.com"
    case "PostHog":
        url = "https://status.posthog.com"
    case "Umami":
        // Umami cloud status
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
