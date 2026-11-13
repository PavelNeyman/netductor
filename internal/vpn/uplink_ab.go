package vpn

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/curve25519"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Uplink AB: operator-only experimental S→P underlays (PLAN-UPLINK-AB-PROFILE).

const (
	uplinkABEnv      = "NETDUCTOR_UPLINK_AB"
	uplinkABUsersEnv = "NETDUCTOR_UPLINK_AB_USERS"
	uplinkABBasePort = 9444 // client → secondary steer ports
	// Primary ports for S→P exp underlays (not client-facing).
	primaryABPortWS     = 10443
	primaryABPortGRPC   = 10444
	primaryABPortAnyTLS = 10445
	primaryABPortWG     = 51820
)

// UplinkABProfile is one experimental path id.
type UplinkABProfile struct {
	ID          string
	InboundTag  string
	OutboundTag string
	Port        int // secondary listen (client→S)
	Fragment    string
	PrimaryPort int // S→P dial port on primary (0 = default core vless)
}

// UplinkABConfig file + env.
type UplinkABConfig struct {
	Enabled  bool     `json:"enabled"`
	Users    []string `json:"users"`
	Profiles []string `json:"profiles"`
	BasePort int      `json:"base_port"`

	// WG: primary holds server key; secondary holds client key (auto-filled by EnsureUplinkABMaterial).
	WGPrimaryPrivateKey   string `json:"wg_primary_private_key,omitempty"`
	WGPrimaryPublicKey    string `json:"wg_primary_public_key,omitempty"`
	WGSecondaryPrivateKey string `json:"wg_secondary_private_key,omitempty"`
	WGSecondaryPublicKey  string `json:"wg_secondary_public_key,omitempty"`
	WGPrimaryAddress      string `json:"wg_primary_address,omitempty"`   // 10.87.90.1/30
	WGSecondaryAddress    string `json:"wg_secondary_address,omitempty"` // 10.87.90.2/30
	WGListenPort          int    `json:"wg_listen_port,omitempty"`
	WGEndpoint            string `json:"wg_endpoint,omitempty"` // host:port secondary dials (defaults publicIP:listen)
	WGMTU                 int    `json:"wg_mtu,omitempty"`

	// Legacy aliases accepted on load
	WGPrivateKey    string `json:"wg_private_key,omitempty"`
	WGPeerPublicKey string `json:"wg_peer_public_key,omitempty"`
	WGLocalAddress  string `json:"wg_local_address,omitempty"`
	WGPeerAddress   string `json:"wg_peer_address,omitempty"`

	SSHUser           string `json:"ssh_user,omitempty"`
	SSHPrivateKeyPath string `json:"ssh_private_key_path,omitempty"`
	SSHHost           string `json:"ssh_host,omitempty"`
	SSHPort           int    `json:"ssh_port,omitempty"`
}

func uplinkABConfigPath() string {
	return filepath.Join(paths.EtcDir(), "uplink_ab.json")
}

// LoadUplinkABConfig merges file + env (env wins for enabled/users).
func LoadUplinkABConfig() UplinkABConfig {
	cfg := UplinkABConfig{
		Profiles:           []string{"nomux", "wg", "anytls", "ws", "grpc", "ssh"},
		BasePort:           uplinkABBasePort,
		WGPrimaryAddress:   "10.87.90.1/30",
		WGSecondaryAddress: "10.87.90.2/30",
		WGListenPort:       primaryABPortWG,
		WGMTU:              1280,
	}
	if b, err := os.ReadFile(uplinkABConfigPath()); err == nil {
		_ = json.Unmarshal(b, &cfg)
	}
	// legacy field mapping
	if cfg.WGSecondaryPrivateKey == "" && cfg.WGPrivateKey != "" {
		cfg.WGSecondaryPrivateKey = cfg.WGPrivateKey
	}
	if cfg.WGPrimaryPublicKey == "" && cfg.WGPeerPublicKey != "" {
		cfg.WGPrimaryPublicKey = cfg.WGPeerPublicKey
	}
	if cfg.WGSecondaryAddress == "" && cfg.WGLocalAddress != "" {
		cfg.WGSecondaryAddress = cfg.WGLocalAddress
	}
	if cfg.WGPrimaryAddress == "" && cfg.WGPeerAddress != "" {
		if !strings.Contains(cfg.WGPeerAddress, "/") {
			cfg.WGPrimaryAddress = cfg.WGPeerAddress + "/30"
		} else {
			cfg.WGPrimaryAddress = cfg.WGPeerAddress
		}
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
	if cfg.WGListenPort <= 0 {
		cfg.WGListenPort = primaryABPortWG
	}
	if cfg.WGMTU <= 0 {
		cfg.WGMTU = 1280
	}
	if cfg.WGPrimaryAddress == "" {
		cfg.WGPrimaryAddress = "10.87.90.1/30"
	}
	if cfg.WGSecondaryAddress == "" {
		cfg.WGSecondaryAddress = "10.87.90.2/30"
	}
	if len(cfg.Profiles) == 0 {
		cfg.Profiles = []string{"nomux", "wg", "anytls", "ws", "grpc", "ssh"}
	}
	return cfg
}

func saveUplinkABConfig(cfg UplinkABConfig) error {
	_ = os.MkdirAll(paths.EtcDir(), 0o755)
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := uplinkABConfigPath() + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, uplinkABConfigPath())
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

func profileWants(cfg UplinkABConfig, id string) bool {
	for _, p := range cfg.Profiles {
		if strings.EqualFold(strings.TrimSpace(p), id) {
			return true
		}
	}
	return false
}

// EnsureUplinkABMaterial generates WG keypairs when wg profile is enabled and keys missing.
func EnsureUplinkABMaterial() error {
	cfg := LoadUplinkABConfig()
	if !cfg.Enabled || !profileWants(cfg, "wg") {
		return nil
	}
	changed := false
	if cfg.WGPrimaryPrivateKey == "" || cfg.WGPrimaryPublicKey == "" {
		priv, pub, err := generateWGKeypair()
		if err != nil {
			return err
		}
		cfg.WGPrimaryPrivateKey = priv
		cfg.WGPrimaryPublicKey = pub
		changed = true
	}
	if cfg.WGSecondaryPrivateKey == "" || cfg.WGSecondaryPublicKey == "" {
		priv, pub, err := generateWGKeypair()
		if err != nil {
			return err
		}
		cfg.WGSecondaryPrivateKey = priv
		cfg.WGSecondaryPublicKey = pub
		changed = true
	}
	if cfg.WGEndpoint == "" {
		if ip := publicIP(); ip != "" {
			cfg.WGEndpoint = fmt.Sprintf("%s:%d", ip, cfg.WGListenPort)
			changed = true
		}
	}
	if changed {
		return saveUplinkABConfig(cfg)
	}
	return nil
}

func generateWGKeypair() (privB64, pubB64 string, err error) {
	var priv, pub [32]byte
	if _, err = rand.Read(priv[:]); err != nil {
		return "", "", err
	}
	// curve25519 clamp
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64
	curve25519.ScalarBaseMult(&pub, &priv)
	return base64.StdEncoding.EncodeToString(priv[:]), base64.StdEncoding.EncodeToString(pub[:]), nil
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
		switch id {
		case "wg":
			if cfg.WGSecondaryPrivateKey == "" || cfg.WGPrimaryPublicKey == "" || cfg.WGEndpoint == "" {
				continue
			}
		case "ssh":
			if cfg.SSHHost == "" || cfg.SSHUser == "" {
				continue
			}
		}
		p := UplinkABProfile{
			ID:          id,
			InboundTag:  "ab-in-" + id,
			OutboundTag: "uplink-ab-" + id,
			Port:        port,
			Fragment:    "nd-up-" + id,
		}
		switch id {
		case "nomux":
			p.PrimaryPort = 0 // core vless
		case "ws":
			p.PrimaryPort = primaryABPortWS
		case "grpc":
			p.PrimaryPort = primaryABPortGRPC
		case "anytls":
			p.PrimaryPort = primaryABPortAnyTLS
		case "wg":
			p.PrimaryPort = cfg.WGListenPort
		case "ssh":
			if cfg.SSHPort > 0 {
				p.PrimaryPort = cfg.SSHPort
			} else {
				p.PrimaryPort = 22
			}
		}
		out = append(out, p)
		port++
	}
	return out
}

// StatusUplinkAB is CLI/JSON summary.
func StatusUplinkAB() map[string]any {
	_ = EnsureUplinkABMaterial()
	cfg := LoadUplinkABConfig()
	profs := ResolveUplinkABProfiles(cfg)
	ids := make([]string, 0, len(profs))
	for _, p := range profs {
		ids = append(ids, fmt.Sprintf("%s:s%d→p%d", p.ID, p.Port, p.PrimaryPort))
	}
	return map[string]any{
		"enabled":   cfg.Enabled,
		"users":     cfg.Users,
		"profiles":  ids,
		"base_port": cfg.BasePort,
		"wg_ready":  cfg.WGPrimaryPrivateKey != "" && cfg.WGSecondaryPrivateKey != "",
		"config":    uplinkABConfigPath(),
	}
}

// appendUplinkABInbounds adds steer inbounds on secondary (allowlisted users only).
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

func coreDialHost(b *SecondaryBundle) string {
	h := primarySPAddrForSecondary()
	if h == "" {
		h = strings.TrimSpace(b.CoreIP)
	}
	return h
}

// appendUplinkABOutbounds adds experimental S→P outbounds on secondary.
func appendUplinkABOutbounds(outbounds []any, b *SecondaryBundle) []any {
	cfg := LoadUplinkABConfig()
	profs := ResolveUplinkABProfiles(cfg)
	if len(profs) == 0 {
		return outbounds
	}
	uplinkTLS := buildSecondaryUplinkTLS(b)
	coreHost := coreDialHost(b)
	for _, p := range profs {
		var ob map[string]any
		switch p.ID {
		case "nomux":
			ob = map[string]any{
				"type": "vless", "tag": p.OutboundTag,
				"server": coreHost, "server_port": b.CoreVless,
				"uuid": b.UplinkUUID, "tls": uplinkTLS,
			}
		case "wg":
			ob = map[string]any{
				"type":            "wireguard",
				"tag":             p.OutboundTag,
				"private_key":     cfg.WGSecondaryPrivateKey,
				"local_address":   []string{cfg.WGSecondaryAddress},
				"mtu":             cfg.WGMTU,
				"peers": []any{
					map[string]any{
						"public_key":  cfg.WGPrimaryPublicKey,
						"allowed_ips": []string{"0.0.0.0/0"},
						"endpoint":    cfg.WGEndpoint,
					},
				},
			}
		case "anytls":
			ob = map[string]any{
				"type": "anytls", "tag": p.OutboundTag,
				"server": coreHost, "server_port": primaryABPortAnyTLS,
				"password": b.UplinkUUID,
				"tls": map[string]any{
					"enabled": true, "server_name": b.CoreSNI,
					"utls": map[string]any{"enabled": true, "fingerprint": "firefox"},
				},
			}
		case "ws":
			ob = map[string]any{
				"type": "vless", "tag": p.OutboundTag,
				"server": coreHost, "server_port": primaryABPortWS,
				"uuid": b.UplinkUUID,
				"transport": map[string]any{
					"type": "ws", "path": "/nd-up-ws",
				},
				"tls": uplinkTLS,
			}
		case "grpc":
			ob = map[string]any{
				"type": "vless", "tag": p.OutboundTag,
				"server": coreHost, "server_port": primaryABPortGRPC,
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

func uplinkABRouteRules() []any {
	profs := ResolveUplinkABProfiles(LoadUplinkABConfig())
	var rules []any
	for _, p := range profs {
		rules = append(rules, map[string]any{
			"inbound":  []string{p.InboundTag},
			"outbound": p.OutboundTag,
		})
	}
	return rules
}

// appendPrimaryUplinkABInbounds adds primary terminators for ws/grpc/anytls (uplink UUID only, no vision).
func appendPrimaryUplinkABInbounds(inbounds []any, uplinkUUID, sniVal, priv, sid string) []any {
	cfg := LoadUplinkABConfig()
	if !cfg.Enabled || uplinkUUID == "" {
		return inbounds
	}
	u := []map[string]any{{"name": RelayUplinkName, "uuid": uplinkUUID, "flow": ""}}
	tlsReality := map[string]any{
		"enabled": true, "server_name": sniVal,
		"reality": map[string]any{
			"enabled":     true,
			"handshake":   map[string]any{"server": sniVal, "server_port": 443},
			"private_key": priv, "short_id": []string{sid},
		},
	}
	if profileWants(cfg, "ws") {
		inbounds = append(inbounds, map[string]any{
			"type": "vless", "tag": "ab-vless-ws", "listen": "::", "listen_port": primaryABPortWS,
			"users": u,
			"transport": map[string]any{
				"type": "ws", "path": "/nd-up-ws",
			},
			"tls": tlsReality,
		})
	}
	if profileWants(cfg, "grpc") {
		inbounds = append(inbounds, map[string]any{
			"type": "vless", "tag": "ab-vless-grpc", "listen": "::", "listen_port": primaryABPortGRPC,
			"users": u,
			"transport": map[string]any{
				"type": "grpc", "service_name": "nd-up-grpc",
			},
			"tls": tlsReality,
		})
	}
	if profileWants(cfg, "anytls") {
		inbounds = append(inbounds, map[string]any{
			"type": "anytls", "tag": "ab-anytls", "listen": "::", "listen_port": primaryABPortAnyTLS,
			"users": []map[string]any{{"name": RelayUplinkName, "password": uplinkUUID}},
			"tls": map[string]any{
				"enabled": true, "server_name": sniVal,
				// anytls typically needs real cert; use reality only if supported — fallback plain tls insecure for AB lab
				"reality": map[string]any{
					"enabled":     true,
					"handshake":   map[string]any{"server": sniVal, "server_port": 443},
					"private_key": priv, "short_id": []string{sid},
				},
			},
		})
	}
	return inbounds
}

// primaryUplinkABEndpoints returns sing-box endpoints (WireGuard server for AB).
func primaryUplinkABEndpoints() []any {
	cfg := LoadUplinkABConfig()
	if !cfg.Enabled || !profileWants(cfg, "wg") {
		return nil
	}
	if cfg.WGPrimaryPrivateKey == "" || cfg.WGSecondaryPublicKey == "" {
		return nil
	}
	return []any{
		map[string]any{
			"type":         "wireguard",
			"tag":          "ab-wg",
			"system":       false,
			"name":         "nd-ab-wg",
			"private_key":  cfg.WGPrimaryPrivateKey,
			"listen_port":  cfg.WGListenPort,
			"address":      []string{cfg.WGPrimaryAddress},
			"mtu":          cfg.WGMTU,
			"peers": []any{
				map[string]any{
					"public_key":  cfg.WGSecondaryPublicKey,
					"allowed_ips": []string{cfg.WGSecondaryAddress},
				},
			},
		},
	}
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
