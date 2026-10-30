package hardening

import "fmt"

// DropInConf is the single source of truth for final sshd netductor harden drop-in
// (key-only on DefaultSSHPort). Used by finalize after new-port verification.
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

// DropInConfPortOnly moves SSH to DefaultSSHPort but leaves password auth alone
// (still governed by distro defaults / previous config) until finalize.
func DropInConfPortOnly() string {
	return fmt.Sprintf(`Port %d
PubkeyAuthentication yes
`, DefaultSSHPort)
}

const rollbackUnit = "netductor-ssh-harden-rollback.service"
const rollbackTimer = "netductor-ssh-harden-rollback.timer"
const backupDir = "/var/lib/netductor/ssh-harden-backup"

// RemoteHardenPrepareScript: backup, open new port, write Port drop-in, sshd -t,
// arm independent rollback timer, restart sshd — does NOT close :22 or disable password.
func RemoteHardenPrepareScript() string {
	port := DefaultSSHPort
	return fmt.Sprintf(`set -e
ND_BAK='%s'
ND_PORT=%d
mkdir -p "$ND_BAK"
# Backup current ssh config (best-effort full tree)
cp -a /etc/ssh/sshd_config "$ND_BAK/sshd_config.bak" 2>/dev/null || true
cp -a /etc/ssh/sshd_config.d "$ND_BAK/sshd_config.d.bak" 2>/dev/null || true
# Allow new port before restart; keep 22 open
export DEBIAN_FRONTEND=noninteractive
apt-get install -y ufw 2>/dev/null || true
if command -v ufw >/dev/null 2>&1; then
  ufw allow "$ND_PORT"/tcp comment netductor-ssh 2>/dev/null || true
  ufw allow 22/tcp comment netductor-ssh-bootstrap 2>/dev/null || true
  ufw allow 443/tcp comment netductor-vless 2>/dev/null || true
  ufw allow 8445/tcp comment nd-wss-ps 2>/dev/null || true
  echo y | ufw --force enable 2>/dev/null || true
fi
mkdir -p /etc/ssh/sshd_config.d
cat > /etc/ssh/sshd_config.d/00-netductor-harden.conf << 'NDSSH'
%sNDSSH
# Comment Port lines in main config so drop-in wins
if [ -f /etc/ssh/sshd_config ]; then
  sed -i 's/^Port /# Port /' /etc/ssh/sshd_config
fi
for f in /etc/ssh/sshd_config.d/*.conf; do
  [ -f "$f" ] || continue
  case "$f" in *00-netductor-harden.conf) continue ;; esac
  sed -i '/^[[:space:]]*Port[[:space:]]/d' "$f" 2>/dev/null || true
done
if command -v sshd >/dev/null 2>&1; then
  sshd -t || { echo "sshd -t failed — refusing restart"; exit 1; }
fi
# Independent rollback: restore backup and restart ssh if not cancelled within 8 minutes
cat > /etc/systemd/system/%s << 'NDRB'
[Unit]
Description=Netductor SSH harden rollback (auto if new port not confirmed)
[Service]
Type=oneshot
ExecStart=/bin/bash -c 'set -e; B=%s; if [ -f "$B/sshd_config.bak" ]; then cp -a "$B/sshd_config.bak" /etc/ssh/sshd_config; fi; if [ -d "$B/sshd_config.d.bak" ]; then rm -rf /etc/ssh/sshd_config.d; cp -a "$B/sshd_config.d.bak" /etc/ssh/sshd_config.d; fi; if command -v sshd >/dev/null 2>&1; then sshd -t || true; fi; systemctl restart sshd 2>/dev/null || systemctl restart ssh 2>/dev/null || true'
NDRB
cat > /etc/systemd/system/%s << 'NDRT'
[Unit]
Description=Timer for Netductor SSH harden rollback
[Timer]
OnActiveSec=8min
Unit=%s
[Install]
WantedBy=timers.target
NDRT
systemctl daemon-reload
systemctl reset-failed %s 2>/dev/null || true
systemctl start %s
systemctl stop ssh.socket 2>/dev/null || true
systemctl disable ssh.socket 2>/dev/null || true
systemctl stop sshd.socket 2>/dev/null || true
systemctl disable sshd.socket 2>/dev/null || true
systemctl enable ssh.service 2>/dev/null || systemctl enable sshd.service 2>/dev/null || true
systemctl restart sshd 2>/dev/null || systemctl restart ssh 2>/dev/null || service ssh restart 2>/dev/null || true
echo "nd-harden-prepare: ok port=$ND_PORT rollback_timer=8m"
`, backupDir, port, DropInConfPortOnly(), rollbackUnit, backupDir, rollbackTimer, rollbackUnit, rollbackUnit, rollbackTimer)
}

// RemoteHardenFinalizeScript: password off, deny :22, cancel rollback timer.
// Call only after operator verified key-only on DefaultSSHPort.
func RemoteHardenFinalizeScript() string {
	port := DefaultSSHPort
	return fmt.Sprintf(`set -e
ND_PORT=%d
mkdir -p /etc/ssh/sshd_config.d
cat > /etc/ssh/sshd_config.d/00-netductor-harden.conf << 'NDSSH'
%sNDSSH
for f in /etc/ssh/sshd_config.d/*.conf; do
  [ -f "$f" ] || continue
  case "$f" in *00-netductor-harden.conf) continue ;; esac
  sed -i 's/PasswordAuthentication yes/PasswordAuthentication no/gi' "$f" 2>/dev/null || true
done
if [ -f /etc/ssh/sshd_config ]; then
  sed -i 's/^#\?PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config
  sed -i 's/^#\?PermitRootLogin.*/PermitRootLogin prohibit-password/' /etc/ssh/sshd_config
fi
if command -v sshd >/dev/null 2>&1; then
  sshd -t || { echo "sshd -t failed — refusing finalize"; exit 1; }
fi
systemctl restart sshd 2>/dev/null || systemctl restart ssh 2>/dev/null || true
# Cancel independent rollback
systemctl stop %s 2>/dev/null || true
systemctl disable %s 2>/dev/null || true
rm -f /etc/systemd/system/%s /etc/systemd/system/%s
systemctl daemon-reload 2>/dev/null || true
# Host monitoring purge (units only — no broad process kill)
for u in zabbix-agent zabbix-agentd zabbix-agent2 telegraf salt-minion node_exporter prometheus-node-exporter datadog-agent snmpd nrpe; do
  systemctl disable --now "$u" 2>/dev/null || true
  systemctl mask "$u" 2>/dev/null || true
done
export DEBIAN_FRONTEND=noninteractive
apt-get remove -y --purge zabbix-agent zabbix-agent2 zabbix-release telegraf salt-minion prometheus-node-exporter snmpd 2>/dev/null || true
if command -v ufw >/dev/null 2>&1; then
  ufw allow "$ND_PORT"/tcp comment netductor-ssh 2>/dev/null || true
  ufw delete allow 22/tcp 2>/dev/null || true
  ufw deny 22/tcp comment netductor-no-ssh22 2>/dev/null || true
  ufw deny 10050/tcp 2>/dev/null || true
  ufw deny 10051/tcp 2>/dev/null || true
  ufw deny 8787/tcp 2>/dev/null || true
  ufw deny 8788/tcp 2>/dev/null || true
  ufw deny 8789/tcp 2>/dev/null || true
fi
echo "nd-harden-finalize: ok"
`, port, DropInConf(), rollbackTimer, rollbackTimer, rollbackUnit, rollbackTimer)
}

// RemoteHardenScript is the legacy one-shot path (prepare+finalize without operator mid-check).
// Prefer Prepare + operator verify + Finalize from provision.
func RemoteHardenScript() string {
	// Keep a combined script for callers that have not been migrated: prepare then finalize
	// without cancel window (still has sshd -t and no pkill).
	return RemoteHardenPrepareScript() + "\n" + RemoteHardenFinalizeScript()
}
