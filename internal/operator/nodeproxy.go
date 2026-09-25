package operator

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ProxyNodeAPI forwards loopback browser requests to the node API (after SSH tunnel).
// Prevents browser CORS issues. Target must be loopback.
func ProxyNodeAPI(w http.ResponseWriter, r *http.Request, opToken string) {
	if !requireToken(r, opToken) {
		http.Error(w, "unauthorized", 401)
		return
	}
	base := strings.TrimSpace(r.Header.Get("X-Node-API-Base"))
	if base == "" {
		base = "http://127.0.0.1:8787"
	}
	u, err := url.Parse(base)
	if err != nil || u.Host == "" {
		http.Error(w, "bad X-Node-API-Base", 400)
		return
	}
	host := u.Hostname()
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		http.Error(w, "node API base must be loopback (use tunnel)", 400)
		return
	}
	suffix := strings.TrimPrefix(r.URL.Path, "/v1/node")
	if suffix == "" {
		suffix = "/"
	}
	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}
	// block weird paths
	if strings.Contains(suffix, "..") {
		http.Error(w, "bad path", 400)
		return
	}
	target := strings.TrimRight(base, "/") + suffix
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}

	body := r.Body
	if r.ContentLength == 0 {
		body = http.NoBody
	}
	req, err := http.NewRequestWithContext(r.Context(), r.Method, target, body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if ct := r.Header.Get("Content-Type"); ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	if sess := strings.TrimSpace(r.Header.Get("X-Node-Session")); sess != "" {
		req.Header.Set("Authorization", "Bearer "+sess)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("node unreachable: %v (tunnel up?)", err), 502)
		return
	}
	defer resp.Body.Close()
	for k, vv := range resp.Header {
		if strings.EqualFold(k, "Transfer-Encoding") || strings.EqualFold(k, "Connection") {
			continue
		}
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(resp.Body, 8<<20))
}

// unused but documents intent
var _ = net.IPv4
