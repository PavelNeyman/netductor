package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// EnsureSSHKeyAndHarden ensures a root ed25519 key, authorized_keys, and key-only SSH.
// Password auth is only for the first provider login before install runs.
func EnsureSSHKeyAndHarden() error {
	sshDir := "/root/.ssh"
	_ = os.MkdirAll(sshDir, 0o700)
	priv := filepath.Join(sshDir, "id_ed25519")
	pub := priv + ".pub"
	if _, err := os.Stat(priv); err != nil {
		cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-f", priv, "-C", "netductor-primary")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("ssh-keygen: %s %w", strings.TrimSpace(string(out)), err)
		}
		fmt.Fprintln(os.Stderr, "ssh: generated /root/.ssh/id_ed25519 (keep a copy; password auth will be disabled)")
	}
	pubBytes, err := os.ReadFile(pub)
	if err != nil {
		return fmt.Errorf("read pubkey: %w", err)
	}
	pubLine := strings.TrimSpace(string(pubBytes))
	ak := filepath.Join(sshDir, "authorized_keys")
	existing, _ := os.ReadFile(ak)
	if !strings.Contains(string(existing), pubLine) {
		f, err := os.OpenFile(ak, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		_, _ = f.WriteString(pubLine + "\n")
		_ = f.Close()
	}
	_ = os.Chmod(sshDir, 0o700)
	_ = os.Chmod(ak, 0o600)
	_ = os.Chmod(priv, 0o600)

	_ = os.MkdirAll("/etc/ssh/sshd_config.d", 0o755)
	// Lexically first so we win over cloud-init 00password.conf (sshd: first obtained value wins)
	drop := "/etc/ssh/sshd_config.d/00-netductor-harden.conf"
	body := "PasswordAuthentication no\nKbdInteractiveAuthentication no\nChallengeResponseAuthentication no\nPermitRootLogin prohibit-password\nPubkeyAuthentication yes\nX11Forwarding no\n"
	if err := os.WriteFile(drop, []byte(body), 0o644); err != nil {
		return err
	}
	// Neutralize other drop-ins that force password yes
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
		if strings.Contains(strings.ToLower(s), "passwordauthentication") && strings.Contains(strings.ToLower(s), "yes") {
			s2 := strings.ReplaceAll(s, "PasswordAuthentication yes", "PasswordAuthentication no")
			s2 = strings.ReplaceAll(s2, "PasswordAuthentication Yes", "PasswordAuthentication no")
			if s2 != s {
				_ = os.WriteFile(path, []byte(s2), 0o644)
			}
		}
	}
	_ = exec.Command("sed", "-i", "s/^#\\?PasswordAuthentication.*/PasswordAuthentication no/", "/etc/ssh/sshd_config").Run()
	_ = exec.Command("sed", "-i", "s/^#\\?PermitRootLogin.*/PermitRootLogin prohibit-password/", "/etc/ssh/sshd_config").Run()
	_ = exec.Command("systemctl", "reload", "sshd").Run()
	_ = exec.Command("systemctl", "reload", "ssh").Run()
	fmt.Fprintln(os.Stderr, "ssh: password auth disabled; key-only root login")
	return nil
}
