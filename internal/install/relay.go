package install

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/nodes"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

// InstallRelay joins a RU VPS to core using bundle.json from core export.
func InstallRelay(bundlePath string) error {
	if bundlePath == "" {
		bundlePath = "bundle.json"
	}
	raw, err := os.ReadFile(bundlePath)
	if err != nil {
		return fmt.Errorf("read bundle: %w", err)
	}
	var b vpn.RelayBundle
	if err := json.Unmarshal(raw, &b); err != nil {
		return fmt.Errorf("parse bundle: %w", err)
	}
	if b.CoreIP == "" || b.UplinkUUID == "" {
		return fmt.Errorf("invalid bundle")
	}

	fmt.Fprintln(os.Stderr, "==> secondary: dirs")
	_ = paths.EnsureLayout()
	_ = os.MkdirAll("/etc/sing-box/certs", 0o755)
	_ = os.MkdirAll("/usr/local/etc/sing-box", 0o755)

	fmt.Fprintln(os.Stderr, "==> secondary: sing-box binary")
	if err := InstallSingBoxBinaryOnly(); err != nil {
		return err
	}

	// Reality keypair for inbound on this RU host
	fmt.Fprintln(os.Stderr, "==> secondary: reality keys")
	priv := readSecret("singbox_reality_private")
	pub := readSecret("singbox_reality_public")
	sid := readSecret("singbox_short_id")
	bin := "/usr/local/bin/sing-box"
	if priv == "" {
		out, err := exec.Command(bin, "generate", "reality-keypair").Output()
		if err != nil {
			return err
		}
		for _, line := range strings.Split(string(out), "\n") {
			f := strings.Fields(line)
			if len(f) >= 2 && strings.Contains(f[0], "Private") {
				priv = f[len(f)-1]
			}
			if len(f) >= 2 && strings.Contains(f[0], "Public") {
				pub = f[len(f)-1]
			}
		}
		_ = writeSecret("singbox_reality_private", priv)
		_ = writeSecret("singbox_reality_public", pub)
	}
	if sid == "" {
		sid = randomHex(8)
		_ = writeSecret("singbox_short_id", sid)
	}
	_ = writeSecret("singbox_reality_sni", b.RelaySNI)

	// store bundle for re-apply
	_ = os.MkdirAll(filepath.Join(paths.StateDir(), "relay"), 0o700)
	_ = os.WriteFile(filepath.Join(paths.StateDir(), "relay", "bundle.json"), raw, 0o600)

	if err := vpn.WriteRelaySingBox(&b, priv, sid); err != nil {
		return err
	}

	unit := `[Unit]
Description=sing-box (Netductor RU secondary)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/sing-box run -c /usr/local/etc/sing-box/config.json
Restart=on-failure
RestartSec=3
LimitNOFILE=1048576
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
`
	if err := writeUnit("sing-box.service", unit); err != nil {
		return err
	}
	if err := enableStart("sing-box"); err != nil {
		return err
	}

	applyHostname("nd-secondary")
	ip := b.CoreIP
	_ = ip
	pubIP := env("PUBLIC_IP", "")
	if pubIP == "" {
		out, _ := exec.Command("curl", "-4", "-fsS", "--max-time", "5", "https://ifconfig.me").Output()
		pubIP = strings.TrimSpace(string(out))
	}
	_ = nodes.SelfRegisterLocal("", "secondary", pubIP)

	// write mobile client links for operator convenience
	outDir := filepath.Join(paths.StateDir(), "relay", "clients")
	_ = os.MkdirAll(outDir, 0o700)
	for _, u := range b.Users {
		link := vpn.ClientLinkForRelay(u.Name, u.UUID, pubIP, pub, sid, b.RelaySNI)
		_ = os.WriteFile(filepath.Join(outDir, u.Name+".txt"), []byte(link+"\n"), 0o600)
	}
	fmt.Fprintf(os.Stderr, "secondary ready · public_ip=%s · SNI=%s · client links in %s\n", pubIP, b.RelaySNI, outDir)
	// install agent
	if b.AgentToken != "" && b.CoreAgentURL != "" {
		_ = os.WriteFile(filepath.Join(paths.EtcDir(), "secrets", "relay_agent_token"), append([]byte(b.AgentToken), 10), 0o600)
		_ = os.WriteFile(filepath.Join(paths.EtcDir(), "secrets", "relay_core_url"), append([]byte(b.CoreAgentURL), 10), 0o600)
		agentUnit := `[Unit]
Description=Netductor secondary agent
After=network-online.target sing-box.service
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/netductor secondary agent
Restart=always
RestartSec=15

[Install]
WantedBy=multi-user.target
`
		_ = writeUnit("netductor-secondary-agent.service", agentUnit)
		_ = writeUnit("netductor-relay-agent.service", agentUnit)
		_ = enableStart("netductor-secondary-agent")
		_ = enableStart("netductor-relay-agent")
		fmt.Fprintln(os.Stderr, "secondary agent started →", b.CoreAgentURL)
	}
	fmt.Fprintln(os.Stderr, "Give mobile users *-relay links; home users keep core links.")
	return nil
}

// InstallSingBoxBinaryOnly ensures sing-box exists (may run full InstallSingBox once).
func InstallSingBoxBinaryOnly() error {
	if st, err := os.Stat("/usr/local/bin/sing-box"); err == nil && !st.IsDir() {
		return nil
	}
	return InstallSingBox()
}

