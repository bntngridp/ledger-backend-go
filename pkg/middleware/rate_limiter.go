package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type clientLimit struct {
	lastSeen time.Time
	tokens   int
}

// IPBasedRateLimiter implements an in-memory token bucket rate limiter per client IP.
func IPBasedRateLimiter(maxTokens, refillRate int, duration time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	clients := make(map[string]*clientLimit)

	// Clean up inactive clients to prevent memory leaks
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			mu.Lock()
			for ip, limit := range clients {
				if time.Since(limit.lastSeen) > 1*time.Hour {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		mu.Lock()
		client, exists := clients[ip]
		now := time.Now()

		if !exists {
			clients[ip] = &clientLimit{
				lastSeen: now,
				tokens:   maxTokens - 1,
			}
			mu.Unlock()
			c.Next()
			return
		}

		elapsed := now.Sub(client.lastSeen)
		refills := int(elapsed/duration) * refillRate
		if refills > 0 {
			client.tokens += refills
			if client.tokens > maxTokens {
				client.tokens = maxTokens
			}
			client.lastSeen = now
		}

		if client.tokens <= 0 {
			mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"status":  http.StatusTooManyRequests,
				"message": "Too many requests. Please try again later.",
			})
			return
		}

		client.tokens--
		mu.Unlock()
		c.Next()
	}
}
