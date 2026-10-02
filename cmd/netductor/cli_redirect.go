package main

import (
	"github.com/PavelNeyman/netductor/internal/vpn"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// runRedirectServe: open-redirect-safe landing for TG url buttons.
// GET /r?u=<base64url(deep-link)> → 302 Location: deep-link
// Optional HTTPS: -tls-cert / -tls-key and -https-listen (default :8443 only if certs given).
func runRedirectServe(args []string) {
	addr := "off"
	httpsAddr := ""
	tlsCert, tlsKey := "", ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-listen", "--listen":
			if i+1 < len(args) {
				addr = args[i+1]
				i++
			}
		case "-https-listen":
			if i+1 < len(args) {
				httpsAddr = args[i+1]
				i++
			}
		case "-tls-cert":
			if i+1 < len(args) {
				tlsCert = args[i+1]
				i++
			}
		case "-tls-key":
			if i+1 < len(args) {
				tlsKey = args[i+1]
				i++
			}
		case "-h", "--help":
			fmt.Println("usage: netductor redirect-serve [-listen 127.0.0.1:80] [-https-listen :8443] [-tls-cert C] [-tls-key K]\n  default listen is loopback; use -listen :80 only if intentional public HTTP")
			return
		}
	}
	// env fallback for TLS
	if tlsCert == "" {
		tlsCert = strings.TrimSpace(os.Getenv("NETDUCTOR_REDIRECT_TLS_CERT"))
	}
	if tlsKey == "" {
		tlsKey = strings.TrimSpace(os.Getenv("NETDUCTOR_REDIRECT_TLS_KEY"))
	}
	if httpsAddr == "" && tlsCert != "" && tlsKey != "" {
		httpsAddr = ":8443"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/r", handleImportRedirect)
	mux.HandleFunc("/profiles/", handleProfileDownload)
	mux.HandleFunc("/sub/", handleSubscription)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	if addr != "" && addr != "off" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Printf("import redirect HTTP on %s", addr)
			errCh <- http.ListenAndServe(addr, mux)
		}()
	}
	if httpsAddr != "" && tlsCert != "" && tlsKey != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Printf("import redirect HTTPS on %s", httpsAddr)
			errCh <- http.ListenAndServeTLS(httpsAddr, tlsCert, tlsKey, mux)
		}()
	}
	err := <-errCh
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	wg.Wait()
}

func handleProfileDownload(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/profiles/")
	name = strings.Trim(name, "/")
	if name == "" || strings.Contains(name, "..") || strings.Contains(name, "/") {
		http.NotFound(w, r)
		return
	}
	candidates := []string{
		filepath.Join("/opt/netductor/profiles", name),
		filepath.Join("/etc/netductor/profiles", name),
	}
	var data []byte
	var err error
	for _, p := range candidates {
		data, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	if err != nil || len(data) == 0 {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+safeAttachmentFilename(name)+"\"")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(data)
}

func handleImportRedirect(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimSpace(r.URL.Query().Get("u"))
	if raw == "" {
		http.Error(w, "missing u", http.StatusBadRequest)
		return
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		b, err = base64.URLEncoding.DecodeString(raw)
	}
	if err != nil {
		if u, e := url.QueryUnescape(raw); e == nil && allowedDeepLink(u) {
			w.Header().Set("Cache-Control", "no-store")
			http.Redirect(w, r, u, http.StatusFound)
			return
		}
		http.Error(w, "bad u", http.StatusBadRequest)
		return
	}
	target := strings.TrimSpace(string(b))
	if !allowedDeepLink(target) {
		http.Error(w, "target not allowed", http.StatusBadRequest)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, target, http.StatusFound)
}

func allowedDeepLink(s string) bool {
	low := strings.ToLower(s)
	for _, p := range []string{
		"shadowrocket://", "happ://", "incy://",
		"vless://", "hysteria2://", "hy2://", "ss://", "trojan://",
	} {
		if strings.HasPrefix(low, p) {
			return true
		}
	}
	return false
}



// In-memory rate limit for public /sub/ (does not change URL or auth contract).
var (
	subRLMu sync.Mutex
	subRL   = map[string]struct {
		n     int
		until int64
	}{}
)

func subClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// allowSubRequest: max per IP per window (default 60/min). Failed auth still counts.
func allowSubRequest(ip string) bool {
	const maxN = 60
	const window = int64(60) // seconds
	if ip == "" {
		ip = "unknown"
	}
	now := time.Now().Unix()
	subRLMu.Lock()
	defer subRLMu.Unlock()
	// opportunistic prune
	if len(subRL) > 10000 {
		for k, v := range subRL {
			if now > v.until {
				delete(subRL, k)
			}
		}
	}
	a, ok := subRL[ip]
	if !ok || now > a.until {
		subRL[ip] = struct {
			n     int
			until int64
		}{n: 1, until: now + window}
		return true
	}
	if a.n >= maxN {
		return false
	}
	a.n++
	subRL[ip] = a
	return true
}

// GET /sub/{token} — base64 subscription body for one VPN user (token auth).
func handleSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method", 405)
		return
	}
	if !allowSubRequest(subClientIP(r)) {
		w.Header().Set("Retry-After", "60")
		http.Error(w, "rate limited", http.StatusTooManyRequests)
		return
	}
	tok := strings.TrimPrefix(r.URL.Path, "/sub/")
	tok = strings.Trim(tok, "/")
	if i := strings.IndexByte(tok, '/'); i >= 0 {
		tok = tok[:i]
	}
	user := vpn.ResolveSubToken(tok)
	if user == "" {
		http.NotFound(w, r)
		return
	}
	b64, err := vpn.SubscriptionBase64(user)
	if err != nil {
		http.Error(w, "unavailable", 503)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write([]byte(b64))
}
