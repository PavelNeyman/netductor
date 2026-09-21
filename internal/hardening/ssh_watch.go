package hardening

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

func knownSSHIPPath() string {
	return filepath.Join(paths.EtcDir(), "secrets", "ssh_known_ips")
}

func LoadKnownSSHIPs() map[string]bool {
	m := map[string]bool{}
	b, err := os.ReadFile(knownSSHIPPath())
	if err != nil {
		return m
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m[line] = true
	}
	return m
}

func RecentSSHLogins() []string {
	out, err := exec.Command("journalctl", "-u", "ssh", "-u", "sshd", "--since", "2 hours ago", "--no-pager", "-o", "cat").CombinedOutput()
	if err != nil {
		out, _ = exec.Command("journalctl", "--since", "2 hours ago", "--no-pager", "-o", "cat").CombinedOutput()
	}
	var ips []string
	seen := map[string]bool{}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := sc.Text()
		if !strings.Contains(line, "Accepted") {
			continue
		}
		parts := strings.Fields(line)
		for i, p := range parts {
			if p == "from" && i+1 < len(parts) {
				ip := parts[i+1]
				if !seen[ip] {
					seen[ip] = true
					ips = append(ips, ip)
				}
			}
		}
	}
	return ips
}

func UnusualSSHAlerts() []string {
	known := LoadKnownSSHIPs()
	if len(known) == 0 {
		return nil
	}
	var msgs []string
	for _, ip := range RecentSSHLogins() {
		if !known[ip] {
			msgs = append(msgs, fmt.Sprintf("SSH login from unusual IP: %s at %s", ip, time.Now().UTC().Format(time.RFC3339)))
		}
	}
	return msgs
}

func EnsureFail2banSSH() error {
	if _, err := exec.LookPath("fail2ban-client"); err != nil {
		_ = exec.Command("apt-get", "install", "-y", "fail2ban").Run()
	}
	jail := `/etc/fail2ban/jail.d/netductor-ssh.conf`
	port := SSHPort()
	content := fmt.Sprintf(`[sshd]
enabled = true
port = %d
filter = sshd
backend = systemd
maxretry = 4
bantime = 1h
findtime = 10m
`, port)
	_ = os.MkdirAll("/etc/fail2ban/jail.d", 0o755)
	if err := os.WriteFile(jail, []byte(content), 0o644); err != nil {
		return err
	}
	_ = exec.Command("systemctl", "enable", "--now", "fail2ban").Run()
	return exec.Command("fail2ban-client", "reload").Run()
}

// DefaultSSHPort is used when deploy does not set NETDUCTOR_SSH_PORT.
// 52222 — non-standard, easy to remember, avoids colliding with common 2222 scanners slightly less than 22.
const DefaultSSHPort = 52222

// SSHPort returns NETDUCTOR_SSH_PORT or DefaultSSHPort.
func SSHPort() int {
	if v := os.Getenv("NETDUCTOR_SSH_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n < 65536 {
			return n
		}
	}
	return DefaultSSHPort
}
