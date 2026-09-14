package vpn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

const RelayUplinkName = "relay-uplink"

// RelayBundle is generated on core and consumed on RU relay VPS.
type RelayBundle struct {
	Version      int         `json:"version"`
	CreatedAt    string      `json:"created_at"`
	CoreIP       string      `json:"core_ip"`
	CoreVless    int         `json:"core_vless_port"`
	CoreSNI      string      `json:"core_sni"`
	CorePBK      string      `json:"core_pbk"`
	CoreSID      string      `json:"core_sid"`
	UplinkUUID   string      `json:"uplink_uuid"`
	RelaySNI     string      `json:"relay_sni"`
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

func ExportRelayBundle(relaySNI string) (*RelayBundle, error) {
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
	b := &RelayBundle{
		Version:    1,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		CoreIP:     publicIP(),
		CoreVless:  vlessPort(),
		CoreSNI:    sni(),
		CorePBK:    secret("singbox_reality_public"),
		CoreSID:    secret("singbox_short_id"),
		UplinkUUID: up,
		RelaySNI:   relaySNI,
		Users:      users,
		ExitPort:   4443,
	}
	// dedicated UUID for core→RU exit feeder
	if eu := secret("relay_exit_uuid"); eu != "" {
		b.ExitUUID = eu
	} else {
		b.ExitUUID = genUUID()
		_ = os.WriteFile(filepath.Join(paths.EtcDir(), "secrets", "relay_exit_uuid"), append([]byte(b.ExitUUID), 10), 0o600)
	}
	if b.CorePBK == "" || b.CoreSID == "" {
		return nil, fmt.Errorf("core Reality secrets missing")
	}
	dir := filepath.Join(paths.StateDir(), "relay")
	_ = os.MkdirAll(dir, 0o700)
	raw, _ := json.MarshalIndent(b, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, "bundle.json"), append(raw, '\n'), 0o600)
	return b, nil
}

func WriteRelaySingBox(b *RelayBundle, privKey, shortID string) error {
	if b == nil {
		return fmt.Errorf("nil bundle")
	}
	if privKey == "" || shortID == "" {
		return fmt.Errorf("relay reality keys required")
	}
	type vu struct {
		UUID string `json:"uuid"`
		Flow string `json:"flow"`
	}
	var users []vu
	for _, u := range b.Users {
		users = append(users, vu{UUID: u.UUID, Flow: "xtls-rprx-vision"})
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
				"enabled": true, "server_name": b.RelaySNI,
				"reality": map[string]any{
					"enabled": true,
					"handshake": map[string]any{
						"server": b.RelaySNI, "server_port": 443,
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
				"enabled": true, "server_name": b.RelaySNI,
				"reality": map[string]any{
					"enabled": true,
					"handshake": map[string]any{
						"server": b.RelaySNI, "server_port": 443,
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
	ruSuffixes := []string{
		"ru", "su", "xn--p1ai", "xn--p1acf",
		"vk.com", "vk.ru", "vk.me", "userapi.com", "vkuservideo.net", "vk-cdn.net",
		"yandex.ru", "yandex.net", "yandex.com", DefaultRealitySNI, "yastatic.net", "yandex.cloud",
		"mail.ru", "imgsmail.ru", "ok.ru", "odnoklassniki.ru",
		"wildberries.ru", "wb.ru", "ozon.ru", "avito.ru", "dns-shop.ru", "citilink.ru",
		"2gis.com", "2gis.ru", "gosuslugi.ru", "mos.ru", "nalog.ru", "cbr.ru",
		"sberbank.ru", "sber.ru", "tinkoff.ru", "tbank.ru", "vtb.ru", "alfabank.ru",
		"mts.ru", "megafon.ru", "beeline.ru", "tele2.ru", "yota.ru",
		"rutube.ru", "ivi.ru", "kinopoisk.ru", "hd.kinopoisk.ru",
		"cloudflare-dns.com", // keep resolvers reachable; actual RU DNS via rule
	}
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
			},
			"final": "quad9",
		},
		"inbounds": inbounds,
		"outbounds": []any{
			map[string]any{
				"type": "vless", "tag": "uplink",
				"server": b.CoreIP, "server_port": b.CoreVless,
				"uuid": b.UplinkUUID, "flow": "xtls-rprx-vision",
				"domain_resolver": "quad9",
				"tls": map[string]any{
					"enabled": true, "server_name": b.CoreSNI,
					"utls":    map[string]any{"enabled": true, "fingerprint": "firefox"},
					"reality": map[string]any{
						"enabled":    true,
						"public_key": b.CorePBK,
						"short_id":   b.CoreSID,
					},
				},
			},
			map[string]any{"type": "direct", "tag": "direct"},
			map[string]any{"type": "block", "tag": "block"},
		},
		"route": map[string]any{
			"rules": []any{
				map[string]any{"action": "sniff"},
				map[string]any{"protocol": "dns", "action": "hijack-dns"},
				map[string]any{"ip_version": 6, "outbound": "block"},
				map[string]any{"inbound": []string{"exit-in"}, "outbound": "direct"},
				map[string]any{"ip_is_private": true, "outbound": "direct"},
				map[string]any{"domain_suffix": ruSuffixes, "outbound": "direct"},
				map[string]any{"inbound": []string{"relay-in"}, "outbound": "uplink"},
			},
			"final":                    "uplink",
			"default_domain_resolver":  "quad9",
			"auto_detect_interface":    true,
		},
	}
	_ = os.MkdirAll(filepath.Dir(singboxConf), 0o755)
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := singboxConf + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, singboxConf)
}

func ClientLinkForRelay(name, uuid, relayIP, pbk, sid, sniName string) string {
	if sniName == "" {
		sniName = DefaultRealitySNI
	}
	return fmt.Sprintf(
		"vless://%s@%s:443?encryption=none&flow=xtls-rprx-vision&security=reality&sni=%s&fp=firefox&pbk=%s&sid=%s&type=tcp#nd-relay",
		uuid, relayIP, sniName, pbk, sid,
	)
}
