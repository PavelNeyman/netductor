package vpn

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// Reverse uplink A/B (operator allowlist only).
//
// Model (honest vs S→P forward matrix):
//   Client → S:9544+ (VLESS Reality steer, same as forward)
//   L3: WireGuard listen on Secondary; Primary dials S (P→S initiator).
//   TCP underlays (ws/grpc/anytls/nomux/ssh): S dials Primary services via
//   reverse WG VIP (10.87.91.1), not public Core IP — so inter-VPS path is
//   the P-initiated WG, not a new S→P TCP to foreign :10443.
//
// Forward nd-up-* profiles are unchanged.

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

// ResolveUplinkABReverseProfiles — steer ports on S + tags for reverse matrix.
func ResolveUplinkABReverseProfiles(cfg UplinkABConfig) []UplinkABProfile {
	if !cfg.Enabled || !cfg.ReverseEnabled {
		return nil
	}
	if !reverseWGReady(cfg) {
		return nil
	}
	want := map[string]bool{}
	for _, id := range cfg.ReverseProfiles {
		want[strings.ToLower(strings.TrimSpace(id))] = true
	}
	var out []UplinkABProfile
	port := cfg.ReverseBasePort
	for _, id := range knownUplinkABOrder() {
		if !want[id] {
			continue
		}
		p := UplinkABProfile{
			ID: id, Reverse: true,
			InboundTag:  "ab-in-rev-" + id,
			OutboundTag: "uplink-ab-rev-" + id,
			Port:        port,
			Fragment:    "nd-up-rev-" + id,
		}
		switch id {
		case "nomux":
			p.PrimaryPort = 0
		case "ws":
			p.PrimaryPort = primaryABPortWS
		case "grpc":
			p.PrimaryPort = primaryABPortGRPC
		case "anytls":
			p.PrimaryPort = primaryABPortAnyTLS
		case "wg":
			p.PrimaryPort = cfg.WGRevListenPort
			p.OutboundTag = "uplink-ab-rev-wg" // endpoint tag
		case "ssh":
			p.PrimaryPort = cfg.SSHPort
			if p.PrimaryPort <= 0 {
				p.PrimaryPort = 22
			}
		}
		out = append(out, p)
		port++
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

func appendUplinkABReverseOutbounds(outbounds []any, b *SecondaryBundle) []any {
	cfg := LoadUplinkABConfig()
	profs := ResolveUplinkABReverseProfiles(cfg)
	if len(profs) == 0 {
		return outbounds
	}
	vip := revPrimaryVIP(cfg)
	if vip == "" || net.ParseIP(vip) == nil {
		return outbounds
	}
	uplinkTLS := buildSecondaryUplinkTLS(b)
	// Point TLS server_name still at core SNI; dial IP is WG VIP.
	for _, p := range profs {
		if p.ID == "wg" {
			continue // endpoint
		}
		var ob map[string]any
		switch p.ID {
		case "nomux":
			port := b.CoreVless
			if port <= 0 {
				port = 443
			}
			ob = map[string]any{
				"type": "vless", "tag": p.OutboundTag,
				"server": vip, "server_port": port,
				"uuid": b.UplinkUUID, "tls": uplinkTLS,
			}
		case "anytls":
			ob = map[string]any{
				"type": "anytls", "tag": p.OutboundTag,
				"server": vip, "server_port": primaryABPortAnyTLS,
				"password": b.UplinkUUID,
				"tls": map[string]any{
					"enabled": true, "server_name": b.CoreSNI, "insecure": true,
				},
			}
		case "ws":
			ob = map[string]any{
				"type": "vless", "tag": p.OutboundTag,
				"server": vip, "server_port": primaryABPortWS,
				"uuid": b.UplinkUUID,
				"transport": map[string]any{"type": "ws", "path": "/nd-up-ws"},
				"tls": map[string]any{
					"enabled": true, "server_name": b.CoreSNI, "insecure": true,
				},
			}
		case "grpc":
			ob = map[string]any{
				"type": "vless", "tag": p.OutboundTag,
				"server": vip, "server_port": primaryABPortGRPC,
				"uuid": b.UplinkUUID,
				"transport": map[string]any{"type": "grpc", "service_name": "nd-up-grpc"},
				"tls": map[string]any{
					"enabled": true, "server_name": b.CoreSNI, "insecure": true,
				},
			}
		case "ssh":
			if cfg.SSHUser == "" || cfg.SSHPrivateKey == "" {
				continue
			}
			ob = map[string]any{
				"type": "ssh", "tag": p.OutboundTag,
				"server": vip, "server_port": p.PrimaryPort,
				"user": cfg.SSHUser, "private_key": cfg.SSHPrivateKey,
			}
		default:
			continue
		}
		outbounds = append(outbounds, ob)
	}
	return outbounds
}

func uplinkABReverseRouteRules() []any {
	profs := ResolveUplinkABReverseProfiles(LoadUplinkABConfig())
	var rules []any
	for _, p := range profs {
		ob := p.OutboundTag
		rules = append(rules, map[string]any{
			"inbound":  []string{p.InboundTag},
			"outbound": ob,
		})
	}
	return rules
}

// secondaryUplinkABReverseEndpoints — WG listen on S; Primary dials (no peer endpoint here).
func secondaryUplinkABReverseEndpoints() []any {
	cfg := LoadUplinkABConfig()
	if !cfg.Enabled || !cfg.ReverseEnabled || !reverseWGReady(cfg) {
		return nil
	}
	if !reverseProfileWants(cfg, "wg") && len(ResolveUplinkABReverseProfiles(cfg)) == 0 {
		return nil
	}
	// Always create reverse WG endpoint if any reverse profile needs the VIP path.
	if len(ResolveUplinkABReverseProfiles(cfg)) == 0 {
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

// primaryUplinkABReverseEndpoints — Primary dials Secondary:revWGPort (P→S initiator).
func primaryUplinkABReverseEndpoints() []any {
	cfg := LoadUplinkABConfig()
	if !cfg.Enabled || !cfg.ReverseEnabled || !reverseWGReady(cfg) {
		return nil
	}
	if len(ResolveUplinkABReverseProfiles(cfg)) == 0 {
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
					// Only secondary VIP — never 0.0.0.0/0 on primary (would blackhole default route).
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
		return // primary dials out; no extra listen for rev WG
	}
	// Secondary: client steer ports + rev WG listen
	for _, p := range ResolveUplinkABReverseProfiles(cfg) {
		_ = exec.Command("ufw", "allow", fmt.Sprintf("%d/tcp", p.Port), "comment", "nd-uplink-ab-rev-steer").Run()
	}
	if reverseWGReady(cfg) {
		_ = exec.Command("ufw", "allow", fmt.Sprintf("%d/udp", cfg.WGRevListenPort), "comment", "nd-uplink-ab-rev-wg").Run()
	}
}

// UplinkABReverseSubscriptionLinks — nd-up-rev-* for allowlisted user only.
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
	var links []string
	for _, p := range profs {
		links = append(links, fmt.Sprintf(
			"vless://%s@%s:%d?encryption=none&flow=xtls-rprx-vision&security=reality&sni=%s&fp=firefox&pbk=%s&sid=%s&type=tcp#%s",
			uuid, host, p.Port, sniName, pbk, sid, p.Fragment,
		))
	}
	return links
}
