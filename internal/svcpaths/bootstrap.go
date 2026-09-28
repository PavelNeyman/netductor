package svcpaths

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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

// Material is shared key material for both ends (no need to keep peer private).
type Material struct {
	SPPrivP string `json:"sp_priv_p"`
	SPPubP  string `json:"sp_pub_p"`
	SPPrivS string `json:"sp_priv_s"`
	SPPubS  string `json:"sp_pub_s"`
	PSPrivP string `json:"ps_priv_p"`
	PSPubP  string `json:"ps_pub_p"`
	PSPrivS string `json:"ps_priv_s"`
	PSPubS  string `json:"ps_pub_s"`
	Primary string `json:"primary_ip"`
	Secondary string `json:"secondary_ip"`
}

func wgGen() (priv, pub string, err error) {
	out, err := exec.Command("wg", "genkey").Output()
	if err != nil {
		return "", "", fmt.Errorf("wg genkey: %w (install wireguard-tools)", err)
	}
	priv = strings.TrimSpace(string(out))
	pubOut, err := exec.Command("bash", "-c", "echo "+shellQuote(priv)+" | wg pubkey").Output()
	if err != nil {
		return "", "", fmt.Errorf("wg pubkey: %w", err)
	}
	return priv, strings.TrimSpace(string(pubOut)), nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
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
	// static binary amd64
	script := `set -e
arch=$(uname -m)
case "$arch" in x86_64) a=amd64;; aarch64) a=arm64;; *) exit 1;; esac
url="https://github.com/erebe/wstunnel/releases/download/v10.1.9/wstunnel_${a}"
# try newer layout
for u in \
  "https://github.com/erebe/wstunnel/releases/download/v10.1.9/wstunnel_10.1.9_linux_${a}.tar.gz" \
  "https://github.com/erebe/wstunnel/releases/download/v9.2.3/wstunnel_9.2.3_linux_${a}.tar.gz"
do
  if curl -fsSL "$u" -o /tmp/wst.tgz 2>/dev/null; then
    tar -xzf /tmp/wst.tgz -C /tmp
    bin=$(find /tmp -maxdepth 2 -type f -name 'wstunnel' | head -1)
    if [ -n "$bin" ]; then
      install -m 755 "$bin" /usr/local/bin/wstunnel
      exit 0
    fi
  fi
done
# direct binary attempt
curl -fsSL "https://github.com/erebe/wstunnel/releases/download/v9.2.3/wstunnel_9.2.3_linux_${a}" -o /usr/local/bin/wstunnel && chmod +x /usr/local/bin/wstunnel
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

func writeFile(path, body string, mode os.FileMode) error {
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
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

// BootstrapPrimary writes confs+units on primary and saves material.
func BootstrapPrimary(m *Material) error {
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
	_ = os.MkdirAll(KeysDir, 0o700)
	b, _ := json.MarshalIndent(m, "", "  ")
	_ = os.WriteFile(filepath.Join(KeysDir, "material.json"), b, 0o600)

	// SP: listen local UDP, peer secondary via WSS client on secondary
	spConf := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s/30
ListenPort = %d
Table = off

[Peer]
PublicKey = %s
AllowedIPs = %s/32
`, m.SPPrivP, AddrP_SP, PortSP, m.SPPubS, AddrS_SP)

	psConf := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s/30
Table = off

[Peer]
PublicKey = %s
AllowedIPs = %s/32
Endpoint = 127.0.0.1:%d
PersistentKeepalive = 25
`, m.PSPrivP, AddrP_PS, m.PSPubS, AddrS_PS, PortPS)

	_ = os.MkdirAll("/etc/wireguard", 0o700)
	_ = writeFile("/etc/wireguard/nd-svc-sp.conf", spConf, 0o600)
	_ = writeFile("/etc/wireguard/nd-svc-ps.conf", psConf, 0o600)

	// WSS: SP server on primary :8444 → local UDP 51830
	_ = unitFile("nd-wss-sp-server",
		fmt.Sprintf("/usr/local/bin/wstunnel server wss://0.0.0.0:%d --restrict-to 127.0.0.1:%d", WSS_SP, PortSP))
	// PS client: tunnel local :51832 to secondary WSS :8445
	_ = unitFile("nd-wss-ps-client",
		fmt.Sprintf("/usr/local/bin/wstunnel client -L udp://127.0.0.1:%d:127.0.0.1:%d?timeout_sec=0 wss://%s:%d",
			PortPS, PortPS, m.Secondary, WSS_PS))

	_ = exec.Command("systemctl", "daemon-reload").Run()
	for _, u := range []string{"nd-wss-sp-server", "nd-wss-ps-client", "wg-quick@nd-svc-sp", "wg-quick@nd-svc-ps"} {
		_ = exec.Command("systemctl", "enable", u).Run()
		_ = exec.Command("systemctl", "restart", u).Run()
	}

	// ufw: WSS SP port + agent plane only from SP peer (not secondary public IP)
	_ = exec.Command("bash", "-c", fmt.Sprintf(
		`command -v ufw >/dev/null && ufw allow %d/tcp comment nd-wss-sp && ufw allow from %s to any port 8789 proto tcp comment nd-svc-sp || true`,
		WSS_SP, AddrS_SP)).Run()
	// Drop interim WAN allow for secondary public IP once SP is the path
	if m.Secondary != "" {
		_ = exec.Command("bash", "-c", fmt.Sprintf(
			`command -v ufw >/dev/null || exit 0
ufw status numbered 2>/dev/null | grep -E '8789.*%s' | head -5
ufw delete allow from %s to any port 8789 proto tcp 2>/dev/null || true
ufw status 2>/dev/null | grep 8789 || true
`, m.Secondary, m.Secondary)).Run()
	}

	// health timer
	_ = installHealthTimer()
	return nil
}

// BootstrapSecondary applies material on secondary.
func BootstrapSecondary(m *Material) error {
	if m == nil {
		return fmt.Errorf("nil material")
	}
	if err := ensureWireguardTools(); err != nil {
		return err
	}
	if err := ensureWstunnel(); err != nil {
		return err
	}
	_ = os.MkdirAll(KeysDir, 0o700)
	b, _ := json.MarshalIndent(m, "", "  ")
	_ = os.WriteFile(filepath.Join(KeysDir, "material.json"), b, 0o600)

	spConf := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s/30
Table = off

[Peer]
PublicKey = %s
AllowedIPs = %s/32
Endpoint = 127.0.0.1:%d
PersistentKeepalive = 25
`, m.SPPrivS, AddrS_SP, m.SPPubP, AddrP_SP, PortSP)

	psConf := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s/30
ListenPort = %d
Table = off

[Peer]
PublicKey = %s
AllowedIPs = %s/32
`, m.PSPrivS, AddrS_PS, PortPS, m.PSPubP, AddrP_PS)

	_ = os.MkdirAll("/etc/wireguard", 0o700)
	_ = writeFile("/etc/wireguard/nd-svc-sp.conf", spConf, 0o600)
	_ = writeFile("/etc/wireguard/nd-svc-ps.conf", psConf, 0o600)

	_ = unitFile("nd-wss-sp-client",
		fmt.Sprintf("/usr/local/bin/wstunnel client -L udp://127.0.0.1:%d:127.0.0.1:%d?timeout_sec=0 wss://%s:%d",
			PortSP, PortSP, m.Primary, WSS_SP))
	_ = unitFile("nd-wss-ps-server",
		fmt.Sprintf("/usr/local/bin/wstunnel server wss://0.0.0.0:%d --restrict-to 127.0.0.1:%d", WSS_PS, PortPS))

	_ = exec.Command("systemctl", "daemon-reload").Run()
	for _, u := range []string{"nd-wss-sp-client", "nd-wss-ps-server", "wg-quick@nd-svc-sp", "wg-quick@nd-svc-ps"} {
		_ = exec.Command("systemctl", "enable", u).Run()
		_ = exec.Command("systemctl", "restart", u).Run()
	}
	_ = exec.Command("bash", "-c", fmt.Sprintf(
		`command -v ufw >/dev/null && ufw allow %d/tcp comment nd-wss-ps || true`, WSS_PS)).Run()

	// agent prefer SP
	_ = os.MkdirAll("/etc/netductor/secrets", 0o700)
	pub := fmt.Sprintf("https://%s:8789", m.Primary)
	_ = os.WriteFile("/etc/netductor/secrets/secondary_core_url.public", []byte(pub+"\n"), 0o600)
	_ = os.WriteFile("/etc/netductor/secrets/secondary_core_url", []byte("https://"+AddrP_SP+":8789\n"), 0o600)
	_ = exec.Command("systemctl", "try-restart", "netductor-secondary-agent").Run()

	_ = installHealthTimer()
	return nil
}

func installHealthTimer() error {
	script := `/usr/local/bin/nd-svc-paths-health`
	// minimal if missing
	if _, err := os.Stat(script); err != nil {
		body := `#!/bin/bash
OUT=/var/lib/netductor/svc-paths-health.json
mkdir -p /var/lib/netductor
sp=0; ps=0
ping -c1 -W2 10.87.10.1 >/dev/null 2>&1 || ping -c1 -W2 10.87.10.2 >/dev/null 2>&1 && sp=1
ping -c1 -W2 10.87.11.1 >/dev/null 2>&1 || ping -c1 -W2 10.87.11.2 >/dev/null 2>&1 && ps=1
printf '{"ts":"%s","svc_sp_up":%s,"svc_ps_up":%s}\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$sp" "$ps" > "$OUT"
`
		_ = writeFile(script, body, 0o755)
	}
	_ = writeFile("/etc/systemd/system/nd-svc-paths-health.service", `[Unit]
Description=Netductor svc-paths health
[Service]
Type=oneshot
ExecStart=/usr/local/bin/nd-svc-paths-health
`, 0o644)
	_ = writeFile("/etc/systemd/system/nd-svc-paths-health.timer", `[Unit]
Description=Netductor svc-paths health timer
[Timer]
OnBootSec=30
OnUnitActiveSec=30
AccuracySec=5
[Install]
WantedBy=timers.target
`, 0o644)
	_ = exec.Command("systemctl", "daemon-reload").Run()
	_ = exec.Command("systemctl", "enable", "--now", "nd-svc-paths-health.timer").Run()
	return nil
}

// BootstrapAll is convenience for operator SSH on each host.
func SaveMaterial(m *Material) error {
	_ = os.MkdirAll(KeysDir, 0o700)
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(KeysDir, "material.json"), b, 0o600)
}

func LoadMaterial() (*Material, error) {
	b, err := os.ReadFile(filepath.Join(KeysDir, "material.json"))
	if err != nil {
		return nil, err
	}
	var m Material
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// WaitPing waits for peer.
func WaitPing(ip string, tries int) bool {
	for i := 0; i < tries; i++ {
		if exec.Command("ping", "-c", "1", "-W", "2", ip).Run() == nil {
			return true
		}
		time.Sleep(2 * time.Second)
	}
	return false
}
