package vpscheck

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"
)

// DPI-oriented probes for RU secondary: TLS handshake with various SNIs,
// TCP to common VPN/Reality ports patterns, DNS pollution hints.

func checkDPI(ctx context.Context, profile Profile) []Result {
	if profile != ProfileSecondary && profile != ProfileGeneric {
		return []Result{{
			ID: "dpi_skip", TitleEN: "DPI suite", TitleRU: "DPI",
			Status: StatusSkip, Detail: "DPI suite primarily for secondary/RU profile",
		}}
	}
	var out []Result
	out = append(out, checkTLSSNI(ctx, "api.vk.me:443", "api.vk.me"))
	out = append(out, checkTLSSNI(ctx, "www.cloudflare.com:443", "www.cloudflare.com"))
	out = append(out, checkTLSSNI(ctx, "www.google.com:443", "www.google.com"))
	out = append(out, checkTLSSNI(ctx, "github.com:443", "github.com"))
	// DNS mismatch: resolve vs expected roughly
	out = append(out, checkDNSPollution(ctx, "www.google.com"))
	out = append(out, checkDNSPollution(ctx, "twitter.com"))
	out = append(out, checkDNSPollution(ctx, "youtube.com"))
	return out
}

func checkTLSSNI(ctx context.Context, addr, sni string) Result {
	start := time.Now()
	r := Result{ID: "dpi_tls_" + sanitizeID(sni), TitleEN: "TLS SNI " + sni, TitleRU: "TLS SNI " + sni}
	d := net.Dialer{Timeout: 8 * time.Second}
	raw, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		r.Status = StatusWarn
		r.Detail = "tcp: " + err.Error()
		r.DurationMs = time.Since(start).Milliseconds()
		return r
	}
	defer raw.Close()
	cfg := &tls.Config{ServerName: sni, InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}
	conn := tls.Client(raw, cfg)
	_ = conn.SetDeadline(time.Now().Add(8 * time.Second))
	err = conn.HandshakeContext(ctx)
	r.DurationMs = time.Since(start).Milliseconds()
	if err != nil {
		r.Status = StatusWarn
		r.Detail = "handshake: " + err.Error()
		return r
	}
	state := conn.ConnectionState()
	r.Status = StatusOK
	r.Detail = fmt.Sprintf("ok ver=%x %dms", state.Version, r.DurationMs)
	_ = conn.Close()
	return r
}

func checkDNSPollution(ctx context.Context, name string) Result {
	start := time.Now()
	r := Result{ID: "dpi_dns_" + sanitizeID(name), TitleEN: "DNS " + name, TitleRU: "DNS " + name}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", name)
	r.DurationMs = time.Since(start).Milliseconds()
	if err != nil {
		r.Status = StatusWarn
		r.Detail = err.Error()
		return r
	}
	var ss []string
	bogus := false
	for _, ip := range ips {
		ss = append(ss, ip.String())
		// classic pollution: 127/0/private for public names
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() {
			bogus = true
		}
	}
	if bogus {
		r.Status = StatusFail
		r.Detail = "suspicious IPs: " + strings.Join(ss, ",")
	} else {
		r.Status = StatusOK
		r.Detail = strings.Join(ss, ",")
	}
	return r
}

func sanitizeID(s string) string {
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, "/", "_")
	return s
}
