package vpn

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// WLCheckResult is a lightweight probe against public WL check APIs (olc / compatible).
type WLCheckResult struct {
	IP     string `json:"ip"`
	OK     bool   `json:"ok"`
	Raw    string `json:"raw,omitempty"`
	Source string `json:"source"`
	Error  string `json:"error,omitempty"`
}

// CheckWhitelistIP queries https://wly.zarazaex.xyz/check?ip= (openlibrecommunity tooling).
// Network-dependent; failure does not mean IP is bad — only that check API was unreachable.
func CheckWhitelistIP(ip string) WLCheckResult {
	ip = strings.TrimSpace(ip)
	res := WLCheckResult{IP: ip, Source: "wly.zarazaex.xyz"}
	if ip == "" {
		res.Error = "empty ip"
		return res
	}
	u := "https://wly.zarazaex.xyz/check?ip=" + url.QueryEscape(ip)
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	res.Raw = strings.TrimSpace(string(body))
	// API shape may evolve; treat HTTP 200 + non-empty as soft signal
	if resp.StatusCode == 200 {
		var m map[string]any
		if json.Unmarshal(body, &m) == nil {
			for _, k := range []string{"ok", "whitelist", "whitelisted", "in_whitelist", "pass"} {
				if v, ok := m[k]; ok {
					switch t := v.(type) {
					case bool:
						res.OK = t
						return res
					case string:
						res.OK = t == "true" || t == "yes" || t == "1"
						return res
					}
				}
			}
		}
		low := strings.ToLower(res.Raw)
		res.OK = strings.Contains(low, "true") || strings.Contains(low, "whitelist") || strings.Contains(low, `"ok":`)
	} else {
		res.Error = fmt.Sprintf("http %d", resp.StatusCode)
	}
	return res
}

// CheckSelfPublicIP checks this host public IP against WL API.
func CheckSelfPublicIP() WLCheckResult {
	ip := publicIP()
	if ip == "" {
		ip = strings.TrimSpace(os.Getenv("NETDUCTOR_PUBLIC_IP"))
	}
	return CheckWhitelistIP(ip)
}
