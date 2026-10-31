package install

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/ndconfig"
	"github.com/PavelNeyman/netductor/internal/nodes"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/version"
)

var currentComps []string

type Options struct {
	Components   []string // empty = default core set
	Force        bool
	SkipHostname bool // recover: do not invent nd-primary-*; restore name from backup after tar
}

func DefaultComponents() []string {
	return []string{"dirs", "hardening", "singbox", "blocky", "vpn-users", "api", "metrics", "telegram", "backup"}
}

func Run(opts Options) error {
	currentComps = opts.Components
	if os.Geteuid() != 0 {
		return fmt.Errorf("install requires root")
	}
	if _, err := os.Stat("/etc/debian_version"); err != nil {
		return fmt.Errorf("only Debian-like hosts supported for full install")
	}
	comps := opts.Components
	if len(comps) == 0 {
		comps = DefaultComponents()
	}
	if err := paths.EnsureLayout(); err != nil {
		return err
	}
	ndconfig.EnsureDefaultsInConf()
	ndconfig.Load()
	_ = copySelfToLocalBin()
	_ = EnsureDomainConfig()
	_ = EnsureClientProfiles()
	if !opts.SkipHostname {
		if err := applyHostname("primary"); err != nil {
			return fmt.Errorf("set primary hostname: %w", err)
		}
	}
	_ = nodes.LocalStableID() // stable node id (UUID), independent of hostname
	var errs []error
	for _, c := range comps {
		fmt.Fprintf(os.Stderr, "==> %s\n", c)
		var err error
		switch c {
		case "dirs":
			err = paths.EnsureLayout()
		case "hardening":
			err = InstallHardening()
		case "singbox", "sing-box":
			err = InstallSingBox()
		case "blocky":
			err = InstallBlocky()
		case "vpn-users", "vpn":
			err = InstallVPNUsers()
		case "api", "netductor-api":
			err = InstallAPI()
		case "metrics":
			err = InstallMetrics()
		case "telegram", "tg":
			_ = EnsureClientProfiles()
			_ = InstallRedirect()
			err = InstallTelegram()
			// Soft: missing release asset must not fail whole install (deploy continues domain/LE).
			// Unit stays inactive until: netductor install telegram (after asset published).
			if err != nil {
				fmt.Fprintf(os.Stderr, "component telegram: %v (soft — install continues)\n", err)
				err = nil
			}
		case "backup":
			err = InstallBackup()
		case "lampac":
			err = InstallLampac()
			// Soft: optional media stack must not fail recover drill / full install.
			if err != nil {
				fmt.Fprintf(os.Stderr, "component lampac: %v (soft — install continues)\n", err)
				err = nil
			}
		case "git":
			// bare repos root; sample repo optional
			err = ensureGitComponent()
			if err != nil {
				fmt.Fprintf(os.Stderr, "component git: %v (soft — install continues)\n", err)
				err = nil
			}
		case "registry":
			err = ensureRegistryComponent()
			if err != nil {
				fmt.Fprintf(os.Stderr, "component registry: %v (soft — install continues)\n", err)
				err = nil
			}
		default:
			fmt.Fprintf(os.Stderr, "skip unknown component %s\n", c)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "component %s: %v (continuing)\n", c, err)
			errs = append(errs, fmt.Errorf("%s: %w", c, err))
			continue
		}
	}
	_ = os.WriteFile(filepath.Join(paths.StateDir(), "installed_version"), []byte(version.Release+"\n"), 0o644)
	writeReady()
	// optional extras (env-gated)
	if os.Getenv("NETDUCTOR_LAMPAC") == "1" {
		fmt.Fprintln(os.Stderr, "==> lampac")
		if err := InstallLampac(); err != nil {
			fmt.Fprintln(os.Stderr, "lampac:", err)
		}
	}
	fmt.Fprintln(os.Stderr, "install finished")
	if len(errs) > 0 {
		parts := make([]string, 0, len(errs))
		for _, e := range errs {
			parts = append(parts, e.Error())
		}
		return fmt.Errorf("install partial: %s", strings.Join(parts, "; "))
	}
	return nil
}

func arch() string {
	switch runtime.GOARCH {
	case "amd64", "arm64":
		return runtime.GOARCH
	case "arm":
		return "armv7"
	default:
		return runtime.GOARCH
	}
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func runOut(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return string(out), err
}

// writeSecretSet stages a related set of secret files before replacing any live value.
// Existing files are retained in a same-filesystem backup until every replacement is durable.
func writeSecretSet(values map[string]string) error {
	if len(values) == 0 {
		return fmt.Errorf("empty secret set")
	}
	dir := filepath.Join(paths.EtcDir(), "secrets")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create secrets directory: %w", err)
	}
	stage, err := os.MkdirTemp(dir, ".secret-stage-*")
	if err != nil {
		return fmt.Errorf("create secret staging directory: %w", err)
	}
	defer os.RemoveAll(stage)
	backup, err := os.MkdirTemp(dir, ".secret-backup-*")
	if err != nil {
		return fmt.Errorf("create secret backup directory: %w", err)
	}
	backupKept := false
	defer func() {
		if !backupKept {
			_ = os.RemoveAll(backup)
		}
	}()
	keys := make([]string, 0, len(values))
	for name := range values {
		if name == "" || name == "." || name == ".." || filepath.Base(name) != name || strings.ContainsAny(name, "/\\\\") {
			return fmt.Errorf("invalid secret filename %q", name)
		}
		keys = append(keys, name)
	}
	sort.Strings(keys)
	for _, name := range keys {
		if err := writeInstallFileDurable(filepath.Join(stage, name), []byte(values[name]+"\\n"), 0o600); err != nil {
			return fmt.Errorf("stage secret %s: %w", name, err)
		}
	}
	if err := syncDirectory(stage); err != nil {
		return fmt.Errorf("sync staged secrets: %w", err)
	}
	backed := make(map[string]bool, len(keys))
	installed := make(map[string]bool, len(keys))
	rollback := func(cause error) error {
		var rollbackErr error
		for i := len(keys) - 1; i >= 0; i-- {
			name := keys[i]
			dest := filepath.Join(dir, name)
			if installed[name] {
				if err := os.Remove(dest); err != nil && !os.IsNotExist(err) && rollbackErr == nil {
					rollbackErr = fmt.Errorf("remove partial secret %s: %w", name, err)
				}
			}
			if backed[name] {
				if err := os.Rename(filepath.Join(backup, name), dest); err != nil && rollbackErr == nil {
					rollbackErr = fmt.Errorf("restore secret %s: %w", name, err)
				}
			}
		}
		if err := syncDirectory(dir); err != nil && rollbackErr == nil {
			rollbackErr = fmt.Errorf("sync secret rollback: %w", err)
		}
		if rollbackErr != nil {
			backupKept = true
			return fmt.Errorf("%w; secret rollback incomplete: %v; backup retained at %s", cause, rollbackErr, backup)
		}
		return cause
	}
	for _, name := range keys {
		dest := filepath.Join(dir, name)
		st, err := os.Lstat(dest)
		if err == nil {
			if !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 {
				return rollback(fmt.Errorf("existing secret %s is not a regular file", name))
			}
			if err := os.Rename(dest, filepath.Join(backup, name)); err != nil {
				return rollback(fmt.Errorf("backup secret %s: %w", name, err))
			}
			backed[name] = true
		} else if !os.IsNotExist(err) {
			return rollback(fmt.Errorf("inspect secret %s: %w", name, err))
		}
		if err := syncDirectory(dir); err != nil {
			return rollback(fmt.Errorf("sync secret backup: %w", err))
		}
	}
	for _, name := range keys {
		if err := os.Rename(filepath.Join(stage, name), filepath.Join(dir, name)); err != nil {
			return rollback(fmt.Errorf("install secret %s: %w", name, err))
		}
		installed[name] = true
		if err := syncDirectory(dir); err != nil {
			return rollback(fmt.Errorf("sync installed secret %s: %w", name, err))
		}
	}
	if err := syncDirectory(dir); err != nil {
		return rollback(fmt.Errorf("sync committed secret set: %w", err))
	}
	if err := os.RemoveAll(backup); err != nil {
		backupKept = true
		return fmt.Errorf("secret set installed but backup cleanup failed; backup retained at %s: %w", backup, err)
	}
	backupKept = true
	return nil
}

func syncDirectory(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func writeSecret(name, val string) error {
	dir := filepath.Join(paths.EtcDir(), "secrets")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create secrets directory: %w", err)
	}
	return writeInstallFileDurable(filepath.Join(dir, name), []byte(val+"\n"), 0o600)
}

func readSecret(name string) string {
	b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "secrets", name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func writeUnit(name, content string) error {
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name || strings.ContainsAny(name, "/\\\\") {
		return fmt.Errorf("invalid systemd unit filename %q", name)
	}
	path := filepath.Join("/etc/systemd/system", name)
	var previous []byte
	var previousMode os.FileMode
	existed := false
	st, statErr := os.Lstat(path)
	if statErr == nil {
		if !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("existing unit %s is not a regular file", name)
		}
		previous, statErr = os.ReadFile(path)
		if statErr != nil {
			return fmt.Errorf("read previous unit %s: %w", name, statErr)
		}
		previousMode = st.Mode().Perm()
		existed = true
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("inspect previous unit %s: %w", name, statErr)
	}
	if err := writeInstallFileDurable(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write unit %s: %w", name, err)
	}
	if err := run("systemctl", "daemon-reload"); err != nil {
		// The new file is already on disk. Restore the prior unit definition before
		// returning so systemd is not left with a failed partially-applied update.
		var restoreErr error
		if existed {
			restoreErr = writeInstallFileDurable(path, previous, previousMode)
		} else {
			restoreErr = os.Remove(path)
			if os.IsNotExist(restoreErr) {
				restoreErr = nil
			}
			if restoreErr == nil {
				restoreErr = syncDirectory(filepath.Dir(path))
			}
		}
		if restoreErr == nil {
			restoreErr = run("systemctl", "daemon-reload")
		}
		if restoreErr != nil {
			return fmt.Errorf("systemctl daemon-reload failed: %w; restoring previous unit %s also failed: %v", err, name, restoreErr)
		}
		return fmt.Errorf("systemctl daemon-reload failed; previous unit %s restored: %w", name, err)
	}
	return nil
}

func enableStart(unit string) error {
	if err := run("systemctl", "enable", unit); err != nil {
		return fmt.Errorf("systemctl enable %s: %w", unit, err)
	}
	if err := run("systemctl", "restart", unit); err != nil {
		return fmt.Errorf("systemctl restart %s: %w", unit, err)
	}
	return nil
}

func writeInstallFileDurable(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil { return err }
	tmp, err := os.CreateTemp(dir, ".netductor-install-*")
	if err != nil { return err }
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	n, err := tmp.Write(data)
	if err != nil { _ = tmp.Close(); return err }
	if n != len(data) { _ = tmp.Close(); return io.ErrShortWrite }
	if err := tmp.Chmod(mode); err != nil { _ = tmp.Close(); return err }
	if err := tmp.Sync(); err != nil { _ = tmp.Close(); return err }
	if err := tmp.Close(); err != nil { return err }
	if err := os.Rename(tmpName, path); err != nil { return err }
	df, err := os.Open(dir)
	if err != nil { return err }
	if err := df.Sync(); err != nil { _ = df.Close(); return err }
	return df.Close()
}

// applyHostname sets a uniform node name.
// Priority: NETDUCTOR_HOSTNAME env > existing /etc/netductor/node_id > auto nd-<role>-<ip-suffix>
func applyHostname(role string) error {
	if role == "" { role = "primary" }
	name := strings.TrimSpace(os.Getenv("NETDUCTOR_HOSTNAME"))
	if name == "" {
		if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "node_id")); err == nil { name = strings.TrimSpace(string(b)) }
	}
	if name == "" {
		ip := detectPublicIP()
		suf := "00"
		if ip != "" {
			parts := strings.Split(ip, ".")
			if len(parts) == 4 { suf = parts[2] + parts[3]; if len(suf) > 6 { suf = suf[len(suf)-6:] } }
		}
		name = fmt.Sprintf("nd-%s-%s", role, suf)
	}
	name = strings.ToLower(name)
	if len(name) == 0 || len(name) > 63 || name[0] == '.' || name[len(name)-1] == '.' { return fmt.Errorf("invalid hostname %q", name) }
	for _, label := range strings.Split(name, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' { return fmt.Errorf("invalid hostname label in %q", name) }
		for _, ch := range label {
			if (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') && ch != '-' { return fmt.Errorf("invalid hostname character in %q", name) }
		}
	}
	if err := os.MkdirAll(paths.EtcDir(), 0o755); err != nil { return fmt.Errorf("create netductor config dir: %w", err) }
	if err := writeInstallFileDurable(filepath.Join(paths.EtcDir(), "node_id"), []byte(name+"\n"), 0o644); err != nil { return fmt.Errorf("write node_id: %w", err) }
	if err := writeInstallFileDurable("/etc/hostname", []byte(name+"\n"), 0o644); err != nil { return fmt.Errorf("write /etc/hostname: %w", err) }
	if err := run("hostnamectl", "set-hostname", name); err != nil { return fmt.Errorf("hostnamectl: %w", err) }
	hosts, err := os.ReadFile("/etc/hosts")
	if err != nil { return fmt.Errorf("read /etc/hosts: %w", err) }
	exists := false
	for _, line := range strings.Split(string(hosts), "\n") {
		fields := strings.Fields(strings.SplitN(line, "#", 2)[0])
		for _, host := range fields[1:] { if host == name { exists = true; break } }
		if exists { break }
	}
	if !exists {
		f, err := os.OpenFile("/etc/hosts", os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil { return fmt.Errorf("open /etc/hosts: %w", err) }
		line := []byte("127.0.1.1 " + name + "\n")
		n, werr := f.Write(line)
		if werr == nil && n != len(line) { werr = io.ErrShortWrite }
		if werr == nil { werr = f.Sync() }
		cerr := f.Close()
		if werr != nil { return fmt.Errorf("append /etc/hosts: %w", werr) }
		if cerr != nil { return fmt.Errorf("close /etc/hosts: %w", cerr) }
	}
	return nil
}

func detectPublicIPv4() string {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "tcp4", addr)
		},
	}
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 { return fmt.Errorf("too many redirects") }
			if req.URL.Scheme != "https" { return fmt.Errorf("refuse non-HTTPS redirect") }
			return nil
		},
	}
	for _, endpoint := range []string{"https://api4.ipify.org", "https://ipv4.icanhazip.com"} {
		resp, err := client.Get(endpoint)
		if err != nil { continue }
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 64))
		closeErr := resp.Body.Close()
		if readErr != nil || closeErr != nil || resp.StatusCode != http.StatusOK { continue }
		ip := strings.TrimSpace(string(body))
		parsed := net.ParseIP(ip)
		if parsed != nil && parsed.To4() != nil { return parsed.To4().String() }
	}
	return ""
}

func detectPublicIP() string {
	if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "public_ip")); err == nil {
		if ip := strings.TrimSpace(string(b)); ip != "" {
			return ip
		}
	}
	client := &http.Client{Timeout: 5 * time.Second}
	for _, u := range []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
	} {
		resp, err := client.Get(u)
		if err != nil {
			continue
		}
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
		resp.Body.Close()
		ip := strings.TrimSpace(string(b))
		if ip != "" && !strings.Contains(ip, "<") {
			_ = os.MkdirAll(paths.EtcDir(), 0o755)
			_ = os.WriteFile(filepath.Join(paths.EtcDir(), "public_ip"), []byte(ip+"\n"), 0o644)
			return ip
		}
	}
	return ""
}

func writeReady() {
	ip := detectPublicIP()
	txt := fmt.Sprintf("Netductor ready\nETC=%s\nSTATE=%s\nOPT=%s\nIP=%s\n",
		paths.EtcDir(), paths.StateDir(), paths.OptDir(), ip)
	_ = os.WriteFile(filepath.Join(paths.EtcDir(), "READY.txt"), []byte(txt), 0o644)
	hn := ""
	if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "node_id")); err == nil {
		hn = strings.TrimSpace(string(b))
	}
	if hn == "" {
		if b, err := os.ReadFile("/etc/hostname"); err == nil {
			hn = strings.TrimSpace(string(b))
		}
	}
	_ = nodes.SelfRegisterLocal(hn, "primary", ip)
}

func copySelfToLocalBin() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	dest := "/usr/local/bin/netductor"
	// resolve symlinks
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	if real, err := filepath.EvalSymlinks(dest); err == nil && real == exe {
		return nil
	}
	if exe == dest {
		return nil
	}
	in, err := os.ReadFile(exe)
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", dest, os.Getpid())
	if err := os.WriteFile(tmp, in, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		// Text file busy: leave side-by-side binary for next restart
		alt := dest + ".new"
		_ = os.Rename(tmp, alt)
		fmt.Fprintf(os.Stderr, "copySelf: dest busy — wrote %s (restart netductor-api to pick up)\n", alt)
		return nil
	}
	return nil
}

func ensureGitComponent() error {
	// thin: create git root; repos restored from backup data under var/lib/netductor/git
	dir := filepath.Join(paths.StateDir(), "git")
	return os.MkdirAll(dir, 0o755)
}

func ensureRegistryComponent() error {
	// docker registry; data under var/lib/netductor/registry restored by tar
	if err := exec.Command("docker", "info").Run(); err != nil {
		return fmt.Errorf("docker required for registry recover: %w", err)
	}
	// start empty registry; blob data already restored from backup if present
	_ = exec.Command("docker", "rm", "-f", "netductor-registry").Run()
	data := filepath.Join(paths.StateDir(), "registry")
	_ = os.MkdirAll(data, 0o755)
	return exec.Command("docker", "run", "-d", "--name", "netductor-registry", "--restart", "unless-stopped",
		"-p", "127.0.0.1:5000:5000", "-v", data+":/var/lib/registry", "registry:2").Run()
}
