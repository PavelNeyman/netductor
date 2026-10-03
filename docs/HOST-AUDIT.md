# Host audit: hoster agents, listening surface, baseline

**EN** · [RU](ru/HOST-AUDIT.md)

How to fully audit a netductor VPS (primary or secondary) for **unwanted monitoring / CM / RMM agents**, unexpected listeners, and drift from a known-good surface. Produce logs you can hand to an operator or AI for analysis.

Detects units, packages (incl. hoster forks e.g. `zabbix-agent-timeweb`), residual paths, and **apt sources/keyrings** (`*zabbix*`, `*timeweb*`). Related code: `scripts/run-host-audit-bundle.sh`, `scripts/audit-hoster-agents.sh`, `scripts/collect-host-audit.sh`, `netductor host-audit`, `internal/install/host_agents.go`, firewall role allow-list, [PORTS.md](PORTS.md), [SECURITY.md](SECURITY.md).

---

## 0. Operator one-shot (copy-paste on the VPS)

Run **as root** on primary and/or secondary. Read-only collection (no purge). Downloads scripts from GitHub `main` (or pin a tag).

```bash
# Full bundle → /tmp/nd-host-audit-<host>-<utc>/ + .tar.gz
curl -fsSL https://raw.githubusercontent.com/PavelNeyman/netductor/main/scripts/run-host-audit-bundle.sh | bash
```

Pin a release/ref:

```bash
curl -fsSL https://raw.githubusercontent.com/PavelNeyman/netductor/main/scripts/run-host-audit-bundle.sh \
  | NETDUCTOR_AUDIT_REF=main bash
```

Then pull the archive to your laptop and send it for review:

```bash
# on laptop (example)
scp -P 52222 root@PRIMARY:/tmp/nd-host-audit-*.tar.gz .
scp -P 52222 root@SECONDARY:/tmp/nd-host-audit-*.tar.gz .
```

### What the archive contains

| File | Content |
|------|---------|
| `host-audit.txt` | Full text bundle (ss, units, packages, firewall, ssh, cron, …) |
| `hoster-agents.txt` | Denylist probe (zabbix/salt/RMM/…) |
| `extra-signals.txt` | Top CPU, denylist processes, established TCP sample, failed units, auth tail |
| `MANIFEST.txt` | host, ref, timestamps |
| `*.stderr` | Script errors if any |

**Do not** put private keys or backup keys in the bundle (scripts redact / skip them).

### Offline / no GitHub from the VPS

```bash
# from a machine that has the repo
scp -P 52222 scripts/run-host-audit-bundle.sh scripts/collect-host-audit.sh scripts/audit-hoster-agents.sh root@HOST:/tmp/nd-audit/
ssh -p 52222 root@HOST 'bash /tmp/nd-audit/run-host-audit-bundle.sh'
```

### Destructive purge (only after review)

```bash
# dry-run style: read-only first (one-shot above)
# then, only if findings are confirmed:
netductor host-audit --purge    # if supported on installed version
# or: bash audit-hoster-agents.sh --purge
```

---


## 1. Principles

| Plane | Policy |
|-------|--------|
| **Inbound** | Default deny; **allow-list only** what the role needs (see PORTS / role matrix). Do not rely on an ever-growing deny list. |
| **Outbound** | Default allow (VPN egress node). Do not default-deny outbound. |
| **Agents** | No hoster monitoring, config-management, or RMM agents on managed nodes. |
| **Baseline** | After successful deploy+harden, snapshot listeners/units/packages; later diffs are WARN/FAIL material. |

---

## 2. Expected listen surface (reference)

### Secondary (RU entry)

| Port | Service |
|------|---------|
| 52222/tcp | SSH (key-only after harden) |
| 443/tcp | sing-box VLESS |
| WSS service port (e.g. 8445) | SP/PS path |
| Loopback only | agent, local health |

### Primary (control plane)

| Port | Service |
|------|---------|
| 52222/tcp | SSH |
| 443/tcp | Reality / uplink |
| 8443/tcp | redirect (LE) |
| 80/tcp | ACME only if used |
| 8444/tcp | WSS SP (typical) |
| 8787, 8789 | **not** on public WAN (loopback / service CIDR) |
| 9118, 5000, 53, … | loopback / service-net per PORTS.md |

Anything else on `0.0.0.0` / `*` is a finding until justified.

---

## 3. Threat / junk catalog (what we look for)

### 3.1 Monitoring

Zabbix agent/agent2, NRPE, check_mk, Netdata, node_exporter, telegraf, collectd, PCP, Datadog, New Relic, Elastic Agent, Splunk UF, OpenTelemetry Collector, Amazon CloudWatch agent, Glances server, Monit HTTP.

**Ports often:** 10050, 10051, 5666, 19999, 9100, 9273, 8125, 2812.

### 3.2 Configuration management

salt-minion/master, puppet-agent, chef-client, cfengine, Landscape.

**Ports:** 4505, 4506, 8140.

### 3.3 RMM / remote support

Mesh agent, Tactical RMM, commercial RMM, AnyDesk, TeamViewer host.

### 3.4 Panel / hypervisor helpers

SolusVM/DirectAdmin-style agents; optional debate: `qemu-guest-agent` / `open-vm-tools` (console convenience vs extra channel — policy choice, document decision).

### 3.5 Distro noise

avahi, cups, rpcbind, unused MTA (exim/postfix) if SMTP product alerts are not enabled.

### 3.6 Compromise signals (extra pass)

Unexpected `authorized_keys`, user systemd units, `ld.so.preload`, miners, hidden listeners (`ss` vs `netstat` disagreement), unknown packages in `/opt`.

Reference IoC lists: [linux-edr-iop](https://github.com/messede-degod/linux-edr-iop).

---

## 4. Step-by-step audit (operator)

Run on **each** VPS as root. Prefer read-only first.

### Step 0 — Prepare

```bash
mkdir -p /tmp/nd-audit && cd /tmp/nd-audit
hostname; date -u; cat /etc/netductor/VERSION 2>/dev/null; netductor version 2>/dev/null
```

### Step 1 — Product collect (if installed)

```bash
# from repo or installed path
bash /path/to/netductor/scripts/collect-vps-state.sh | tee nd-state-$(hostname).txt
```

### Step 2 — Hoster agent script

```bash
curl -fsSL -o audit-hoster-agents.sh \
  https://raw.githubusercontent.com/PavelNeyman/netductor/main/scripts/audit-hoster-agents.sh
# or copy from repo
sudo bash audit-hoster-agents.sh 2>&1 | tee hoster-agents-$(hostname).txt
echo EXIT:$?
```

Exit code `1` = findings. **Do not** pass `--purge` until you review the log.

### Step 3 — Full host audit bundle

```bash
sudo bash /path/to/netductor/scripts/collect-host-audit.sh 2>&1 | tee host-audit-$(hostname).txt
```

This script (see §6) gathers listeners, units, packages, crons, ssh keys summary, firewall status, and denylist hits into one file.

### Step 4 — Optional deeper tools (not required for first pass)

```bash
# if you choose to install temporarily
apt-get install -y rkhunter aide 2>/dev/null || true
rkhunter --check --sk 2>&1 | tee rkhunter-$(hostname).txt
# AIDE only after a trusted baseline exists
```

### Step 5 — Package and send

```bash
tar -czf nd-audit-$(hostname)-$(date -u +%Y%m%dT%H%MZ).tgz \
  nd-state-*.txt hoster-agents-*.txt host-audit-*.txt rkhunter-*.txt 2>/dev/null
ls -la *.tgz
```

Copy the archive off-box (SCP to Mac). **Redact** nothing required for analysis except you may strip private key material if any script ever dumped it (these scripts must not dump private keys).

### Step 6 — Analysis handoff

Provide the `tgz` or the three main text files to the operator/AI with:

- role of host (primary / secondary);
- whether harden/ufw was expected;
- any recent reinstall or hoster “support” login.

---

## 5. What the AI / operator should conclude from logs

| Finding | Typical action |
|---------|----------------|
| zabbix/telegraf/… active | `host-audit --purge` or harden re-run; verify unit masked |
| port 10050/9100/… listening | purge agent; confirm inbound allow-list |
| ufw missing / inactive | install ufw or iptables fallback; `firewall apply`; doctor FAIL |
| unexpected public :8789 | restrict to service CIDR; disarm api-public |
| qemu-ga only | policy: leave or remove; not auto-critical |
| CLEAN hoster script + extra listener on :631 | disable cups/avahi |

After purge, re-run Step 2–3 until `RESULT: CLEAN` and listen list matches role matrix.

---

## 6. Scripts

| Script | Role |
|--------|------|
| `scripts/audit-hoster-agents.sh` | Denylist units/pkgs/ports/procs; optional `--purge` |
| `scripts/collect-host-audit.sh` | Full text bundle for offline analysis |
| `scripts/collect-vps-state.sh` | Product/systemd/stack-oriented state |
| `netductor host-audit [--purge]` | Same denylist from Go on node |
| `netductor firewall status\|apply` | Inbound allow-list |
| `netductor doctor` | Must FAIL on foreign agents / missing firewall backend |

---

## 7. Baseline snapshot (recommended after clean deploy)

```bash
mkdir -p /var/lib/netductor/baseline
ss -tulnp > /var/lib/netductor/baseline/ss.txt
systemctl list-unit-files --state=enabled > /var/lib/netductor/baseline/enabled-units.txt
dpkg -l > /var/lib/netductor/baseline/dpkg.txt
ufw status verbose > /var/lib/netductor/baseline/ufw.txt 2>/dev/null || iptables-save > /var/lib/netductor/baseline/iptables.txt
date -u > /var/lib/netductor/baseline/created
```

Future doctor enhancement: diff current `ss`/units against baseline (OPEN_ITEM until implemented).

Optional: AIDE init **after** baseline when primary has disk budget.

---

## 8. Implementation checklist (product)

- [x] Initial denylist + `audit-hoster-agents.sh` + `host-audit` CLI (0.9.192+)  
- [x] Firewall module with ufw preferred + iptables fallback direction  
- [ ] Expand denylist (RMM names, otel, cloudwatch, avahi/cups) — keep in sync script ↔ Go  
- [x] `collect-host-audit.sh` + `run-host-audit-bundle.sh` one-shot  
- [ ] Doctor FAIL: foreign agent OR missing fw backend OR denylist port on non-loopback  
- [ ] TG/Web firewall status block  
- [ ] Baseline snapshot at end of install  
- [ ] Document qemu-ga policy explicitly in SECURITY.md  

---

## 9. Safety

- `--purge` is destructive to matching packages; run read-only first.  
- Do not purge `cloud-init` by default (provider rebuild).  
- Auto-heal of firewall from a timer: **alert first**; apply only on explicit operator action (risk of lockout).
