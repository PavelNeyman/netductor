package vpn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Uplink AB: operator-only experimental S→P underlays (PLAN-UPLINK-AB-PROFILE).

const (
	uplinkABEnv       = "NETDUCTOR_UPLINK_AB"
	uplinkABUsersEnv  = "NETDUCTOR_UPLINK_AB_USERS"
	uplinkABBasePort  = 9444
)

// UplinkABProfile is one experimental path id.
type UplinkABProfile struct {
	ID          string // nomux, wg, anytls, ws, grpc, ssh
	InboundTag  string
	OutboundTag string
	Port        int
	Fragment    string // URI #name
}

// UplinkABConfig file + env.
type UplinkABConfig struct {
	Enabled  bool     `json:"enabled"`
	Users    []string `json:"users"`    // allowlist; empty → operator only when enabled
	Profiles []string `json:"profiles"` // subset of known ids; empty → default set
	BasePort int      `json:"base_port"`
	// Optional WG material (Phase 2); if missing, wg profile is skipped at generate time.
	WGPrivateKey     string `json:"wg_private_key,omitempty"`
	WGPeerPublicKey  string `json:"wg_peer_public_key,omitempty"`
	WGLocalAddress   string `json:"wg_local_address,omitempty"` // e.g. 10.87.90.2/30
	WGPeerAddress    string `json:"wg_peer_address,omitempty"`  // 10.87.90.1
	WGEndpoint       string `json:"wg_endpoint,omitempty"`      // host:port of primary
	WGMTU            int    `json:"wg_mtu,omitempty"`
	SSHUser          string `json:"ssh_user,omitempty"`
	SSHPrivateKeyPath string `json:"ssh_private_key_path,omitempty"`
	SSHHost          string `json:"ssh_host,omitempty"`
	SSHPort          int    `json:"ssh_port,omitempty"`
}

func uplinkABConfigPath() string {
	return filepath.Join(paths.EtcDir(), "uplink_ab.json")
}

// LoadUplinkABConfig merges file + env (env wins for enabled/users).
func LoadUplinkABConfig() UplinkABConfig {
	cfg := UplinkABConfig{
		Profiles: []string{"nomux", "wg", "anytls", "ws", "grpc", "ssh"},
		BasePort: uplinkABBasePort,
	}
	if b, err := os.ReadFile(uplinkABConfigPath()); err == nil {
		_ = json.Unmarshal(b, &cfg)
	}
	if v := strings.TrimSpace(os.Getenv(uplinkABEnv)); v != "" {
		cfg.Enabled = v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "on")
	}
	if v := strings.TrimSpace(os.Getenv(uplinkABUsersEnv)); v != "" {
		cfg.Users = nil
		for _, p := range strings.Split(v, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				cfg.Users = append(cfg.Users, p)
			}
		}
	}
	if cfg.BasePort < 1024 || cfg.BasePort > 60000 {
		cfg.BasePort = uplinkABBasePort
	}
	if len(cfg.Profiles) == 0 {
		cfg.Profiles = []string{"nomux", "wg", "anytls", "ws", "grpc", "ssh"}
	}
	return cfg
}

// UplinkABEnabled reports feature flag.
func UplinkABEnabled() bool {
	return LoadUplinkABConfig().Enabled
}

// UplinkABUserAllowed reports whether name gets exp URIs / may use exp inbounds.
func UplinkABUserAllowed(name string) bool {
	cfg := LoadUplinkABConfig()
	if !cfg.Enabled {
		return false
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	users := cfg.Users
	if len(users) == 0 {
		// default allowlist: operator only
		return strings.EqualFold(name, "operator")
	}
	for _, u := range users {
		if strings.EqualFold(strings.TrimSpace(u), name) {
			return true
		}
	}
	return false
}

func knownUplinkABOrder() []string {
	return []string{"nomux", "wg", "anytls", "ws", "grpc", "ssh"}
}

// ResolveUplinkABProfiles returns concrete profiles that can be generated now.
func ResolveUplinkABProfiles(cfg UplinkABConfig) []UplinkABProfile {
	if !cfg.Enabled {
		return nil
	}
	want := map[string]bool{}
	for _, id := range cfg.Profiles {
		want[strings.ToLower(strings.TrimSpace(id))] = true
	}
	var out []UplinkABProfile
	port := cfg.BasePort
	for _, id := range knownUplinkABOrder() {
		if !want[id] {
			continue
		}
		if id == "wg" && (cfg.WGPrivateKey == "" || cfg.WGPeerPublicKey == "" || cfg.WGEndpoint == "") {
			continue // material required
		}
		if id == "ssh" && (cfg.SSHHost == "" || cfg.SSHUser == "") {
			continue
		}
		p := UplinkABProfile{
			ID:          id,
			InboundTag:  "ab-in-" + id,
			OutboundTag: "uplink-ab-" + id,
			Port:        port,
			Fragment:    "nd-up-" + id,
		}
		out = append(out, p)
		port++
	}
	return out
}

// StatusUplinkAB is CLI/JSON summary.
func StatusUplinkAB() map[string]any {
	cfg := LoadUplinkABConfig()
	profs := ResolveUplinkABProfiles(cfg)
	ids := make([]string, 0, len(profs))
	for _, p := range profs {
		ids = append(ids, fmt.Sprintf("%s:%d", p.ID, p.Port))
	}
	return map[string]any{
		"enabled":  cfg.Enabled,
		"users":    cfg.Users,
		"profiles": ids,
		"base_port": cfg.BasePort,
		"config":   uplinkABConfigPath(),
	}
}

// appendUplinkABInbounds adds steer inbounds (allowlisted users only).
func appendUplinkABInbounds(inbounds []any, b *SecondaryBundle, privKey, shortID string, allUsers []secondaryUser) []any {
	cfg := LoadUplinkABConfig()
	profs := ResolveUplinkABProfiles(cfg)
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
		// still allow by UUID list empty → skip inbounds (no one allowed)
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

// appendUplinkABOutbounds adds experimental S→P outbounds.
func appendUplinkABOutbounds(outbounds []any, b *SecondaryBundle) []any {
	cfg := LoadUplinkABConfig()
	profs := ResolveUplinkABProfiles(cfg)
	if len(profs) == 0 {
		return outbounds
	}
	uplinkTLS := buildSecondaryUplinkTLS(b)
	coreHost := primarySPAddrForSecondary()
	if coreHost == "" {
		coreHost = strings.TrimSpace(b.CoreIP)
	}
	for _, p := range profs {
		var ob map[string]any
		switch p.ID {
		case "nomux":
			ob = map[string]any{
				"type": "vless", "tag": p.OutboundTag,
				"server": coreHost, "server_port": b.CoreVless,
				"uuid": b.UplinkUUID, "tls": uplinkTLS,
				// no multiplex
			}
		case "wg":
			mtu := cfg.WGMTU
			if mtu <= 0 {
				mtu = 1280
			}
			local := cfg.WGLocalAddress
			if local == "" {
				local = "10.87.90.2/30"
			}
			ob = map[string]any{
				"type": "wireguard", "tag": p.OutboundTag,
				"private_key": cfg.WGPrivateKey,
				"local_address": []string{local},
				"peers": []any{
					map[string]any{
						"public_key": cfg.WGPeerPublicKey,
						"allowed_ips": []string{"0.0.0.0/0"},
						"endpoint":   cfg.WGEndpoint,
					},
				},
				"mtu": mtu,
			}
		case "anytls":
			// AnyTLS outbound toward primary public IP:443-ish — uses core host + password=uplink UUID as placeholder
			// Primary must expose anytls inbound when this is used; otherwise dial fails (expected until P side ready).
			ob = map[string]any{
				"type": "anytls", "tag": p.OutboundTag,
				"server": coreHost, "server_port": b.CoreVless,
				"password": b.UplinkUUID,
				"tls": map[string]any{
					"enabled": true, "server_name": b.CoreSNI,
					"utls": map[string]any{"enabled": true, "fingerprint": "firefox"},
				},
			}
		case "ws":
			ob = map[string]any{
				"type": "vless", "tag": p.OutboundTag,
				"server": coreHost, "server_port": b.CoreVless,
				"uuid": b.UplinkUUID,
				"transport": map[string]any{
					"type": "ws", "path": "/nd-up-ws",
				},
				"tls": uplinkTLS,
			}
		case "grpc":
			ob = map[string]any{
				"type": "vless", "tag": p.OutboundTag,
				"server": coreHost, "server_port": b.CoreVless,
				"uuid": b.UplinkUUID,
				"transport": map[string]any{
					"type": "grpc", "service_name": "nd-up-grpc",
				},
				"tls": uplinkTLS,
			}
		case "ssh":
			port := cfg.SSHPort
			if port <= 0 {
				port = 22
			}
			host := cfg.SSHHost
			if host == "" {
				host = coreHost
			}
			ob = map[string]any{
				"type": "ssh", "tag": p.OutboundTag,
				"server": host, "server_port": port,
				"user": cfg.SSHUser,
			}
			if cfg.SSHPrivateKeyPath != "" {
				ob["private_key_path"] = cfg.SSHPrivateKeyPath
			}
		default:
			continue
		}
		outbounds = append(outbounds, ob)
	}
	return outbounds
}

// uplinkABRouteRules maps ab-in-* → uplink-ab-*.
func uplinkABRouteRules() []any {
	cfg := LoadUplinkABConfig()
	profs := ResolveUplinkABProfiles(cfg)
	var rules []any
	for _, p := range profs {
		rules = append(rules, map[string]any{
			"inbound":  []string{p.InboundTag},
			"outbound": p.OutboundTag,
		})
	}
	return rules
}

// UplinkABSubscriptionLinks returns extra VLESS URIs for allowlisted user.
func UplinkABSubscriptionLinks(name, uuid, relayIP, pbk, sid, sniName string) []string {
	if !UplinkABUserAllowed(name) {
		return nil
	}
	cfg := LoadUplinkABConfig()
	profs := ResolveUplinkABProfiles(cfg)
	if len(profs) == 0 {
		return nil
	}
	if sniName == "" {
		sniName = DefaultRealitySNI
	}
	host := strings.TrimSpace(os.Getenv("NETDUCTOR_VPN_HOST"))
	if host == "" {
		if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "vpn_hostname")); err == nil {
			host = strings.TrimSpace(string(b))
		}
	}
	if host == "" {
		host = relayIP
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
