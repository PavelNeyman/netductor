package main

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/session"
)

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// withSecurity wraps the API mux with baseline headers and request body limit.
func withSecurity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}

func readJSON(r *http.Request) map[string]any {
	defer r.Body.Close()
	var m map[string]any
	_ = json.NewDecoder(io.LimitReader(r.Body, MaxBodyBytes)).Decode(&m)
	if m == nil {
		m = map[string]any{}
	}
	return m
}

func bearer(r *http.Request) string {
	return session.TokenFromAuth(r.Header.Get("Authorization"), r.Header.Get("Cookie"))
}

func securityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
}

func requireSession(w http.ResponseWriter, r *http.Request) bool {
	securityHeaders(w)
	tok := bearer(r)
	if tok == "" || !session.Valid(tok) {
		writeJSON(w, 401, map[string]string{"error": "unauthorized"})
		return false
	}
	return true
}

func adminRoot() string {
	return paths.AdminRoot()
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func isLoopback(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// MaxBodyBytes default request body limit for JSON APIs.
const MaxBodyBytes = 16 << 20 // 16 MiB — JSON APIs + modest uploads

// clientIP returns the remote IP (X-Real-IP / X-Forwarded-For first hop / RemoteAddr).

func clientIP(r *http.Request) string {
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


// safeAttachmentFilename strips path and header-injection characters from Content-Disposition names.
func safeAttachmentFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == '"' || r == '\\' || r == '/' {
			return -1
		}
		return r
	}, name)
	if name == "" || name == "." || name == ".." {
		return "download"
	}
	if len(name) > 180 {
		name = name[:180]
	}
	return name
}
