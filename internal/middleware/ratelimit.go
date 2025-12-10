package middleware

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter.
type RateLimiter struct {
	mu          sync.RWMutex
	clients     map[string]*clientLimiter
	rate        int           // requests per window
	window      time.Duration // time window
	cleanupTick *time.Ticker
	stopCleanup chan bool
}

type clientLimiter struct {
	tokens     int
	lastUpdate time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter.
// rate: number of requests allowed
// window: time window for the rate limit
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients:     make(map[string]*clientLimiter),
		rate:        rate,
		window:      window,
		cleanupTick: time.NewTicker(5 * time.Minute),
		stopCleanup: make(chan bool),
	}

	// Start cleanup goroutine to remove old entries
	go rl.cleanup()

	return rl
}

// cleanup periodically removes old client entries to prevent memory leaks.
func (rl *RateLimiter) cleanup() {
	for {
		select {
		case <-rl.cleanupTick.C:
			rl.mu.Lock()
			now := time.Now()
			for key, limiter := range rl.clients {
				limiter.mu.Lock()
				// Remove if last update was more than 1 hour ago
				if now.Sub(limiter.lastUpdate) > time.Hour {
					delete(rl.clients, key)
				}
				limiter.mu.Unlock()
			}
			rl.mu.Unlock()
		case <-rl.stopCleanup:
			return
		}
	}
}

// Stop stops the cleanup goroutine.
func (rl *RateLimiter) Stop() {
	rl.cleanupTick.Stop()
	rl.stopCleanup <- true
}

// Allow checks if a request from the given key should be allowed.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.RLock()
	limiter, exists := rl.clients[key]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		// Double-check after acquiring write lock
		limiter, exists = rl.clients[key]
		if !exists {
			limiter = &clientLimiter{
				tokens:     rl.rate,
				lastUpdate: time.Now(),
			}
			rl.clients[key] = limiter
		}
		rl.mu.Unlock()
	}

	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(limiter.lastUpdate)

	// Refill tokens based on elapsed time
	if elapsed >= rl.window {
		limiter.tokens = rl.rate
		limiter.lastUpdate = now
	} else {
		// Calculate tokens to add based on elapsed time
		tokensToAdd := int(float64(rl.rate) * elapsed.Seconds() / rl.window.Seconds())
		if tokensToAdd > 0 {
			limiter.tokens = min(limiter.tokens+tokensToAdd, rl.rate)
			limiter.lastUpdate = now
		}
	}

	if limiter.tokens > 0 {
		limiter.tokens--
		return true
	}

	return false
}

// GetClientKey extracts a client identifier from the request.
// Uses IP address as the key.
func GetClientKey(r *http.Request) string {
	// Check X-Forwarded-For header (for proxies/load balancers)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP (original client)
		return forwarded
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// RateLimitMiddleware creates a middleware that rate limits requests.
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := GetClientKey(r)

			if !limiter.Allow(key) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-RateLimit-Limit", "100")
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error": "rate limit exceeded"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
