package svcpaths

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Primary↔Build dual WG (same pattern as SP/PS, own subnets — no VPN/secondary roles).
const (
	AddrP_PB = "10.87.12.1"
	AddrB_PB = "10.87.12.2"
	AddrP_BP = "10.87.13.1"
	AddrB_BP = "10.87.13.2"
	PortPB   = 51834
	PortBP   = 51836
	WSS_PB   = 8446
	WSS_BP   = 8447
	IfacePB  = "nd-svc-pb"
	IfaceBP  = "nd-svc-bp"
)

// BuildMaterial dual path keys between primary and build-host.
type BuildMaterial struct {
	PBPrivP string `json:"pb_priv_p"`
	PBPubP  string `json:"pb_pub_p"`
	PBPrivB string `json:"pb_priv_b"`
	PBPubB  string `json:"pb_pub_b"`
	BPPrivP string `json:"bp_priv_p"`
	BPPubP  string `json:"bp_pub_p"`
	BPPrivB string `json:"bp_priv_b"`
	BPPubB  string `json:"bp_pub_b"`
	Primary string `json:"primary_ip"`
	Build   string `json:"build_ip"`
}

func GenerateBuildMaterial(primaryIP, buildIP string) (*BuildMaterial, error) {
	if err := ensureWireguardTools(); err != nil {
		return nil, err
	}
	m := &BuildMaterial{Primary: primaryIP, Build: buildIP}
	var err error
	if m.PBPrivP, m.PBPubP, err = wgGen(); err != nil {
		return nil, err
	}
	if m.PBPrivB, m.PBPubB, err = wgGen(); err != nil {
		return nil, err
	}
	if m.BPPrivP, m.BPPubP, err = wgGen(); err != nil {
		return nil, err
	}
	if m.BPPrivB, m.BPPubB, err = wgGen(); err != nil {
		return nil, err
	}
	return m, nil
}

func SaveBuildMaterial(m *BuildMaterial) error {
	_ = os.MkdirAll(KeysDir, 0o700)
	b, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(filepath.Join(KeysDir, "build-material.json"), b, 0o600)
}

// BootstrapPrimaryBuild applies PB/BP on primary (listen PB server side, BP via wstunnel).
func BootstrapPrimaryBuild(m *BuildMaterial) error {
	if m == nil {
		return fmt.Errorf("nil build material")
	}
	if err := ensureWireguardTools(); err != nil {
		return err
	}
	if err := ensureWstunnel(); err != nil {
		return err
	}
	_ = SaveBuildMaterial(m)

	// PB: primary listens WG, build connects via WSS client on build side
	pbConf := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s/30
ListenPort = %d
Table = off

[Peer]
PublicKey = %s
AllowedIPs = %s/32
`, m.PBPrivP, AddrP_PB, PortPB, m.PBPubB, AddrB_PB)

	// BP: primary is client toward build's listen (via local wstunnel)
	bpConf := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s/30
Table = off

[Peer]
PublicKey = %s
AllowedIPs = %s/32
Endpoint = 127.0.0.1:%d
PersistentKeepalive = 25
`, m.BPPrivP, AddrP_BP, m.BPPubB, AddrB_BP, PortBP)

	_ = os.MkdirAll("/etc/wireguard", 0o700)
	_ = writeFile("/etc/wireguard/nd-svc-pb.conf", pbConf, 0o600)
	_ = writeFile("/etc/wireguard/nd-svc-bp.conf", bpConf, 0o600)

	_ = unitFile("nd-wss-pb-server",
		fmt.Sprintf("/usr/local/bin/wstunnel server wss://0.0.0.0:%d --restrict-to 127.0.0.1:%d", WSS_PB, PortPB))
	_ = unitFile("nd-wss-bp-client",
		fmt.Sprintf("/usr/local/bin/wstunnel client -L udp://127.0.0.1:%d:127.0.0.1:%d?timeout_sec=0 wss://%s:%d",
			PortBP, PortBP, m.Build, WSS_BP))

	_ = exec.Command("systemctl", "daemon-reload").Run()
	for _, u := range []string{"nd-wss-pb-server", "nd-wss-bp-client", "wg-quick@nd-svc-pb", "wg-quick@nd-svc-bp"} {
		_ = exec.Command("systemctl", "enable", u).Run()
		_ = exec.Command("systemctl", "restart", u).Run()
	}
	_ = exec.Command("bash", "-c", fmt.Sprintf(
		`command -v ufw >/dev/null && ufw allow %d/tcp comment nd-wss-pb || true`, WSS_PB)).Run()
	if m.Build != "" {
		_ = exec.Command("bash", "-c", fmt.Sprintf(
			`command -v ufw >/dev/null && ufw allow from %s to any port 52222 proto tcp comment nd-build-ssh || true`, m.Build)).Run()
	}
	return nil
}

// BootstrapBuildHost applies PB/BP on build VPS.
func BootstrapBuildHost(m *BuildMaterial) error {
	if m == nil {
		return fmt.Errorf("nil build material")
	}
	if err := ensureWireguardTools(); err != nil {
		return err
	}
	if err := ensureWstunnel(); err != nil {
		return err
	}
	_ = os.MkdirAll(KeysDir, 0o700)
	b, _ := json.MarshalIndent(m, "", "  ")
	_ = os.WriteFile(filepath.Join(KeysDir, "build-material.json"), b, 0o600)

	pbConf := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s/30
Table = off

[Peer]
PublicKey = %s
AllowedIPs = %s/32
Endpoint = 127.0.0.1:%d
PersistentKeepalive = 25
`, m.PBPrivB, AddrB_PB, m.PBPubP, AddrP_PB, PortPB)

	bpConf := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s/30
ListenPort = %d
Table = off

[Peer]
PublicKey = %s
AllowedIPs = %s/32
`, m.BPPrivB, AddrB_BP, PortBP, m.BPPubP, AddrP_BP)

	_ = os.MkdirAll("/etc/wireguard", 0o700)
	_ = writeFile("/etc/wireguard/nd-svc-pb.conf", pbConf, 0o600)
	_ = writeFile("/etc/wireguard/nd-svc-bp.conf", bpConf, 0o600)

	_ = unitFile("nd-wss-pb-client",
		fmt.Sprintf("/usr/local/bin/wstunnel client -L udp://127.0.0.1:%d:127.0.0.1:%d?timeout_sec=0 wss://%s:%d",
			PortPB, PortPB, m.Primary, WSS_PB))
	_ = unitFile("nd-wss-bp-server",
		fmt.Sprintf("/usr/local/bin/wstunnel server wss://0.0.0.0:%d --restrict-to 127.0.0.1:%d", WSS_BP, PortBP))

	_ = exec.Command("systemctl", "daemon-reload").Run()
	for _, u := range []string{"nd-wss-pb-client", "nd-wss-bp-server", "wg-quick@nd-svc-pb", "wg-quick@nd-svc-bp"} {
		_ = exec.Command("systemctl", "enable", u).Run()
		_ = exec.Command("systemctl", "restart", u).Run()
	}
	_ = exec.Command("bash", "-c", fmt.Sprintf(
		`command -v ufw >/dev/null && ufw allow %d/tcp comment nd-wss-bp || true`, WSS_BP)).Run()
	return nil
}

// BuildPathsStatus short line for doctor.
func BuildPathsStatus() string {
	up := []string{}
	for _, iface := range []string{IfacePB, IfaceBP} {
		if out, err := exec.Command("ip", "-br", "link", "show", iface).Output(); err == nil && strings.Contains(string(out), "UP") {
			up = append(up, iface)
		}
	}
	if len(up) == 0 {
		return "build-paths: none"
	}
	return "build-paths: " + strings.Join(up, ",")
}

// ApplyBuildAgentURL sets tunnel-only core URL for ROLE=build agent (PB preferred, BP fallback).
func ApplyBuildAgentURL(preferPB bool) error {
	path := "/etc/netductor/secrets/build_core_url"
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	want := "https://" + AddrP_PB + ":8789"
	if !preferPB {
		want = "https://" + AddrP_BP + ":8789"
	}
	return os.WriteFile(path, []byte(want+"\n"), 0o600)
}

// ProbeBuildPaths returns pbUp, bpUp via ping to peer VIP from this host.
func ProbeBuildPaths() (pbUp, bpUp bool) {
	pbUp = exec.Command("ping", "-c", "1", "-W", "2", AddrP_PB).Run() == nil ||
		exec.Command("ping", "-c", "1", "-W", "2", AddrB_PB).Run() == nil
	bpUp = exec.Command("ping", "-c", "1", "-W", "2", AddrP_BP).Run() == nil ||
		exec.Command("ping", "-c", "1", "-W", "2", AddrB_BP).Run() == nil
	return
}

// BuildPathsExpected is true when PB/BP were bootstrapped (material on disk).
// Without material there is no build VPS path to monitor — do not alert.
func BuildPathsExpected() bool {
	_, err := os.Stat(filepath.Join(KeysDir, "build-material.json"))
	return err == nil
}

// TickBuildPathsFailover on primary or build: update agent URL file + return summary.
// Empty string when paths were never configured (no build-host deploy).
func TickBuildPathsFailover() string {
	if !BuildPathsExpected() {
		return ""
	}
	pb, bp := ProbeBuildPaths()
	preferPB := pb
	if !pb && bp {
		preferPB = false
	}
	_ = ApplyBuildAgentURL(preferPB)
	return fmt.Sprintf("pb_up=%v bp_up=%v prefer_pb=%v", pb, bp, preferPB)
}
