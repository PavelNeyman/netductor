# Agent handoff (read first in a new chat)

**Repo:** https://github.com/PavelNeyman/netductor  
**Release tag:** `v0.7.0-dev`  
**Binaries:** `netductor-linux-amd64`, `netductor-tg-linux-amd64`, `netductor-agent-*`  
**Owner language:** Russian OK; docs must stay **EN + RU** for user-facing behaviour.

## Product in one paragraph

Personal **production** control plane (single operator): abroad **primary** VPS runs source of truth (users, TG bot, admin, backups); optional RU **secondary** is the default **VPN entry** under carrier whitelist and hosts warm services (Lampac). VPN tunnels are **not** load-balanced. OpenWrt/MikroTik edge is agent-based (outbound to primary).

## Fleet model (do not invert without owner OK)

| Name | Typical | Responsibility |
|------|---------|----------------|
| **primary** | Abroad | Control plane, TG bot **active**, API, users/secrets, backup authority. Hostname `nd-primary` |
| **secondary** | RU | VPN entry (WL), warm Lampac, TG bot **standby** via SOCKS→primary. Hostname `nd-secondary` |

Internal VPN agent code may still say `role=relay` — same process, operator-facing name is **secondary**.

## Clean deploy sequence (both VPS wiped)

### A. Primary (abroad)

```bash
wget -qO /usr/local/bin/netductor \
  https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-linux-amd64
chmod 755 /usr/local/bin/netductor

mkdir -p /etc/netductor/secrets
echo 'BOT_TOKEN' > /etc/netductor/secrets/telegram_bot_token
echo 'TG_USER_ID' > /etc/netductor/secrets/telegram_admin_id
chmod 600 /etc/netductor/secrets/*

netductor install
# optional: netductor install lampac   # prefer secondary later
netductor doctor
netductor fleet bootstrap   # after secondary exists; or set-primary only first
```

Preferred Reality SNI for RU WL experiments: **`api.vk.me`** (also `ya.ru` preset). Client links should prefer **secondary IP** when secondary is online.

### B. Secondary (RU) — from primary

```bash
netductor fleet provision-secondary \
  --host RU_IP --password '…' [--sni api.vk.me]
```

Does: VPN join + fleet secondary + sync + Lampac + bot standby + sync timer.  
Low-level VPN-only: `netductor relay provision --host … --password …`

### C. Recover primary from encrypted backup

Artifacts (keep offline): `*.ndenc`, `BACKUP_KEY.txt`, `COMPONENTS.txt` next to archive.

```bash
netductor recover --key "$(cat BACKUP_KEY.txt)" /path/to/backup.ndenc
```

Installs components from manifest, restores data, enables services. Keys inside backup are restored; if secondary Reality keys drifted, re-run `provision-secondary` or wait for agent + `vpn refresh-links`.

## Important paths

| Path | Purpose |
|------|---------|
| `/etc/netductor` | secrets, users, clients |
| `/var/lib/netductor` | state, registry, fleet policy, backups |
| `/opt/netductor` | bins, lampac volume, admin |
| `internal/fleet/` | primary/secondary policy, sync, provision-secondary, bot failover |
| `internal/relay/` | VPN secondary agent plane (legacy name “relay”) |
| `cmd/netductor/serve.go` | API :8787 localhost; agent :8788 includes `/api/bot-status` |

## Commands cheat sheet

```bash
netductor install|doctor|status|vpn|fleet|relay|backup|recover|edge|addons|tui
netductor fleet status|bootstrap|provision-secondary|sync|apply-lampac|bot-failover
netductor vpn link <user> [vless|hy2]
netductor backup peer-set root@SECONDARY:/var/lib/netductor/backups/peers/core/
```

## Hard constraints

- Go-only control plane; no new Python/shell installers.
- No secrets in git.
- Docs EN+RU when behaviour changes.
- Subscription **removed** (single links only).
- Lampac **localhost only** (VPN/SSH to reach).
- TG rich HTML in-message buttons preferred over reply_markup clutter.
- Do not expose admin API on WAN without TLS + explicit flag.

## Open / next (owner)

- Full clean dual-VPS smoke after wipe (this handoff’s goal).
- OpenWrt + MikroTik e2e on real hardware.
- HTTPS / domain optional later.
- TG bot standby needs secondary able to reach primary `:8788` and SSH for SOCKS when promoting.

## Test accounts (ephemeral — owner rotates)

Do **not** commit live passwords. Owner provides root passwords in chat for test VPS only.

## File to update when architecture changes

1. This handoff  
2. `docs/FLEET.md` + `docs/ru/FLEET.md`  
3. `AGENTS.md` frozen table if product decision changes  
4. `docs/ROADMAP.md`
