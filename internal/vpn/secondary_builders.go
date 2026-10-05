package vpn

import "strings"

// Secondary sing-box builders (R1): WriteSecondarySingBox stays orchestration-only.

type secondaryUser struct {
	Name string `json:"name,omitempty"`
	UUID string `json:"uuid"`
	Flow string `json:"flow"`
}

func buildSecondaryInbounds(b *SecondaryBundle, privKey, shortID string, users []secondaryUser) []any {
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
			"users": []secondaryUser{{UUID: b.ExitUUID, Flow: "xtls-rprx-vision"}},
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

func buildSecondaryUplinkTLS(b *SecondaryBundle) map[string]any {
	return map[string]any{
		"enabled": true, "server_name": b.CoreSNI,
		"utls": map[string]any{"enabled": true, "fingerprint": "firefox"},
		"reality": map[string]any{
			"enabled":    true,
			"public_key": b.CorePBK,
			"short_id":   b.CoreSID,
		},
	}
}

// buildSecondaryOutboundsAndSvcRules returns outbounds (uplink + per-user svc + direct/block)
// and service ACL rules for 10.88.0.0/24.
func buildSecondaryOutboundsAndSvcRules(b *SecondaryBundle) (outbounds []any, svcRules []any) {
	uplinkTLS := buildSecondaryUplinkTLS(b)
	uplink := map[string]any{
		"type": "vless", "tag": "uplink",
		"server": "10.87.10.1", "server_port": b.CoreVless,
		// no vision: required for multiplex (vision ⊕ mux unsupported)
		"uuid":            b.UplinkUUID,
		"domain_resolver": "quad9",
		"tls":             uplinkTLS,
	}
	if mx := uplinkMultiplexObject(); mx != nil {
		uplink["multiplex"] = mx
	}
	outbounds = []any{uplink}
	for _, u := range b.Users {
		name := strings.TrimSpace(u.Name)
		if name == "" || u.UUID == "" {
			continue
		}
		tag := "uplink-svc-" + sanitizeTag(name)
		outbounds = append(outbounds, map[string]any{
			"type": "vless", "tag": tag,
			"server": "10.87.10.1", "server_port": 9443,
			"uuid":            u.UUID,
			"domain_resolver": "quad9",
			"tls":             uplinkTLS,
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
	outbounds = append(outbounds,
		map[string]any{"type": "direct", "tag": "direct"},
		map[string]any{"type": "block", "tag": "block"},
	)
	return outbounds, svcRules
}

func buildSecondaryDNS(ruSuffixes []string) map[string]any {
	return map[string]any{
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
	}
}

func buildSecondaryRoute(svcRules []any, ruSuffixes []string) map[string]any {
	return map[string]any{
		// domain_suffix list (fast, no download) + geoip-ru rule-set (IP ranges).
		"rule_set": []any{
			map[string]any{
				"tag":    "geoip-ru",
				"type":   "remote",
				"format": "binary",
				"url":    "https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-ru.srs",
				// direct: uplink is Reality (SNI=api.vk.me) and breaks TLS to githubusercontent
				"download_detour": "direct",
			},
		},
		"rules":                   assembleSecondaryRouteRules(svcRules, ruSuffixes),
		"final":                   "uplink",
		"default_domain_resolver": "quad9",
		"auto_detect_interface":   true,
	}
}

// assembleSecondaryRouteRules keeps rule order explicit (R1 / R11b): service → sniff → DNS → geo → uplink.
func assembleSecondaryRouteRules(svcRules []any, ruSuffixes []string) []any {
	rules := append([]any{}, svcRules...)
	rules = append(rules,
		map[string]any{"action": "sniff"},
		map[string]any{"protocol": "dns", "action": "hijack-dns"},
		map[string]any{"ip_version": 6, "outbound": "block"},
		map[string]any{"inbound": []string{"exit-in"}, "outbound": "direct"},
		map[string]any{"ip_is_private": true, "outbound": "direct"},
		map[string]any{"domain_suffix": ruSuffixes, "outbound": "direct"},
		map[string]any{"domain_keyword": RuDirectKeywords(), "outbound": "direct"},
		map[string]any{"rule_set": []string{"geoip-ru"}, "outbound": "direct"},
		map[string]any{"inbound": []string{"relay-in"}, "outbound": "uplink"},
	)
	return rules
}
