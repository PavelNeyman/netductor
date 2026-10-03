// Package servicenet provides a small primary-only internal net (default 10.88.0.0/24)
// for policy-controlled access to loopback-bound apps via VIP + DNAT.
// See docs/PLAN-SERVICE-ACCESS-POLICY.md (service-net / P4).
package servicenet

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/PavelNeyman/netductor/internal/ndconfig"
	"github.com/PavelNeyman/netductor/internal/policy"
)

const (
	Iface   = "nd-svc"
	Gateway = "10.88.0.1"
	Prefix  = 24
	CIDR    = "10.88.0.0/24"
)

// DefaultVIP maps catalog service id → VIP on nd-svc (host side DNAT to loopback).
var DefaultVIP = map[string]string{
	"lampac":   "10.88.0.10",
	"git":      "10.88.0.11",
	"registry": "10.88.0.12",
	"nvr":      "10.88.0.13",
}

// Status is JSON-friendly.
type Status struct {
	Enabled bool     `json:"enabled"`
	Iface   string   `json:"iface"`
	Gateway string   `json:"gateway"`
	CIDR    string   `json:"cidr"`
	Addrs   []string `json:"addrs,omitempty"`
	OK      bool     `json:"ok"`
	Detail  string   `json:"detail,omitempty"`
}

// Enabled reports whether service-net should be configured (env NETDUCTOR_SERVICE_NET=0 disables).
func Enabled() bool {
	v := strings.TrimSpace(os.Getenv("NETDUCTOR_SERVICE_NET"))
	return v != "0" && v != "false" && v != "off"
}

// Collect reads current iface state.
func Collect() Status {
	st := Status{Iface: Iface, Gateway: Gateway, CIDR: CIDR, Enabled: Enabled()}
	if !st.Enabled {
		st.Detail = "disabled by NETDUCTOR_SERVICE_NET"
		st.OK = true
		return st
	}
	out, err := exec.Command("ip", "-o", "addr", "show", "dev", Iface).CombinedOutput()
	if err != nil {
		st.Detail = "iface missing"
		st.OK = false
		return st
	}
	s := string(out)
	st.Detail = strings.TrimSpace(s)
	for _, line := range strings.Split(s, "\n") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if f == "inet" && i+1 < len(fields) {
				st.Addrs = append(st.Addrs, fields[i+1])
			}
		}
	}
	st.OK = strings.Contains(s, Gateway)
	return st
}

// Ensure brings up nd-svc, assigns gateway + per-service VIPs, installs DNAT to loopback ports.
func Ensure() error {
	if !Enabled() {
		return nil
	}
	if err := ensureIface(); err != nil {
		return err
	}
	if err := ensureAddr(Gateway + "/24"); err != nil {
		return err
	}
	for _, vip := range DefaultVIP {
		_ = ensureAddr(vip + "/32")
	}
	cat, err := policy.EnsureCatalog()
	if err != nil {
		return err
	}
	_ = MergeCatalogServiceNet(cat)
	_ = policy.SaveCatalog(cat)
	if err := applyDNAT(cat); err != nil {
		return err
	}
	return nil
}

func ensureIface() error {
	if out, err := exec.Command("ip", "link", "show", Iface).CombinedOutput(); err != nil {
		_ = out
		if out, err := exec.Command("ip", "link", "add", Iface, "type", "dummy").CombinedOutput(); err != nil {
			// dummy may need modprobe
			_ = exec.Command("modprobe", "dummy").Run()
			if out2, err2 := exec.Command("ip", "link", "add", Iface, "type", "dummy").CombinedOutput(); err2 != nil {
				return fmt.Errorf("ip link add %s: %v (%s; %s)", Iface, err2, out, out2)
			}
		}
	}
	if out, err := exec.Command("ip", "link", "set", Iface, "up").CombinedOutput(); err != nil {
		return fmt.Errorf("ip link set up: %v (%s)", err, out)
	}
	return nil
}

func ensureAddr(cidr string) error {
	// idempotent: ignore "File exists"
	out, err := exec.Command("ip", "addr", "add", cidr, "dev", Iface).CombinedOutput()
	if err != nil {
		s := string(out)
		if strings.Contains(s, "File exists") || strings.Contains(s, "already") {
			return nil
		}
		return fmt.Errorf("ip addr add %s: %v (%s)", cidr, err, s)
	}
	return nil
}

func applyDNAT(cat *policy.Catalog) error {
	if cat == nil {
		return nil
	}
	_ = exec.Command("iptables", "-t", "nat", "-N", "NETDUCTOR_SVC").Run()
	_ = exec.Command("iptables", "-t", "nat", "-F", "NETDUCTOR_SVC").Run()
	for _, hook := range []string{"PREROUTING", "OUTPUT"} {
		if err := exec.Command("iptables", "-t", "nat", "-C", hook, "-j", "NETDUCTOR_SVC").Run(); err != nil {
			_ = exec.Command("iptables", "-t", "nat", "-A", hook, "-j", "NETDUCTOR_SVC").Run()
		}
	}
	for _, svc := range cat.Services {
		if svc.Disabled || svc.Kind != policy.KindInternal {
			continue
		}
		vip := DefaultVIP[svc.ID]
		if vip == "" {
			continue
		}
		for _, ep := range svc.Endpoints {
			if ep.Port <= 0 {
				continue
			}
			dst := "127.0.0.1"
			port := ep.Port
			if ep.Network == "loopback" && ep.Addr != "" {
				if ip := net.ParseIP(ep.Addr); ip != nil {
					dst = ep.Addr
				}
			}
			if svc.ID == "lampac" {
				if p := ndconfig.LampacPort(); p != "" {
					fmt.Sscanf(p, "%d", &port)
				}
			}
			proto := strings.ToLower(ep.Proto)
			if proto == "" {
				proto = "tcp"
			}
			// only one DNAT rule per vip:port from loopback-oriented endpoints
			if ep.Network == "service" {
				continue
			}
			spec := fmt.Sprintf("%s:%d", dst, port)
			_ = exec.Command("iptables", "-t", "nat", "-A", "NETDUCTOR_SVC",
				"-d", vip, "-p", proto, "--dport", fmt.Sprintf("%d", port),
				"-j", "DNAT", "--to-destination", spec).Run()
		}
	}
	return nil
}

// MergeCatalogServiceNet adds service-net endpoints alongside loopback without removing existing.
func MergeCatalogServiceNet(cat *policy.Catalog) bool {
	if cat == nil {
		return false
	}
	changed := false
	for i := range cat.Services {
		s := &cat.Services[i]
		vip := DefaultVIP[s.ID]
		if vip == "" || s.Kind != policy.KindInternal {
			continue
		}
		port := 0
		proto := "tcp"
		for _, ep := range s.Endpoints {
			if ep.Port > 0 {
				port = ep.Port
				if ep.Proto != "" {
					proto = ep.Proto
				}
				break
			}
		}
		if s.ID == "lampac" {
			if p := ndconfig.LampacPort(); p != "" {
				fmt.Sscanf(p, "%d", &port)
			}
			if port == 0 {
				port = 9118
			}
		}
		if port == 0 {
			continue
		}
		have := false
		for _, ep := range s.Endpoints {
			if ep.Network == "service" && ep.Addr == vip {
				have = true
				break
			}
		}
		if !have {
			s.Endpoints = append(s.Endpoints, policy.Endpoint{
				Network: "service",
				Addr:    vip,
				Port:    port,
				Proto:   proto,
			})
			changed = true
		}
	}
	return changed
}
