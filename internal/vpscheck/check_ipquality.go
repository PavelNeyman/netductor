package vpscheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// IP quality / blocklists without third-party bash scripts.
// Combines: ip-api proxy flags, multi-DNSBL, free reputation endpoints.

func checkIPQuality(ctx context.Context, publicIP string) []Result {
	var out []Result
	if publicIP == "" {
		out = append(out, Result{ID: "ipq_skip", TitleEN: "IP quality", TitleRU: "IP quality", Status: StatusSkip, Detail: "no public IP"})
		return out
	}
	out = append(out, checkDNSBL(ctx, publicIP))
	out = append(out, checkProxyFlags(ctx, publicIP))
	out = append(out, checkIPAPIAbuseHints(ctx, publicIP))
	out = append(out, checkHTTPFromIP(ctx, publicIP))
	return out
}

// Spamhaus ZEN + a few common lists via dig (reverse DNSBL).
func checkDNSBL(ctx context.Context, ip string) Result {
	start := time.Now()
	r := Result{ID: "ipq_dnsbl", TitleEN: "DNSBL", TitleRU: "DNSBL (спам-листы)"}
	rev, err := reverseIPv4(ip)
	if err != nil {
		r.Status = StatusSkip
		r.Detail = err.Error()
		return r
	}
	lists := []struct {
		name, zone string
	}{
		{"zen", "zen.spamhaus.org"},
		{"sbl", "sbl.spamhaus.org"},
		{"xbl", "xbl.spamhaus.org"},
		{"cbl", "cbl.abuseat.org"},
		{"b.barracudacentral.org", "b.barracudacentral.org"},
	}
	var hits, checked, unknown []string
	for _, l := range lists {
		q := rev + "." + l.zone
		res := dnsblLookup(ctx, q)
		checked = append(checked, l.name)
		switch res {
		case dnsblListed:
			hits = append(hits, l.name)
		case dnsblUnknown:
			unknown = append(unknown, l.name)
		}
	}
	r.DurationMs = time.Since(start).Milliseconds()
	switch {
	case len(hits) > 0:
		r.Status = StatusFail
		r.Detail = "LISTED: " + strings.Join(hits, ",") + " (checked " + strings.Join(checked, ",") + ")"
	case len(unknown) == len(checked):
		r.Status = StatusWarn
		r.Detail = "DNSBL unreachable (all queries failed): " + strings.Join(unknown, ",")
	case len(unknown) > 0:
		r.Status = StatusWarn
		r.Detail = "clean on some; unknown: " + strings.Join(unknown, ",") + " (checked " + strings.Join(checked, ",") + ")"
	default:
		r.Status = StatusOK
		r.Detail = "clean on " + strings.Join(checked, ",")
	}
	return r
}


func reverseIPv4(ip string) (string, error) {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "", fmt.Errorf("bad IP %s", ip)
	}
	v4 := parsed.To4()
	if v4 == nil {
		return "", fmt.Errorf("DNSBL only IPv4 for now")
	}
	return fmt.Sprintf("%d.%d.%d.%d", v4[3], v4[2], v4[1], v4[0]), nil
}

// dnsblResult: listed / clean / unknown
type dnsblResult int

const (
	dnsblClean dnsblResult = iota
	dnsblListed
	dnsblUnknown
)

func dnsblLookup(ctx context.Context, name string) dnsblResult {
	if _, err := exec.LookPath("dig"); err == nil {
		cmd := exec.CommandContext(ctx, "dig", "+short", "+time=3", "+tries=1", name, "A")
		out, err := cmd.CombinedOutput()
		s := strings.TrimSpace(string(out))
		if err != nil && s == "" {
			return dnsblUnknown
		}
		if s == "" {
			return dnsblClean
		}
		for _, line := range strings.Split(s, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "127.0.0.") {
				return dnsblListed
			}
		}
		return dnsblUnknown
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", name)
	if err != nil {
		if dnsErr, ok := err.(*net.DNSError); ok && dnsErr.IsNotFound {
			return dnsblClean
		}
		return dnsblUnknown
	}
	for _, ip := range ips {
		if ip.IsLoopback() {
			return dnsblListed
		}
	}
	return dnsblUnknown
}


func checkProxyFlags(ctx context.Context, ip string) Result {
	start := time.Now()
	r := Result{ID: "ipq_proxy", TitleEN: "Proxy/VPN flags", TitleRU: "Флаги proxy/VPN"}
	client := &http.Client{Timeout: 12 * time.Second}
	// ip-api fields already used; free endpoint without key
	url := "http://ip-api.com/json/" + ip + "?fields=status,proxy,hosting,mobile,query"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := client.Do(req)
	r.DurationMs = time.Since(start).Milliseconds()
	if err != nil {
		r.Status = StatusWarn
		r.Detail = err.Error()
		return r
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	proxy := m["proxy"] == true
	hosting := m["hosting"] == true
	mobile := m["mobile"] == true
	r.Detail = fmt.Sprintf("proxy=%v hosting=%v mobile=%v", proxy, hosting, mobile)
	switch {
	case proxy:
		r.Status = StatusFail
		r.Detail += " — IP marked as proxy"
	case mobile:
		r.Status = StatusWarn
	case hosting:
		r.Status = StatusInfo // normal for VPS
	default:
		r.Status = StatusOK
	}
	return r
}

func checkIPAPIAbuseHints(ctx context.Context, ip string) Result {
	start := time.Now()
	r := Result{ID: "ipq_org", TitleEN: "ASN / org", TitleRU: "ASN / org"}
	client := &http.Client{Timeout: 12 * time.Second}
	url := "http://ip-api.com/json/" + ip + "?fields=status,as,org,isp,reverse"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := client.Do(req)
	r.DurationMs = time.Since(start).Milliseconds()
	if err != nil {
		r.Status = StatusWarn
		r.Detail = err.Error()
		return r
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if fmt.Sprint(m["status"]) != "success" {
		r.Status = StatusWarn
		r.Detail = string(b)
		return r
	}
	r.Status = StatusOK
	r.Detail = fmt.Sprintf("as=%v org=%v isp=%v rdns=%v", m["as"], m["org"], m["isp"], m["reverse"])
	return r
}

// fetch ipify/ifconfig from this host — already in http checks; here compare advertised IP
func checkHTTPFromIP(ctx context.Context, expectIP string) Result {
	start := time.Now()
	r := Result{ID: "ipq_egress", TitleEN: "Egress IP match", TitleRU: "Исходящий IP"}
	client := &http.Client{Timeout: 12 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.ipify.org", nil)
	resp, err := client.Do(req)
	r.DurationMs = time.Since(start).Milliseconds()
	if err != nil {
		r.Status = StatusWarn
		r.Detail = err.Error()
		return r
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
	got := strings.TrimSpace(string(b))
	if got == expectIP {
		r.Status = StatusOK
		r.Detail = got
	} else {
		r.Status = StatusWarn
		r.Detail = fmt.Sprintf("expected %s got %s (NAT/proxy?)", expectIP, got)
	}
	return r
}
