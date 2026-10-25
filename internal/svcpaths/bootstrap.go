package svcpaths

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	AddrP_SP = "10.87.10.1"
	AddrS_SP = "10.87.10.2"
	AddrP_PS = "10.87.11.1"
	AddrS_PS = "10.87.11.2"
	PortSP   = 51830
	PortPS   = 51832
	WSS_SP   = 8444
	WSS_PS   = 8445
)

// Material holds WG keypairs for SP/PS. Primary keeps the full set on disk.
// Secondary must only receive MaterialForSecondary (no primary private keys).
type Material struct {
	SPPrivP   string `json:"sp_priv_p"`
	SPPubP    string `json:"sp_pub_p"`
	SPPrivS   string `json:"sp_priv_s"`
	SPPubS    string `json:"sp_pub_s"`
	PSPrivP   string `json:"ps_priv_p"`
	PSPubP    string `json:"ps_pub_p"`
	PSPrivS   string `json:"ps_priv_s"`
	PSPubS    string `json:"ps_pub_s"`
	Primary   string `json:"primary_ip"`
	Secondary string `json:"secondary_ip"`
	PeerID    string `json:"peer_id,omitempty"`
	Slot      int    `json:"slot,omitempty"`
	AddrP_SP  string `json:"addr_p_sp,omitempty"`
	AddrS_SP  string `json:"addr_s_sp,omitempty"`
	AddrP_PS  string `json:"addr_p_ps,omitempty"`
	AddrS_PS  string `json:"addr_s_ps,omitempty"`
	PortSP    int    `json:"port_sp,omitempty"`
	PortPS    int    `json:"port_ps,omitempty"`
	WSS_SP    int    `json:"wss_sp,omitempty"`
	WSS_PS    int    `json:"wss_ps,omitempty"`
	IfaceSP   string `json:"iface_sp,omitempty"`
	IfacePS   string `json:"iface_ps,omitempty"`
}

// ForSecondary returns a copy safe to ship to the secondary host.
// Omits primary private keys (SPPrivP, PSPrivP). Secondary only needs its own
// privates and the primary peer public keys.
func (m *Material) ForSecondary() *Material {
	if m == nil {
		return nil
	}
	return &Material{
		SPPubP: m.SPPubP, SPPrivS: m.SPPrivS, SPPubS: m.SPPubS,
		PSPubP: m.PSPubP, PSPrivS: m.PSPrivS, PSPubS: m.PSPubS,
		Primary: m.Primary, Secondary: m.Secondary,
		PeerID: m.PeerID, Slot: m.Slot,
		AddrP_SP: m.AddrP_SP, AddrS_SP: m.AddrS_SP, AddrP_PS: m.AddrP_PS, AddrS_PS: m.AddrS_PS,
		PortSP: m.PortSP, PortPS: m.PortPS, WSS_SP: m.WSS_SP, WSS_PS: m.WSS_PS,
		IfaceSP: m.IfaceSP, IfacePS: m.IfacePS,
	}
}

func wgGen() (priv, pub string, err error) {
	out, err := exec.Command("wg", "genkey").Output()
	if err != nil {
		return "", "", fmt.Errorf("wg genkey: %w (install wireguard-tools)", err)
	}
	priv = strings.TrimSpace(string(out))
	// stdin, not argv: private key must not appear in /proc/<pid>/cmdline.
	cmd := exec.Command("wg", "pubkey")
	cmd.Stdin = strings.NewReader(priv + "\n")
	pubOut, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("wg pubkey: %w", err)
	}
	return priv, strings.TrimSpace(string(pubOut)), nil
}

func ensureWireguardTools() error {
	if _, err := exec.LookPath("wg"); err == nil {
		if _, err2 := exec.LookPath("wg-quick"); err2 == nil {
			return nil
		}
	}
	_ = exec.Command("bash", "-c", "apt-get update -qq && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq wireguard wireguard-tools").Run()
	if _, err := exec.LookPath("wg"); err != nil {
		return fmt.Errorf("wireguard-tools missing")
	}
	return nil
}

func ensureWstunnel() error {
	if _, err := exec.LookPath("wstunnel"); err == nil {
		return nil
	}
	// Pinned v10.1.9 + official checksums.txt SHA256 (linux amd64/arm64 only).
	script := `set -e
arch=$(uname -m)
case "$arch" in
  x86_64) a=amd64; want=d4c6178e7e7756a49386ba6f419345da4b8b53996390100cfafb2323fc853fbb;;
  aarch64) a=arm64; want=77e256ccef8ab02d21752207fbdb8f27587cf53eafa98f7e878db8979a8a130f;;
  *) echo "unsupported arch $arch" >&2; exit 1;;
esac
u="https://github.com/erebe/wstunnel/releases/download/v10.1.9/wstunnel_10.1.9_linux_${a}.tar.gz"
curl -fsSL "$u" -o /tmp/wst.tgz
got=$(sha256sum /tmp/wst.tgz | awk '{print $1}')
test "$got" = "$want" || { echo "wstunnel checksum mismatch: got $got want $want" >&2; exit 1; }
rm -rf /tmp/wst-extract
mkdir -p /tmp/wst-extract
tar -xzf /tmp/wst.tgz -C /tmp/wst-extract
bin=$(find /tmp/wst-extract -type f -name 'wstunnel' | head -1)
test -n "$bin"
install -m 755 "$bin" /usr/local/bin/wstunnel
rm -rf /tmp/wst.tgz /tmp/wst-extract
`
	if out, err := exec.Command("bash", "-c", script).CombinedOutput(); err != nil {
		return fmt.Errorf("wstunnel install: %v %s", err, strings.TrimSpace(string(out)))
	}
	if _, err := exec.LookPath("wstunnel"); err != nil {
		return fmt.Errorf("wstunnel not on PATH after install")
	}
	return nil
}

// GenerateMaterial creates keypairs (call once on primary or operator).
func GenerateMaterial(primaryIP, secondaryIP string) (*Material, error) {
	if err := ensureWireguardTools(); err != nil {
		return nil, err
	}
	m := &Material{Primary: primaryIP, Secondary: secondaryIP}
	var err error
	if m.SPPrivP, m.SPPubP, err = wgGen(); err != nil {
		return nil, err
	}
	if m.SPPrivS, m.SPPubS, err = wgGen(); err != nil {
		return nil, err
	}
	if m.PSPrivP, m.PSPubP, err = wgGen(); err != nil {
		return nil, err
	}
	if m.PSPrivS, m.PSPubS, err = wgGen(); err != nil {
		return nil, err
	}
	return m, nil
}

// LoadMaterialStrict reads legacy KeysDir/material.json.
// os.ErrNotExist if absent; any other error = corrupt/incomplete — never treat as missing.
func LoadMaterialStrict() (*Material, error) {
	path := filepath.Join(keysRoot, "material.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Material
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("material.json corrupt (refuse regen): %w", err)
	}
	if m.SPPrivP == "" || m.SPPrivS == "" || m.PSPrivP == "" || m.PSPrivS == "" {
		return nil, fmt.Errorf("material.json incomplete (refuse regen)")
	}
	return &m, nil
}

// LoadMaterial aliases LoadMaterialStrict.
func LoadMaterial() (*Material, error) { return LoadMaterialStrict() }

// LoadOrGenerateMaterial generates only when file is truly absent.
func LoadOrGenerateMaterial(primaryIP, secondaryIP string, force bool) (*Material, bool, error) {
	existing, err := LoadMaterialStrict()
	if err == nil && existing != nil {
		if existing.Secondary == secondaryIP || secondaryIP == "" {
			if primaryIP != "" && primaryIP != "0.0.0.0" {
				existing.Primary = primaryIP
			}
			if secondaryIP != "" {
				existing.Secondary = secondaryIP
			}
			return existing, false, nil
		}
		if !force {
			return nil, false, fmt.Errorf(
				"legacy material peers %s; refusing %s without --force-new-material (or: svc-paths peer add)",
				existing.Secondary, secondaryIP)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return nil, false, err
	}
	m, err := GenerateMaterial(primaryIP, secondaryIP)
	if err != nil {
		return nil, false, err
	}
	return m, true, nil
}

func (m *Material) resolvedAddrs() (pSP, sSP, pPS, sPS string, portSP, portPS, wssSP, wssPS int, ifaceSP, ifacePS string) {
	pSP, sSP, pPS, sPS = AddrP_SP, AddrS_SP, AddrP_PS, AddrS_PS
	portSP, portPS, wssSP, wssPS = PortSP, PortPS, WSS_SP, WSS_PS
	ifaceSP, ifacePS = "nd-svc-sp", "nd-svc-ps"
	if m != nil && m.AddrP_SP != "" {
		pSP, sSP, pPS, sPS = m.AddrP_SP, m.AddrS_SP, m.AddrP_PS, m.AddrS_PS
	}
	if m != nil && m.PortSP != 0 {
		portSP = m.PortSP
	}
	if m != nil && m.PortPS != 0 {
		portPS = m.PortPS
	}
	if m != nil && m.WSS_SP != 0 {
		wssSP = m.WSS_SP
	}
	if m != nil && m.WSS_PS != 0 {
		wssPS = m.WSS_PS
	}
	if m != nil && m.IfaceSP != "" {
		ifaceSP = m.IfaceSP
	}
	if m != nil && m.IfacePS != "" {
		ifacePS = m.IfacePS
	}
	return
}

func writeFile(path, body string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), mode)
}

func unitFile(name, execStart string) error {
	body := fmt.Sprintf(`[Unit]
Description=Netductor %s
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
`, name, execStart)
	return writeFile("/etc/systemd/system/"+name+".service", body, 0o644)
}

func enableRestart(units ...string) error {
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("daemon-reload: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	for _, u := range units {
		if out, err := exec.Command("systemctl", "enable", u).CombinedOutput(); err != nil {
			return fmt.Errorf("enable %s: %v (%s)", u, err, strings.TrimSpace(string(out)))
		}
		if out, err := exec.Command("systemctl", "restart", u).CombinedOutput(); err != nil {
			return fmt.Errorf("restart %s: %v (%s)", u, err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

// BootstrapPrimaryOpts controls side effects when applying a peer on primary.
type BootstrapPrimaryOpts struct {
	// WriteLegacyMaterial writes KeysDir/material.json (only for first/legacy peer).
	WriteLegacyMaterial bool
	// ResetPolicy writes DefaultPolicy (never on multi-peer add).
	ResetPolicy bool
	// InstallHealthTimer installs health timer once.
	InstallHealthTimer bool
}

// BootstrapPrimary writes confs+units on primary (legacy single-peer: writes material.json + default policy).
func BootstrapPrimary(m *Material) error {
	return BootstrapPrimaryWith(m, BootstrapPrimaryOpts{
		WriteLegacyMaterial: true,
		ResetPolicy:         true,
		InstallHealthTimer:  true,
	})
}

// BootstrapPrimaryWith applies one peer's WG/WSS on primary with explicit side effects.
func BootstrapPrimaryWith(m *Material, o BootstrapPrimaryOpts) error {
	if m == nil {
		return fmt.Errorf("nil material")
	}
	if m.Primary == "" || m.Primary == "0.0.0.0" {
		if b, err := os.ReadFile("/etc/netductor/secrets/public_ip"); err == nil {
			m.Primary = strings.TrimSpace(string(b))
		}
	}
	if err := ensureWireguardTools(); err != nil {
		return err
	}
	if err := ensureWstunnel(); err != nil {
		return err
	}
	if err := os.MkdirAll(KeysDir, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if o.WriteLegacyMaterial {
		if err := os.WriteFile(filepath.Join(KeysDir, "material.json"), b, 0o600); err != nil {
			return fmt.Errorf("material.json: %w", err)
		}
	}

	pSP, sSP, pPS, sPS, portSP, portPS, wssSP, wssPS, ifaceSP, ifacePS := m.resolvedAddrs()
	spConf := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s/30
ListenPort = %d
Table = off

[Peer]
PublicKey = %s
AllowedIPs = %s/32
`, m.SPPrivP, pSP, portSP, m.SPPubS, sSP)

	psConf := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s/30
Table = off

[Peer]
PublicKey = %s
AllowedIPs = %s/32
Endpoint = 127.0.0.1:%d
PersistentKeepalive = 25
`, m.PSPrivP, pPS, m.PSPubS, sPS, portPS)

	if err := os.MkdirAll("/etc/wireguard", 0o700); err != nil {
		return err
	}
	if err := writeFile("/etc/wireguard/"+ifaceSP+".conf", spConf, 0o600); err != nil {
		return fmt.Errorf("%s.conf: %w", ifaceSP, err)
	}
	if err := writeFile("/etc/wireguard/"+ifacePS+".conf", psConf, 0o600); err != nil {
		return fmt.Errorf("%s.conf: %w", ifacePS, err)
	}
	// Per-peer SP WSS server (unique port per slot) — adding B must not rewrite A's unit.
	spServerUnit := "nd-wss-sp-server"
	if ifaceSP != "nd-svc-sp" {
		spServerUnit = "nd-wss-sp-server-" + ifaceSP
	}
	if err := unitFile(spServerUnit,
		fmt.Sprintf("/usr/local/bin/wstunnel server wss://0.0.0.0:%d --restrict-to 127.0.0.1:%d", wssSP, portSP)); err != nil {
		return err
	}
	psClientUnit := "nd-wss-ps-client"
	if ifacePS != "nd-svc-ps" {
		psClientUnit = "nd-wss-ps-client-" + ifacePS
	}
	if err := unitFile(psClientUnit,
		fmt.Sprintf("/usr/local/bin/wstunnel client -L udp://127.0.0.1:%d:127.0.0.1:%d?timeout_sec=0 wss://%s:%d",
			portPS, portPS, m.Secondary, wssPS)); err != nil {
		return err
	}
	if err := enableRestart(spServerUnit, psClientUnit, "wg-quick@"+ifaceSP, "wg-quick@"+ifacePS); err != nil {
		return err
	}

	// UFW: WSS_SP only from secondary public IP; mTLS 8789 from secondary SP WG IP.
	// When ufw is active, fail if open WSS remains or allow rules cannot be applied.
	if err := applyPrimaryWSSFirewall(m.Secondary, sSP, wssSP); err != nil {
		return err
	}

	if o.InstallHealthTimer {
		if err := installHealthTimer(); err != nil {
			return err
		}
	}
	if o.ResetPolicy {
		if err := SavePolicy(DefaultPolicy()); err != nil {
			return err
		}
	}
	return nil
}

// BootstrapSecondary applies material on secondary.
// Always persists ForSecondary() so primary privates never land on disk here.
func BootstrapSecondary(m *Material) error {
	if m == nil {
		return fmt.Errorf("nil material")
	}
	if m.SPPrivS == "" || m.PSPrivS == "" || m.SPPubP == "" || m.PSPubP == "" {
		return fmt.Errorf("material incomplete for secondary (need SPPrivS, PSPrivS, SPPubP, PSPubP)")
	}
	if err := ensureWireguardTools(); err != nil {
		return err
	}
	if err := ensureWstunnel(); err != nil {
		return err
	}
	safe := m.ForSecondary()
	if err := os.MkdirAll(KeysDir, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(safe, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(KeysDir, "material.json"), b, 0o600); err != nil {
		return fmt.Errorf("material.json: %w", err)
	}

	pSP, sSP, pPS, sPS, portSP, portPS, wssSP, wssPS, ifaceSP, ifacePS := m.resolvedAddrs()
	spConf := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s/30
Table = off

[Peer]
PublicKey = %s
AllowedIPs = %s/32
Endpoint = 127.0.0.1:%d
PersistentKeepalive = 25
`, m.SPPrivS, sSP, m.SPPubP, pSP, portSP)

	psConf := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s/30
ListenPort = %d
Table = off

[Peer]
PublicKey = %s
AllowedIPs = %s/32
`, m.PSPrivS, sPS, portPS, m.PSPubP, pPS)

	if err := os.MkdirAll("/etc/wireguard", 0o700); err != nil {
		return err
	}
	if err := writeFile("/etc/wireguard/"+ifaceSP+".conf", spConf, 0o600); err != nil {
		return fmt.Errorf("%s.conf: %w", ifaceSP, err)
	}
	if err := writeFile("/etc/wireguard/"+ifacePS+".conf", psConf, 0o600); err != nil {
		return fmt.Errorf("%s.conf: %w", ifacePS, err)
	}
	if err := unitFile("nd-wss-sp-client",
		fmt.Sprintf("/usr/local/bin/wstunnel client -L udp://127.0.0.1:%d:127.0.0.1:%d?timeout_sec=0 wss://%s:%d",
			portSP, portSP, m.Primary, wssSP)); err != nil {
		return err
	}
	if err := unitFile("nd-wss-ps-server",
		fmt.Sprintf("/usr/local/bin/wstunnel server wss://0.0.0.0:%d --restrict-to 127.0.0.1:%d", wssPS, portPS)); err != nil {
		return err
	}
	if err := enableRestart("nd-wss-sp-client", "nd-wss-ps-server", "wg-quick@"+ifaceSP, "wg-quick@"+ifacePS); err != nil {
		return err
	}
	if err := applySecondaryWSSFirewall(m.Primary, wssPS); err != nil {
		return err
	}

	if err := os.MkdirAll("/etc/netductor/secrets", 0o700); err != nil {
		return err
	}
	pub := fmt.Sprintf("https://%s:8789", m.Primary)
	if err := os.WriteFile("/etc/netductor/secrets/secondary_core_url.public", []byte(pub+"\n"), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile("/etc/netductor/secrets/secondary_core_url", []byte("https://"+pSP+":8789\n"), 0o600); err != nil {
		return err
	}
	// Agent must pick up secondary_core_url (SP). Restart failures are fatal unless unit is absent (pre-join).
	if out, err := exec.Command("systemctl", "try-restart", "netductor-secondary-agent").CombinedOutput(); err != nil {
		st := string(out)
		if !strings.Contains(st, "not found") && !strings.Contains(st, "not-found") && !strings.Contains(st, "could not be found") {
			return fmt.Errorf("try-restart netductor-secondary-agent: %w: %s", err, strings.TrimSpace(st))
		}
	}

	if err := installHealthTimer(); err != nil {
		return err
	}
	if err := SavePolicy(DefaultPolicy()); err != nil {
		return err
	}
	return nil
}

func installHealthTimer() error {
	script := `/usr/local/bin/nd-svc-paths-health`
	// Always rewrite so bootstrap upgrades the tick policy (do not keep silent-success legacy).
	body := `#!/bin/bash
set -u
# Prefer agent as sole owner of failover samples; timer is fallback if agent is down.
if systemctl is-active --quiet netductor-secondary-agent 2>/dev/null; then
  exit 0
fi
if command -v netductor >/dev/null 2>&1; then
  if ! out=$(netductor svc-paths failover tick 2>&1); then
    echo "nd-svc-paths-health: failover tick failed: $out" >&2
    logger -t nd-svc-paths-health "failover tick failed: $out" 2>/dev/null || true
    exit 1
  fi
  exit 0
fi
OUT=/var/lib/netductor/svc-paths-health.json
mkdir -p /var/lib/netductor
sp=0; ps=0
ping -c1 -W2 10.87.10.1 >/dev/null 2>&1 || ping -c1 -W2 10.87.10.2 >/dev/null 2>&1 && sp=1
ping -c1 -W2 10.87.11.1 >/dev/null 2>&1 || ping -c1 -W2 10.87.11.2 >/dev/null 2>&1 && ps=1
printf '{"ts":"%s","svc_sp_up":%s,"svc_ps_up":%s}\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$sp" "$ps" > "$OUT"
`
	if err := writeFile(script, body, 0o755); err != nil {
		return err
	}
	if err := writeFile("/etc/systemd/system/nd-svc-paths-health.service", `[Unit]
Description=Netductor svc-paths health
[Service]
Type=oneshot
ExecStart=/usr/local/bin/nd-svc-paths-health
`, 0o644); err != nil {
		return err
	}
	if err := writeFile("/etc/systemd/system/nd-svc-paths-health.timer", `[Unit]
Description=Netductor svc-paths health timer
[Timer]
OnBootSec=30
OnUnitActiveSec=30
AccuracySec=5
[Install]
WantedBy=timers.target
`, 0o644); err != nil {
		return err
	}
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("daemon-reload (health timer): %w: %s", err, strings.TrimSpace(string(out)))
	}
	if out, err := exec.Command("systemctl", "enable", "--now", "nd-svc-paths-health.timer").CombinedOutput(); err != nil {
		return fmt.Errorf("enable nd-svc-paths-health.timer: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if st, err := exec.Command("systemctl", "is-active", "nd-svc-paths-health.timer").CombinedOutput(); err != nil || strings.TrimSpace(string(st)) != "active" {
		return fmt.Errorf("nd-svc-paths-health.timer not active: %s", strings.TrimSpace(string(st)))
	}
	return nil
}

// ufwOpenPortAnywherePattern matches a wide-open UFW status line for port (IPv4 and IPv6).
// Examples:
//
//	8444/tcp                   ALLOW IN    Anywhere
//	8444/tcp (v6)              ALLOW IN    Anywhere (v6)
//	8444 ALLOW Anywhere
func ufwOpenPortAnywherePattern(port int) string {
	return fmt.Sprintf(`(^|[[:space:]])%d(/tcp)?([[:space:]]+\(v6\))?[[:space:]]+ALLOW([[:space:]]+IN)?[[:space:]]+Anywhere`, port)
}

// applyPrimaryWSSFirewall restricts WSS_SP to secondary public IP when ufw is active.
func applyPrimaryWSSFirewall(secondaryPub, sSP string, wssSP int) error {
	if secondaryPub == "" {
		return nil
	}
	openRE := ufwOpenPortAnywherePattern(wssSP)
	script := fmt.Sprintf(
		"set -e; "+
			"command -v ufw >/dev/null || { echo ufw-not-installed; exit 0; }; "+
			"if ! ufw status 2>/dev/null | grep -qi 'Status: active'; then echo ufw-inactive; exit 0; fi; "+
			"ufw --force delete allow %d/tcp 2>/dev/null || true; "+
			"ufw allow from %s to any port %d proto tcp comment nd-wss-sp; "+
			"ufw allow from %s to any port 8789 proto tcp comment nd-svc-sp; "+
			"if ufw status | grep -E '%s' >/dev/null 2>&1; then echo open-wss-still-present; exit 1; fi; "+
			"echo primary-ufw-wss-ok",
		wssSP, secondaryPub, wssSP, sSP, openRE)
	out, err := exec.Command("bash", "-c", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("primary ufw WSS: %w: %s", err, out)
	}
	return nil
}

func applySecondaryWSSFirewall(primaryPub string, wssPS int) error {
	if primaryPub == "" {
		return nil
	}
	openRE := ufwOpenPortAnywherePattern(wssPS)
	script := fmt.Sprintf(
		"set -e; "+
			"command -v ufw >/dev/null || { echo ufw-not-installed; exit 0; }; "+
			"if ! ufw status 2>/dev/null | grep -qi 'Status: active'; then echo ufw-inactive; exit 0; fi; "+
			"ufw --force delete allow %d/tcp 2>/dev/null || true; "+
			"ufw allow from %s to any port %d proto tcp comment nd-wss-ps; "+
			"if ufw status | grep -E '%s' >/dev/null 2>&1; then echo open-wss-ps-still-present; exit 1; fi; "+
			"echo secondary-ufw-wss-ok",
		wssPS, primaryPub, wssPS, openRE)
	out, err := exec.Command("bash", "-c", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("secondary ufw WSS: %w: %s", err, out)
	}
	return nil
}

// ufwStatusHasOpenPort reports whether ufw status text still allows PORT from Anywhere
// (IPv4 and/or IPv6 status lines).
func ufwStatusHasOpenPort(status string, port int) bool {
	ok, _ := regexp.MatchString(ufwOpenPortAnywherePattern(port), status)
	return ok
}
