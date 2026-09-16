package secondary

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// DefaultCatalogURLs checks what is reachable from secondary's RU IP.
var DefaultCatalogURLs = []string{
	"https://ya.ru/",
	"https://www.google.com/generate_204",
	"https://github.com/",
	"https://api.telegram.org/",
	"https://www.cloudflare.com/cdn-cgi/trace",
}

type CatalogResult struct {
	URL    string `json:"url"`
	OK     bool   `json:"ok"`
	Status int    `json:"status,omitempty"`
	MS     int64  `json:"ms"`
	Error  string `json:"error,omitempty"`
}

// RunCatalogHTTP runs from the machine where this is called (run on secondary).
func RunCatalogHTTP(urls []string, timeout time.Duration) []CatalogResult {
	if len(urls) == 0 {
		urls = DefaultCatalogURLs
	}
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	client := &http.Client{Timeout: timeout, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	}}
	var out []CatalogResult
	for _, u := range urls {
		t0 := time.Now()
		resp, err := client.Get(u)
		ms := time.Since(t0).Milliseconds()
		if err != nil {
			out = append(out, CatalogResult{URL: u, OK: false, MS: ms, Error: err.Error()})
			continue
		}
		resp.Body.Close()
		ok := resp.StatusCode > 0 && resp.StatusCode < 500
		out = append(out, CatalogResult{URL: u, OK: ok, Status: resp.StatusCode, MS: ms})
	}
	return out
}

func FormatCatalog(results []CatalogResult) string {
	var b strings.Builder
	b.WriteString("secondary reachability\n")
	for _, r := range results {
		mark := "❌"
		if r.OK {
			mark = "✅"
		}
		b.WriteString(fmt.Sprintf("%s %s (%dms", mark, r.URL, r.MS))
		if r.Status > 0 {
			b.WriteString(fmt.Sprintf(" http %d", r.Status))
		}
		if r.Error != "" {
			b.WriteString(" " + r.Error)
		}
		b.WriteString(")\n")
	}
	return b.String()
}
