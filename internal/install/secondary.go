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

	"github.com/PavelNeyman/netductor/internal/mtls"
	"github.com/PavelNeyman/netductor/internal/nodes"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/version"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

// InstallRelay joins a RU VPS to core using bundle.json from core export.
func InstallRelay(bundlePath string) (retErr error) {
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
		coreURL.Hostname() != strings.TrimSpace(b.CoreIP) || coreURL.Port() != mtls.AgentTLSPort ||
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

	// Validate every value that can fail deterministically before any host mutation.
	if len(b.Users) == 0 {
		return fmt.Errorf("invalid bundle: no end users")
	}
	if strings.TrimSpace(b.SecondarySNI) == "" {
		return fmt.Errorf("invalid bundle: relay_sni is empty")
	}
	seenUsers := make(map[string]struct{}, len(b.Users))
	for _, u := range b.Users {
		name := strings.TrimSpace(u.Name)
		if name == "" || name == "." || name == ".." || filepath.Base(name) != name || strings.ContainsAny(name, "/\\\\") {
			return fmt.Errorf("invalid secondary client link name %q", u.Name)
		}
		if strings.TrimSpace(u.UUID) == "" {
			return fmt.Errorf("invalid bundle: user %q has empty UUID", name)
		}
		if _, exists := seenUsers[name]; exists {
			return fmt.Errorf("invalid bundle: duplicate user name %q", name)
		}
		seenUsers[name] = struct{}{}
	}

	// Establish and verify the host firewall baseline before optional cleanup.
	// Hoster purge is destructive and has no reliable inverse; defer it until
	// the relay services have passed their startup checks.
	if err := EnsureHostBaseline("secondary"); err != nil {
		return fmt.Errorf("secondary host baseline: %w", err)
	}
	fmt.Fprintln(os.Stderr, "==> secondary: dirs")
	if err := paths.EnsureLayout(); err != nil { return fmt.Errorf("secondary state layout: %w", err) }
	// Snapshot every persistent file this install path can replace before the
	// first hostname, secret, config, registry or client-link mutation.
	stateTxn, err := beginSecondaryFileTransaction(secondaryInstallMutationPaths(b.Users))
	if err != nil { return fmt.Errorf("snapshot secondary install state: %w", err) }
	defer func() {
		if retErr == nil { return }
		if rollbackErr := stateTxn.rollback(); rollbackErr != nil {
			retErr = fmt.Errorf("%w; secondary state rollback incomplete: %v", retErr, rollbackErr)
		}
	}()
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
	newPair := priv == "" || pub == ""
	if newPair {
		out, err := exec.Command(bin, "generate", "reality-keypair").Output()
		if err != nil {
			return fmt.Errorf("generate Reality keypair: %w", err)
		}
		priv, pub = "", ""
		for _, line := range strings.Split(string(out), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && strings.Contains(fields[0], "Private") {
				priv = fields[len(fields)-1]
			}
			if len(fields) >= 2 && strings.Contains(fields[0], "Public") {
				pub = fields[len(fields)-1]
			}
		}
		if priv == "" || pub == "" {
			return fmt.Errorf("sing-box returned incomplete Reality keypair")
		}
	}
	if sid == "" {
		sid = randomHex(8)
	}
	// Replace the related Reality values as one rollback-capable file operation.
	if err := writeSecretSet(map[string]string{
		"singbox_reality_private": priv,
		"singbox_reality_public":  pub,
		"singbox_short_id":        sid,
		"singbox_reality_sni":     b.SecondarySNI,
	}); err != nil {
		return fmt.Errorf("save Reality secret set: %w", err)
	}

	// store bundle for re-apply
	if err := os.MkdirAll(paths.SecondaryDir(), 0o700); err != nil { return fmt.Errorf("create secondary state directory: %w", err) }
	if err := writeInstallFileDurable(filepath.Join(paths.SecondaryDir(), "bundle.json"), raw, 0o600); err != nil { return fmt.Errorf("save secondary bundle: %w", err) }

	if err := vpn.WriteSecondarySingBox(&b, priv, sid); err != nil {
		return err
	}

	// Preserve prior unit definitions and enablement until activation succeeds.
	unitTxn, err := beginSecondaryUnitTransaction()
	if err != nil { return fmt.Errorf("snapshot secondary systemd units: %w", err) }
	unitTxnTouched := false
	defer func() {
		if retErr == nil || !unitTxnTouched { return }
		if rollbackErr := unitTxn.rollback(); rollbackErr != nil {
			retErr = fmt.Errorf("%w; secondary unit rollback incomplete: %v", retErr, rollbackErr)
		}
	}()

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
	unitTxnTouched = true
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

	// This purge removes packages, configuration and files and has no reliable
	// inverse. Run it only after the relay and agent are confirmed healthy.
	PurgeHostMonitoring()
	fmt.Fprintf(os.Stderr, "secondary ready · public_ip=%s · SNI=%s · client links in %s\n", pubIP, b.SecondarySNI, outDir)
	fmt.Fprintln(os.Stderr, "Give mobile users *-relay links; home users keep core links.")
	return nil
}

// InstallSingBoxBinaryOnly ensures sing-box exists (may run full InstallSingBox once).
func InstallSingBoxBinaryOnly() error {
	const destination = "/usr/local/bin/sing-box"
	st, err := os.Lstat(destination)
	if err == nil {
		if !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("existing sing-box binary is not a regular file")
		}
		if st.Mode().Perm()&0o111 == 0 {
			return fmt.Errorf("existing sing-box binary is not executable")
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("inspect sing-box binary: %w", err)
	}

	// Do not call InstallSingBox here: that routine also writes secrets/config,
	// installs and starts a systemd unit, and would escape InstallRelay's transaction.
	stage, err := os.MkdirTemp("/usr/local/bin", ".sing-box-stage-*")
	if err != nil {
		return fmt.Errorf("create sing-box staging directory: %w", err)
	}
	defer os.RemoveAll(stage)
	tag := resolveSingBoxTag()
	if err := downloadSingBoxVersion(stage, tag); err != nil {
		return fmt.Errorf("download sing-box %s: %w", tag, err)
	}
	binary, err := os.ReadFile(filepath.Join(stage, "sing-box"))
	if err != nil {
		return fmt.Errorf("read staged sing-box binary: %w", err)
	}
	if len(binary) == 0 {
		return fmt.Errorf("downloaded sing-box binary is empty")
	}
	if err := writeInstallFileDurable(destination, binary, 0o755); err != nil {
		return fmt.Errorf("install sing-box binary: %w", err)
	}
	return nil
}


type secondaryUnitSnapshot struct {
	name string
	path string
	data []byte
	mode os.FileMode
	exists bool
	enabledState string
	activeState string
}

type secondaryUnitTransaction struct { units []secondaryUnitSnapshot }

func beginSecondaryUnitTransaction() (*secondaryUnitTransaction, error) {
	t := &secondaryUnitTransaction{}
	for _, name := range []string{"sing-box.service", "netductor-secondary-agent.service"} {
		path := filepath.Join("/etc/systemd/system", name)
		u := secondaryUnitSnapshot{name: name, path: path}
		st, err := os.Lstat(path)
		if err == nil {
			if !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 { return nil, fmt.Errorf("unit %s is not a regular file", name) }
			u.data, err = os.ReadFile(path)
			if err != nil { return nil, fmt.Errorf("read %s: %w", path, err) }
			u.mode, u.exists = st.Mode().Perm(), true
		} else if !os.IsNotExist(err) { return nil, fmt.Errorf("inspect %s: %w", path, err) }
		unit := strings.TrimSuffix(name, ".service")
		out, enabledErr := exec.Command("systemctl", "is-enabled", unit).CombinedOutput()
		u.enabledState = strings.TrimSpace(string(out))
		if enabledErr != nil {
			switch u.enabledState {
			case "disabled", "not-found", "static", "indirect", "masked", "masked-runtime", "generated", "transient":
			default: return nil, fmt.Errorf("query enablement for %s: %s: %w", name, u.enabledState, enabledErr)
			}
		}
		activeOut, activeErr := exec.Command("systemctl", "is-active", unit).CombinedOutput()
		u.activeState = strings.TrimSpace(string(activeOut))
		if activeErr != nil && u.activeState != "inactive" && u.activeState != "failed" {
			return nil, fmt.Errorf("query active state for %s: %s: %w", name, u.activeState, activeErr)
		}
		if u.activeState != "active" && u.activeState != "inactive" && u.activeState != "failed" {
			return nil, fmt.Errorf("query active state for %s: unexpected state %q", name, u.activeState)
		}
		t.units = append(t.units, u)
	}
	return t, nil
}

func (t *secondaryUnitTransaction) rollback() error {
	var errs []string
	// Stop managed units before restoring definitions and files.
	for _, unit := range []string{"netductor-secondary-agent", "sing-box"} {
		if out, err := exec.Command("systemctl", "stop", unit).CombinedOutput(); err != nil {
			errs = append(errs, fmt.Sprintf("stop %s: %s: %v", unit, strings.TrimSpace(string(out)), err))
		}
	}
	restoreFailed := make(map[string]bool, len(t.units))
	for i := len(t.units)-1; i >= 0; i-- {
		u := t.units[i]
		var err error
		if u.exists {
			err = writeInstallFileDurable(u.path, u.data, u.mode)
		} else {
			err = os.Remove(u.path)
			if os.IsNotExist(err) { err = nil }
			if err == nil { err = syncDirectory(filepath.Dir(u.path)) }
		}
		if err != nil {
			restoreFailed[u.name] = true
			errs = append(errs, fmt.Sprintf("restore %s: %v", u.name, err))
		}
	}
	reloadErr := run("systemctl", "daemon-reload")
	if reloadErr != nil { errs = append(errs, fmt.Sprintf("daemon-reload: %v", reloadErr)) }
	for _, u := range t.units {
		unit := strings.TrimSuffix(u.name, ".service")
		var err error
		switch u.enabledState {
		case "enabled", "enabled-runtime":
			err = run("systemctl", "enable", unit)
		case "masked", "masked-runtime":
			_ = run("systemctl", "unmask", unit)
			if u.enabledState == "masked-runtime" {
				err = run("systemctl", "mask", "--runtime", "--force", unit)
			} else {
				err = run("systemctl", "mask", "--force", unit)
			}
		default:
			err = run("systemctl", "disable", unit)
			if err == nil && u.enabledState == "masked" {
				err = run("systemctl", "mask", "--force", unit)
			}
		}
		if err != nil { errs = append(errs, fmt.Sprintf("restore enablement %s (%s): %v", unit, u.enabledState, err)) }
		if u.activeState == "active" && reloadErr == nil && !restoreFailed[u.name] {
			if err := run("systemctl", "start", unit); err != nil {
				errs = append(errs, fmt.Sprintf("restore active state %s: %v", unit, err))
			}
		}
	}
	if len(errs) != 0 { return fmt.Errorf("%s", strings.Join(errs, "; ")) }
	return nil
}


type secondaryFileSnapshot struct {
	path string
	data []byte
	mode os.FileMode
	exists bool
}

type secondaryFileTransaction struct { files []secondaryFileSnapshot; runtimeHostname string }

func secondaryInstallMutationPaths(users []vpn.RelayUser) []string {
	etc := paths.EtcDir()
	state := paths.StateDir()
	out := []string{
		filepath.Join(etc, "node_id"),
		filepath.Join(etc, "public_ip"),
		filepath.Join(etc, "node_uuid"),
		"/etc/hostname",
		"/etc/hosts",
		filepath.Join(etc, "secrets", "singbox_reality_private"),
		filepath.Join(etc, "secrets", "singbox_reality_public"),
		filepath.Join(etc, "secrets", "singbox_short_id"),
		filepath.Join(etc, "secrets", "singbox_reality_sni"),
		filepath.Join(etc, "secrets", "secondary_agent_token"),
		filepath.Join(etc, "secrets", "secondary_core_url"),
		filepath.Join(paths.SecondaryDir(), "bundle.json"),
		filepath.Join("/usr/local/etc/sing-box", "config.json"),
		"/usr/local/bin/sing-box",
		filepath.Join(state, "nodes", "registry.json"),
	}
	for _, u := range users {
		out = append(out, filepath.Join(paths.SecondaryDir(), "clients", u.Name+".txt"))
	}
	return out
}

func beginSecondaryFileTransaction(pathsToCapture []string) (*secondaryFileTransaction, error) {
	t := &secondaryFileTransaction{}
	var err error
	t.runtimeHostname, err = os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("capture runtime hostname: %w", err)
	}
	if strings.TrimSpace(t.runtimeHostname) == "" {
		return nil, fmt.Errorf("capture runtime hostname: empty hostname")
	}
	seen := make(map[string]struct{}, len(pathsToCapture))
	for _, path := range pathsToCapture {
		if _, ok := seen[path]; ok { continue }
		seen[path] = struct{}{}
		s := secondaryFileSnapshot{path: path}
		st, err := os.Lstat(path)
		if err == nil {
			if !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 {
				return nil, fmt.Errorf("refuse to snapshot non-regular file %s", path)
			}
			s.data, err = os.ReadFile(path)
			if err != nil { return nil, fmt.Errorf("read %s: %w", path, err) }
			s.mode, s.exists = st.Mode().Perm(), true
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("inspect %s: %w", path, err)
		}
		t.files = append(t.files, s)
	}
	return t, nil
}

func (t *secondaryFileTransaction) rollback() error {
	var errs []string
	for i := len(t.files)-1; i >= 0; i-- {
		s := t.files[i]
		var err error
		if s.exists {
			err = writeInstallFileDurable(s.path, s.data, s.mode)
		} else {
			_, statErr := os.Lstat(s.path)
			if os.IsNotExist(statErr) {
				continue
			}
			if statErr != nil {
				err = statErr
			} else {
				err = os.Remove(s.path)
				if err == nil { err = syncDirectory(filepath.Dir(s.path)) }
			}
		}
		if err != nil { errs = append(errs, fmt.Sprintf("restore %s: %v", s.path, err)) }
	}
	// Restore the kernel's runtime hostname only after the on-disk hostname is restored.
	if t.runtimeHostname != "" {
		if err := run("hostnamectl", "set-hostname", t.runtimeHostname); err != nil {
			errs = append(errs, fmt.Sprintf("restore runtime hostname %s: %v", t.runtimeHostname, err))
		}
	}
	if len(errs) != 0 { return fmt.Errorf("%s", strings.Join(errs, "; ")) }
	return nil
}
