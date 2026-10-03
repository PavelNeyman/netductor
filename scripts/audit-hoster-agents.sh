#!/usr/bin/env bash
# Audit VPS for hoster monitoring / remote config-management agents.
# Safe read-only by default. --purge attempts removal (needs root).
# Usage:
#   sudo bash audit-hoster-agents.sh
#   sudo bash audit-hoster-agents.sh --purge
set -euo pipefail
PURGE=0
[[ "${1:-}" == "--purge" ]] && PURGE=1

UNITS=(
  zabbix-agent zabbix-agentd zabbix-agent2
  telegraf datadog-agent newrelic-infra amazon-cloudwatch-agent
  node_exporter prometheus-node-exporter collectd netdata
  salt-minion salt-master salt-api
  puppet puppet-agent pxp-agent
  chef-client landscape-client
  nrpe nagios-nrpe-server snmpd monit glances
  wazuh-agent ossec anydesk teamviewerd
)
PKGS=(
  zabbix-agent zabbix-agent2 zabbix-release telegraf datadog-agent
  prometheus-node-exporter collectd netdata
  salt-minion salt-master salt-common puppet-agent chef
  landscape-client nagios-nrpe-server snmpd monit wazuh-agent
)
PORTS=(10050 10051 9100 9273 8125 161 162 4505 4506 8140 5666 19999 2812)
PROCS=(zabbix_agentd zabbix_agent2 salt-minion telegraf datadog-agent node_exporter ossec-agentd wazuh-agentd puppet chef-client)

echo "=== hoster agent audit $(hostname) $(date -u +%Y-%m-%dT%H:%MZ) ==="
echo
echo "-- systemd units --"
found_u=0
for u in "${UNITS[@]}"; do
  st=$(systemctl is-active "$u" 2>/dev/null || echo n/a)
  en=$(systemctl is-enabled "$u" 2>/dev/null || echo n/a)
  if [[ "$st" == "active" || "$en" == "enabled" ]]; then
    echo "FIND unit=$u active=$st enabled=$en"
    found_u=1
  fi
done
[[ $found_u -eq 0 ]] && echo "(none of watched units active/enabled)"

echo
echo "-- packages (dpkg) --"
found_p=0
if command -v dpkg >/dev/null 2>&1; then
  for p in "${PKGS[@]}"; do
    if dpkg -l "$p" 2>/dev/null | grep -q '^ii'; then
      echo "FIND package=$p installed"
      found_p=1
    fi
  done
fi
[[ $found_p -eq 0 ]] && echo "(none of watched packages installed)"

echo
echo "-- listening ports --"
found_port=0
SS=$(ss -tuln 2>/dev/null || netstat -tuln 2>/dev/null || true)
for port in "${PORTS[@]}"; do
  if echo "$SS" | grep -Eq ":${port}([[:space:]]|$)"; then
    echo "FIND port=$port listening"
    echo "$SS" | grep -E ":${port}([[:space:]]|$)" || true
    found_port=1
  fi
done
[[ $found_port -eq 0 ]] && echo "(none of watched ports listening)"

echo
echo "-- processes --"
found_pr=0
PS=$(ps ax -o comm= 2>/dev/null || true)
for pr in "${PROCS[@]}"; do
  if echo "$PS" | grep -qx "$pr" || echo "$PS" | grep -q "$pr"; then
    echo "FIND process=$pr"
    found_pr=1
  fi
done
[[ $found_pr -eq 0 ]] && echo "(none of watched process names)"

echo
echo "-- suspicious paths (names only) --"
for d in /opt/zabbix /etc/zabbix /etc/salt /etc/puppet /etc/chef /opt/datadog-agent /etc/telegraf /usr/local/nagios; do
  if [[ -e "$d" ]]; then
    echo "FIND path=$d"
  fi
done

echo
if [[ $PURGE -eq 1 ]]; then
  echo "==> purge requested"
  for u in "${UNITS[@]}"; do
    systemctl disable --now "$u" 2>/dev/null || true
    systemctl mask "$u" 2>/dev/null || true
  done
  export DEBIAN_FRONTEND=noninteractive
  apt-get remove -y --purge "${PKGS[@]}" 2>/dev/null || true
  if command -v netductor >/dev/null 2>&1; then
    netductor host-audit --purge 2>/dev/null || true
  fi
  echo "purge done — re-run without --purge to verify"
else
  echo "read-only. To remove findings: sudo bash $0 --purge"
  echo "or: netductor host-audit --purge"
fi

if [[ $found_u -eq 1 || $found_p -eq 1 || $found_port -eq 1 || $found_pr -eq 1 ]]; then
  echo
  echo "RESULT: FINDINGS"
  exit 1
fi
echo
echo "RESULT: CLEAN"
exit 0
