package vpn

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/PavelNeyman/netductor/internal/revbridge"
)

// Reverse uplink: Primary initiates underlay toward Secondary; client traffic
// enters on Secondary and is injected into those sessions toward Primary.
//
// Paths:
//   - wg: sing-box WireGuard (listen S, dial P) — no revbridge
//   - ws/grpc/anytls/nomux: revbridge mux (P dials S public port; S local port for sing-box)

const (
	revSteerWG     = "wg"
	revSteerWS     = "ws"
	revSteerGRPC   = "grpc"
	revSteerAnyTLS = "anytls"
	revSteerNomux  = "nomux"
)

func knownReverseUplinkOrder() []string {
	return []string{revSteerWG, revSteerWS, revSteerGRPC, revSteerAnyTLS, revSteerNomux}
}

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

func revKindsFromConfig(cfg UplinkABConfig) []revbridge.Kind {
	list := cfg.ReverseProfiles
	if len(list) == 0 {
		list = knownReverseUplinkOrder()
	}
	return revbridge.ParseKinds(list)
}

// ResolveUplinkABReverseProfiles — client steer ports on Secondary.
func ResolveUplinkABReverseProfiles(cfg UplinkABConfig) []UplinkABProfile {
	if !cfg.Enabled || !cfg.ReverseEnabled {
		return nil
	}
	want := map[string]bool{}
	for _, id := range cfg.ReverseProfiles {
		want[strings.ToLower(strings.TrimSpace(id))] = true
	}
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
				InboundTag: "ab-in-rev-wg", OutboundTag: "uplink-ab-rev-wg",
				Port: port, Fragment: "nd-up-rev-wg", PrimaryPort: cfg.WGRevListenPort,
			})
			port++
		case revSteerWS, revSteerGRPC, revSteerAnyTLS, revSteerNomux:
			plan, ok := revbridge.PlanFor(revbridge.Kind(id))
			if !ok {
				continue
			}
			out = append(out, UplinkABProfile{
				ID: id, Reverse: true,
				InboundTag: "ab-in-rev-" + id, OutboundTag: "uplink-ab-rev-" + id,
				Port: port, Fragment: "nd-up-rev-" + id, PrimaryPort: plan.PublicPort,
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
		inbounds = append(inbounds, map[string]any{
			"type": "vless", "tag": p.InboundTag, "listen": "::", "listen_port": p.Port,
			"users": abUsers,
			"tls": map[string]any{
				"enabled": true, "server_name": b.SecondarySNI,
				"reality": map[string]any{
					"enabled": true,
					"handshake": map[string]any{"server": b.SecondarySNI, "server_port": 443},
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
	uplinkTLS := buildSecondaryUplinkTLS(b)
	for _, p := range profs {
		if p.ID == revSteerWG {
			continue // endpoint
		}
		plan, ok := revbridge.PlanFor(revbridge.Kind(p.ID))
		if !ok {
			continue
		}
		// Dial local revbridge; through tunnel Primary terminates real underlay.
		switch p.ID {
		case revSteerWS:
			outbounds = append(outbounds, map[string]any{
				"type": "vless", "tag": p.OutboundTag,
				"server": "127.0.0.1", "server_port": plan.LocalPort,
				"uuid": b.UplinkUUID,
				"transport": map[string]any{"type": "ws", "path": "/nd-up-ws"},
				"tls": map[string]any{"enabled": true, "server_name": b.CoreSNI, "insecure": true},
			})
		case revSteerGRPC:
			outbounds = append(outbounds, map[string]any{
				"type": "vless", "tag": p.OutboundTag,
				"server": "127.0.0.1", "server_port": plan.LocalPort,
				"uuid": b.UplinkUUID,
				"transport": map[string]any{"type": "grpc", "service_name": "nd-up-grpc"},
				"tls": map[string]any{"enabled": true, "server_name": b.CoreSNI, "insecure": true},
			})
		case revSteerAnyTLS:
			outbounds = append(outbounds, map[string]any{
				"type": "anytls", "tag": p.OutboundTag,
				"server": "127.0.0.1", "server_port": plan.LocalPort,
				"password": b.UplinkUUID,
				"tls": map[string]any{"enabled": true, "server_name": b.CoreSNI, "insecure": true},
			})
		case revSteerNomux:
			port := b.CoreVless
			if port <= 0 {
				port = 443
			}
			_ = port // actual port is primary side of bridge (443); local is plan.LocalPort
			outbounds = append(outbounds, map[string]any{
				"type": "vless", "tag": p.OutboundTag,
				"server": "127.0.0.1", "server_port": plan.LocalPort,
				"uuid": b.UplinkUUID, "tls": uplinkTLS,
			})
		}
	}
	return outbounds
}

func uplinkABReverseRouteRules() []any {
	profs := ResolveUplinkABReverseProfiles(LoadUplinkABConfig())
	var rules []any
	for _, p := range profs {
		rules = append(rules, map[string]any{
			"inbound":  []string{p.InboundTag},
			"outbound": p.OutboundTag,
		})
	}
	return rules
}

func secondaryUplinkABReverseEndpoints() []any {
	cfg := LoadUplinkABConfig()
	if !cfg.Enabled || !cfg.ReverseEnabled || !reverseWGReady(cfg) {
		return nil
	}
	has := false
	for _, p := range ResolveUplinkABReverseProfiles(cfg) {
		if p.ID == revSteerWG {
			has = true
			break
		}
	}
	if !has {
		return nil
	}
	return []any{
		map[string]any{
			"type": "wireguard", "tag": "uplink-ab-rev-wg", "system": false, "name": "nd-ab-rev-wg-s",
			"private_key": cfg.WGRevSecondaryPrivateKey, "listen_port": cfg.WGRevListenPort,
			"address": []string{cfg.WGRevSecondaryAddress}, "mtu": cfg.WGMTU,
			"peers": []any{
				map[string]any{"public_key": cfg.WGRevPrimaryPublicKey, "allowed_ips": []string{"0.0.0.0/0"}},
			},
		},
	}
}

func primaryUplinkABReverseEndpoints() []any {
	cfg := LoadUplinkABConfig()
	if !cfg.Enabled || !cfg.ReverseEnabled || !reverseWGReady(cfg) {
		return nil
	}
	has := false
	for _, p := range ResolveUplinkABReverseProfiles(cfg) {
		if p.ID == revSteerWG {
			has = true
			break
		}
	}
	if !has {
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
			"type": "wireguard", "tag": "ab-rev-wg", "system": false, "name": "nd-ab-rev-wg-p",
			"private_key": cfg.WGRevPrimaryPrivateKey, "address": []string{cfg.WGRevPrimaryAddress}, "mtu": cfg.WGMTU,
			"peers": []any{
				map[string]any{
					"address": h, "port": pt, "public_key": cfg.WGRevSecondaryPublicKey,
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
	for _, k := range revKindsFromConfig(cfg) {
		if plan, ok := revbridge.PlanFor(k); ok {
			_ = exec.Command("ufw", "allow", fmt.Sprintf("%d/tcp", plan.PublicPort), "comment", "nd-revbridge").Run()
		}
	}
}

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
		links = append(links, fmt.Sprintf(
			"vless://%s@%s:%d?encryption=none&flow=xtls-rprx-vision&security=reality&sni=%s&fp=firefox&pbk=%s&sid=%s&type=tcp#%s",
			uuid, host, p.Port, sniName, pbk, sid, p.Fragment,
		))
	}
	return links
}

// EnsureReverseBridge starts in-process bridge on this host (primary client or secondary server).
func EnsureReverseBridge() {
	cfg := LoadUplinkABConfig()
	if !cfg.Enabled || !cfg.ReverseEnabled {
		return
	}
	kinds := revKindsFromConfig(cfg)
	if isLikelyPrimaryHost() {
		host := ""
		if h, _, err := net.SplitHostPort(cfg.WGRevEndpoint); err == nil {
			host = h
		}
		if host == "" {
			host = strings.TrimSpace(cfg.WGRevEndpoint)
		}
		if host == "" {
			for _, sec := range loadOnlineSecondarys() {
				ip := strings.TrimSpace(sec.PublicIP)
				if ip == "" {
					ip = strings.TrimSpace(sec.Host)
				}
				if ip != "" && net.ParseIP(ip) != nil {
					host = ip
					break
				}
			}
		}
		if host == "" {
			fmt.Fprintln(os.Stderr, "revbridge: primary: no secondary host yet")
			return
		}
		if err := revbridge.StartClientPrimary(host, kinds); err != nil {
			fmt.Fprintln(os.Stderr, "revbridge primary:", err)
		}
		return
	}
	if err := revbridge.StartServerSecondary(kinds); err != nil {
		fmt.Fprintln(os.Stderr, "revbridge secondary:", err)
	}
}
