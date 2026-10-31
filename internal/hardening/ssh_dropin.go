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
	return fmt.Sprintf(`set -euo pipefail
ND_BAK='%s'
ND_PORT=%d
ND_RB_UNIT='%s'
ND_RB_TIMER='%s'
ND_RB_SCRIPT="$ND_BAK/rollback.sh"
# Permanent lock outside removable backup dir (concurrent Prepare safety)
PREP_LOCK=/var/lib/netductor/ssh-harden.lock
mkdir -p /var/lib/netductor
exec 8>"$PREP_LOCK"
flock 8

# --- Stop any previous rollback before touching backup dir ---
timeout 10 systemctl stop "$ND_RB_TIMER" 2>/dev/null || systemctl kill "$ND_RB_TIMER" 2>/dev/null || true
timeout 10 systemctl stop "$ND_RB_UNIT" 2>/dev/null || systemctl kill "$ND_RB_UNIT" 2>/dev/null || true
systemctl disable "$ND_RB_TIMER" 2>/dev/null || true
# Wait for previous unit to leave active (bounded)
for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do
  if ! systemctl is-active "$ND_RB_UNIT" 2>/dev/null; then
    break
  fi
  sleep 1
done
if systemctl is-active "$ND_RB_UNIT" 2>/dev/null; then
  echo "previous rollback service still active — refuse prepare" >&2
  exit 1
fi
if systemctl is-active "$ND_RB_TIMER" 2>/dev/null; then
  echo "previous rollback timer still active — refuse prepare" >&2
  exit 1
fi
rm -f /etc/systemd/system/${ND_RB_UNIT} /etc/systemd/system/${ND_RB_TIMER}
systemctl daemon-reload 2>/dev/null || true

# Safe to recreate backup dir only after old rollback is gone
rm -rf "$ND_BAK"
mkdir -p "$ND_BAK"
LOCK="$ND_BAK/lock"
# rollback/cancel still share ND_BAK/lock for critical section
exec 9>"$LOCK"
flock 9

# --- Mandatory SSH backup ---
test -f /etc/ssh/sshd_config
cp -a /etc/ssh/sshd_config "$ND_BAK/sshd_config.bak"
if [ -d /etc/ssh/sshd_config.d ]; then
  cp -a /etc/ssh/sshd_config.d "$ND_BAK/sshd_config.d.bak"
else
  mkdir -p "$ND_BAK/sshd_config.d.bak"
fi
test -f "$ND_BAK/sshd_config.bak"

# --- Socket unit state (enabled / active / masked) ---
for u in ssh.socket sshd.socket; do
  if systemctl cat "$u" >/dev/null 2>&1; then
    set +e
    en_out=$(systemctl is-enabled "$u" 2>/dev/null)
    en_rc=$?
    ac_out=$(systemctl is-active "$u" 2>/dev/null)
    ac_rc=$?
    set -e
    # Prefer stdout even when exit code is non-zero (e.g. masked)
    if [ -n "$en_out" ]; then
      echo "$en_out" > "$ND_BAK/${u}.enabled"
    elif [ $en_rc -ne 0 ]; then
      echo "disabled" > "$ND_BAK/${u}.enabled"
    else
      echo "unknown" > "$ND_BAK/${u}.enabled"
    fi
    if [ -n "$ac_out" ]; then
      echo "$ac_out" > "$ND_BAK/${u}.active"
    else
      echo "inactive" > "$ND_BAK/${u}.active"
    fi
  else
    echo absent > "$ND_BAK/${u}.enabled"
    echo absent > "$ND_BAK/${u}.active"
  fi
done

# --- UFW policy snapshot (validated) ---
if ! command -v ufw >/dev/null 2>&1; then
  echo absent > "$ND_BAK/ufw-policy"
else
  if ! ST=$(ufw status 2>&1); then
    echo "ufw status failed: $ST" >&2
    exit 1
  fi
  HEAD=$(echo "$ST" | head -1)
  if echo "$HEAD" | grep -qi inactive; then
    echo inactive > "$ND_BAK/ufw-policy"
  elif echo "$HEAD" | grep -qi active; then
    echo active > "$ND_BAK/ufw-policy"
  else
    echo "cannot determine ufw status: $HEAD" >&2
    exit 1
  fi
  mkdir -p "$ND_BAK/ufw"
  if [ -f /etc/ufw/user.rules ]; then
    cp -a /etc/ufw/user.rules "$ND_BAK/ufw/user.rules"
  fi
  if [ -f /etc/ufw/user6.rules ]; then
    cp -a /etc/ufw/user6.rules "$ND_BAK/ufw/user6.rules"
  fi
  if grep -qx active "$ND_BAK/ufw-policy"; then
    test -f "$ND_BAK/ufw/user.rules"
    # if host has user6.rules, snapshot must include it
    if [ -f /etc/ufw/user6.rules ]; then
      test -f "$ND_BAK/ufw/user6.rules"
    fi
  fi
  echo "$ST" > "$ND_BAK/ufw-status.txt"
fi

# --- Arm rollback BEFORE mutations ---
# Protocol: flock $B/lock; re-check CANCELLED under lock before each SSH mutation.
cat > "$ND_RB_SCRIPT" << 'NDRBSH'
#!/bin/bash
set -e
B=/var/lib/netductor/ssh-harden-backup
LOCK="$B/lock"
mkdir -p "$B"
exec 9>"$LOCK"
# Exclusive lock for entire critical section (cancel acquires same lock)
flock 9
if [ -f "$B/CANCELLED" ]; then
  echo "nd-harden-rollback: CANCELLED under lock — no-op"
  exit 0
fi

# SSH restore under lock
test -f "$B/sshd_config.bak"
cp -a "$B/sshd_config.bak" /etc/ssh/sshd_config
if [ -d "$B/sshd_config.d.bak" ]; then
  rm -rf /etc/ssh/sshd_config.d
  cp -a "$B/sshd_config.d.bak" /etc/ssh/sshd_config.d
fi
if [ -f "$B/CANCELLED" ]; then
  echo "nd-harden-rollback: CANCELLED after config write — stop before restart"
  exit 0
fi
if command -v sshd >/dev/null 2>&1; then
  if ! sshd -t; then
    echo "nd-harden-rollback: sshd -t failed after restore" >&2
    exit 1
  fi
fi
if [ -f "$B/CANCELLED" ]; then
  exit 0
fi
systemctl restart sshd 2>/dev/null || systemctl restart ssh

# Sockets (best-effort but covers masked/disabled/enabled)
for u in ssh.socket sshd.socket; do
  enf="$B/${u}.enabled"
  [ -f "$enf" ] || continue
  en=$(cat "$enf" 2>/dev/null || true)
  case "$en" in
    absent) continue ;;
    masked)
      systemctl mask "$u" 2>/dev/null || true
      ;;
    enabled)
      systemctl unmask "$u" 2>/dev/null || true
      systemctl enable "$u" 2>/dev/null || true
      ;;
    disabled|static|indirect|alias|*)
      systemctl unmask "$u" 2>/dev/null || true
      systemctl disable "$u" 2>/dev/null || true
      ;;
  esac
  acf="$B/${u}.active"
  ac=$(cat "$acf" 2>/dev/null || true)
  case "$ac" in
    active) systemctl start "$u" 2>/dev/null || true ;;
    inactive|failed|absent) systemctl stop "$u" 2>/dev/null || true ;;
  esac
done

# UFW: restore files only (no heuristic allow 22). Failure does not undo SSH.
set +e
if [ -f "$B/ufw-policy" ] && command -v ufw >/dev/null 2>&1; then
  pol=$(cat "$B/ufw-policy")
  case "$pol" in
    absent|inactive)
      [ -f "$B/ufw/user.rules" ] && cp -a "$B/ufw/user.rules" /etc/ufw/user.rules
      [ -f "$B/ufw/user6.rules" ] && cp -a "$B/ufw/user6.rules" /etc/ufw/user6.rules
      ufw --force disable
      ;;
    active)
      [ -f "$B/ufw/user.rules" ] && cp -a "$B/ufw/user.rules" /etc/ufw/user.rules
      [ -f "$B/ufw/user6.rules" ] && cp -a "$B/ufw/user6.rules" /etc/ufw/user6.rules
      ufw reload
      ;;
  esac
fi
exit 0
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
  systemctl start ${ND_RB_UNIT} || true
  exit 1
fi

echo "nd-harden-prepare: ok port=$ND_PORT rollback_timer=8m"
`, BackupDir, port, RollbackUnit, RollbackTimer, DropInConfPortOnly())
}

// RemoteHardenFinalizeScript: password off + deny :22 only.
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

// RemoteHardenCancelRollbackScript: flock + CANCELLED + stop timer/unit + verify.
func RemoteHardenCancelRollbackScript() string {
	return fmt.Sprintf(`set -euo pipefail
ND_BAK='%s'
ND_RB_UNIT='%s'
ND_RB_TIMER='%s'
LOCK="$ND_BAK/lock"
mkdir -p "$ND_BAK"
exec 9>"$LOCK"
# Block until rollback critical section finishes (or free)
flock 9
touch "$ND_BAK/CANCELLED"
chmod 600 "$ND_BAK/CANCELLED"

# Bounded stop (systemctl stop itself can hang)
timeout 10 systemctl stop "$ND_RB_TIMER" 2>/dev/null || systemctl kill "$ND_RB_TIMER" 2>/dev/null || true
timeout 10 systemctl stop "$ND_RB_UNIT" 2>/dev/null || systemctl kill "$ND_RB_UNIT" 2>/dev/null || true
systemctl disable "$ND_RB_TIMER" 2>/dev/null || true

for i in 1 2 3 4 5; do
  if ! systemctl is-active "$ND_RB_UNIT" 2>/dev/null && ! systemctl is-active "$ND_RB_TIMER" 2>/dev/null; then
    break
  fi
  sleep 1
done

if systemctl is-active "$ND_RB_TIMER" 2>/dev/null; then
  echo "rollback timer still active after cancel"
  exit 1
fi
if systemctl is-active "$ND_RB_UNIT" 2>/dev/null; then
  echo "rollback service still active after cancel"
  exit 1
fi

rm -f /etc/systemd/system/${ND_RB_UNIT} /etc/systemd/system/${ND_RB_TIMER}
systemctl daemon-reload
echo "nd-harden-rollback-cancel: ok"
`, BackupDir, RollbackUnit, RollbackTimer)
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

// RemoteHardenScript disabled — would skip independent key-only checks.
func RemoteHardenScript() string {
	return `echo "RemoteHardenScript disabled: use phased HardenSSHPrepare/Finalize/CancelRollback" >&2; exit 1`
}
