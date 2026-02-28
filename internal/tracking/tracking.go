package tracking

import (
	"log"
	"sync"

	"github.com/bventy/backend/internal/config"
	"github.com/posthog/posthog-go"
)

var (
	client posthog.Client
	once   sync.Once
)

// Init initializes the PostHog client
func Init(cfg *config.Config) {
	once.Do(func() {
		if cfg.PostHogAPIKey == "" {
			log.Println("⚠️  PostHog API Key not found, tracking disabled")
			return
		}

		var err error
		client, err = posthog.NewWithConfig(
			cfg.PostHogAPIKey,
			posthog.Config{
				Endpoint: cfg.PostHogHost,
			},
		)
		if err != nil {
			log.Printf("❌ Failed to initialize PostHog: %v\n", err)
		} else {
			log.Println("✅ PostHog tracking initialized")
		}
	})
}

// Capture sends an event to PostHog
func Capture(userId string, event string, properties map[string]interface{}) {
	if client == nil {
		return
	}

	err := client.Enqueue(posthog.Capture{
		DistinctId: userId,
		Event:      event,
		Properties: properties,
	})
	if err != nil {
		log.Printf("❌ Failed to enqueue PostHog event: %v\n", err)
	}
}

// Identify sends an identify event to PostHog
func Identify(userId string, properties map[string]interface{}) {
	if client == nil {
		return
	}

	err := client.Enqueue(posthog.Identify{
		DistinctId: userId,
		Properties: properties,
	})
	if err != nil {
		log.Printf("❌ Failed to enqueue PostHog identify: %v\n", err)
	}
}

// Flush ensures all events are sent
func Flush() {
	if client != nil {
		client.Close()
	}
}
