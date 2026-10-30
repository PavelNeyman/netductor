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
// until finalize.
func DropInConfPortOnly() string {
	return fmt.Sprintf(`Port %d
PubkeyAuthentication yes
`, DefaultSSHPort)
}

const RollbackUnit = "netductor-ssh-harden-rollback.service"
const RollbackTimer = "netductor-ssh-harden-rollback.timer"
const BackupDir = "/var/lib/netductor/ssh-harden-backup"

// RemoteHardenPrepareScript: mandatory backup (SSH + UFW), open new port, write Port drop-in,
// sshd -t, arm independent rollback timer+unit, restart sshd (errors not swallowed).
// Does NOT close :22 or disable password.
func RemoteHardenPrepareScript() string {
	port := DefaultSSHPort
	return fmt.Sprintf(`set -euo pipefail
ND_BAK='%s'
ND_PORT=%d
ND_RB_UNIT='%s'
ND_RB_TIMER='%s'

# Fresh backup directory — never reuse stale snapshot
rm -rf "$ND_BAK"
mkdir -p "$ND_BAK"

# Mandatory SSH backup
test -f /etc/ssh/sshd_config
cp -a /etc/ssh/sshd_config "$ND_BAK/sshd_config.bak"
if [ -d /etc/ssh/sshd_config.d ]; then
  cp -a /etc/ssh/sshd_config.d "$ND_BAK/sshd_config.d.bak"
else
  mkdir -p "$ND_BAK/sshd_config.d.bak"
fi
test -f "$ND_BAK/sshd_config.bak"

# Mandatory UFW snapshot when ufw is present (status + rules.d)
if command -v ufw >/dev/null 2>&1; then
  ufw status numbered > "$ND_BAK/ufw-status.txt" 2>/dev/null || true
  if [ -d /etc/ufw ]; then
    cp -a /etc/ufw "$ND_BAK/ufw" 2>/dev/null || true
  fi
  # Record whether 22 was allowed before we touch rules
  if ufw status | grep -qE '22/tcp.*ALLOW'; then
    echo allowed > "$ND_BAK/ufw-22-state"
  else
    echo not-allowed > "$ND_BAK/ufw-22-state"
  fi
fi

# Allow new port before restart; keep 22 open during transition
export DEBIAN_FRONTEND=noninteractive
if ! command -v ufw >/dev/null 2>&1; then
  apt-get install -y ufw
fi
command -v ufw >/dev/null 2>&1
ufw allow "$ND_PORT"/tcp comment netductor-ssh
ufw allow 22/tcp comment netductor-ssh-bootstrap
ufw allow 443/tcp comment netductor-vless || true
ufw allow 8445/tcp comment nd-wss-ps || true
echo y | ufw --force enable
ufw status | grep -qE "$ND_PORT/tcp"

mkdir -p /etc/ssh/sshd_config.d
cat > /etc/ssh/sshd_config.d/00-netductor-harden.conf << 'NDSSH'
%sNDSSH
if [ -f /etc/ssh/sshd_config ]; then
  sed -i 's/^Port /# Port /' /etc/ssh/sshd_config
fi
for f in /etc/ssh/sshd_config.d/*.conf; do
  [ -f "$f" ] || continue
  case "$f" in *00-netductor-harden.conf) continue ;; esac
  sed -i '/^[[:space:]]*Port[[:space:]]/d' "$f" 2>/dev/null || true
done

command -v sshd >/dev/null 2>&1
sshd -t

# Independent rollback: restore SSH + UFW and restart sshd (fail-closed on sshd -t)
cat > /etc/systemd/system/${ND_RB_UNIT} << NDRB
[Unit]
Description=Netductor SSH harden rollback
[Service]
Type=oneshot
ExecStart=/bin/bash -c 'set -e; B=%s; \
  if [ -f "$$B/sshd_config.bak" ]; then cp -a "$$B/sshd_config.bak" /etc/ssh/sshd_config; fi; \
  if [ -d "$$B/sshd_config.d.bak" ]; then rm -rf /etc/ssh/sshd_config.d; cp -a "$$B/sshd_config.d.bak" /etc/ssh/sshd_config.d; fi; \
  if [ -d "$$B/ufw" ] && [ -d /etc/ufw ]; then \
    cp -a "$$B/ufw/user.rules" /etc/ufw/user.rules 2>/dev/null || true; \
    cp -a "$$B/ufw/user6.rules" /etc/ufw/user6.rules 2>/dev/null || true; \
    ufw reload 2>/dev/null || true; \
  fi; \
  if [ -f "$$B/ufw-22-state" ] && grep -q allowed "$$B/ufw-22-state"; then ufw allow 22/tcp comment netductor-ssh-bootstrap 2>/dev/null || true; fi; \
  if command -v sshd >/dev/null 2>&1; then sshd -t; fi; \
  systemctl restart sshd 2>/dev/null || systemctl restart ssh'
NDRB

cat > /etc/systemd/system/${ND_RB_TIMER} << NDRT
[Unit]
Description=Timer for Netductor SSH harden rollback
[Timer]
OnActiveSec=8min
Unit=${ND_RB_UNIT}
[Install]
WantedBy=timers.target
NDRT

systemctl daemon-reload
systemctl reset-failed ${ND_RB_UNIT} 2>/dev/null || true
systemctl start ${ND_RB_TIMER}
systemctl is-active ${ND_RB_TIMER}

systemctl stop ssh.socket 2>/dev/null || true
systemctl disable ssh.socket 2>/dev/null || true
systemctl stop sshd.socket 2>/dev/null || true
systemctl disable sshd.socket 2>/dev/null || true
systemctl enable ssh.service 2>/dev/null || systemctl enable sshd.service

if systemctl restart sshd; then
  :
elif systemctl restart ssh; then
  :
else
  echo "sshd restart failed"
  exit 1
fi

echo "nd-harden-prepare: ok port=$ND_PORT rollback_timer=8m"
`, BackupDir, port, RollbackUnit, RollbackTimer, DropInConfPortOnly(), BackupDir)
}

// RemoteHardenFinalizeScript: password off, deny :22 — does NOT cancel rollback timer.
// Caller must cancel after independent key-only success (RemoteHardenCancelRollbackScript).
func RemoteHardenFinalizeScript() string {
	port := DefaultSSHPort
	return fmt.Sprintf(`set -euo pipefail
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
command -v sshd >/dev/null 2>&1
sshd -t
if systemctl restart sshd; then
  :
elif systemctl restart ssh; then
  :
else
  echo "sshd restart failed after finalize"
  exit 1
fi
# Firewall: deny 22 only after successful restart (rollback timer still armed until cancel)
if command -v ufw >/dev/null 2>&1; then
  ufw allow "$ND_PORT"/tcp comment netductor-ssh
  ufw delete allow 22/tcp 2>/dev/null || true
  ufw deny 22/tcp comment netductor-no-ssh22
  ufw deny 10050/tcp 2>/dev/null || true
  ufw deny 10051/tcp 2>/dev/null || true
  ufw deny 8787/tcp 2>/dev/null || true
  ufw deny 8788/tcp 2>/dev/null || true
  ufw deny 8789/tcp 2>/dev/null || true
fi
# Monitoring units — no broad process kill
for u in zabbix-agent zabbix-agentd zabbix-agent2 telegraf salt-minion node_exporter prometheus-node-exporter datadog-agent snmpd nrpe; do
  systemctl disable --now "$u" 2>/dev/null || true
  systemctl mask "$u" 2>/dev/null || true
done
export DEBIAN_FRONTEND=noninteractive
apt-get remove -y --purge zabbix-agent zabbix-agent2 zabbix-release telegraf salt-minion prometheus-node-exporter snmpd 2>/dev/null || true
echo "nd-harden-finalize: ok (rollback still armed until cancel)"
`, port, DropInConf())
}

// RemoteHardenCancelRollbackScript stops and removes the rollback timer/unit.
// Call only after post-finalize independent key-only success.
func RemoteHardenCancelRollbackScript() string {
	return fmt.Sprintf(`set -euo pipefail
systemctl stop %s 2>/dev/null || true
systemctl disable %s 2>/dev/null || true
rm -f /etc/systemd/system/%s /etc/systemd/system/%s
systemctl daemon-reload
echo "nd-harden-rollback-cancel: ok"
`, RollbackTimer, RollbackTimer, RollbackUnit, RollbackTimer)
}

// RemoteHardenScript is the legacy one-shot path (not recommended for production).
// Prefer Prepare → verify → Finalize → verify → CancelRollback.
func RemoteHardenScript() string {
	return RemoteHardenPrepareScript() + "\n" + RemoteHardenFinalizeScript() + "\n" + RemoteHardenCancelRollbackScript()
}
