package install

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/nodes"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/version"
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
	var b vpn.SecondaryBundle
	if err := json.Unmarshal(raw, &b); err != nil {
		return fmt.Errorf("parse bundle: %w", err)
	}
	if b.CoreIP == "" || b.UplinkUUID == "" {
		return fmt.Errorf("invalid bundle: missing core_ip or uplink_uuid")
	}
	if ip := net.ParseIP(strings.TrimSpace(b.CoreIP)); ip == nil || ip.To4() == nil {
		return fmt.Errorf("invalid bundle: core_ip must be a public IPv4 address")
	}
	if len(strings.TrimSpace(b.AgentToken)) < 16 || strings.TrimSpace(b.AgentID) == "" {
		return fmt.Errorf("invalid bundle: secondary agent identity/token is missing")
	}
	coreURL, err := url.Parse(strings.TrimSpace(b.CoreAgentURL))
	if err != nil || coreURL.Scheme != "https" || coreURL.Host == "" || coreURL.User != nil ||
		coreURL.Hostname() != strings.TrimSpace(b.CoreIP) || coreURL.Port() != "8789" ||
		(coreURL.Path != "" && coreURL.Path != "/") || coreURL.RawQuery != "" || coreURL.Fragment != "" {
		return fmt.Errorf("invalid bundle: core_agent_url must target core_ip on mTLS port 8789")
	}
	const supportedSchema = 1
	if b.Version != supportedSchema {
		return fmt.Errorf("bundle schema version %d not supported (need %d; missing/0 rejected)", b.Version, supportedSchema)
	}
	if b.MinAgentVersion != "" {
		if !version.ValidDotVersion(b.MinAgentVersion) {
			return fmt.Errorf("bundle min_agent_version %q invalid (need major.minor.patch)", b.MinAgentVersion)
		}
		if version.Less(version.Release, b.MinAgentVersion) {
			return fmt.Errorf("agent %s older than bundle min_agent_version %s — upgrade secondary binary first", version.Release, b.MinAgentVersion)
		}
	}

	PurgeHostMonitoring()
	if err := EnsureHostBaseline("secondary"); err != nil {
		return fmt.Errorf("secondary host baseline: %w", err)
	}
	fmt.Fprintln(os.Stderr, "==> secondary: dirs")
	if err := paths.EnsureLayout(); err != nil { return fmt.Errorf("secondary state layout: %w", err) }
	if err := applyHostname("secondary"); err != nil { return fmt.Errorf("set secondary hostname: %w", err) }
	if err := os.MkdirAll("/etc/sing-box/certs", 0o755); err != nil { return fmt.Errorf("create sing-box cert directory: %w", err) }
	if err := os.MkdirAll("/usr/local/etc/sing-box", 0o755); err != nil { return fmt.Errorf("create sing-box config directory: %w", err) }

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
	if priv == "" || pub == "" {
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
		if priv == "" || pub == "" { return fmt.Errorf("sing-box returned incomplete Reality keypair") }
		if err := writeSecret("singbox_reality_private", priv); err != nil { return fmt.Errorf("save Reality private key: %w", err) }
		if err := writeSecret("singbox_reality_public", pub); err != nil { return fmt.Errorf("save Reality public key: %w", err) }
	}
	if sid == "" {
		sid = randomHex(8)
		if err := writeSecret("singbox_short_id", sid); err != nil { return fmt.Errorf("save Reality short ID: %w", err) }
	}
	if err := writeSecret("singbox_reality_sni", b.SecondarySNI); err != nil { return fmt.Errorf("save Reality SNI: %w", err) }

	// store bundle for re-apply
	if err := os.MkdirAll(paths.SecondaryDir(), 0o700); err != nil { return fmt.Errorf("create secondary state directory: %w", err) }
	if err := writeInstallFileDurable(filepath.Join(paths.SecondaryDir(), "bundle.json"), raw, 0o600); err != nil { return fmt.Errorf("save secondary bundle: %w", err) }

	if err := vpn.WriteSecondarySingBox(&b, priv, sid); err != nil {
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
	if out, err := exec.Command("systemctl", "is-active", "sing-box").CombinedOutput(); err != nil || strings.TrimSpace(string(out)) != "active" {
		return fmt.Errorf("sing-box did not become active: %s (%v)", strings.TrimSpace(string(out)), err)
	}

	pubIP := strings.TrimSpace(env("PUBLIC_IP", ""))
	if pubIP == "" { pubIP = strings.TrimSpace(detectPublicIPv4()) }
	parsedIP := net.ParseIP(pubIP)
	if parsedIP == nil || parsedIP.To4() == nil { return fmt.Errorf("secondary public IPv4 could not be determined (got %q)", pubIP) }
	if err := nodes.SelfRegisterLocal("", "secondary", pubIP); err != nil { return fmt.Errorf("register secondary node: %w", err) }

	// write mobile client links for operator convenience
	outDir := filepath.Join(paths.SecondaryDir(), "clients")
	if err := os.MkdirAll(outDir, 0o700); err != nil { return fmt.Errorf("create secondary client links directory: %w", err) }
	for _, u := range b.Users {
		if u.Name == "" || u.Name == "." || u.Name == ".." || filepath.Base(u.Name) != u.Name || strings.ContainsAny(u.Name, "/\\\\") {
			return fmt.Errorf("invalid secondary client link name %q", u.Name)
		}
		link := vpn.ClientLinkForSecondary(u.Name, u.UUID, pubIP, pub, sid, b.SecondarySNI)
		if err := writeInstallFileDurable(filepath.Join(outDir, u.Name+".txt"), []byte(link+"\n"), 0o600); err != nil {
			return fmt.Errorf("write secondary client link %q: %w", u.Name, err)
		}
	}
	// install agent
	if b.AgentToken != "" && b.CoreAgentURL != "" {
		if err := paths.WriteSecret("secondary_agent_token", b.AgentToken); err != nil { return fmt.Errorf("save secondary agent token: %w", err) }
		if err := paths.WriteSecret("secondary_core_url", b.CoreAgentURL); err != nil { return fmt.Errorf("save secondary core URL: %w", err) }
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
		if err := writeUnit("netductor-secondary-agent.service", agentUnit); err != nil { return err }
		if err := enableStart("netductor-secondary-agent"); err != nil { return err }
		if out, err := exec.Command("systemctl", "is-active", "netductor-secondary-agent").CombinedOutput(); err != nil || strings.TrimSpace(string(out)) != "active" {
			return fmt.Errorf("secondary agent did not become active: %s (%v)", strings.TrimSpace(string(out)), err)
		}
		fmt.Fprintln(os.Stderr, "secondary agent started →", b.CoreAgentURL)
	}
	fmt.Fprintf(os.Stderr, "secondary ready · public_ip=%s · SNI=%s · client links in %s\n", pubIP, b.SecondarySNI, outDir)
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
