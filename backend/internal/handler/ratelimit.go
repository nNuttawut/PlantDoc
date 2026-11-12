package handler

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter limits requests per client IP (token bucket). It protects the
// Gemini quota from a single client hammering the API.
type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*client
	limit   rate.Limit
	burst   int
}

type client struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter allows `perMinute` requests per minute per IP, with bursts up to `burst`.
func NewRateLimiter(perMinute, burst int) *RateLimiter {
	rl := &RateLimiter{
		clients: map[string]*client{},
		limit:   rate.Limit(float64(perMinute) / 60),
		burst:   burst,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	c, ok := rl.clients[ip]
	if !ok {
		c = &client{lim: rate.NewLimiter(rl.limit, rl.burst)}
		rl.clients[ip] = c
	}
	c.lastSeen = time.Now()
	return c.lim.Allow()
}

func (rl *RateLimiter) cleanup() {
	for range time.Tick(5 * time.Minute) {
		rl.mu.Lock()
		for ip, c := range rl.clients {
			if time.Since(c.lastSeen) > 10*time.Minute {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(clientIP(r)) {
			w.Header().Set("Retry-After", "60")
			writeError(w, http.StatusTooManyRequests, "ถามบ่อยเกินไป กรุณารอสักครู่แล้วลองใหม่")
			return
		}
		next(w, r)
	}
}

// clientIP uses the first X-Forwarded-For entry (set by Render/Fly/Koyeb proxies),
// falling back to the TCP peer address.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
