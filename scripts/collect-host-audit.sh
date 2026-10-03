#!/usr/bin/env bash
# Full host audit text bundle for offline analysis (primary or secondary).
# Safe read-only. Does not print private keys or backup encryption keys.
# Usage: sudo bash collect-host-audit.sh | tee host-audit-$(hostname).txt
set -u
echo "=== netductor collect-host-audit ==="
echo "host=$(hostname) date=$(date -u +%Y-%m-%dT%H:%MZ)"
echo "role_hint=$(grep -E '^(ROLE|NETDUCTOR_ROLE)=' /etc/netductor/netductor.conf 2>/dev/null || echo unknown)"
echo

section() { echo; echo "##### $1 #####"; }

section "versions"
cat /etc/netductor/VERSION 2>/dev/null || true
command -v netductor >/dev/null && netductor version 2>/dev/null || true
command -v netductor-tg >/dev/null && netductor-tg version 2>/dev/null || true
uname -a
cat /etc/os-release 2>/dev/null | head -5

section "ss -tulnp"
ss -tulnp 2>/dev/null || netstat -tulnp 2>/dev/null || true

section "listening summary (non-localhost)"
ss -tuln 2>/dev/null | awk 'NR>1 && $5 !~ /127\.0\.0\.1/ && $5 !~ /\[::1\]/ {print}' || true

section "systemctl running services"
systemctl list-units --type=service --state=running --no-pager 2>/dev/null | head -200

section "systemctl enabled"
systemctl list-unit-files --state=enabled --no-pager 2>/dev/null | head -200

section "netductor units"
systemctl list-units --all --no-pager 2>/dev/null | grep -i netductor || true
systemctl list-units --all --no-pager 2>/dev/null | grep -iE 'sing-box|blocky|docker|zabbix|telegraf|salt|puppet|netdata|node_exporter' || true

section "hoster denylist probe"
if [[ -f /usr/local/share/netductor/audit-hoster-agents.sh ]]; then
  bash /usr/local/share/netductor/audit-hoster-agents.sh 2>/dev/null || true
elif [[ -f "$(dirname "$0")/audit-hoster-agents.sh" ]]; then
  bash "$(dirname "$0")/audit-hoster-agents.sh" 2>/dev/null || true
elif command -v netductor >/dev/null; then
  netductor host-audit 2>/dev/null || true
else
  echo "(audit-hoster-agents.sh not found; install from repo scripts/)"
fi

section "dpkg interesting"
dpkg -l 2>/dev/null | grep -iE 'zabbix|telegraf|netdata|node.exporter|salt|puppet|chef|datadog|newrelic|elastic|nrpe|collectd|avahi|cups|rpcbind|qemu-guest|open-vm|landscape|monit|wazuh|ossec|otel|cloudwatch|snmpd' || echo "(none matched)"

section "firewall"
if command -v ufw >/dev/null; then
  echo "ufw_bin=$(command -v ufw)"
  ufw status verbose 2>/dev/null || true
else
  echo "ufw: not installed"
fi
if command -v netductor >/dev/null; then
  netductor firewall status 2>/dev/null || true
fi
iptables-save 2>/dev/null | head -120 || true
nft list ruleset 2>/dev/null | head -80 || true

section "ssh"
sshd -T 2>/dev/null | grep -iE 'port |passwordauthentication|permitrootlogin|pubkeyauthentication' || true
ls -la /root/.ssh 2>/dev/null | sed 's/\(id_\|key\).*/[redacted name]/' || true
wc -l /root/.ssh/authorized_keys 2>/dev/null || true

section "cron / timers"
ls -la /etc/cron.* 2>/dev/null || true
systemctl list-timers --all --no-pager 2>/dev/null | head -40

section "users uid0"
awk -F: '($3==0){print $1}' /etc/passwd 2>/dev/null

section "opt / usr/local bins (names)"
ls /opt 2>/dev/null || true
ls /usr/local/bin 2>/dev/null | head -80

section "docker"
command -v docker >/dev/null && docker ps -a 2>/dev/null | head -30 || echo "no docker"

section "baseline present"
ls -la /var/lib/netductor/baseline 2>/dev/null || echo "(no baseline dir)"

section "end"
echo "collect-host-audit done"
exit 0
