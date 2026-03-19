package middleware

import (
	"fmt"
	"maps"
	"net/http"
	"sync"
	"time"
)

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    int
	window   time.Duration
}

type visitor struct {
	count    int
	resetAt  time.Time
}

func NewRateLimiter(requestsPerMinute int) func(http.Handler) http.Handler {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		limit:    requestsPerMinute,
		window:   time.Minute,
	}

	go rl.cleanup()

	return rl.middleware
}

func (rl *rateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := rl.getKey(r)

		rl.mu.Lock()
		v, exists := rl.visitors[key]
		now := time.Now()

		if !exists || now.After(v.resetAt) {
			rl.visitors[key] = &visitor{count: 1, resetAt: now.Add(rl.window)}
			rl.mu.Unlock()
			next.ServeHTTP(w, r)
			return
		}

		v.count++
		if v.count > rl.limit {
			remaining := v.resetAt.Sub(now).Seconds()
			rl.mu.Unlock()
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", remaining))
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		rl.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

func (rl *rateLimiter) getKey(r *http.Request) string {
	if userID := r.Context().Value(UserIDKey); userID != nil {
		return fmt.Sprintf("user:%v", userID)
	}
	return "ip:" + r.RemoteAddr
}

func (rl *rateLimiter) cleanup() {
	for {
		time.Sleep(5 * time.Minute)
		rl.mu.Lock()
		now := time.Now()
		maps.DeleteFunc(rl.visitors, func(_ string, v *visitor) bool {
			return now.After(v.resetAt)
		})
		rl.mu.Unlock()
	}
}
