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
	body := fmt.Sprintf("Port %d\nPasswordAuthentication no\nKbdInteractiveAuthentication no\nChallengeResponseAuthentication no\nPermitRootLogin prohibit-password\nPubkeyAuthentication yes\nX11Forwarding no\n", port)
	if err := os.WriteFile(drop, []byte(body), 0o644); err != nil {
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
	fmt.Fprintf(os.Stderr, "ssh: harden drop-in Port %d key-only\n", port)
	return nil
}
