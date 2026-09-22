package edgeagent

import (
	"fmt"
	"strings"
)

// DesiredUCI extracts uci set lines from template JSON-like map.
// network: lan_ip, lan_mask, wan_proto (dhcp|static|pppoe), wan_* / pppoe_*
// dhcp: start, limit
// wifi: ssid/key (both radios) or ssid_24/key_24 + ssid_5/key_5 (empty band inherits the other)
func DesiredUCI(tmpl map[string]any) []string {
	var lines []string
	if net, ok := tmpl["network"].(map[string]any); ok {
		if ip := str(net["lan_ip"]); ip != "" {
			lines = append(lines, "network.lan.ipaddr="+ip)
		}
		if mask := str(net["lan_mask"]); mask != "" {
			lines = append(lines, "network.lan.netmask="+mask)
		}
		proto := strings.ToLower(strings.TrimSpace(str(net["wan_proto"])))
		switch proto {
		case "static", "dhcp", "pppoe":
			lines = append(lines, "network.wan.proto="+proto)
		}
		switch proto {
		case "static":
			if ip := str(net["wan_ip"]); ip != "" {
				lines = append(lines, "network.wan.ipaddr="+ip)
			}
			if mask := str(net["wan_mask"]); mask != "" {
				lines = append(lines, "network.wan.netmask="+mask)
			}
			if gw := str(net["wan_gateway"]); gw != "" {
				lines = append(lines, "network.wan.gateway="+gw)
			}
			if dns := str(net["wan_dns"]); dns != "" {
				lines = append(lines, "network.wan.dns="+dns)
			}
		case "pppoe":
			if u := str(net["pppoe_user"]); u != "" {
				lines = append(lines, "network.wan.username="+u)
			}
			if p := str(net["pppoe_pass"]); p != "" {
				lines = append(lines, "network.wan.password="+p)
			}
			if svc := str(net["pppoe_service"]); svc != "" {
				lines = append(lines, "network.wan.service="+svc)
			}
			if ac := str(net["pppoe_ac"]); ac != "" {
				lines = append(lines, "network.wan.ac="+ac)
			}
			if dns := str(net["wan_dns"]); dns != "" {
				lines = append(lines, "network.wan.dns="+dns)
			}
		}
	}
	if dhcp, ok := tmpl["dhcp"].(map[string]any); ok {
		if start := strOrNum(dhcp["start"]); start != "" {
			lines = append(lines, "dhcp.lan.start="+start)
		}
		if lim := strOrNum(dhcp["limit"]); lim != "" {
			lines = append(lines, "dhcp.lan.limit="+lim)
		}
	}
	if wifi, ok := tmpl["wifi"].(map[string]any); ok {
		ssid24, key24, ssid5, key5 := resolveBands(wifi)
		enc := str(wifi["encryption"])
		if enc == "" {
			enc = "psk2"
		}
		if ssid24 != "" {
			lines = append(lines,
				"wireless.default_radio0.ssid="+ssid24,
				"wireless.default_radio0.encryption="+enc,
			)
			if key24 != "" {
				lines = append(lines, "wireless.default_radio0.key="+key24)
			}
		}
		if ssid5 != "" {
			lines = append(lines,
				"wireless.default_radio1.ssid="+ssid5,
				"wireless.default_radio1.encryption="+enc,
			)
			if key5 != "" {
				lines = append(lines, "wireless.default_radio1.key="+key5)
			}
		}
	}
	return lines
}

// resolveBands: fill empty 2.4 from 5 and vice versa; legacy ssid/key apply to both when band-specific empty.
func resolveBands(wifi map[string]any) (ssid24, key24, ssid5, key5 string) {
	ssid24 = str(wifi["ssid_24"])
	key24 = str(wifi["key_24"])
	ssid5 = str(wifi["ssid_5"])
	key5 = str(wifi["key_5"])
	legacySSID := str(wifi["ssid"])
	legacyKey := str(wifi["key"])
	if ssid24 == "" && ssid5 == "" && legacySSID != "" {
		ssid24, ssid5 = legacySSID, legacySSID
	}
	if key24 == "" && key5 == "" && legacyKey != "" {
		key24, key5 = legacyKey, legacyKey
	}
	if ssid24 == "" {
		ssid24 = ssid5
	}
	if ssid5 == "" {
		ssid5 = ssid24
	}
	if key24 == "" {
		key24 = key5
	}
	if key5 == "" {
		key5 = key24
	}
	return
}

func str(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func strOrNum(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case float64:
		return fmt.Sprintf("%d", int(x))
	case int:
		return fmt.Sprintf("%d", x)
	default:
		return ""
	}
}

func DiffUCI(desired []string, current map[string]string) (toSet []string) {
	for _, line := range desired {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		if current[parts[0]] == parts[1] {
			continue
		}
		toSet = append(toSet, line)
	}
	return toSet
}

func FormatApplyReport(toSet []string) string {
	if len(toSet) == 0 {
		return "no changes"
	}
	return fmt.Sprintf("changing %d keys", len(toSet))
}

func ShellApply(desired []string) string {
	if len(desired) == 0 {
		return "true"
	}
	var b strings.Builder
	b.WriteString("set -e\n")
	for _, line := range desired {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		b.WriteString("uci set ")
		b.WriteString(parts[0])
		b.WriteString("=")
		b.WriteString("'" + strings.ReplaceAll(parts[1], "'", `'"'"'`) + "'")
		b.WriteByte('\n')
	}
	b.WriteString("uci commit network 2>/dev/null || true\n")
	b.WriteString("uci commit wireless 2>/dev/null || true\n")
	b.WriteString("uci commit dhcp 2>/dev/null || true\n")
	b.WriteString("/etc/init.d/network reload 2>/dev/null || true\n")
	b.WriteString("wifi reload 2>/dev/null || true\n")
	return b.String()
}
