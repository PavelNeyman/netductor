package vpn

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Route / inbound builders (R1): keep ApplyConfig thin; edit order of rules carefully.

const svcPlanePort = 9443

// knownSPPrimary mirrors svcpaths.SlotTable AddrP_SP (avoid import cycle vpn↔svcpaths).
var knownSPPrimary = []string{
	"10.87.10.1", // slot 0
	"10.87.20.1", // slot 1
	"10.87.22.1", // slot 2
	"10.87.24.1", // slot 3
}

func localIPv4Present(addr string) bool {
	ip := net.ParseIP(addr)
	if ip == nil {
		return false
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			var cand net.IP
			switch v := a.(type) {
			case *net.IPNet:
				cand = v.IP
			case *net.IPAddr:
				cand = v.IP
			}
			if cand != nil && cand.Equal(ip) {
				return true
			}
		}
	}
	return false
}

// materialAddrP_SP reads addr_p_sp from host material.json (slot-aware).
func materialAddrP_SP() string {
	candidates := []string{
		filepath.Join(paths.EtcDir(), "svc-paths", "material.json"),
		"/etc/netductor/svc-paths/material.json",
	}
	// also first peer material under state
	if st := paths.StateDir(); st != "" {
		candidates = append(candidates, filepath.Join(st, "svc-paths", "material.json"))
	}
	for _, path := range candidates {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var m struct {
			AddrP_SP string `json:"addr_p_sp"`
		}
		if json.Unmarshal(b, &m) == nil && m.AddrP_SP != "" {
			return m.AddrP_SP
		}
	}
	return ""
}

// primarySPListenAddrs: SP primary IPs present on this host (multi-slot safe).
func primarySPListenAddrs() []string {
	seen := map[string]bool{}
	var out []string
	if a := materialAddrP_SP(); a != "" && localIPv4Present(a) {
		seen[a] = true
		out = append(out, a)
	}
	for _, a := range knownSPPrimary {
		if seen[a] || !localIPv4Present(a) {
			continue
		}
		seen[a] = true
		out = append(out, a)
	}
	return out
}

// primarySPAddrForSecondary is the primary address secondary should dial over SP WG.
func primarySPAddrForSecondary() string {
	if a := materialAddrP_SP(); a != "" {
		return a
	}
	return knownSPPrimary[0]
}

func buildPrimaryInbounds(vusers, svcUsers any, sniVal, priv, sid string) []any {
	inbounds := []any{
		map[string]any{
			"type": "vless", "tag": "vless-reality", "listen": "::", "listen_port": vlessPort(),
			"users": vusers,
			"multiplex": map[string]any{
				"enabled": true,
				"padding": true,
			},
			"tls": map[string]any{
				"enabled": true, "server_name": sniVal,
				"reality": map[string]any{
					"enabled":     true,
					"handshake":   map[string]any{"server": sniVal, "server_port": 443},
					"private_key": priv, "short_id": []string{sid},
				},
			},
		},
	}
	spAddrs := primarySPListenAddrs()
	if len(spAddrs) == 0 {
		fmt.Fprintln(os.Stderr, "vpn: skip vless-svc — no local SP primary address (service-plane WG)")
	}
	for i, addr := range spAddrs {
		tag := "vless-svc"
		if i > 0 {
			tag = fmt.Sprintf("vless-svc-%d", i)
		}
		inbounds = append(inbounds, map[string]any{
			"type": "vless", "tag": tag, "listen": addr, "listen_port": svcPlanePort,
			"users": svcUsers,
			"tls": map[string]any{
				"enabled": true, "server_name": sniVal,
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

func buildOutboundsAndRoute() (outbounds []any, routeRules []any, finalOut string) {
	outbounds = []any{
		map[string]any{"type": "direct", "tag": "direct"},
		map[string]any{"type": "block", "tag": "block"},
	}
	// Service ACL before sniff: sniff on the vless-svc→lampac path ate the HTTP response
	// (client saw empty reply while loopback already had 200).
	routeRules = append([]any{}, policyServiceRules()...)
	routeRules = append(routeRules,
		map[string]any{"action": "sniff"},
		map[string]any{"protocol": "dns", "action": "hijack-dns"},
	)
	finalOut = "direct"
	exitOn, ip, pbk, sid, sniR := readExitTarget()
	exitUUID := secret("secondary_exit_uuid")
	if !exitOn || ip == "" || pbk == "" || exitUUID == "" {
		return
	}
	if sniR == "" {
		sniR = DefaultRealitySNI
	}
	outbounds = append(outbounds, map[string]any{
		"type": "vless", "tag": "ru-exit",
		"server": ip, "server_port": 4443,
		"uuid": exitUUID, "flow": "xtls-rprx-vision",
		"tls": map[string]any{
			"enabled": true, "server_name": sniR,
			"utls": map[string]any{"enabled": true, "fingerprint": "chrome"},
			"reality": map[string]any{
				"enabled": true, "public_key": pbk, "short_id": sid,
			},
		},
	})
	finalOut = "ru-exit"
	return
}
