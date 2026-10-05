package vpn

// Route / inbound builders (R1): keep ApplyConfig thin; edit order of rules carefully.

func buildPrimaryInbounds(vusers, svcUsers any, sniVal, priv, sid string) []any {
	return []any{
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
		map[string]any{
			"type": "vless", "tag": "vless-svc", "listen": "10.87.10.1", "listen_port": 9443,
			"users": svcUsers,
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
