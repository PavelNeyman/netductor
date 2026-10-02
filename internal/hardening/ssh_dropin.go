package hardening

import "fmt"

// DropInConf is the single source of truth for sshd netductor harden drop-in.
// Always DefaultSSHPort (52222) — never inherit NETDUCTOR_SSH_PORT from operator
// deploy (secondary provision temporarily sets that env to 22 for first hop).
func DropInConf() string {
	return fmt.Sprintf(`Port %d
PasswordAuthentication no
KbdInteractiveAuthentication no
ChallengeResponseAuthentication no
PermitRootLogin prohibit-password
PubkeyAuthentication yes
X11Forwarding no
`, DefaultSSHPort)
}

// RemoteHardenScript applies DropInConf on a remote Debian host (secondary provision).
// Always hardens to DefaultSSHPort; disables ssh.socket so :22 cannot linger.
func RemoteHardenScript() string {
	port := DefaultSSHPort
	return fmt.Sprintf(`set -e
mkdir -p /etc/ssh/sshd_config.d
cat > /etc/ssh/sshd_config.d/00-netductor-harden.conf << 'NDSSH'
%sNDSSH
for f in /etc/ssh/sshd_config.d/*.conf; do
  [ -f "$f" ] || continue
  case "$f" in *00-netductor-harden.conf) continue ;; esac
  sed -i 's/PasswordAuthentication yes/PasswordAuthentication no/gi' "$f" 2>/dev/null || true
  # drop any extra Port lines outside our drop-in (incl. cloud-init)
  sed -i '/^[[:space:]]*Port[[:space:]]/d' "$f" 2>/dev/null || true
done
if [ -f /etc/ssh/sshd_config ]; then
  sed -i 's/^#\?PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config
  sed -i 's/^#\?PermitRootLogin.*/PermitRootLogin prohibit-password/' /etc/ssh/sshd_config
  sed -i 's/^Port /# Port /' /etc/ssh/sshd_config
fi
if command -v ufw >/dev/null 2>&1; then
  ufw allow %d/tcp comment netductor-ssh 2>/dev/null || true
  ufw delete allow 22/tcp 2>/dev/null || true
  ufw deny 22/tcp comment netductor-no-ssh22 2>/dev/null || true
fi
# purge hoster monitoring agents (zabbix)
for u in zabbix-agent zabbix-agentd zabbix-agent2; do
  systemctl disable --now "$u" 2>/dev/null || true
  systemctl mask "$u" 2>/dev/null || true
done
export DEBIAN_FRONTEND=noninteractive
apt-get remove -y --purge zabbix-agent zabbix-agent2 zabbix-release 2>/dev/null || true
if command -v ufw >/dev/null 2>&1; then
  ufw deny 10050/tcp 2>/dev/null || true
  ufw deny 10051/tcp 2>/dev/null || true
fi
pkill -f zabbix_agent 2>/dev/null || true
systemctl stop ssh.socket 2>/dev/null || true
systemctl disable ssh.socket 2>/dev/null || true
systemctl stop sshd.socket 2>/dev/null || true
systemctl disable sshd.socket 2>/dev/null || true
systemctl enable ssh.service 2>/dev/null || systemctl enable sshd.service 2>/dev/null || true
systemctl restart sshd 2>/dev/null || systemctl restart ssh 2>/dev/null || service ssh restart 2>/dev/null || true
`, DropInConf(), port)
}
