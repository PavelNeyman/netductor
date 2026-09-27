// Package svcpaths reports dual service WG-over-WSS paths (nd-svc-sp / nd-svc-ps).
package svcpaths

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	IfaceSP    = "nd-svc-sp"
	IfacePS    = "nd-svc-ps"
	HealthJSON = "/var/lib/netductor/svc-paths-health.json"
	KeysDir    = "/etc/netductor/svc-paths"
)

// StatusReport is machine-readable status for CLI/doctor.
type StatusReport struct {
	HostRole   string            `json:"host_role,omitempty"`
	SP         PathStatus        `json:"sp"`
	PS         PathStatus        `json:"ps"`
	HealthFile map[string]any    `json:"health_file,omitempty"`
	Units      map[string]string `json:"units"`
	KeysDirOK  bool              `json:"keys_dir_ok"`
	CheckedAt  string            `json:"checked_at"`
}

type PathStatus struct {
	Iface      string `json:"iface"`
	Up         bool   `json:"up"`
	HasAddr    bool   `json:"has_addr"`
	Addr       string `json:"addr,omitempty"`
	PeerPingOK *bool  `json:"peer_ping_ok,omitempty"`
}

func detectRole() string {
	if hasIP(IfaceSP, "10.87.10.1") || hasIP(IfacePS, "10.87.11.1") {
		return "primary"
	}
	if hasIP(IfaceSP, "10.87.10.2") || hasIP(IfacePS, "10.87.11.2") {
		return "secondary"
	}
	return "unknown"
}

func hasIP(iface, want string) bool {
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return false
	}
	addrs, err := ifi.Addrs()
	if err != nil {
		return false
	}
	for _, a := range addrs {
		if strings.HasPrefix(a.String(), want+"/") || a.String() == want {
			return true
		}
	}
	return false
}

func pathStatus(iface string) PathStatus {
	ps := PathStatus{Iface: iface}
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return ps
	}
	ps.Up = ifi.Flags&net.FlagUp != 0
	addrs, _ := ifi.Addrs()
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && ipnet.IP.To4() != nil {
			ps.HasAddr = true
			ps.Addr = ipnet.String()
			break
		}
	}
	peer := peerOf(ps.Addr)
	if peer != "" && ps.Up {
		ok := pingOnce(peer)
		ps.PeerPingOK = &ok
	}
	return ps
}

func peerOf(cidr string) string {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return ""
	}
	ones, bits := ipnet.Mask.Size()
	if ones != 30 || bits != 32 {
		return ""
	}
	b := ip.To4()
	if b == nil {
		return ""
	}
	last := b[3]
	base := last & 0xFC
	var peer net.IP
	if last == base+1 {
		peer = net.IPv4(b[0], b[1], b[2], base+2)
	} else if last == base+2 {
		peer = net.IPv4(b[0], b[1], b[2], base+1)
	} else {
		return ""
	}
	return peer.String()
}

func pingOnce(ip string) bool {
	cmd := exec.Command("ping", "-c", "1", "-W", "2", ip)
	return cmd.Run() == nil
}

func unitActive(name string) string {
	cmd := exec.Command("systemctl", "is-active", name)
	out, err := cmd.Output()
	s := strings.TrimSpace(string(out))
	if err != nil && s == "" {
		return "inactive"
	}
	if s == "" {
		return "unknown"
	}
	return s
}

// Status collects iface + unit + health JSON.
func Status() StatusReport {
	r := StatusReport{
		HostRole:  detectRole(),
		SP:        pathStatus(IfaceSP),
		PS:        pathStatus(IfacePS),
		Units:     map[string]string{},
		KeysDirOK: false,
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if st, err := os.Stat(KeysDir); err == nil && st.IsDir() {
		r.KeysDirOK = true
	}
	if b, err := os.ReadFile(HealthJSON); err == nil {
		var m map[string]any
		if json.Unmarshal(b, &m) == nil {
			r.HealthFile = m
		}
	}
	role := r.HostRole
	units := []string{
		"wg-quick@nd-svc-sp",
		"wg-quick@nd-svc-ps",
	}
	if role == "primary" {
		units = append(units, "nd-wss-sp-server", "nd-wss-ps-client")
	} else if role == "secondary" {
		units = append(units, "nd-wss-sp-client", "nd-wss-ps-server")
	} else {
		units = append(units, "nd-wss-sp-server", "nd-wss-ps-client", "nd-wss-sp-client", "nd-wss-ps-server")
	}
	for _, u := range units {
		r.Units[u] = unitActive(u)
	}
	return r
}

// Apply brings up wg-quick units and WSS helpers if conf exists.
func Apply() (string, error) {
	var msgs []string
	confDir := "/etc/wireguard"
	for _, iface := range []string{IfaceSP, IfacePS} {
		conf := filepath.Join(confDir, iface+".conf")
		if _, err := os.Stat(conf); err != nil {
			msgs = append(msgs, fmt.Sprintf("skip %s: no %s", iface, conf))
			continue
		}
		unit := "wg-quick@" + iface
		_ = exec.Command("systemctl", "enable", unit).Run()
		if out, err := exec.Command("systemctl", "restart", unit).CombinedOutput(); err != nil {
			return strings.Join(msgs, "\n"), fmt.Errorf("%s: %v (%s)", unit, err, strings.TrimSpace(string(out)))
		}
		msgs = append(msgs, unit+" restarted")
	}
	role := detectRole()
	wss := map[string][]string{
		"primary":   {"nd-wss-sp-server", "nd-wss-ps-client"},
		"secondary": {"nd-wss-sp-client", "nd-wss-ps-server"},
	}
	for _, u := range wss[role] {
		_ = exec.Command("systemctl", "enable", u).Run()
		if out, err := exec.Command("systemctl", "restart", u).CombinedOutput(); err != nil {
			msgs = append(msgs, fmt.Sprintf("%s: %v %s", u, err, strings.TrimSpace(string(out))))
			continue
		}
		msgs = append(msgs, u+" restarted")
	}
	return strings.Join(msgs, "\n"), nil
}
