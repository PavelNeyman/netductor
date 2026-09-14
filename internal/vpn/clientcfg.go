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

func loadOnlineRelays() []struct{ Host, PBK, SID, SNI string } {
	p := filepath.Join(paths.StateDir(), "relay", "devices.json")
	b, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var f relayDevFile
	if json.Unmarshal(b, &f) != nil {
		return nil
	}
	now := time.Now().UTC()
	var out []struct{ Host, PBK, SID, SNI string }
	for _, d := range f.Devices {
		if d.PublicIP == "" || d.PBK == "" {
			continue
		}
		if !d.LastSeen.IsZero() && now.Sub(d.LastSeen) > 3*time.Minute {
			continue
		}
		sniName := d.SNI
		if sniName == "" {
			sniName = DefaultRealitySNI
		}
		out = append(out, struct{ Host, PBK, SID, SNI string }{d.PublicIP, d.PBK, d.SID, sniName})
	}
	return out
}

func loadOnlineRelay() (host, pbk, sid, sniName string) {
	rels := loadOnlineRelays()
	if len(rels) == 0 {
		return
	}
	return rels[0].Host, rels[0].PBK, rels[0].SID, rels[0].SNI
}

func ResolveClientEndpoints(name, uuid string) ClientEndpoints {
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
		sniName = DefaultRealitySNI
	}
	return fmt.Sprintf(
		"vless://%s@%s:443?encryption=none&flow=xtls-rprx-vision&security=reality&sni=%s&fp=%s&pbk=%s&sid=%s&type=tcp#nd-relay",
		uuid, relayIP, sniName, DefaultUTLSFingerprint, pbk, sid,
	)
}

func vlessOutbound(tag, host string, port int, uuid, sniName, pbk, sid string, detour string) map[string]any {
	o := map[string]any{
		"type": "vless", "tag": tag,
		"server": host, "server_port": port,
		"uuid": uuid, "flow": "xtls-rprx-vision",
		"tls": map[string]any{
			"enabled": true, "server_name": sniName,
			"utls":    map[string]any{"enabled": true, "fingerprint": DefaultUTLSFingerprint},
			"reality": map[string]any{"enabled": true, "public_key": pbk, "short_id": sid},
		},
	}
	if detour != "" {
		o["detour"] = detour
	}
	return o
}

// SingBoxClientJSON builds a WL-oriented client:
// - IPv4 preferred DNS/route
// - block UDP/443 and IPv6
// - RU DNS via Yandex when possible
// - dual-hop: core detours via relay when relay is online (dialerProxy analogue)
func SingBoxClientJSON(e ClientEndpoints) ([]byte, error) {
	if e.UUID == "" {
		return nil, fmt.Errorf("uuid required")
	}
	if e.CoreSNI == "" {
		e.CoreSNI = DefaultRealitySNI
	}
	if e.RelaySNI == "" {
		e.RelaySNI = DefaultRealitySNI
	}
	outbounds := []any{
		map[string]any{"type": "direct", "tag": "direct"},
		map[string]any{"type": "block", "tag": "block"},
	}
	final := "direct"
	hasRelay := e.RelayHost != "" && e.RelayPBK != ""
	hasCore := e.CoreHost != "" && e.CorePBK != ""

	if hasRelay {
		outbounds = append(outbounds, vlessOutbound("relay", e.RelayHost, e.RelayPort, e.UUID, e.RelaySNI, e.RelayPBK, e.RelaySID, ""))
		final = "relay"
	}
	if hasCore {
		detour := ""
		tag := "core"
		if hasRelay {
			// second hop: dial core through established relay (commercial dual-hop pattern)
			detour = "relay"
			tag = "core-via-relay"
			outbounds = append(outbounds, vlessOutbound(tag, e.CoreHost, e.CorePort, e.UUID, e.CoreSNI, e.CorePBK, e.CoreSID, detour))
			// also plain core for home ISP without WL
			outbounds = append(outbounds, vlessOutbound("core", e.CoreHost, e.CorePort, e.UUID, e.CoreSNI, e.CorePBK, e.CoreSID, ""))
			final = tag
		} else {
			outbounds = append(outbounds, vlessOutbound("core", e.CoreHost, e.CorePort, e.UUID, e.CoreSNI, e.CorePBK, e.CoreSID, ""))
			final = "core"
		}
	}

	cfg := map[string]any{
		"log": map[string]any{"level": "warn"},
		"dns": map[string]any{
			"servers": []any{
				map[string]any{"type": "udp", "tag": "ya", "server": "77.88.8.8"},
				map[string]any{"type": "udp", "tag": "quad9", "server": "9.9.9.9", "detour": final},
				map[string]any{"type": "udp", "tag": "google", "server": "8.8.8.8", "detour": final},
			},
			"rules": []any{
				map[string]any{"domain_suffix": []string{".ru", ".su", "vk.com", "yandex.ru", "ya.ru", "vk.me"}, "server": "ya"},
			},
			"final":          "quad9",
			"strategy":       "ipv4_only",
			"independent_cache": true,
		},
		"inbounds": []any{
			map[string]any{
				"type": "tun", "tag": "tun-in",
				"address": []string{"172.19.0.1/30"},
				"auto_route": true, "strict_route": true,
				"sniff": true,
			},
		},
		"outbounds": outbounds,
		"route": map[string]any{
			"rules": []any{
				map[string]any{"action": "sniff"},
				map[string]any{"protocol": "dns", "action": "hijack-dns"},
				map[string]any{"ip_is_private": true, "outbound": "direct"},
				// WL tip: UDP/443 (QUIC) is usually dropped — avoid stalls
				map[string]any{"network": "udp", "port": 443, "outbound": "block"},
				map[string]any{"ip_version": 6, "outbound": "block"},
				// RU-ish domains: direct when possible (home / relay exit-RU later)
				map[string]any{
					"domain_suffix": []string{".ru", ".su", ".xn--p1ai"},
					"outbound":      "direct",
				},
			},
			"final":                 final,
			"auto_detect_interface": true,
			"default_domain_resolver": "ya",
		},
	}
	return json.MarshalIndent(cfg, "", "  ")
}

// ShadowrocketJSON helper + URIs (relay first).
func ShadowrocketJSON(e ClientEndpoints) ([]byte, error) {
	if e.UUID == "" {
		return nil, fmt.Errorf("uuid required")
	}
	uris := []string{}
	seen := map[string]bool{}
	for _, r := range loadOnlineRelays() {
		if seen[r.Host] {
			continue
		}
		seen[r.Host] = true
		uris = append(uris, ClientLinkForRelayLocal(e.Name, e.UUID, r.Host, r.PBK, r.SID, r.SNI))
	}
	if e.RelayHost != "" && !seen[e.RelayHost] {
		uris = append(uris, ClientLinkForRelayLocal(e.Name, e.UUID, e.RelayHost, e.RelayPBK, e.RelaySID, e.RelaySNI))
	}
	if e.CoreHost != "" {
		uris = append(uris, VLESSLink(e.Name, e.UUID))
	}
	doc := map[string]any{
		"remarks": "netductor WL profile",
		"note":    "Primary URI is relay when online. HY2 optional (not for carrier WL). Keep flow=xtls-rprx-vision. See docs/WL.md.",
		"uris":    uris,
	}
	return json.MarshalIndent(doc, "", "  ")
}

// WriteClientConfigs writes sing-box + shadowrocket helper files under client dir.
func WriteClientConfigs(name, uuid string) error {
	e := ResolveClientEndpoints(name, uuid)
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
	_ = os.WriteFile(filepath.Join(dir, "shadowrocket-uris.txt"), []byte(uris), 0o600)
	vless := PreferredVLESSLink(name, uuid)
	_ = os.WriteFile(filepath.Join(dir, "link-vless.txt"), []byte(vless+nl), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "link.txt"), []byte(vless+nl), 0o600)
	core := VLESSLink(name, uuid)
	_ = os.WriteFile(filepath.Join(dir, "link-vless-core.txt"), []byte(core+nl), 0o600)
	// subscription intentionally not advertised; dual single links only
	_ = os.Remove(filepath.Join(dir, "subscription.txt"))
	_ = os.Remove(filepath.Join(dir, "subscription.b64"))
	_ = os.Remove(filepath.Join(dir, "qr-subscription.png"))
	return nil
}
