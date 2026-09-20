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

	"github.com/PavelNeyman/netductor/internal/nodes"
	"github.com/PavelNeyman/netductor/internal/paths"
)

type Options struct {
	Components []string // empty = default core set
	Force      bool
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
	_ = copySelfToLocalBin()
	applyHostname("core")
	_ = nodes.LocalStableID() // stable node id (UUID), independent of hostname
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
			err = InstallTelegram()
		case "backup":
			err = InstallBackup()
		case "lampac":
			err = InstallLampac()
		default:
			fmt.Fprintf(os.Stderr, "skip unknown component %s\n", c)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", c, err)
		}
	}
	_ = os.WriteFile(filepath.Join(paths.StateDir(), "installed_version"), []byte("0.8.0\n"), 0o644)
	writeReady()
	// optional extras (env-gated)
	if os.Getenv("NETDUCTOR_LAMPAC") == "1" {
		fmt.Fprintln(os.Stderr, "==> lampac")
		if err := InstallLampac(); err != nil {
			fmt.Fprintln(os.Stderr, "lampac:", err)
		}
	}
	fmt.Fprintln(os.Stderr, "install finished")
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
	_ = os.MkdirAll(dir, 0o700)
	return os.WriteFile(filepath.Join(dir, name), []byte(val+"\n"), 0o600)
}

func readSecret(name string) string {
	b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "secrets", name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func writeUnit(name, content string) error {
	path := filepath.Join("/etc/systemd/system", name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	_ = run("systemctl", "daemon-reload")
	return nil
}

func enableStart(unit string) error {
	_ = run("systemctl", "enable", unit)
	return run("systemctl", "restart", unit)
}


// applyHostname sets a uniform node name.
// Priority: NETDUCTOR_HOSTNAME env > existing /etc/netductor/node_id > auto nd-<role>-<ip-suffix>
func applyHostname(role string) {
	if role == "" {
		role = "core"
	}
	name := strings.TrimSpace(os.Getenv("NETDUCTOR_HOSTNAME"))
	if name == "" {
		if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "node_id")); err == nil {
			name = strings.TrimSpace(string(b))
		}
	}
	if name == "" {
		ip := detectPublicIP()
		suf := "00"
		if ip != "" {
			parts := strings.Split(ip, ".")
			if len(parts) == 4 {
				suf = parts[2] + parts[3]
				if len(suf) > 6 {
					suf = suf[len(suf)-6:]
				}
			}
		}
		name = fmt.Sprintf("nd-%s-%s", role, suf)
	}
	name = strings.ToLower(name)
	_ = os.MkdirAll(paths.EtcDir(), 0o755)
	_ = os.WriteFile(filepath.Join(paths.EtcDir(), "node_id"), []byte(name+"\n"), 0o644)
	_ = os.WriteFile("/etc/hostname", []byte(name+"\n"), 0o644)
	_ = run("hostnamectl", "set-hostname", name)
	// ensure hosts entry
	_ = run("bash", "-c", fmt.Sprintf(
		`grep -q '%s' /etc/hosts || echo '127.0.1.1 %s' >> /etc/hosts`, name, name))
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
	_ = nodes.SelfRegisterLocal(hn, "core", ip)
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
