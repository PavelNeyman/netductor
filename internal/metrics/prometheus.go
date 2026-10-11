package metrics

import (
	"fmt"
	"strings"
)

// PrometheusText renders Collect() as Prometheus exposition format (text/plain).
// Session-gated by the HTTP handler; suitable for scrape via operator tunnel.
func PrometheusText() string {
	m := Collect()
	var b strings.Builder
	b.WriteString("# HELP netductor_up 1 if collect succeeded\n# TYPE netductor_up gauge\n")
	b.WriteString("netductor_up 1\n")

	if la, ok := m["loadavg"].(map[string]float64); ok {
		b.WriteString("# HELP netductor_loadavg System load average\n# TYPE netductor_loadavg gauge\n")
		for _, k := range []string{"1", "5", "15"} {
			if v, ok := la[k]; ok {
				fmt.Fprintf(&b, "netductor_loadavg{window=%q} %g\n", k, v)
			}
		}
	}
	if mem, ok := m["mem"].(map[string]int64); ok {
		b.WriteString("# HELP netductor_mem_bytes Memory bytes\n# TYPE netductor_mem_bytes gauge\n")
		for _, k := range []string{"total", "available", "used"} {
			if v, ok := mem[k]; ok {
				fmt.Fprintf(&b, "netductor_mem_bytes{state=%q} %d\n", k, v)
			}
		}
	}
	if disk, ok := m["disk"].(map[string]any); ok {
		b.WriteString("# HELP netductor_disk_bytes Root filesystem bytes\n# TYPE netductor_disk_bytes gauge\n")
		for _, k := range []string{"total", "used", "available"} {
			if v, ok := asInt64(disk[k]); ok {
				fmt.Fprintf(&b, "netductor_disk_bytes{state=%q} %d\n", k, v)
			}
		}
		if pct, ok := asInt64(disk["use_pct"]); ok {
			b.WriteString("# HELP netductor_disk_use_ratio Root filesystem used ratio\n# TYPE netductor_disk_use_ratio gauge\n")
			fmt.Fprintf(&b, "netductor_disk_use_ratio %g\n", float64(pct)/100)
		}
	}
	if svcs, ok := m["services"].(map[string]string); ok {
		b.WriteString("# HELP netductor_service_up 1 if systemd unit is active\n# TYPE netductor_service_up gauge\n")
		for name, st := range svcs {
			up := 0
			if st == "active" {
				up = 1
			}
			fmt.Fprintf(&b, "netductor_service_up{name=%q} %d\n", name, up)
		}
	}
	if fw, ok := m["firewall"].(map[string]any); ok {
		b.WriteString("# HELP netductor_firewall_ok 1 if firewall status is OK\n# TYPE netductor_firewall_ok gauge\n")
		okv := 0
		if v, _ := fw["ok"].(bool); v {
			okv = 1
		}
		fmt.Fprintf(&b, "netductor_firewall_ok %d\n", okv)
	}
	if n, ok := asInt64(m["vpn_endpoints"]); ok {
		b.WriteString("# HELP netductor_vpn_endpoints Historical unique VPN client endpoints\n# TYPE netductor_vpn_endpoints gauge\n")
		fmt.Fprintf(&b, "netductor_vpn_endpoints %d\n", n)
	}
	if n, ok := asInt64(m["path_e2e_ok"]); ok {
		b.WriteString("# HELP netductor_path_e2e_ok Path e2e probes OK\n# TYPE netductor_path_e2e_ok gauge\n")
		fmt.Fprintf(&b, "netductor_path_e2e_ok %d\n", n)
	}
	if n, ok := asInt64(m["path_e2e_total"]); ok {
		b.WriteString("# HELP netductor_path_e2e_total Path e2e probes total\n# TYPE netductor_path_e2e_total gauge\n")
		fmt.Fprintf(&b, "netductor_path_e2e_total %d\n", n)
	}
	if n, ok := asInt64(m["local_release_age_sec"]); ok {
		b.WriteString("# HELP netductor_local_release_age_seconds Age of latest local release tag\n# TYPE netductor_local_release_age_seconds gauge\n")
		fmt.Fprintf(&b, "netductor_local_release_age_seconds %d\n", n)
	}
	return b.String()
}

func asInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		return int64(n), true
	default:
		return 0, false
	}
}
