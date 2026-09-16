package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"os"
	"strings"
	"sync"
)

// runRedirectServe: open-redirect-safe landing for TG url buttons.
// GET /r?u=<base64url(deep-link)> → 302 Location: deep-link
// Optional HTTPS: -tls-cert / -tls-key and -https-listen (default :8443 only if certs given).
func runRedirectServe(args []string) {
	addr := ":80"
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
			fmt.Println("usage: netductor redirect-serve [-listen :80] [-https-listen :8443] [-tls-cert C] [-tls-key K]")
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
	w.Header().Set("Content-Disposition", "attachment; filename="+name)
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
