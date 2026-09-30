**EN** · [RU](ru/RUNBOOK-FORCE-UPDATE.md)

# Force-update node binaries (broken release / OOM)

Use when the node is stuck on an old build (e.g. **0.9.111**), `netductor update` fails, or the Telegram bot / API cannot self-upgrade safely.

Target: **v0.9.121** (or later tag — replace `VER` below).

Assets: https://github.com/PavelNeyman/netductor/releases/tag/v0.9.121

SSH: port **52222**, operator key (e.g. `~/.ssh/netductor`).

---

## Primary (control plane)

```bash
VER=0.9.121
KEY=~/.ssh/netductor
HOST=2.27.118.70   # your primary IP

ssh -i "$KEY" -p 52222 root@$HOST bash -s <<EOF
set -euo pipefail
VER=$VER
cd /tmp
curl -fsSL -o netductor "https://github.com/PavelNeyman/netductor/releases/download/v\${VER}/netductor-linux-amd64"
curl -fsSL -o netductor-tg "https://github.com/PavelNeyman/netductor/releases/download/v\${VER}/netductor-tg-linux-amd64"
chmod 755 netductor netductor-tg
systemctl stop netductor-telegram-bot netductor-api 2>/dev/null || true
install -m 755 netductor /usr/local/bin/netductor
install -m 755 netductor-tg /usr/local/bin/netductor-tg
echo "\$VER" > /etc/netductor/VERSION
systemctl start netductor-api netductor-telegram-bot
netductor version
netductor stack watchdog-install 2>/dev/null || true
netductor stack status 2>/dev/null || true
netductor doctor 2>/dev/null | tail -20 || true
EOF
```

Configs, VPN users, LE, secrets under `/etc/netductor` and `/var/lib/netductor` are **not** wiped.

---

## Secondary (RU entry + agent)

```bash
VER=0.9.121
KEY=~/.ssh/netductor
HOST=92.255.77.253   # your secondary IP

ssh -i "$KEY" -p 52222 root@$HOST bash -s <<EOF
set -euo pipefail
VER=$VER
cd /tmp
curl -fsSL -o netductor "https://github.com/PavelNeyman/netductor/releases/download/v\${VER}/netductor-linux-amd64"
curl -fsSL -o netductor-agent "https://github.com/PavelNeyman/netductor/releases/download/v\${VER}/netductor-agent-linux-amd64"
chmod 755 netductor netductor-agent
install -m 755 netductor /usr/local/bin/netductor
install -m 755 netductor-agent /usr/local/bin/netductor-agent
echo "\$VER" > /etc/netductor/VERSION 2>/dev/null || true
systemctl restart netductor-secondary-agent sing-box 2>/dev/null || true
netductor version 2>/dev/null || true
netductor-agent version 2>/dev/null || true
# optional DR local backup path
netductor backup secondary-local-timer 2>/dev/null || true
EOF
```

---

## Mac operator

```bash
brew reinstall netductor
netductor-op version   # expect 0.9.121
```

Formula tracks Release assets; after `brew reinstall` you get **netductor-op** only (node stays on VPS).

---

## Verify

| Check | Command / UI |
|-------|----------------|
| Versions | `netductor version` on both nodes |
| Stack | `netductor stack status` on primary; TG **Tools → Stack**; Web Control → Overview → Stack |
| Bot | message bot /status |
| VPN | phone client via secondary |
| Agent | primary nodes list shows secondary online |

---

## Notes

- Prefer force-update over full wipe when only the binary is bad.
- After primary is on ≥0.9.120, future upgrades can use `stack apply` / TG Updates with pre-backup.
- Do **not** run concurrent `update apply` while manually replacing binaries.
