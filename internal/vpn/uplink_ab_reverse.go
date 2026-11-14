package vpn

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// Reverse uplink A/B — only underlays that are truly P→S on the wire.
//
// Plan (PLAN-UPLINK-AB-PROFILE): forward matrix stays S→P dial.
// Reverse is for A/B "who initiates inter-VPS path", not labels on one WG.
//
// Real reverse paths shipped:
//
//  1. rev-wg — WireGuard listen on Secondary; Primary dials S (endpoint on P).
//     Client → S:reverseBasePort → endpoint uplink-ab-rev-wg → P → egress.
//
//  2. rev-ps — user bulk via existing service-plane PS (Primary-initiated WSS/WG
//     between P and S). Outbound VLESS to primary PS VIP when material is present.
//     Outer path is the same PS plane already used for control; initiator is P.
//
// Fake "rev-ws / rev-grpc / … inside reverse WG" were removed: outside the
// path is still one WG and A/B of transport is meaningless for DPI tests.
// True reverse-WS/gRPC (P dials S, S injects user into that session) needs a
// reverse-bridge (frp/ssh -R/custom mux) — not sing-box outbound flip.

const (
	revSteerWG = "wg"
	revSteerPS = "ps"
)

func reverseProfileWants(cfg UplinkABConfig, id string) bool {
	if !cfg.Enabled || !cfg.ReverseEnabled {
		return false
	}
	id = strings.ToLower(strings.TrimSpace(id))
	for _, p := range cfg.ReverseProfiles {
		if strings.EqualFold(strings.TrimSpace(p), id) {
			return true
		}
	}
	return false
}

func reverseWGReady(cfg UplinkABConfig) bool {
	return cfg.WGRevPrimaryPrivateKey != "" && cfg.WGRevSecondaryPrivateKey != "" &&
		cfg.WGRevPrimaryPublicKey != "" && cfg.WGRevSecondaryPublicKey != ""
}

func revPrimaryVIP(cfg UplinkABConfig) string {
	a := cfg.WGRevPrimaryAddress
	if i := strings.IndexByte(a, '/'); i > 0 {
		return a[:i]
	}
	return strings.TrimSpace(a)
}

// knownReverseUplinkOrder — only real P-initiated paths.
func knownReverseUplinkOrder() []string {
	return []string{revSteerWG, revSteerPS}
}

// ResolveUplinkABReverseProfiles — steer ports on S for real reverse paths only.
func ResolveUplinkABReverseProfiles(cfg UplinkABConfig) []UplinkABProfile {
	if !cfg.Enabled || !cfg.ReverseEnabled {
		return nil
	}
	want := map[string]bool{}
	for _, id := range cfg.ReverseProfiles {
		want[strings.ToLower(strings.TrimSpace(id))] = true
	}
	// Default when reverse on but list empty / legacy junk: only real paths.
	if len(want) == 0 {
		for _, id := range knownReverseUplinkOrder() {
			want[id] = true
		}
	}
	var out []UplinkABProfile
	port := cfg.ReverseBasePort
	if port < 1024 {
		port = uplinkABReverseBasePort
	}
	for _, id := range knownReverseUplinkOrder() {
		if !want[id] {
			continue
		}
		switch id {
		case revSteerWG:
			if !reverseWGReady(cfg) {
				continue
			}
			out = append(out, UplinkABProfile{
				ID: id, Reverse: true,
				InboundTag:  "ab-in-rev-wg",
				OutboundTag: "uplink-ab-rev-wg",
				Port:        port,
				Fragment:    "nd-up-rev-wg",
				PrimaryPort: cfg.WGRevListenPort,
			})
			port++
		case revSteerPS:
			// PS VIP path; material checked when building outbound.
			out = append(out, UplinkABProfile{
				ID: id, Reverse: true,
				InboundTag:  "ab-in-rev-ps",
				OutboundTag: "uplink-ab-rev-ps",
				Port:        port,
				Fragment:    "nd-up-rev-ps",
				PrimaryPort: 0,
			})
			port++
		}
	}
	return out
}

func appendUplinkABReverseInbounds(inbounds []any, b *SecondaryBundle, privKey, shortID string, allUsers []secondaryUser) []any {
	cfg := LoadUplinkABConfig()
	profs := ResolveUplinkABReverseProfiles(cfg)
	if len(profs) == 0 {
		return inbounds
	}
	var abUsers []secondaryUser
	for _, u := range allUsers {
		if UplinkABUserAllowed(u.Name) {
			abUsers = append(abUsers, u)
		}
	}
	if len(abUsers) == 0 {
		return inbounds
	}
	for _, p := range profs {
		// Skip PS inbound if we cannot build outbound (no VIP).
		if p.ID == revSteerPS {
			if _, _, ok := reversePSDialTarget(b); !ok {
				continue
			}
		}
		inbounds = append(inbounds, map[string]any{
			"type": "vless", "tag": p.InboundTag, "listen": "::", "listen_port": p.Port,
			"users": abUsers,
			"tls": map[string]any{
				"enabled": true, "server_name": b.SecondarySNI,
				"reality": map[string]any{
					"enabled": true,
					"handshake": map[string]any{
						"server": b.SecondarySNI, "server_port": 443,
					},
					"private_key": privKey,
					"short_id":    []string{shortID},
				},
			},
		})
	}
	return inbounds
}

// reversePSDialTarget — primary address reachable over P-initiated PS plane.
func reversePSDialTarget(b *SecondaryBundle) (host string, port int, ok bool) {
	// Slot-0 PS primary VIP (svcpaths.AddrP_PS). Hardcoded to avoid vpn↔svcpaths import cycle.
	host = "10.87.11.1"
	if net.ParseIP(host) == nil {
		return "", 0, false
	}
	port = 443
	if b != nil && b.CoreVless > 0 {
		port = b.CoreVless
	}
	return host, port, true
}

func appendUplinkABReverseOutbounds(outbounds []any, b *SecondaryBundle) []any {
	cfg := LoadUplinkABConfig()
	profs := ResolveUplinkABReverseProfiles(cfg)
	if len(profs) == 0 {
		return outbounds
	}
	uplinkTLS := buildSecondaryUplinkTLS(b)
	for _, p := range profs {
		switch p.ID {
		case revSteerWG:
			// endpoint, not outbound
			continue
		case revSteerPS:
			host, port, ok := reversePSDialTarget(b)
			if !ok || b.UplinkUUID == "" {
				continue
			}
			outbounds = append(outbounds, map[string]any{
				"type": "vless", "tag": p.OutboundTag,
				"server": host, "server_port": port,
				"uuid": b.UplinkUUID, "tls": uplinkTLS,
			})
		}
	}
	return outbounds
}

func uplinkABReverseRouteRules() []any {
	cfg := LoadUplinkABConfig()
	profs := ResolveUplinkABReverseProfiles(cfg)
	var rules []any
	for _, p := range profs {
		if p.ID == revSteerPS {
			if _, _, ok := reversePSDialTarget(nil); !ok {
				continue
			}
		}
		rules = append(rules, map[string]any{
			"inbound":  []string{p.InboundTag},
			"outbound": p.OutboundTag,
		})
	}
	return rules
}

// secondaryUplinkABReverseEndpoints — WG listen on S; Primary dials (no peer endpoint).
func secondaryUplinkABReverseEndpoints() []any {
	cfg := LoadUplinkABConfig()
	if !cfg.Enabled || !cfg.ReverseEnabled || !reverseWGReady(cfg) {
		return nil
	}
	if !reverseProfileWants(cfg, revSteerWG) && !reverseListImpliesWG(cfg) {
		return nil
	}
	// Always ship endpoint if reverse-wg profile resolves.
	ok := false
	for _, p := range ResolveUplinkABReverseProfiles(cfg) {
		if p.ID == revSteerWG {
			ok = true
			break
		}
	}
	if !ok {
		return nil
	}
	return []any{
		map[string]any{
			"type":        "wireguard",
			"tag":         "uplink-ab-rev-wg",
			"system":      false,
			"name":        "nd-ab-rev-wg-s",
			"private_key": cfg.WGRevSecondaryPrivateKey,
			"listen_port": cfg.WGRevListenPort,
			"address":     []string{cfg.WGRevSecondaryAddress},
			"mtu":         cfg.WGMTU,
			"peers": []any{
				map[string]any{
					"public_key":  cfg.WGRevPrimaryPublicKey,
					"allowed_ips": []string{"0.0.0.0/0"},
				},
			},
		},
	}
}

func reverseListImpliesWG(cfg UplinkABConfig) bool {
	if len(cfg.ReverseProfiles) == 0 {
		return true
	}
	return reverseProfileWants(cfg, revSteerWG)
}

// primaryUplinkABReverseEndpoints — Primary dials Secondary (P→S initiator).
func primaryUplinkABReverseEndpoints() []any {
	cfg := LoadUplinkABConfig()
	if !cfg.Enabled || !cfg.ReverseEnabled || !reverseWGReady(cfg) {
		return nil
	}
	hasWG := false
	for _, p := range ResolveUplinkABReverseProfiles(cfg) {
		if p.ID == revSteerWG {
			hasWG = true
			break
		}
	}
	if !hasWG {
		return nil
	}
	h, pt := splitHostPortDefault(cfg.WGRevEndpoint, cfg.WGRevListenPort)
	if h == "" || net.ParseIP(h) == nil {
		return nil
	}
	if pt <= 0 {
		pt = cfg.WGRevListenPort
	}
	peerIP := cfg.WGRevSecondaryAddress
	if i := strings.IndexByte(peerIP, '/'); i > 0 {
		peerIP = peerIP[:i] + "/32"
	} else if peerIP != "" && !strings.Contains(peerIP, "/") {
		peerIP = peerIP + "/32"
	}
	return []any{
		map[string]any{
			"type":        "wireguard",
			"tag":         "ab-rev-wg",
			"system":      false,
			"name":        "nd-ab-rev-wg-p",
			"private_key": cfg.WGRevPrimaryPrivateKey,
			"address":     []string{cfg.WGRevPrimaryAddress},
			"mtu":         cfg.WGMTU,
			"peers": []any{
				map[string]any{
					"address":     h,
					"port":        pt,
					"public_key":  cfg.WGRevSecondaryPublicKey,
					"allowed_ips": []string{peerIP},
				},
			},
		},
	}
}

func openUplinkABReverseFirewall() {
	if _, err := exec.LookPath("ufw"); err != nil {
		return
	}
	cfg := LoadUplinkABConfig()
	if !cfg.Enabled || !cfg.ReverseEnabled {
		return
	}
	if isLikelyPrimaryHost() {
		return
	}
	for _, p := range ResolveUplinkABReverseProfiles(cfg) {
		_ = exec.Command("ufw", "allow", fmt.Sprintf("%d/tcp", p.Port), "comment", "nd-uplink-ab-rev-steer").Run()
	}
	if reverseWGReady(cfg) {
		_ = exec.Command("ufw", "allow", fmt.Sprintf("%d/udp", cfg.WGRevListenPort), "comment", "nd-uplink-ab-rev-wg").Run()
	}
}

// UplinkABReverseSubscriptionLinks — nd-up-rev-wg / nd-up-rev-ps only.
func UplinkABReverseSubscriptionLinks(name, uuid, relayIP, pbk, sid, sniName string) []string {
	if !UplinkABUserAllowed(name) {
		return nil
	}
	_ = EnsureUplinkABMaterial()
	cfg := LoadUplinkABConfig()
	profs := ResolveUplinkABReverseProfiles(cfg)
	if len(profs) == 0 {
		return nil
	}
	host := strings.TrimSpace(relayIP)
	if host == "" {
		return nil
	}
	if sniName == "" {
		sniName = DefaultRealitySNI
	}
	var links []string
	for _, p := range profs {
		if p.ID == revSteerPS {
			if _, _, ok := reversePSDialTarget(nil); !ok {
				continue
			}
		}
		links = append(links, fmt.Sprintf(
			"vless://%s@%s:%d?encryption=none&flow=xtls-rprx-vision&security=reality&sni=%s&fp=firefox&pbk=%s&sid=%s&type=tcp#%s",
			uuid, host, p.Port, sniName, pbk, sid, p.Fragment,
		))
	}
	return links
}
