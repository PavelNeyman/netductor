package edgeagent

import (
	"fmt"
	"strings"
)

// DesiredUCI extracts uci set lines from template JSON-like map.
// Supports network (lan + wan), dhcp.lan, wifi (same SSID on radio0/radio1).
func DesiredUCI(tmpl map[string]any) []string {
	var lines []string
	if net, ok := tmpl["network"].(map[string]any); ok {
		if ip, ok := net["lan_ip"].(string); ok && ip != "" {
			lines = append(lines, "network.lan.ipaddr="+ip)
		}
		if mask, ok := net["lan_mask"].(string); ok && mask != "" {
			lines = append(lines, "network.lan.netmask="+mask)
		}
		proto, _ := net["wan_proto"].(string)
		proto = strings.ToLower(strings.TrimSpace(proto))
		if proto == "static" || proto == "dhcp" {
			lines = append(lines, "network.wan.proto="+proto)
		}
		if proto == "static" {
			if ip, ok := net["wan_ip"].(string); ok && ip != "" {
				lines = append(lines, "network.wan.ipaddr="+ip)
			}
			if mask, ok := net["wan_mask"].(string); ok && mask != "" {
				lines = append(lines, "network.wan.netmask="+mask)
			}
			if gw, ok := net["wan_gateway"].(string); ok && gw != "" {
				lines = append(lines, "network.wan.gateway="+gw)
			}
			if dns, ok := net["wan_dns"].(string); ok && dns != "" {
				lines = append(lines, "network.wan.dns="+dns)
			}
		}
	}
	if dhcp, ok := tmpl["dhcp"].(map[string]any); ok {
		if start, ok := dhcp["start"].(string); ok && start != "" {
			lines = append(lines, "dhcp.lan.start="+start)
		}
		if lim, ok := dhcp["limit"].(string); ok && lim != "" {
			lines = append(lines, "dhcp.lan.limit="+lim)
		}
		// also accept numbers from JSON
		if start, ok := dhcp["start"].(float64); ok {
			lines = append(lines, fmt.Sprintf("dhcp.lan.start=%d", int(start)))
		}
		if lim, ok := dhcp["limit"].(float64); ok {
			lines = append(lines, fmt.Sprintf("dhcp.lan.limit=%d", int(lim)))
		}
	}
	if wifi, ok := tmpl["wifi"].(map[string]any); ok {
		ssid, _ := wifi["ssid"].(string)
		key, _ := wifi["key"].(string)
		enc, _ := wifi["encryption"].(string)
		if enc == "" {
			enc = "psk2"
		}
		if ssid != "" {
			for _, radio := range []string{"default_radio0", "default_radio1"} {
				lines = append(lines,
					"wireless."+radio+".ssid="+ssid,
					"wireless."+radio+".encryption="+enc,
				)
				if key != "" {
					lines = append(lines, "wireless."+radio+".key="+key)
				}
			}
		}
	}
	return lines
}

// DiffUCI returns sets needed when current values differ (current map path->value).
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

// ShellApply builds a safe-ish OpenWrt shell snippet from desired uci lines.
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
		// uci set path=value — value may need quoting
		b.WriteString("uci set ")
		b.WriteString(parts[0])
		b.WriteString("=")
		b.WriteString(shellSingle(parts[1]))
		b.WriteByte('\n')
	}
	b.WriteString("uci commit network 2>/dev/null || true\n")
	b.WriteString("uci commit wireless 2>/dev/null || true\n")
	b.WriteString("uci commit dhcp 2>/dev/null || true\n")
	b.WriteString("/etc/init.d/network reload 2>/dev/null || true\n")
	b.WriteString("wifi reload 2>/dev/null || true\n")
	return b.String()
}

func shellSingle(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
