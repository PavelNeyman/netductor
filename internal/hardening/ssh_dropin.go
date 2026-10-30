package hardening

import "fmt"

// DropInConf is the final sshd drop-in (key-only on DefaultSSHPort).
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

// DropInConfPortOnly moves SSH to DefaultSSHPort; password left to distro until finalize.
func DropInConfPortOnly() string {
	return fmt.Sprintf(`Port %d
PubkeyAuthentication yes
`, DefaultSSHPort)
}

const RollbackUnit = "netductor-ssh-harden-rollback.service"
const RollbackTimer = "netductor-ssh-harden-rollback.timer"
const BackupDir = "/var/lib/netductor/ssh-harden-backup"

// RemoteHardenPrepareScript: backup → arm rollback → only then mutate firewall/SSH.
func RemoteHardenPrepareScript() string {
	port := DefaultSSHPort
	// Rollback script written on remote as file in systemd ExecStart single-quoted bash -c is hard;
	// write rollback script to disk first, then unit points at it.
	return fmt.Sprintf(`set -euo pipefail
ND_BAK='%s'
ND_PORT=%d
ND_RB_UNIT='%s'
ND_RB_TIMER='%s'
ND_RB_SCRIPT="$ND_BAK/rollback.sh"

rm -rf "$ND_BAK"
mkdir -p "$ND_BAK"

# --- Mandatory SSH backup (before any mutation) ---
test -f /etc/ssh/sshd_config
cp -a /etc/ssh/sshd_config "$ND_BAK/sshd_config.bak"
if [ -d /etc/ssh/sshd_config.d ]; then
  cp -a /etc/ssh/sshd_config.d "$ND_BAK/sshd_config.d.bak"
else
  mkdir -p "$ND_BAK/sshd_config.d.bak"
fi
test -f "$ND_BAK/sshd_config.bak"

# --- UFW policy snapshot ---
if ! command -v ufw >/dev/null 2>&1; then
  echo absent > "$ND_BAK/ufw-policy"
else
  if ufw status 2>/dev/null | head -1 | grep -qi inactive; then
    echo inactive > "$ND_BAK/ufw-policy"
  else
    echo active > "$ND_BAK/ufw-policy"
  fi
  mkdir -p "$ND_BAK/ufw"
  if [ -f /etc/ufw/user.rules ]; then cp -a /etc/ufw/user.rules "$ND_BAK/ufw/user.rules"; fi
  if [ -f /etc/ufw/user6.rules ]; then cp -a /etc/ufw/user6.rules "$ND_BAK/ufw/user6.rules"; fi
  ufw status numbered > "$ND_BAK/ufw-status.txt" 2>/dev/null || true
  if ufw status 2>/dev/null | grep -E '^[[:space:]]*22/tcp' | grep -qi allow; then
    echo allowed > "$ND_BAK/ufw-22-state"
  else
    echo not-allowed > "$ND_BAK/ufw-22-state"
  fi
fi

# --- Arm rollback BEFORE mutations (script + systemd) ---
cat > "$ND_RB_SCRIPT" << 'NDRBSH'
#!/bin/bash
set -e
B=/var/lib/netductor/ssh-harden-backup
test -f "$B/sshd_config.bak"
cp -a "$B/sshd_config.bak" /etc/ssh/sshd_config
if [ -d "$B/sshd_config.d.bak" ]; then
  rm -rf /etc/ssh/sshd_config.d
  cp -a "$B/sshd_config.d.bak" /etc/ssh/sshd_config.d
fi
if [ -f "$B/ufw-policy" ]; then
  pol=$(cat "$B/ufw-policy")
  case "$pol" in
    absent)
      if command -v ufw >/dev/null 2>&1; then ufw --force disable || true; fi
      ;;
    inactive)
      if command -v ufw >/dev/null 2>&1; then
        [ -f "$B/ufw/user.rules" ] && cp -a "$B/ufw/user.rules" /etc/ufw/user.rules
        [ -f "$B/ufw/user6.rules" ] && cp -a "$B/ufw/user6.rules" /etc/ufw/user6.rules
        ufw --force disable || true
      fi
      ;;
    active)
      if command -v ufw >/dev/null 2>&1; then
        [ -f "$B/ufw/user.rules" ] && cp -a "$B/ufw/user.rules" /etc/ufw/user.rules
        [ -f "$B/ufw/user6.rules" ] && cp -a "$B/ufw/user6.rules" /etc/ufw/user6.rules
        ufw reload
        if [ -f "$B/ufw-22-state" ] && grep -qx allowed "$B/ufw-22-state"; then
          ufw allow 22/tcp comment netductor-ssh-bootstrap || true
        fi
      fi
      ;;
  esac
fi
command -v sshd >/dev/null 2>&1
sshd -t
systemctl restart sshd 2>/dev/null || systemctl restart ssh
NDRBSH
chmod 700 "$ND_RB_SCRIPT"
test -x "$ND_RB_SCRIPT"

cat > /etc/systemd/system/${ND_RB_UNIT} << NDRB
[Unit]
Description=Netductor SSH harden rollback
[Service]
Type=oneshot
ExecStart=$ND_RB_SCRIPT
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

# --- Mutations only after rollback is armed ---
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
  # trigger rollback immediately
  systemctl start ${ND_RB_UNIT} || true
  exit 1
fi

echo "nd-harden-prepare: ok port=$ND_PORT rollback_timer=8m"
`, BackupDir, port, RollbackUnit, RollbackTimer, DropInConfPortOnly())
}

// RemoteHardenFinalizeScript: password off + deny :22 only. No package purge.
// Does not cancel rollback timer.
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
if command -v ufw >/dev/null 2>&1; then
  ufw allow "$ND_PORT"/tcp comment netductor-ssh
  ufw delete allow 22/tcp 2>/dev/null || true
  ufw deny 22/tcp comment netductor-no-ssh22
fi
echo "nd-harden-finalize: ok (rollback still armed until cancel)"
`, port, DropInConf())
}

// RemoteHardenCancelRollbackScript verifies timer is stopped.
func RemoteHardenCancelRollbackScript() string {
	return fmt.Sprintf(`set -euo pipefail
systemctl stop %s
systemctl disable %s 2>/dev/null || true
rm -f /etc/systemd/system/%s /etc/systemd/system/%s
systemctl daemon-reload
# Must not remain active
if systemctl is-active %s 2>/dev/null; then
  echo "rollback timer still active after cancel"
  exit 1
fi
echo "nd-harden-rollback-cancel: ok"
`, RollbackTimer, RollbackTimer, RollbackUnit, RollbackTimer, RollbackTimer)
}

// RemoteHostAgentPurgeScript runs after SSH access is proven and rollback cancelled.
func RemoteHostAgentPurgeScript() string {
	return `set -euo pipefail
for u in zabbix-agent zabbix-agentd zabbix-agent2 telegraf salt-minion node_exporter prometheus-node-exporter datadog-agent snmpd nrpe; do
  systemctl disable --now "$u" 2>/dev/null || true
  systemctl mask "$u" 2>/dev/null || true
done
export DEBIAN_FRONTEND=noninteractive
apt-get remove -y --purge zabbix-agent zabbix-agent2 zabbix-release telegraf salt-minion prometheus-node-exporter snmpd 2>/dev/null || true
if command -v ufw >/dev/null 2>&1; then
  ufw deny 10050/tcp 2>/dev/null || true
  ufw deny 10051/tcp 2>/dev/null || true
  ufw deny 8787/tcp 2>/dev/null || true
  ufw deny 8788/tcp 2>/dev/null || true
  ufw deny 8789/tcp 2>/dev/null || true
fi
echo "nd-host-agent-purge: ok"
`
}

// RemoteHardenScript legacy one-shot (prepare+finalize+cancel+purge). Prefer phased API.
func RemoteHardenScript() string {
	return RemoteHardenPrepareScript() + "\n" + RemoteHardenFinalizeScript() + "\n" +
		RemoteHardenCancelRollbackScript() + "\n" + RemoteHostAgentPurgeScript()
}
