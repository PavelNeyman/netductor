package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// runRedirectServe: minimal open-redirect-safe landing for TG url buttons.
// GET /r?u=<base64url(deep-link)>  → 302 Location: deep-link
// Allowed targets: shadowrocket:// happ:// incy:// vless:// hysteria2:// hy2://
func runRedirectServe(args []string) {
	addr := ":80"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-listen", "--listen":
			if i+1 < len(args) {
				addr = args[i+1]
				i++
			}
		case "-h", "--help":
			fmt.Println("usage: netductor redirect-serve [-listen :80]")
			return
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/r", handleImportRedirect)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("netductor import redirect\n"))
	})
	log.Printf("import redirect listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func handleImportRedirect(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimSpace(r.URL.Query().Get("u"))
	if raw == "" {
		http.Error(w, "missing u", http.StatusBadRequest)
		return
	}
	// accept standard or raw base64url
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		b, err = base64.URLEncoding.DecodeString(raw)
	}
	if err != nil {
		// also allow percent-encoded deep link directly
		if u, e := url.QueryUnescape(raw); e == nil && allowedDeepLink(u) {
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
	// no logging of secrets
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, target, http.StatusFound)
}

func allowedDeepLink(s string) bool {
	low := strings.ToLower(s)
	for _, p := range []string{
		"shadowrocket://",
		"happ://",
		"incy://",
		"vless://",
		"hysteria2://",
		"hy2://",
		"ss://",
		"trojan://",
	} {
		if strings.HasPrefix(low, p) {
			return true
		}
	}
	return false
}

// RedirectPublicBase returns public base for TG buttons (env or empty).
func RedirectPublicBase() string {
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_REDIRECT_BASE")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return ""
}

// BuildImportRedirectURL builds http(s)://host/r?u=base64url(deep)
func BuildImportRedirectURL(base, deep string) string {
	base = strings.TrimRight(base, "/")
	if base == "" || deep == "" {
		return ""
	}
	enc := base64.RawURLEncoding.EncodeToString([]byte(deep))
	return base + "/r?u=" + enc
}

// ensure used if compiled with deadcode tools
