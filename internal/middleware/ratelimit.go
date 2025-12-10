package middleware

import (
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	mu          sync.RWMutex
	clients     map[string]*clientLimiter
	rate        int
	window      time.Duration
	cleanupTick *time.Ticker
	stopCleanup chan bool
}

type clientLimiter struct {
	tokens     int
	lastUpdate time.Time
	mu         sync.Mutex
}

func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients:     make(map[string]*clientLimiter),
		rate:        rate,
		window:      window,
		cleanupTick: time.NewTicker(5 * time.Minute),
		stopCleanup: make(chan bool),
	}

	go rl.cleanup()

	return rl
}
func (rl *RateLimiter) cleanup() {
	for {
		select {
		case <-rl.cleanupTick.C:
			rl.mu.Lock()
			now := time.Now()
			for key, limiter := range rl.clients {
				limiter.mu.Lock()
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

func (rl *RateLimiter) Stop() {
	rl.cleanupTick.Stop()
	rl.stopCleanup <- true
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.RLock()
	limiter, exists := rl.clients[key]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
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

	if elapsed >= rl.window {
		limiter.tokens = rl.rate
		limiter.lastUpdate = now
	} else {
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

func GetClientKey(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return forwarded
	}

	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	return r.RemoteAddr
}

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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
