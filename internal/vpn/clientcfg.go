package vpn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// ClientEndpoints holds resolved core + preferred relay for a user.
type ClientEndpoints struct {
	Name      string
	UUID      string
	CoreHost  string
	CorePort  int
	CoreSNI   string
	CorePBK   string
	CoreSID   string
	RelayHost string
	RelayPort int
	RelaySNI  string
	RelayPBK  string
	RelaySID  string
}

type relayDevFile struct {
	Devices []struct {
		PublicIP string    `json:"public_ip"`
		PBK      string    `json:"pbk"`
		SID      string    `json:"sid"`
		SNI      string    `json:"sni"`
		LastSeen time.Time `json:"last_seen"`
	} `json:"devices"`
}

func loadOnlineRelay() (host, pbk, sid, sniName string) {
	p := filepath.Join(paths.StateDir(), "relay", "devices.json")
	b, err := os.ReadFile(p)
	if err != nil {
		return
	}
	var f relayDevFile
	if json.Unmarshal(b, &f) != nil {
		return
	}
	now := time.Now().UTC()
	for _, d := range f.Devices {
		if d.PublicIP == "" || d.PBK == "" {
			continue
		}
		if !d.LastSeen.IsZero() && now.Sub(d.LastSeen) > 3*time.Minute {
			continue
		}
		sniName = d.SNI
		if sniName == "" {
			sniName = "ya.ru"
		}
		return d.PublicIP, d.PBK, d.SID, sniName
	}
	return
}

func resolveClientEndpoints(name, uuid string) ClientEndpoints {
	e := ClientEndpoints{
		Name: name, UUID: uuid,
		CoreHost: publicIP(), CorePort: vlessPort(),
		CoreSNI: sni(), CorePBK: secret("singbox_reality_public"), CoreSID: secret("singbox_short_id"),
		RelayPort: 443,
	}
	e.RelayHost, e.RelayPBK, e.RelaySID, e.RelaySNI = loadOnlineRelay()
	return e
}

func ClientLinkForRelayLocal(name, uuid, relayIP, pbk, sid, sniName string) string {
	if sniName == "" {
		sniName = "ya.ru"
	}
	return fmt.Sprintf(
		"vless://%s@%s:443?encryption=none&flow=xtls-rprx-vision&security=reality&sni=%s&fp=firefox&pbk=%s&sid=%s&type=tcp#%s-relay",
		uuid, relayIP, sniName, pbk, sid, name,
	)
}

// SingBoxClientJSON minimal dual-outbound (relay primary, core backup). Always Vision.
func SingBoxClientJSON(e ClientEndpoints) ([]byte, error) {
	if e.UUID == "" {
		return nil, fmt.Errorf("uuid required")
	}
	outbounds := []any{
		map[string]any{"type": "direct", "tag": "direct"},
		map[string]any{"type": "block", "tag": "block"},
	}
	final := "direct"
	if e.RelayHost != "" && e.RelayPBK != "" {
		outbounds = append(outbounds, map[string]any{
			"type": "vless", "tag": "relay",
			"server": e.RelayHost, "server_port": e.RelayPort,
			"uuid": e.UUID, "flow": "xtls-rprx-vision",
			"tls": map[string]any{
				"enabled": true, "server_name": e.RelaySNI,
				"utls":    map[string]any{"enabled": true, "fingerprint": "firefox"},
				"reality": map[string]any{"enabled": true, "public_key": e.RelayPBK, "short_id": e.RelaySID},
			},
		})
		final = "relay"
	}
	if e.CoreHost != "" && e.CorePBK != "" {
		outbounds = append(outbounds, map[string]any{
			"type": "vless", "tag": "core",
			"server": e.CoreHost, "server_port": e.CorePort,
			"uuid": e.UUID, "flow": "xtls-rprx-vision",
			"tls": map[string]any{
				"enabled": true, "server_name": e.CoreSNI,
				"utls":    map[string]any{"enabled": true, "fingerprint": "chrome"},
				"reality": map[string]any{"enabled": true, "public_key": e.CorePBK, "short_id": e.CoreSID},
			},
		})
		if final == "direct" {
			final = "core"
		}
	}
	cfg := map[string]any{
		"log": map[string]any{"level": "info"},
		"dns": map[string]any{
			"servers": []any{map[string]any{"type": "udp", "tag": "local", "server": "8.8.8.8"}},
			"final":   "local",
		},
		"inbounds": []any{
			map[string]any{"type": "tun", "tag": "tun-in", "address": []string{"172.19.0.1/30"}, "auto_route": true, "strict_route": true},
		},
		"outbounds": outbounds,
		"route": map[string]any{
			"rules": []any{
				map[string]any{"action": "sniff"},
				map[string]any{"protocol": "dns", "action": "hijack-dns"},
				map[string]any{"ip_is_private": true, "outbound": "direct"},
			},
			"final":                 final,
			"auto_detect_interface": true,
		},
	}
	return json.MarshalIndent(cfg, "", "  ")
}

// ShadowrocketJSON helper doc + URIs (import URIs; keep flow).
func ShadowrocketJSON(e ClientEndpoints) ([]byte, error) {
	if e.UUID == "" {
		return nil, fmt.Errorf("uuid required")
	}
	uris := []string{}
	if e.RelayHost != "" {
		uris = append(uris, ClientLinkForRelayLocal(e.Name, e.UUID, e.RelayHost, e.RelayPBK, e.RelaySID, e.RelaySNI))
	}
	if e.CoreHost != "" {
		uris = append(uris, VLESSLink(e.Name, e.UUID))
	}
	doc := map[string]any{
		"remarks": "netductor minimal — RU split is on relay; no external balancers",
		"note":    "Import uris into Shadowrocket. Every VLESS must keep flow=xtls-rprx-vision.",
		"uris":    uris,
	}
	return json.MarshalIndent(doc, "", "  ")
}

// WriteClientConfigs writes sing-box + shadowrocket helper files under client dir.
func WriteClientConfigs(name, uuid string) error {
	e := resolveClientEndpoints(name, uuid)
	dir := filepath.Join(Clients(), name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	sb, err := SingBoxClientJSON(e)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "sing-box-client.json"), append(sb, 10), 0o600); err != nil {
		return err
	}
	sr, err := ShadowrocketJSON(e)
	if err != nil {
		return err
	}
	_ = os.WriteFile(filepath.Join(dir, "shadowrocket-helper.json"), append(sr, 10), 0o600)
	nl := string([]byte{10})
	var uris string
	if e.RelayHost != "" {
		uris += ClientLinkForRelayLocal(name, uuid, e.RelayHost, e.RelayPBK, e.RelaySID, e.RelaySNI) + nl
	}
	uris += VLESSLink(name, uuid) + nl
	return os.WriteFile(filepath.Join(dir, "shadowrocket-uris.txt"), []byte(uris), 0o600)
}
