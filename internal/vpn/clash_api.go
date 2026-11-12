package vpn

import (
	"os"
	"strconv"
	"strings"
)

// DefaultClashAPIAddr is loopback-only sing-box clash/stats controller (C11).
// Never bind 0.0.0.0 — channel users and agent scrape 127.0.0.1 only.
const DefaultClashAPIAddr = "127.0.0.1:9090"

// ClashAPIListenAddr returns host:port for experimental.clash_api.external_controller.
// Override: NETDUCTOR_CLASH_API=127.0.0.1:9090 or empty to disable.
func ClashAPIListenAddr() string {
	v := strings.TrimSpace(os.Getenv("NETDUCTOR_CLASH_API"))
	if v == "0" || v == "off" || v == "false" || v == "no" {
		return ""
	}
	if v != "" {
		// force loopback prefix if operator passed bare port
		if !strings.Contains(v, ":") {
			if _, err := strconv.Atoi(v); err == nil {
				return "127.0.0.1:" + v
			}
		}
		if strings.HasPrefix(v, "0.0.0.0:") || strings.HasPrefix(v, "[::]:") || strings.HasPrefix(v, "*:") {
			// refuse non-loopback — rewrite to 127.0.0.1 with same port
			if i := strings.LastIndex(v, ":"); i > 0 {
				return "127.0.0.1" + v[i:]
			}
			return DefaultClashAPIAddr
		}
		return v
	}
	return DefaultClashAPIAddr
}

// ExperimentalClashAPI returns the sing-box "experimental" object, or nil if disabled.
func ExperimentalClashAPI() map[string]any {
	addr := ClashAPIListenAddr()
	if addr == "" {
		return nil
	}
	return map[string]any{
		"clash_api": map[string]any{
			"external_controller": addr,
			// empty secret: localhost only; firewall must deny WAN :9090
			"secret": "",
		},
	}
}

// AttachExperimentalClashAPI sets cfg["experimental"] when enabled (idempotent for Apply).
func AttachExperimentalClashAPI(cfg map[string]any) {
	if cfg == nil {
		return
	}
	exp := ExperimentalClashAPI()
	if exp == nil {
		return
	}
	cfg["experimental"] = exp
}
