## Version

## Secondary role (locked 2026-09-17)

Full plan: [PLAN-SECONDARY-VPN-ONLY.md](PLAN-SECONDARY-VPN-ONLY.md).

- **RU secondary = VPN entry only** (VLESS/Reality + thin agent). Not a full mirror of primary.
- **No Lampac/docker on secondary** unless RU IP is explicitly required (needs ≥2 GiB RAM).
- **Edge/OpenWrt agents enroll to primary** (outbound). Routers in RU still reach primary API; do **not** move edge control plane to RU (splits source of truth).
- **Bug fixed:** adding a VPN user must bump `secondary config_ver` so agent pulls new user UUIDs onto `relay-in`. Otherwise links point at `vpn.*` (secondary) but only the first user UUID was on RU → second user “VPN dead”.
- After `vpn add` / `ApplyConfig`, secondary agent applies `ExportRelayBundle` within ~30–90s. Operator can force: `netductor relay sync`.



**0.7.3-dev** — TG Access rich QR + import redirect on :80; core/vpn advertise hosts; mTLS agent plane.

# Agent handoff (read first in a new chat)

**Repo:** https://github.com/PavelNeyman/netductor  
**Release tag:** `v0.7.3-dev` (check Releases if tag name differs)  
**Binaries:** `netductor-linux-amd64`, `netductor-tg-linux-amd64`, `netductor-agent-*`  
**Owner language:** Russian OK; docs **EN + RU** for user-facing behaviour.

Also read: [AGENTS.md](../AGENTS.md) · [FLEET.md](FLEET.md) · [DEPLOY.md](DEPLOY.md) · [ROADMAP.md](ROADMAP.md)

---

## Product (one paragraph)

Personal **production** control plane (single operator): abroad **primary** = source of truth (users, active TG bot, admin API, backups); RU **secondary** = default **VPN entry** under carrier whitelist (**no** Lampac/mirror). VPN is **not** load-balanced. OpenWrt/MikroTik via outbound agents to primary.

---

## Fleet model (do not invert without owner OK)

| Name | Where | Role |
|------|--------|------|
| **primary** | Abroad | Control plane, TG bot **active**, API, users/secrets, backup authority. Hostname `nd-primary` |
| **secondary** | RU | VPN entry (WL), Lampac preferred, TG **standby** via SOCKS→primary. Hostname `nd-secondary` |

Internal VPN agent code may still use `role=relay` / `netductor relay *` — same plane, operator name is **secondary**.

Default Reality SNI for WL: **`api.vk.me`**. Client VLESS links prefer **secondary IP** when online.

---

## SSH policy

1. Password only for **first** provider login.
2. `netductor install` (primary) generates `/root/.ssh/id_ed25519`, installs pubkey, **disables password** (`sshd_config.d/00-netductor-harden.conf` + neutralize cloud-init `PasswordAuthentication yes`).
3. `fleet provision-secondary` / `relay provision` installs **same** primary pubkey on secondary and disables password there.
4. Operator must keep a copy of the private key (workspace often has `artifacts/netductor_vps_id_ed25519`).

---

## Clean deploy (both VPS wiped)

### A. Primary

```bash
wget -qO /usr/local/bin/netductor \
  https://github.com/PavelNeyman/netductor/releases/download/v0.7.3-dev/netductor-linux-amd64
chmod 755 /usr/local/bin/netductor

mkdir -p /etc/netductor/secrets
echo 'BOT_TOKEN' > /etc/netductor/secrets/telegram_bot_token
echo 'TG_USER_ID' > /etc/netductor/secrets/telegram_admin_id
chmod 600 /etc/netductor/secrets/*

netductor install
netductor vpn set-sni api.vk.me
netductor doctor
# copy /root/.ssh/id_ed25519 off-box before relying on key-only SSH
```

### B. Secondary (run on primary)

```bash
netductor fleet provision-secondary \
  --host RU_IP --password '…' --sni api.vk.me
netductor fleet status
netductor vpn refresh-links
netductor vpn link operator vless   # expect @RU_IP
```

**Critical bug fixed:** post-provision must **not** `RemoveByPublicIP` before agent heartbeat (wiped agent token → unauthorized). Code keeps issued tokens; only prune very old offline ghosts.

### C. Recover

Keep offline: `*.ndenc`, `BACKUP_KEY.txt`, `COMPONENTS.txt` (sidecar).

```bash
netductor recover --key "$(cat BACKUP_KEY.txt)" /path/to/backup.ndenc
```

---

## Telegram bot UX (locked)

| Location | Content |
|----------|---------|
| **Under message** (`reply_markup` / inline keyboard) | **Navigation only**: Main menu, Users, Nodes, back |
| **In message body** (`<tg-button>`, tables, HTML) | **Screen actions**: user access/rename/enable, VLESS/Core/HY2 switch, node metrics/upgrade… |

Do **not** duplicate Add/Menu in both places. Code: `cmd/netductor-tg/keyboards.go` + `format.go`.

TG bot active on **primary** only. Secondary bot-failover/standby **removed** (VPN-entry model).

---

## TUI

`netductor tui` — **Setup wizard** (primary / secondary / OpenWrt / MikroTik) then questions → automatic apply; separate **Tools** (doctor, fleet, backup, sync, VPN…).

---

## Routing (secondary)

RU domains (`.ru`, `.su`, xn--p1ai, major RU services) → **direct**; else → **uplink** to primary. See secondary `/usr/local/etc/sing-box/config.json` route rules.

---

## Important paths

| Path | Purpose |
|------|---------|
| `/etc/netductor` | secrets, users, clients |
| `/var/lib/netductor` | state, fleet policy, relay devices, backups |
| `/opt/netductor` | bins, lampac volume, admin |
| `internal/fleet/` | primary/secondary, sync, provision-secondary, bot failover |
| `internal/relay/` | VPN secondary agent (legacy name) |
| `internal/install/ssh_harden.go` | primary key-only SSH |
| `cmd/netductor-tg/` | operator Telegram bot |
| `:8787` | Admin API localhost |
| `:8788` | Agent plane + `/api/bot-status` |

---

## Commands

```bash
netductor install|doctor|status|vpn|fleet|relay|backup|recover|edge|addons|tui
netductor fleet status|bootstrap|provision-secondary|disable-legacy
netductor install lampac   # primary only
netductor relay sync       # VPN users → secondary
netductor vpn link <user> [vless|hy2|core]
netductor backup peer-set root@SECONDARY:/var/lib/netductor/backups/peers/core/
```

---

## Hard constraints

- Go-only control plane; no new Python/shell installers as runtime.
- No secrets in git.
- Docs EN+RU when behaviour changes.
- Subscription **removed** (single links only).
- Lampac **localhost only** (reach via VPN/SSH).
- No VPN load-balancer VPS.
- Do not expose admin API on WAN without TLS + explicit flag.

---

## Recent verified state (2026-09-15 smoke)

- Dual VPS clean install + provision-secondary worked after token-wipe fix.
- Bot failover: stop primary bot → standby on secondary; restore → demote.
- Cross-peer backup OK; fleet sync OK; Lampac on secondary OK.
- SSH password disabled both nodes after harden.

---

## Open / next

- [ ] OpenWrt + MikroTik e2e on real hardware
- [ ] Optional HTTPS / domain
- [ ] Optional: more TG screens under same nav-vs-actions rule
- [ ] Path B: limited end-user bot (documented idea only)

---

## For the next agent

1. Read this file + AGENTS.md.  
2. Prefer GitHub `main` + release assets over ad-hoc VPS edits.  
3. After code changes: `go test` / `go build`, push sources + update release assets for the tag in VERSION / Releases.  
4. Live VPS passwords are **ephemeral** — owner provides them; never commit them. SSH key may live in operator workspace only.


---

## Security hardening (locked)

- SSH: password off, `X11Forwarding no`, key-only root
- Admin API `:8787` → localhost only
- Agent plane `:8788` → world-facing, **token required**; prefer nft limit to secondary IP
- Blocky DNS → `127.0.0.1:53` only (not public — amp risk)
- Lampac → localhost only
- No Zabbix / hoster agents
- sing-box config.json mode `600`
- Agent remote cmds: allowlist only (`reboot`, `upgrade`, `metrics`, `journal`, `restart:<allowed-unit>`)


## Hostnames / domain
- `NETDUCTOR_PUBLIC_HOSTNAME` or `/etc/netductor/public_hostname` → mTLS SAN + agent URL host
- `NETDUCTOR_VPN_HOST` or `/etc/netductor/vpn_hostname` → VLESS link host (e.g. `vpn.netductor.work.gd`)
- Secrets: prefer `secondary_agent_token` / `secondary_core_url` (legacy `relay_*` still read)
- Devices registry: `/var/lib/netductor/secondary/devices.json` (migrates from `relay/`)

### mTLS / TLS CLI
- `netductor mtls ensure` / `mtls issue-client <id>`
- `netductor tls self-signed [host]` for lab HTTPS admin

---

## Locked decisions (2026-09-16)

1. **TG UI:** navigation under message; screen actions in body — [TG-UI.md](TG-UI.md).
2. **Access screen:** rich message with QR (`tg://photo`), URI in `<pre><code>`, VLESS/Core/HY2 in body; nav under.
3. **App import buttons:** Telegram forbids custom schemes in url-buttons → **`netductor redirect-serve` on :80** + `NETDUCTOR_REDIRECT_BASE`.
4. **Advertise hosts:** `coreAdvertiseHost` (primary / netductor.work.gd) ≠ `vpnAdvertiseHost` (secondary / vpn.…); never mix Reality pbk/sid across hops.
5. **Agent plane:** mTLS :8789; nft/UFW allow only secondary→primary; zabbix agent not ours — remove if present.
6. **LE on \*.work.gd:** rate-limited in test; self-signed OK until real domain / reinstall.


## TG Access edit policy

Access uses **delete + sendRichWithPhoto** so QR and buttons stay a single new message (editMessageMedia/rich is flaky across text↔photo). Do not add a second “link only” message on the happy path — that was an earlier experiment and clutters chat. Optional future: in-place rich edit **without** extra messages.

## 2026-09-16 updates

- VPN users on live test: **Pavel**, Nelya (not `operator`). SR Config profile button only for Pavel/operator.
- **SR Config**: callback `u:workcfg:` → **Telegram document** (`nd-oc.conf`), no public URL required for download.
- Default SSH port **52222**. SMTP alerts: `NETDUCTOR_SMTP_*` env (no Apprise).
- Status-page / family ping button: **deferred**.
- Flow mismatch: only public IP in journal; use per-device VPN users for identity.

## 2026-09-16 evening

### DNS
- Lists verified: doubleclick.net → NXDOMAIN via blocky; example.com resolves.
- UI gold standard: rich table + in-cell buttons; toggle writes config; **Reload** fetches lists.

### Menu
- Main: Status, Users, Fleet, **Tools**, Operator, Lang, Help.
- Tools: Guest, DNS, Backup, Locations, Updates.

### Backup
- Paths: etc/netductor, etc/blocky, etc/sing-box, var/lib/netductor, opt/netductor/{lampac,profiles}.
- COMPONENTS.txt drives reinstall; lampac image re-pulled, data from archive.
- UI: schedule, run now, list, restore, keep N. prune uses BackupKeepCount (default 14).

### Updates
- `internal/update`: GitHub latest tag + download assets; TG Tools → Updates → Update primary.
- Agent fleet update (notify + per-device / update-all) — designed, UI stub notes edge versions when registry reports them.

### Locations
- List/card/rename/delete; edge bind still when agents enroll.

## TUI (2026-09-17+)

- LAG-style tabs: **Wizard / Tools / Ops / Settings / Mode**. Chips = real keys (`l`, `tab`, …). No 1–9 jump.
- Locale: default **auto** — macOS **AppleLanguages** first (not LANG; terminals often force en_US), then AppleLocale, then LANG; `NETDUCTOR_LANG=ru|en` override; `l` forces ru/en.
- **Settings**: SSH connection (host, user, key path, password), language, save to `~/.config/netductor/tui.yaml`.
- CLI: `netductor tui --remote HOST [--remote-user root] [--remote-key ~/.ssh/id_ed25519] [--remote-password …]`
- Env still works: `NETDUCTOR_REMOTE`, `NETDUCTOR_REMOTE_USER`.
- Mode **Manage** (was “Day-2/Operator”): ops on an already-installed node.
- Interactive actions (vpn-add, edge, build, connection) use **in-TUI forms** — no exit to huh for those paths.
- Ops run local or via `ssh` to configured remote.



## Ops note (2026-09-17)

- Brief VPN blips without operator action: check secondary journal for `dial tcp PRIMARY:443: i/o timeout` (path RU→abroad). Services may stay active.
- Flow mismatch from home IP = client without `xtls-rprx-vision`.
