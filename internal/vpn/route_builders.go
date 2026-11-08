package vpn

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Route / inbound builders (R1): keep ApplyConfig thin; edit order of rules carefully.

const svcPlanePort = 9443

// peerMaterialAddrP_SPList scans multi-peer materials (primary) + host material (secondary).
// Root material.json is often legacy without addr_* fields; slot addresses live under peers/<id>/.
func peerMaterialAddrP_SPList() []string {
	seen := map[string]bool{}
	var out []string
	add := func(a string) {
		a = strings.TrimSpace(a)
		if a == "" || seen[a] {
			return
		}
		seen[a] = true
		out = append(out, a)
	}
	roots := []string{
		filepath.Join(paths.EtcDir(), "svc-paths"),
		"/etc/netductor/svc-paths",
	}
	if st := paths.StateDir(); st != "" {
		roots = append(roots, filepath.Join(st, "svc-paths"))
	}
	type mat struct {
		AddrP_SP string `json:"addr_p_sp"`
	}
	readMat := func(path string) {
		b, err := os.ReadFile(path)
		if err != nil {
			return
		}
		var m mat
		if json.Unmarshal(b, &m) == nil {
			add(m.AddrP_SP)
		}
	}
	for _, root := range roots {
		readMat(filepath.Join(root, "material.json"))
		entries, err := os.ReadDir(filepath.Join(root, "peers"))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				readMat(filepath.Join(root, "peers", e.Name(), "material.json"))
			}
		}
	}
	return out
}

// localSPPrimaryFromIfaces returns IPv4 addresses on nd-svc-sp* interfaces.
func localSPPrimaryFromIfaces() []string {
	seen := map[string]bool{}
	var out []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return out
	}
	for _, iface := range ifaces {
		if !strings.HasPrefix(iface.Name, "nd-svc-sp") {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.To4() == nil {
				continue
			}
			s := ip.To4().String()
			if seen[s] {
				continue
			}
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
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

// primarySPListenAddrs — bind vless-svc on:
// 1) live nd-svc-sp* addresses (source of truth on primary)
// 2) peer materials whose addr_p_sp is currently local
func primarySPListenAddrs() []string {
	seen := map[string]bool{}
	var out []string
	add := func(a string) {
		if a == "" || seen[a] {
			return
		}
		seen[a] = true
		out = append(out, a)
	}
	for _, a := range localSPPrimaryFromIfaces() {
		add(a)
	}
	for _, a := range peerMaterialAddrP_SPList() {
		if localIPv4Present(a) {
			add(a)
		}
	}
	return out
}

// primarySPAddrForSecondary — SP primary IP secondary dials over WG (from material).
func primarySPAddrForSecondary() string {
	list := peerMaterialAddrP_SPList()
	if len(list) > 0 {
		return list[0]
	}
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_PRIMARY_SP_ADDR")); v != "" {
		return v
	}
	return ""
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
