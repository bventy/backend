package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	LastActivity     time.Time
	activityMu       sync.RWMutex
	initialTime      = time.Now()
)

func init() {
	LastActivity = initialTime
}

// ActivityTracker updates the last activity timestamp
func ActivityTracker() gin.HandlerFunc {
	return func(c *gin.Context) {
		activityMu.Lock()
		LastActivity = time.Now()
		activityMu.Unlock()
		c.Next()
	}
}

// GetLastActivity returns the timestamp of the last request
func GetLastActivity() time.Time {
	activityMu.RLock()
	defer activityMu.RUnlock()
	return LastActivity
}
