package vpn

import (
	"fmt"
	"net"
	"os"
)

// Route / inbound builders (R1): keep ApplyConfig thin; edit order of rules carefully.

const svcPlaneListen = "10.87.10.1"
const svcPlanePort = 9443

// localIPv4Present reports whether addr is assigned on a local interface.
// Used so vless-svc does not crash sing-box when service-plane WG is down.
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

func buildPrimaryInbounds(vusers, svcUsers any, sniVal, priv, sid string) []any {
	inbounds := []any{
		map[string]any{
			"type": "vless", "tag": "vless-reality", "listen": "::", "listen_port": vlessPort(),
			"users": vusers,
			// multiplex for secondary uplink (no vision). End-user vision streams stay non-mux.
			// inbound mux (sing-box ≥1.10): no max_connections here — those are outbound-only
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
	// Service-plane inbound only when 10.87.10.1 is up. Binding a missing address
	// is FATAL in sing-box and crash-loops the whole process (public :443 dies too).
	if localIPv4Present(svcPlaneListen) {
		inbounds = append(inbounds, map[string]any{
			"type": "vless", "tag": "vless-svc", "listen": svcPlaneListen, "listen_port": svcPlanePort,
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
	} else {
		fmt.Fprintln(os.Stderr, "vpn: skip vless-svc —", svcPlaneListen, "not assigned (bring up service-plane WG)")
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
