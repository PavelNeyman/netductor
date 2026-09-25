package httpx

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// PlaneLimiter is a per-IP sliding window for the agent plane (:8789).
// After banAfter consecutive rate-limit denials within banWindow, IP is banned for banTTL.
type PlaneLimiter struct {
	mu        sync.Mutex
	hits      map[string][]time.Time
	denies    map[string][]time.Time
	banned    map[string]time.Time
	max       int
	window    time.Duration
	banAfter  int
	banWindow time.Duration
	banTTL    time.Duration
}

func NewPlaneLimiter(maxPerWindow int, window time.Duration) *PlaneLimiter {
	if maxPerWindow <= 0 {
		maxPerWindow = 120
	}
	if window <= 0 {
		window = time.Minute
	}
	banAfter := 5
	if v := os.Getenv("NETDUCTOR_PLANE_BAN_AFTER"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			banAfter = n
		}
	}
	banTTL := 15 * time.Minute
	if v := os.Getenv("NETDUCTOR_PLANE_BAN_TTL_MIN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			banTTL = time.Duration(n) * time.Minute
		}
	}
	return &PlaneLimiter{
		hits: map[string][]time.Time{}, denies: map[string][]time.Time{}, banned: map[string]time.Time{},
		max: maxPerWindow, window: window,
		banAfter: banAfter, banWindow: window, banTTL: banTTL,
	}
}

func clientIP(r *http.Request) string {
	// Only trust proxy headers when explicitly enabled (prevents rate-limit bypass).
	if os.Getenv("NETDUCTOR_TRUST_PROXY") == "1" {
		if x := r.Header.Get("X-Real-IP"); x != "" {
			return strings.TrimSpace(x)
		}
		if x := r.Header.Get("X-Forwarded-For"); x != "" {
			if i := strings.IndexByte(x, ','); i >= 0 {
				return strings.TrimSpace(x[:i])
			}
			return strings.TrimSpace(x)
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}


func trimTimes(arr []time.Time, cut time.Time) []time.Time {
	n := 0
	for _, t := range arr {
		if t.After(cut) {
			arr[n] = t
			n++
		}
	}
	return arr[:n]
}

func (p *PlaneLimiter) Allow(ip string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	if until, ok := p.banned[ip]; ok {
		if now.Before(until) {
			return false
		}
		delete(p.banned, ip)
	}
	cut := now.Add(-p.window)
	arr := trimTimes(p.hits[ip], cut)
	if len(arr) >= p.max {
		p.hits[ip] = arr
		dcut := now.Add(-p.banWindow)
		den := trimTimes(p.denies[ip], dcut)
		den = append(den, now)
		p.denies[ip] = den
		if len(den) >= p.banAfter {
			p.banned[ip] = now.Add(p.banTTL)
			p.denies[ip] = nil
		}
		return false
	}
	p.hits[ip] = append(arr, now)
	return true
}

func (p *PlaneLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !p.Allow(ip) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "rate_limited", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
