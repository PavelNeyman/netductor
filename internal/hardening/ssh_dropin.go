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
if ! flock -w 30 8; then echo "SSH prepare: permanent lock timeout" >&2; exit 1; fi

# Serialize the complete prepare setup with other prepares, and share the
# persistent backup lock with rollback/finalize/cancel. Never unlink this lock.
mkdir -p "$ND_BAK"
test ! -L "$ND_BAK"
chmod 700 "$ND_BAK"
LOCK="$ND_BAK/lock"
test ! -L "$LOCK"
exec 9>"$LOCK"
chmod 600 "$LOCK"
if ! flock -w 30 9; then echo "SSH prepare: transaction lock timeout" >&2; exit 1; fi

# A previous prepare may still own the rollback window even though its SSH
# script has exited. Do not replace its backup while timer/service is active.
timer_state=$(systemctl is-active "$ND_RB_TIMER" 2>/dev/null || true)
unit_state=$(systemctl is-active "$ND_RB_UNIT" 2>/dev/null || true)
if [ "$timer_state" != "inactive" ] || [ "$unit_state" != "inactive" ]; then
  echo "previous SSH rollback is not terminal (timer=$timer_state service=$unit_state) — refuse concurrent prepare" >&2
  exit 1
fi

# Legacy versions removed the rollback units after successful cancel. Accept
# that terminal state, but refuse an orphaned backup with installed units and
# no terminal marker; it may represent a failed rollback requiring recovery.
if [ -f "$ND_RB_SCRIPT" ] && [ ! -f "$ND_BAK/CANCELLED" ] && [ ! -f "$ND_BAK/ROLLED_BACK" ]; then
  if [ -e "/etc/systemd/system/$ND_RB_UNIT" ] || [ -e "/etc/systemd/system/$ND_RB_TIMER" ]; then
    echo "previous SSH rollback has no terminal marker — inspect/recover before preparing again" >&2
    exit 1
  fi
fi

# Previous transaction is terminal (or prepare never reached rollback script).
# Keep the lock inode; remove only transaction payload while holding the lock.
find "$ND_BAK" -mindepth 1 -maxdepth 1 ! -name lock -exec rm -rf -- {} +
rm -f "/etc/systemd/system/$ND_RB_UNIT" "/etc/systemd/system/$ND_RB_TIMER"
systemctl daemon-reload

# Serialize all firewall reads and writes across the snapshot/prepare window.
mkdir -p /run/lock
test ! -L /run/lock/netductor-firewall.lock
exec 7>/run/lock/netductor-firewall.lock
if ! flock -w 30 7; then echo "SSH prepare: firewall lock timeout" >&2; exit 1; fi

# --- Mandatory SSH backup ---
test -f /etc/ssh/sshd_config
cp -a /etc/ssh/sshd_config "$ND_BAK/sshd_config.bak"
if [ -d /etc/ssh/sshd_config.d ]; then
  cp -a /etc/ssh/sshd_config.d "$ND_BAK/sshd_config.d.bak"
else
  mkdir -p "$ND_BAK/sshd_config.d.bak"
fi
test -f "$ND_BAK/sshd_config.bak"

# Record whether the factory SSH listener exists before changing sshd.
# A previously hardened :52222-only host must not acquire a temporary :22 hole.
if command -v ss >/dev/null 2>&1; then
  if ! SS_LISTENERS=$(ss -H -ltn); then
    echo "cannot inspect TCP listeners before SSH hardening" >&2
    exit 1
  fi
  if printf '%%s\n' "$SS_LISTENERS" | awk '$4 ~ /:22$/ { found=1 } END { exit !found }'; then
    : > "$ND_BAK/factory-ssh-present"
  fi
else
  if [ ! -r /proc/net/tcp ]; then
    echo "cannot inspect TCP listeners before SSH hardening" >&2
    exit 1
  fi
  if awk 'FNR==1 { next } $4=="0A" { split($2,a,":"); if (toupper(a[2])=="0016") found=1 } END { exit !found }' /proc/net/tcp /proc/net/tcp6; then
    : > "$ND_BAK/factory-ssh-present"
  fi
fi

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
  HEAD=$(printf '%s\n' "$ST" | awk 'NR==1 { print; exit }')
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
    if [ -f "$ND_BAK/factory-ssh-present" ] && ! ufw status numbered | grep -Fq 'netductor-ssh-bootstrap'; then
      echo "active UFW is missing the baseline factory SSH transition rule; refusing unsafe rollback snapshot" >&2
      exit 1
    fi
    test -f "$ND_BAK/ufw/user.rules"
    # if host has user6.rules, snapshot must include it
    if [ -f /etc/ufw/user6.rules ]; then
      test -f "$ND_BAK/ufw/user6.rules"
    fi
  fi
  echo "$ST" > "$ND_BAK/ufw-status.txt"
fi

# Persist the management port for the independent rollback service. Older
# transactions without this file fall back to the established Netductor port.
printf '%%s\n' "$ND_PORT" > "$ND_BAK/management-port"
chmod 600 "$ND_BAK/management-port"

# --- Arm rollback BEFORE mutations ---
# Protocol: flock $B/lock; re-check CANCELLED under lock before each SSH mutation.
cat > "$ND_RB_SCRIPT" << 'NDRBSH'
#!/bin/bash
set -e
B=/var/lib/netductor/ssh-harden-backup
LOCK="$B/lock"
mkdir -p "$B"
exec 9>"$LOCK"
# Exclusive lock for entire critical section (cancel/finalize acquire same lock).
# A hung prepare/finalize must not suppress recovery forever: expose the
# management ports and fail this attempt; systemd retries until the lock clears.
if ! flock -w 60 9; then
  echo "nd-harden-rollback: transaction lock timeout; preserving emergency SSH ingress" >&2
  emergency_port=52222
  if [ -f "$B/management-port" ]; then
    candidate=$(cat "$B/management-port")
    case "$candidate" in ''|*[!0-9]*) candidate= ;; esac
    if [ -n "$candidate" ] && [ "$candidate" -ge 1 ] && [ "$candidate" -le 65535 ]; then
      emergency_port=$candidate
    fi
  fi
  if command -v ufw >/dev/null 2>&1; then
    ufw allow "$emergency_port"/tcp comment netductor-ssh-rollback-emergency || true
    ufw allow 22/tcp comment netductor-ssh-rollback-emergency || true
  elif command -v iptables >/dev/null 2>&1; then
    iptables -C INPUT -p tcp --dport "$emergency_port" -m comment --comment netductor-ssh-rollback-emergency -j ACCEPT 2>/dev/null || iptables -I INPUT 1 -p tcp --dport "$emergency_port" -m comment --comment netductor-ssh-rollback-emergency -j ACCEPT || true
    iptables -C INPUT -p tcp --dport 22 -m comment --comment netductor-ssh-rollback-emergency -j ACCEPT 2>/dev/null || iptables -I INPUT 1 -p tcp --dport 22 -m comment --comment netductor-ssh-rollback-emergency -j ACCEPT || true
    command -v netfilter-persistent >/dev/null 2>&1 && netfilter-persistent save || true
  fi
  touch "$B/ROLLBACK_FAILED"
  chmod 600 "$B/ROLLBACK_FAILED"
  exit 1
fi
mkdir -p /run/lock
test ! -L /run/lock/netductor-firewall.lock
exec 8>/run/lock/netductor-firewall.lock
# Recovery must not be skipped if a concurrent firewall apply holds the lock.
# Wait briefly, then prioritize restoring SSH/firewall over serialization.
if ! flock -w 15 8; then
  echo "nd-harden-rollback: firewall lock timeout; proceeding with emergency restore" >&2
fi
if [ -f "$B/CANCELLED" ]; then
  echo "nd-harden-rollback: CANCELLED under lock — no-op"
  exit 0
fi
# Once rollback starts, finalize/cancel must not overwrite or mask its result.
touch "$B/ROLLBACK_STARTED"
chmod 600 "$B/ROLLBACK_STARTED"

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

# Restore the exact UFW snapshot. Never mark rollback complete when
# restoring the firewall failed: the old sshd config may listen only on :22.
FW_RESTORE_OK=1
if [ ! -f "$B/ufw-policy" ]; then
  echo "nd-harden-rollback: UFW snapshot state is missing" >&2
  FW_RESTORE_OK=0
else
  pol=$(cat "$B/ufw-policy")
  case "$pol" in
    absent)
      # No UFW backend existed when the transaction was armed.
      ;;
    inactive)
      if command -v ufw >/dev/null 2>&1; then
        if [ -f "$B/ufw/user.rules" ] && ! cp -a "$B/ufw/user.rules" /etc/ufw/user.rules; then
          echo "nd-harden-rollback: failed to restore UFW IPv4 rules" >&2
          FW_RESTORE_OK=0
        fi
        if [ -f "$B/ufw/user6.rules" ] && ! cp -a "$B/ufw/user6.rules" /etc/ufw/user6.rules; then
          echo "nd-harden-rollback: failed to restore UFW IPv6 rules" >&2
          FW_RESTORE_OK=0
        fi
        if ! ufw --force disable; then
          echo "nd-harden-rollback: failed to restore inactive UFW state" >&2
          FW_RESTORE_OK=0
        fi
      fi
      ;;
    active)
      if ! command -v ufw >/dev/null 2>&1; then
        echo "nd-harden-rollback: UFW was active but its command is unavailable" >&2
        FW_RESTORE_OK=0
      else
        if [ ! -f "$B/ufw/user.rules" ]; then
          echo "nd-harden-rollback: UFW IPv4 snapshot is missing" >&2
          FW_RESTORE_OK=0
        elif ! cp -a "$B/ufw/user.rules" /etc/ufw/user.rules; then
          echo "nd-harden-rollback: failed to restore UFW IPv4 rules" >&2
          FW_RESTORE_OK=0
        fi
        if [ -f /etc/ufw/user6.rules ]; then
          if [ ! -f "$B/ufw/user6.rules" ] || ! cp -a "$B/ufw/user6.rules" /etc/ufw/user6.rules; then
            echo "nd-harden-rollback: failed to restore UFW IPv6 rules" >&2
            FW_RESTORE_OK=0
          fi
        elif [ -f "$B/ufw/user6.rules" ] && ! cp -a "$B/ufw/user6.rules" /etc/ufw/user6.rules; then
          echo "nd-harden-rollback: failed to restore UFW IPv6 rules" >&2
          FW_RESTORE_OK=0
        fi
        if ! ufw reload; then
          echo "nd-harden-rollback: failed to reload restored UFW policy" >&2
          FW_RESTORE_OK=0
        fi
      fi
      ;;
    *)
      echo "nd-harden-rollback: unknown UFW snapshot state: $pol" >&2
      FW_RESTORE_OK=0
      ;;
  esac
fi

# Remove only our tagged iptables emergency rules after the original
# firewall policy has been restored successfully. Keep them if restore failed.
if [ "$FW_RESTORE_OK" -eq 1 ] && command -v iptables >/dev/null 2>&1; then
  emergency_port=52222
  if [ -f "$B/management-port" ]; then
    candidate=$(cat "$B/management-port")
    case "$candidate" in ''|*[!0-9]*) candidate= ;; esac
    if [ -n "$candidate" ] && [ "$candidate" -ge 1 ] && [ "$candidate" -le 65535 ]; then emergency_port=$candidate; fi
  fi
  for port in "$emergency_port" 22; do
    while iptables -C INPUT -p tcp --dport "$port" -m comment --comment netductor-ssh-rollback-emergency -j ACCEPT 2>/dev/null; do
      if ! iptables -D INPUT -p tcp --dport "$port" -m comment --comment netductor-ssh-rollback-emergency -j ACCEPT; then
        FW_RESTORE_OK=0
        break
      fi
    done
  done
  if [ "$FW_RESTORE_OK" -eq 1 ]; then
    if command -v netfilter-persistent >/dev/null 2>&1; then
      netfilter-persistent save || FW_RESTORE_OK=0
    else
      FW_RESTORE_OK=0
    fi
  fi
fi

if [ "$FW_RESTORE_OK" -ne 1 ]; then
  # Best-effort emergency management access. This is intentionally an ALLOW
  # only when exact rollback failed; it is tagged for console-side cleanup.
  if command -v ufw >/dev/null 2>&1; then
    emergency_port=52222
    if [ -f "$B/management-port" ]; then
      candidate=$(cat "$B/management-port")
      case "$candidate" in ''|*[!0-9]*) candidate= ;; esac
      if [ -n "$candidate" ] && [ "$candidate" -ge 1 ] && [ "$candidate" -le 65535 ]; then
        emergency_port=$candidate
      fi
    fi
    if [ -f "$B/factory-ssh-present" ]; then
      emergency_port=22
    fi
    if ! ufw status numbered | grep -Fq 'netductor-ssh-rollback-emergency'; then
      if ufw insert 1 allow "$emergency_port"/tcp comment netductor-ssh-rollback-emergency; then
        ufw status numbered | grep -Fq 'netductor-ssh-rollback-emergency' || true
      fi
    fi
  fi
  touch "$B/ROLLBACK_FAILED"
  chmod 600 "$B/ROLLBACK_FAILED"
  echo "nd-harden-rollback: firewall restore incomplete; rollback remains failed and may be retried from console" >&2
  exit 1
fi

rm -f "$B/ROLLBACK_FAILED"
touch "$B/ROLLED_BACK"
chmod 600 "$B/ROLLED_BACK"
exit 0
NDRBSH
chmod 700 "$ND_RB_SCRIPT"
test -x "$ND_RB_SCRIPT"

cat > /etc/systemd/system/${ND_RB_UNIT} << NDRB
[Unit]
Description=Netductor SSH harden rollback
StartLimitIntervalSec=0
[Service]
Type=oneshot
ExecStart=$ND_RB_SCRIPT
TimeoutStartSec=90s
Restart=on-failure
RestartSec=15s
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
# Firewall lock was acquired before the snapshot and remains held.
# Do not install/switch firewall backends here. EnsureHostBaseline already
# selected and applied one backend; introducing UFW now could leave two
# independent policies active with different defaults.
if command -v ufw >/dev/null 2>&1; then
  ufw allow "$ND_PORT"/tcp comment netductor-ssh || true
  if [ -f "$ND_BAK/factory-ssh-present" ] && ! ufw status numbered | grep -Fq 'netductor-ssh-bootstrap'; then
    # UFW was inactive at snapshot time; install the transition exception
    # before enabling it. Rollback will restore the inactive snapshot.
    ufw insert 1 allow 22/tcp comment netductor-ssh-bootstrap || true
  fi
  ufw allow 443/tcp comment netductor-vless || true
  # WSS is opened later by service-plane bootstrap for the selected peer IP only.
  ufw --force enable
  ufw status | grep -qE "$ND_PORT/tcp"
elif command -v iptables >/dev/null 2>&1; then
  if [ -f "$ND_BAK/factory-ssh-present" ]; then
    iptables -C INPUT -p tcp --dport 22 -m comment --comment netductor-ssh-bootstrap -j ACCEPT
  fi
  iptables -C INPUT -p tcp --dport "$ND_PORT" -j ACCEPT
  command -v netfilter-persistent >/dev/null 2>&1
  netfilter-persistent save
else
  echo "no firewall backend after secondary baseline" >&2
  exit 1
fi

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
  systemctl start --no-block ${ND_RB_UNIT} || true
  exit 1
fi

echo "nd-harden-prepare: ok port=$ND_PORT rollback_timer=8m"
`, BackupDir, port, RollbackUnit, RollbackTimer, DropInConfPortOnly())
}

// RemoteHardenFinalizeScript: password off + deny :22 only.
func RemoteHardenFinalizeScript() string {
	port := DefaultSSHPort
	return fmt.Sprintf(`set -euo pipefail
ND_BAK='%s'
ND_PORT=%d
mkdir -p "$ND_BAK"
exec 7>"$ND_BAK/lock"
if ! flock -w 30 7; then echo "SSH finalize: transaction lock timeout" >&2; exit 1; fi
if [ -f "$ND_BAK/ROLLBACK_STARTED" ] || [ -f "$ND_BAK/ROLLED_BACK" ]; then
  echo "SSH rollback already started/completed; refusing to re-finalize" >&2
  exit 1
fi
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
mkdir -p /run/lock
test ! -L /run/lock/netductor-firewall.lock
exec 8>/run/lock/netductor-firewall.lock
if ! flock -w 30 8; then echo "SSH finalize: firewall lock timeout" >&2; exit 1; fi
if command -v ufw >/dev/null 2>&1; then
  # ufw exits 1 for both "already exists" and a successful delete. The exit
  # code is not the contract; the numbered status after the sequence is.
  ufw allow "$ND_PORT"/tcp comment netductor-ssh || true
  # pipefail + "| head" dies on SIGPIPE (exit 141) after ufw already succeeded.
  # Read the full status, then parse it. No early-close pipelines.
  ufw_status=$(ufw status numbered || true)
  first22=$(printf '%%s\n' "$ufw_status" | awk '/^[[:space:]]*\[[^]]+\][[:space:]]+22\/tcp([[:space:]]|$)/ { if (!seen++) print }')
  case "$first22" in
    *"DENY IN"*) ;;
    *) ufw insert 1 deny 22/tcp comment netductor-no-ssh22 || true ;;
  esac
  # Delete every bootstrap rule. Re-read after each delete; numbers shift.
  for _ in 1 2 3 4 5 6 7 8; do
    ufw_status=$(ufw status numbered || true)
    line=$(printf '%%s\n' "$ufw_status" | awk '/netductor-ssh-bootstrap/ { print; exit }')
    [ -n "$line" ] || break
    num=$(printf '%%s\n' "$line" | sed -nE 's/^[[:space:]]*\[[[:space:]]*([0-9]+)\].*/\1/p')
    [ -n "$num" ] || { echo "cannot parse bootstrap rule: $line" >&2; exit 1; }
    ufw --force delete "$num" || true
  done
  ufw_status=$(ufw status numbered || true)
  first22=$(printf '%%s\n' "$ufw_status" | awk '/^[[:space:]]*\[[^]]+\][[:space:]]+22\/tcp([[:space:]]|$)/ { if (!seen++) print }')
  case "$first22" in
    *"DENY IN"*) ;;
    *) echo "final SSH firewall verification failed: port 22 is not denied first: [$first22]" >&2; exit 1 ;;
  esac
elif command -v iptables >/dev/null 2>&1; then
  # Keep the temporary :22 INPUT exception while rollback is armed. The
  # rollback snapshot does not include kernel iptables state; cancel removes
  # this tagged exception only after post-finalize key-only access is proven.
  chain=$(iptables -S INPUT | awk '$1=="-A" && $2=="INPUT" && $3=="-j" && ($4=="NETDUCTOR_A" || $4=="NETDUCTOR_B") {print $4}')
  test "$(printf '%%s\n' "$chain" | sed '/^$/d' | wc -l)" -eq 1
  iptables -C "$chain" -p tcp --dport 22 -j DROP
  if [ -f "$ND_BAK/factory-ssh-present" ]; then
    iptables -C INPUT -p tcp --dport 22 -m comment --comment netductor-ssh-bootstrap -j ACCEPT
  fi
  netfilter-persistent save
fi
echo "nd-harden-finalize: ok (rollback still armed until cancel)"
`, BackupDir, port, DropInConf())
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
# Wait for rollback critical section, but fail boundedly rather than hang deploy.
if ! flock -w 60 9; then echo "SSH rollback cancel: transaction lock timeout" >&2; exit 1; fi
if [ -f "$ND_BAK/ROLLBACK_STARTED" ] || [ -f "$ND_BAK/ROLLED_BACK" ]; then
  echo "SSH rollback already started/completed; cannot cancel it" >&2
  exit 1
fi
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

# The rollback timer is now definitively stopped and post-finalize key-only SSH
# was verified by the caller. Only now close the temporary factory SSH path.
mkdir -p /run/lock
test ! -L /run/lock/netductor-firewall.lock
exec 8>/run/lock/netductor-firewall.lock
if ! flock -w 30 8; then echo "SSH rollback cancel: firewall lock timeout" >&2; exit 1; fi
if command -v ufw >/dev/null 2>&1; then
  if ufw status numbered | grep -Fq 'netductor-ssh-bootstrap'; then
    echo "UFW SSH bootstrap rule unexpectedly remains after finalize" >&2
    exit 1
  fi
  first22=$(ufw status numbered | grep -E '^[[:space:]]*\[[^]]+\][[:space:]]+22/tcp([[:space:]]|$)' | head -1)
  case "$first22" in
    *"DENY IN"*) ;;
    *) echo "UFW port 22 is not denied after rollback cancellation" >&2; exit 1 ;;
  esac
elif command -v iptables >/dev/null 2>&1; then
  while iptables -C INPUT -p tcp --dport 22 -m comment --comment netductor-ssh-bootstrap -j ACCEPT 2>/dev/null; do
    iptables -D INPUT -p tcp --dport 22 -m comment --comment netductor-ssh-bootstrap -j ACCEPT
  done
  netfilter-persistent save
  if iptables -C INPUT -p tcp --dport 22 -m comment --comment netductor-ssh-bootstrap -j ACCEPT 2>/dev/null; then
    echo "iptables SSH bootstrap exception remains after cancellation" >&2
    exit 1
  fi
  chain=$(iptables -S INPUT | awk '$1=="-A" && $2=="INPUT" && $3=="-j" && ($4=="NETDUCTOR_A" || $4=="NETDUCTOR_B") {print $4}')
  test "$(printf '%%s\n' "$chain" | sed '/^$/d' | wc -l)" -eq 1
  iptables -C "$chain" -p tcp --dport 22 -j DROP
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
  mkdir -p /run/lock
  test ! -L /run/lock/netductor-firewall.lock
  exec 8>/run/lock/netductor-firewall.lock
  if ! flock -w 30 8; then echo "host agent purge: firewall lock timeout" >&2; exit 1; fi
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
