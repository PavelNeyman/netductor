package install

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	if err := writeInstallFileDurable(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write unit %s: %w", name, err)
	}
	if err := run("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w", err)
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
