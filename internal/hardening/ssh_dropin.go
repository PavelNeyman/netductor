package hardening

import "fmt"

// DropInConf is the single source of truth for sshd netductor harden drop-in.
// Used by primary install (local write) and secondary provision (remote script).
func DropInConf() string {
	return fmt.Sprintf(`Port %d
PasswordAuthentication no
KbdInteractiveAuthentication no
ChallengeResponseAuthentication no
PermitRootLogin prohibit-password
PubkeyAuthentication yes
X11Forwarding no
`, SSHPort())
}

// RemoteHardenScript applies DropInConf on a remote Debian host (secondary provision).
// Restarts sshd so Port takes effect.
func RemoteHardenScript() string {
	port := SSHPort()
	return fmt.Sprintf(`set -e
mkdir -p /etc/ssh/sshd_config.d
cat > /etc/ssh/sshd_config.d/00-netductor-harden.conf << 'NDSSH'
%sNDSSH
for f in /etc/ssh/sshd_config.d/*.conf; do
  [ -f "$f" ] || continue
  case "$f" in *netductor*) continue ;; esac
  sed -i 's/PasswordAuthentication yes/PasswordAuthentication no/gi' "$f" 2>/dev/null || true
done
if [ -f /etc/ssh/sshd_config ]; then
  sed -i 's/^#\?PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config
  sed -i 's/^#\?PermitRootLogin.*/PermitRootLogin prohibit-password/' /etc/ssh/sshd_config
  sed -i 's/^Port /# Port /' /etc/ssh/sshd_config
fi
if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -qi active; then
  ufw allow %d/tcp comment netductor-ssh 2>/dev/null || true
fi
systemctl restart sshd 2>/dev/null || systemctl restart ssh 2>/dev/null || service ssh restart 2>/dev/null || true
`, DropInConf(), port)
}
