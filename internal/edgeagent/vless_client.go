package edgeagent

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// ClientOpts controls edge sing-box client JSON generation.
type ClientOpts struct {
	Mode string // "tun" (default) or "socks"/"mixed"
	// SoftFallback: urltest proxy+direct — when secondary/VLESS is down, LAN uses ISP WAN.
	SoftFallback bool
	// PrimaryHost: force direct (agent mTLS / enroll to primary must not go through user VLESS).
	PrimaryHost string
	// DNSMode: "vpn" (default) hijack DNS + resolve via proxy; "wan" local/ISP; "off" no dns block.
	DNSMode string
}

// VLESSClientConfig builds sing-box client JSON (legacy: mode only).
func VLESSClientConfig(link string, mode string) ([]byte, error) {
	return VLESSClientConfigOpts(link, ClientOpts{Mode: mode, SoftFallback: true, DNSMode: "vpn"})
}

// VLESSClientConfigOpts builds sing-box client JSON for OpenWrt edge.
func VLESSClientConfigOpts(link string, opt ClientOpts) ([]byte, error) {
	link = strings.TrimSpace(link)
	if !strings.HasPrefix(link, "vless://") {
		return nil, fmt.Errorf("not vless link")
	}
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	uuid := u.User.Username()
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "443"
	}
	q := u.Query()
	sni := q.Get("sni")
	if sni == "" {
		sni = q.Get("serverName")
	}
	if sni == "" {
		sni = host
	}
	pbk := q.Get("pbk")
	sid := q.Get("sid")
	flow := q.Get("flow")
	fp := q.Get("fp")
	if fp == "" {
		fp = "chrome"
	}
	mode := opt.Mode
	if mode == "" {
		mode = "tun"
	}
	dnsMode := strings.ToLower(strings.TrimSpace(opt.DNSMode))
	if dnsMode == "" {
		dnsMode = "vpn"
	}
	soft := opt.SoftFallback
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_EDGE_SOFT_FALLBACK")); v == "0" || v == "false" {
		soft = false
	} else if v == "1" || v == "true" {
		soft = true
	}

	var inbounds []any
	if mode == "tun" {
		strict := !soft
		inbounds = []any{
			map[string]any{
				"type":                      "tun",
				"tag":                       "tun-in",
				"interface_name":            "nd-tun",
				"inet4_address":             "172.19.0.1/30",
				"auto_route":                true,
				"strict_route":              strict,
				"stack":                     "system",
				"sniff":                     true,
				"sniff_override_destination": true,
			},
		}
	} else {
		listen := "127.0.0.1"
		if v := strings.TrimSpace(os.Getenv("NETDUCTOR_EDGE_MIXED_LISTEN")); v != "" {
			listen = v
		}
		inbounds = []any{
			map[string]any{
				"type": "mixed", "tag": "mixed-in",
				"listen": listen, "listen_port": 7890,
			},
		}
	}

	outbounds := []any{
		map[string]any{
			"type":        "vless",
			"tag":         "proxy",
			"server":      host,
			"server_port": atoi(port),
			"uuid":        uuid,
			"flow":        flow,
			"tls": map[string]any{
				"enabled":     true,
				"server_name": sni,
				"utls":        map[string]any{"enabled": true, "fingerprint": fp},
				"reality": map[string]any{
					"enabled":    true,
					"public_key": pbk,
					"short_id":   sid,
				},
			},
		},
		map[string]any{"type": "direct", "tag": "direct"},
		map[string]any{"type": "block", "tag": "block"},
	}
	final := "proxy"
	if soft {
		outbounds = append(outbounds, map[string]any{
			"type":      "urltest",
			"tag":       "auto",
			"outbounds": []string{"proxy", "direct"},
			"url":       "https://www.gstatic.com/generate_204",
			"interval":  "3m",
			"tolerance": 150,
		})
		final = "auto"
	}

	routeRules := []any{
		map[string]any{"action": "sniff"},
	}
	if dnsMode == "vpn" {
		routeRules = append(routeRules, map[string]any{"protocol": "dns", "action": "hijack-dns"})
	}
	routeRules = append(routeRules, map[string]any{"ip_is_private": true, "outbound": "direct"})
	if host != "" {
		routeRules = append(routeRules, map[string]any{"ip_cidr": []string{host + "/32"}, "outbound": "direct"})
		routeRules = append(routeRules, map[string]any{"domain": []string{host}, "outbound": "direct"})
	}
	if ph := strings.TrimSpace(opt.PrimaryHost); ph != "" {
		routeRules = append(routeRules, map[string]any{"domain": []string{ph}, "outbound": "direct"})
		// only add /32 if looks like IPv4
		if strings.Count(ph, ".") == 3 && !strings.Contains(ph, ":") {
			routeRules = append(routeRules, map[string]any{"ip_cidr": []string{ph + "/32"}, "outbound": "direct"})
		}
	}
	routeRules = append(routeRules, map[string]any{"network": "udp", "port": 443, "outbound": "block"})

	cfg := map[string]any{
		"log":       map[string]any{"level": "warn"},
		"inbounds":  inbounds,
		"outbounds": outbounds,
		"route": map[string]any{
			"rules":                 routeRules,
			"final":                 final,
			"auto_detect_interface": true,
		},
	}
	if dnsMode == "vpn" {
		cfg["dns"] = map[string]any{
			"servers": []any{
				map[string]any{"type": "udp", "tag": "ya", "server": "77.88.8.8"},
				map[string]any{"type": "udp", "tag": "remote", "server": "9.9.9.9", "detour": final},
				map[string]any{"type": "udp", "tag": "remote2", "server": "1.1.1.1", "detour": final},
			},
			"rules": []any{
				map[string]any{"domain_suffix": []string{".ru", ".рф", ".su"}, "server": "ya"},
			},
			"final":             "remote",
			"strategy":          "ipv4_only",
			"independent_cache": true,
		}
	}
	return json.MarshalIndent(cfg, "", "  ")
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	if n == 0 {
		return 443
	}
	return n
}
