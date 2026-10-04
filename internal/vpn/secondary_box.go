package vpn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// UplinkMuxMode: on (default) | off | h2mux — NETDUCTOR_UPLINK_MUX or /etc/netductor/uplink_mux_mode
func UplinkMuxMode() string {
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_UPLINK_MUX")); v != "" {
		return strings.ToLower(v)
	}
	b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "uplink_mux_mode"))
	if err == nil {
		return strings.ToLower(strings.TrimSpace(string(b)))
	}
	return "on"
}

func uplinkMultiplexObject() map[string]any {
	mode := UplinkMuxMode()
	if mode == "off" || mode == "0" || mode == "false" || mode == "no" {
		return nil
	}
	m := map[string]any{
		"enabled":         true,
		"padding":         true,
		"max_connections": 4,
		"min_streams":     4,
		"max_streams":     32,
	}
	if mode == "h2mux" || mode == "h2" {
		m["protocol"] = "h2mux"
	}
	return m
}


// Secondary is trusted for service identity: it dials vless-svc as the end-user UUID.
// A compromised secondary can present any registered user. Do not treat uplink auth_user as proof beyond that trust boundary.
const RelayUplinkName = "relay-uplink"

// SecondaryBundle is generated on core and consumed on RU relay VPS.
type SecondaryBundle struct {
	Version      int         `json:"version"`
	CreatedAt    string      `json:"created_at"`
	CoreIP       string      `json:"core_ip"`
	CoreVless    int         `json:"core_vless_port"`
	CoreSNI      string      `json:"core_sni"`
	CorePBK      string      `json:"core_pbk"`
	CoreSID      string      `json:"core_sid"`
	UplinkUUID   string      `json:"uplink_uuid"`
	SecondarySNI string      `json:"relay_sni"`
	Users        []RelayUser `json:"users"`
	AgentToken   string      `json:"agent_token,omitempty"`
	AgentID      string      `json:"agent_id,omitempty"`
	CoreAgentURL string      `json:"core_agent_url,omitempty"`
	ExitUUID     string      `json:"exit_uuid,omitempty"`
	ExitPort     int         `json:"exit_port,omitempty"`
}

type RelayUser struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

func EnsureRelayUplink() (uuid string, err error) {
	r, err := loadRegistry()
	if err != nil {
		return "", err
	}
	if u := findUser(r, RelayUplinkName); u != nil {
		return u.UUID, nil
	}
	msg, err := AddNative(RelayUplinkName, "RU relay uplink — do not give to end users")
	if err != nil {
		return "", fmt.Errorf("%s: %w", msg, err)
	}
	_ = msg
	r, err = loadRegistry()
	if err != nil {
		return "", err
	}
	u := findUser(r, RelayUplinkName)
	if u == nil {
		return "", fmt.Errorf("uplink user missing after create")
	}
	return u.UUID, nil
}

func ExportSecondaryBundle(relaySNI string) (*SecondaryBundle, error) {
	if relaySNI == "" {
		relaySNI = DefaultRealitySNI
	}
	up, err := EnsureRelayUplink()
	if err != nil {
		return nil, err
	}
	r, err := loadRegistry()
	if err != nil {
		return nil, err
	}
	var users []RelayUser
	for _, u := range r.Users {
		if !u.Enabled || u.Name == RelayUplinkName {
			continue
		}
		users = append(users, RelayUser{Name: u.Name, UUID: u.UUID})
	}
	b := &SecondaryBundle{
		Version:      1,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		CoreIP:       publicIP(),
		CoreVless:    vlessPort(),
		CoreSNI:      sni(),
		CorePBK:      secret("singbox_reality_public"),
		CoreSID:      secret("singbox_short_id"),
		UplinkUUID:   up,
		SecondarySNI: relaySNI,
		Users:        users,
		ExitPort:     4443,
	}
	// dedicated UUID for core→RU exit feeder
	if eu := secret("secondary_exit_uuid"); eu != "" {
		b.ExitUUID = eu
	} else {
		b.ExitUUID = genUUID()
		_ = os.WriteFile(filepath.Join(paths.EtcDir(), "secrets", "secondary_exit_uuid"), append([]byte(b.ExitUUID), 10), 0o600)
	}
	if b.CorePBK == "" || b.CoreSID == "" {
		return nil, fmt.Errorf("core Reality secrets missing")
	}
	dir := paths.SecondaryDir()
	_ = os.MkdirAll(dir, 0o700)
	raw, _ := json.MarshalIndent(b, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, "bundle.json"), append(raw, '\n'), 0o600)
	return b, nil
}

func WriteSecondarySingBox(b *SecondaryBundle, privKey, shortID string) error {
	if b == nil {
		return fmt.Errorf("nil bundle")
	}
	if privKey == "" || shortID == "" {
		return fmt.Errorf("relay reality keys required")
	}
	type vu struct {
		Name string `json:"name,omitempty"`
		UUID string `json:"uuid"`
		Flow string `json:"flow"`
	}
	var users []vu
	for _, u := range b.Users {
		users = append(users, vu{Name: u.Name, UUID: u.UUID, Flow: "xtls-rprx-vision"})
	}
	if len(users) == 0 {
		return fmt.Errorf("bundle has no end users — add vpn users on core first")
	}
	exitPort := b.ExitPort
	if exitPort <= 0 {
		exitPort = 4443
	}
	inbounds := []any{
		map[string]any{
			"type": "vless", "tag": "relay-in", "listen": "::", "listen_port": 443,
			"users": users,
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
		},
	}
	if b.ExitUUID != "" {
		inbounds = append(inbounds, map[string]any{
			"type": "vless", "tag": "exit-in", "listen": "::", "listen_port": exitPort,
			"users": []vu{{UUID: b.ExitUUID, Flow: "xtls-rprx-vision"}},
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
	// RU split: these domains + private IP exit direct (RU IP).
	// Everything else from relay-in goes uplink → core (foreign exit).
	// Clients abroad can use exit-in (4443) for RU-IP egress when toggled on.
	ruSuffixes := RuDirectSuffixes()
	uplinkTLS := map[string]any{
		"enabled": true, "server_name": b.CoreSNI,
		"utls": map[string]any{"enabled": true, "fingerprint": "firefox"},
		"reality": map[string]any{
			"enabled":    true,
			"public_key": b.CorePBK,
			"short_id":   b.CoreSID,
		},
	}
	// Shared uplink: internet only. Primary sees auth_user=relay-uplink.
	uplink := map[string]any{
		"type": "vless", "tag": "uplink",
		"server": b.CoreIP, "server_port": b.CoreVless,
		// no vision: required for multiplex (vision ⊕ mux unsupported)
		"uuid":            b.UplinkUUID,
		"domain_resolver": "quad9",
		"tls":             uplinkTLS,
	}
	if mx := uplinkMultiplexObject(); mx != nil {
		uplink["multiplex"] = mx
	}
	// Per-user service uplink: dial primary as the real user (vision, no mux)
	// so ACL sees Pavel, not relay-uplink. Only 10.88.0.0/24 uses these.
	outbounds := []any{uplink}
	var svcRules []any
	for _, u := range b.Users {
		name := strings.TrimSpace(u.Name)
		if name == "" || u.UUID == "" {
			continue
		}
		tag := "uplink-svc-" + sanitizeTag(name)
		outbounds = append(outbounds, map[string]any{
			"type": "vless", "tag": tag,
			"server": "10.87.10.1", "server_port": 9443,
			"uuid":   u.UUID,
			"domain_resolver": "quad9",
			"tls":    uplinkTLS,
		})
		svcRules = append(svcRules, map[string]any{
			"auth_user": []string{name},
			"ip_cidr":   []string{"10.88.0.0/24"},
			"outbound":  tag,
		})
	}
	svcRules = append(svcRules, map[string]any{
		"ip_cidr":  []string{"10.88.0.0/24"},
		"outbound": "block",
	})
	cfg := map[string]any{
		"log": map[string]any{"level": "info", "timestamp": true},
		"dns": map[string]any{
			"servers": []any{
				map[string]any{"type": "udp", "tag": "ru-dns", "server": "77.88.8.8"},
				map[string]any{"type": "udp", "tag": "quad9", "server": "9.9.9.9"},
				map[string]any{"type": "udp", "tag": "cf", "server": "1.1.1.1"},
				map[string]any{"type": "local", "tag": "local"},
			},
			"rules": []any{
				map[string]any{"domain_suffix": ruSuffixes, "server": "ru-dns"},
				map[string]any{"domain_keyword": RuDirectKeywords(), "server": "ru-dns"},
			},
			"final": "quad9",
		},
		"inbounds": inbounds,
		"outbounds": append(outbounds,
			map[string]any{"type": "direct", "tag": "direct"},
			map[string]any{"type": "block", "tag": "block"},
		),
		"route": map[string]any{
			// domain_suffix list (fast, no download) + geoip-ru rule-set (IP ranges).
			"rule_set": []any{
				map[string]any{
					"tag":             "geoip-ru",
					"type":            "remote",
					"format":          "binary",
					"url":             "https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-ru.srs",
					// direct: uplink is Reality (SNI=api.vk.me) and breaks TLS to githubusercontent
					"download_detour": "direct",
				},
			},
			"rules": append(append([]any{
				map[string]any{"action": "sniff"},
				map[string]any{"protocol": "dns", "action": "hijack-dns"},
				map[string]any{"ip_version": 6, "outbound": "block"},
				map[string]any{"inbound": []string{"exit-in"}, "outbound": "direct"},
			}, svcRules...),
				map[string]any{"ip_is_private": true, "outbound": "direct"},
				map[string]any{"domain_suffix": ruSuffixes, "outbound": "direct"},
				map[string]any{"domain_keyword": RuDirectKeywords(), "outbound": "direct"},
				map[string]any{"rule_set": []string{"geoip-ru"}, "outbound": "direct"},
				map[string]any{"inbound": []string{"relay-in"}, "outbound": "uplink"},
			),
			"final":                   "uplink",
			"default_domain_resolver": "quad9",
			"auto_detect_interface":   true,
		},
	}
	_ = os.MkdirAll(filepath.Dir(singboxConf), 0o755)
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := singboxConf + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return err
	}
	_ = os.Chmod(tmp, 0o600)
	if err := os.Rename(tmp, singboxConf); err != nil {
		return err
	}
	return os.Chmod(singboxConf, 0o600)
}

func sanitizeTag(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		return "user"
	}
	return s
}

func ClientLinkForSecondary(name, uuid, relayIP, pbk, sid, sniName string) string {
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
	tag := "nd-secondary"
	if name != "" {
		tag = "nd-" + name
	}
	return fmt.Sprintf(
		"vless://%s@%s:443?encryption=none&flow=xtls-rprx-vision&security=reality&sni=%s&fp=firefox&pbk=%s&sid=%s&type=tcp#%s",
		uuid, host, sniName, pbk, sid, tag,
	)
}
