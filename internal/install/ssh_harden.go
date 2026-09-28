package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/hardening"
)

// EnsureSSHKeyAndHarden locks SSH to pubkey-only and sets Port (default 52222).
// Operator (Mac) pubkey is expected already in authorized_keys from DeployPrimary.
// We do NOT generate /root/.ssh/id_ed25519 for mesh SSH — primary does not need to SSH
// to secondary/edge after provision (agent HTTP/mTLS). Keygen only if authorized_keys
// is empty, so a bare `netductor install` on the box cannot lock root out.
// Only Port SSHPort() listens — default :22 is commented out and denied in ufw.
func EnsureSSHKeyAndHarden() error {
	sshDir := "/root/.ssh"
	_ = os.MkdirAll(sshDir, 0o700)
	ak := filepath.Join(sshDir, "authorized_keys")
	existing, _ := os.ReadFile(ak)
	hasKey := false
	for _, line := range strings.Split(string(existing), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			hasKey = true
			break
		}
	}
	if !hasKey {
		priv := filepath.Join(sshDir, "id_ed25519")
		pub := priv + ".pub"
		if _, err := os.Stat(priv); err != nil {
			cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-f", priv, "-C", "netductor-primary-local")
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("ssh-keygen: %s %w", strings.TrimSpace(string(out)), err)
			}
			fmt.Fprintln(os.Stderr, "ssh: generated /root/.ssh/id_ed25519 (authorized_keys was empty; copy .pub off-box before relying on key-only)")
		}
		pubBytes, err := os.ReadFile(pub)
		if err != nil {
			return fmt.Errorf("read pubkey: %w (authorized_keys empty and no local key)", err)
		}
		pubLine := strings.TrimSpace(string(pubBytes))
		if err := os.WriteFile(ak, []byte(pubLine+"\n"), 0o600); err != nil {
			return err
		}
		_ = os.Chmod(priv, 0o600)
	} else {
		fmt.Fprintln(os.Stderr, "ssh: using existing authorized_keys (no local id_ed25519 generated)")
	}
	_ = os.Chmod(sshDir, 0o700)
	_ = os.Chmod(ak, 0o600)

	_ = os.MkdirAll("/etc/ssh/sshd_config.d", 0o755)
	drop := "/etc/ssh/sshd_config.d/00-netductor-harden.conf"
	port := hardening.SSHPort()
	if err := os.WriteFile(drop, []byte(hardening.DropInConf()), 0o644); err != nil {
		return err
	}
	entries, _ := os.ReadDir("/etc/ssh/sshd_config.d")
	for _, e := range entries {
		name := e.Name()
		if name == "00-netductor-harden.conf" {
			continue
		}
		path := filepath.Join("/etc/ssh/sshd_config.d", name)
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		s := string(b)
		changed := false
		if strings.Contains(strings.ToLower(s), "passwordauthentication") && strings.Contains(strings.ToLower(s), "yes") {
			s2 := strings.ReplaceAll(s, "PasswordAuthentication yes", "PasswordAuthentication no")
			s2 = strings.ReplaceAll(s2, "PasswordAuthentication Yes", "PasswordAuthentication no")
			if s2 != s {
				s = s2
				changed = true
			}
		}
		// remove Port lines from other drop-ins so only our Port applies
		lines := strings.Split(s, "\n")
		var out []string
		for _, ln := range lines {
			trim := strings.TrimSpace(ln)
			if strings.HasPrefix(trim, "Port ") || strings.HasPrefix(trim, "Port\t") {
				changed = true
				continue
			}
			out = append(out, ln)
		}
		if changed {
			_ = os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644)
		}
	}
	_ = exec.Command("sed", "-i", "s/^#\\?PasswordAuthentication.*/PasswordAuthentication no/", "/etc/ssh/sshd_config").Run()
	_ = exec.Command("sed", "-i", "s/^#\\?PermitRootLogin.*/PermitRootLogin prohibit-password/", "/etc/ssh/sshd_config").Run()
	_ = exec.Command("sed", "-i", "s/^Port /# Port /", "/etc/ssh/sshd_config").Run()

	// ufw: allow only hardened port; deny classic :22
	if _, err := exec.LookPath("ufw"); err == nil {
		_ = exec.Command("ufw", "allow", fmt.Sprintf("%d/tcp", port), "comment", "netductor-ssh").Run()
		_ = exec.Command("ufw", "delete", "allow", "22/tcp").Run()
		_ = exec.Command("ufw", "deny", "22/tcp", "comment", "netductor-no-ssh22").Run()
	}

	_ = exec.Command("systemctl", "restart", "sshd").Run()
	_ = exec.Command("systemctl", "restart", "ssh").Run()
	fmt.Fprintf(os.Stderr, "ssh: harden drop-in Port %d key-only (no :22)\n", port)
	return nil
}
