package httpx

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// PlaneLimiter is a simple per-IP sliding window for the agent plane (:8789).
// Limits probe noise; mTLS still required for authenticated work.
type PlaneLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	max    int
	window time.Duration
}

func NewPlaneLimiter(maxPerWindow int, window time.Duration) *PlaneLimiter {
	if maxPerWindow <= 0 {
		maxPerWindow = 120
	}
	if window <= 0 {
		window = time.Minute
	}
	return &PlaneLimiter{hits: map[string][]time.Time{}, max: maxPerWindow, window: window}
}

func clientIP(r *http.Request) string {
	if x := r.Header.Get("X-Forwarded-For"); x != "" {
		return strings.TrimSpace(strings.Split(x, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (p *PlaneLimiter) Allow(ip string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	cut := now.Add(-p.window)
	arr := p.hits[ip]
	n := 0
	for _, t := range arr {
		if t.After(cut) {
			arr[n] = t
			n++
		}
	}
	arr = arr[:n]
	if len(arr) >= p.max {
		p.hits[ip] = arr
		return false
	}
	p.hits[ip] = append(arr, now)
	return true
}

func (p *PlaneLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !p.Allow(clientIP(r)) {
			http.Error(w, "rate_limited", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
