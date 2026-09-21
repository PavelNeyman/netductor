# Agent handoff — netductor

**Version:** 0.8.19  
**Repo:** https://github.com/PavelNeyman/netductor  
**Last full review:** `docs/REVIEW-2026-09-21-FULL.md`  
**Residual risks:** `docs/RESIDUAL_RISKS.md`  
**UI parity:** `docs/UI-PARITY.md` · `bash scripts/check-ui-parity.sh`

## Locked architecture

| Role | Name | Role |
|------|------|------|
| Primary | abroad VPS | Control plane, API loopback, TG bot, blocky, NVR optional, VPN core |
| Secondary | RU VPS | VLESS entry + agent; not a full mirror of primary services |
| Edge | OpenWrt | Agent outbound mTLS `:8789`, enroll/approve, recovery LAN page |

- Deploy from **Mac TUI** workstation wizards (`deploy.DeployPrimary` / secondary / `DeployEdge`).
- Agent plane **mTLS only** on `:8789` (plain `:8788` emergency only).
- Admin API default **127.0.0.1:8787** — not world-open.
- VPN public ports encrypted (Reality/HY2). SSH key-only after harden.

## Recent fixes (0.8.17–0.8.19)

- Recovery HTTP: no public fallback; loopback if no private IP
- redirect-serve default `127.0.0.1:80`
- Secondary + **primary** node upgrade: pure Go, version from `deploy.Release`
- Admin i18n pass; TG `m:secondary:*`
- cli18n doctor/vpn/mtls

## Next for human

1. Hardware e2e on Cudy / cameras / MikroTik site flow  
2. Domain when ready → HTTPS redirect  
3. Reinstall drill with v0.8.19 binaries from Releases  

## Commands

```bash
brew reinstall netductor
netductor version   # 0.8.19
NETDUCTOR_LANG=ru netductor doctor
bash scripts/check-ui-parity.sh
bash scripts/check-version-pins.sh
```

---

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
**Release tag:** `v0.8.19` (check Releases if tag name differs)  
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
  https://github.com/PavelNeyman/netductor/releases/download/v0.8.19/netductor-linux-amd64
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


## Uplink multiplex (0.7.13-dev)
- Secondary uplink: **no vision**, `multiplex.smux` (max_connections=4).
- Primary `vless-reality`: multiplex enabled; `relay-uplink` user has empty flow.
- Vision remains for end-user clients. Vision ⊕ mux is unsupported.

## Path quality secondary→primary (2026-09-18 load test)

After uplink **smux multiplex** (no vision on relay-uplink):

| Test | Result |
|------|--------|
| ICMP 20–40 pkt | **0% loss**, ~34 ms |
| TCP :443 connect ×100 | **100/100 ok** (earlier was ~10–20% fail) |
| iperf3 TCP 4-stream 10s | **~950 Mbit/s** sum, some Retr |
| iperf3 reverse 8s | **~960 Mbit/s** |
| `i/o timeout` since mux | **~2** (was 500+/6h before) |
| flow mismatch | still high — client profiles without vision (separate issue) |

Bare path is capacity-rich; instability was **many short TCP+Reality dials**. Mux addresses that. UFW on primary defaults DROP — only open test ports temporarily.


## Family messenger

See [MESSENGER-EVAL.md](MESSENGER-EVAL.md). Shortlist: **Snikket** (primary candidate), Tinode (geo?), Matrix fallback; **Guardyn** watch when mobile ships; Seclettr/Delta out.

## Family messenger pilot (2026-09-18)

**Not** part of netductor binary — manual pilot on **primary** (+ proxies on **secondary**). See also [MESSENGER-EVAL.md](MESSENGER-EVAL.md).

### Deployed

| Component | Where | Port | Notes |
|-----------|--------|------|--------|
| **Databag** | primary Docker `databag` | **7000** | Admin env `ADMIN`; data volume `databag_databag_data` |
| **Databag proxy** | secondary `databag-proxy.service` (socat) | **7000** | → `2.27.118.70:7000` |
| **SMP** | primary `smp-server.service` | **5223** | `/etc/opt/simplex/`; user `smp` |
| **XFTP** | primary `xftp-server.service` | **5224** | `/etc/opt/simplex-xftp/`, files `/srv/xftp` quota 20GB |
| **SMP/XFTP proxy** | secondary `simplex-5223-proxy` / `simplex-5224-proxy` | 5223/5224 | → primary; useful **without** VPN |

Paths/creds on primary: `/opt/databag/`, `/opt/simplex/client-servers.txt`, `/opt/messenger-pilot.txt`.

### Access rules (important)

- Primary **UFW**: 7000/5223/5224 allowed from **secondary** (`92.255.77.253`), localhost, and primary self (hairpin). **Not** world-open (Docker-USER drop for Databag 7000 from others).
- Client on **secondary VPN**: use **primary public IP** in URLs (`2.27.118.70`), **not** secondary IP — hairpin to `92.255.77.253` from the tunnel often fails.
- Client **without** VPN: Databag via secondary proxy `http://92.255.77.253:7000`; SMP/XFTP secondary proxies may work from internet (socat public listen) — review firewall if that is undesired.
- Databag admin: **cog (settings) → admin password only** (not main user login). User accounts created via invite link from admin UI.

### SimpleX addresses (VPN → secondary, then to primary)

```
smp://DtvLSkd71GwRROIU4j0L_VDsi0ak0ArKWxo7G49EHnU=:PASSWORD@2.27.118.70:5223
xftp://73FiJ0clO6ZvINV6uG0u5yG0bg1KLAeW0gig5wyn1zg=:PASSWORD@2.27.118.70:5224
```

Passwords: `/opt/simplex/queue.pass`, `/opt/simplex/xftp.pass` on primary (rotate if this handoff is shared widely).

App: Network & servers → add **both** SMP and XFTP; media errors clear after XFTP added. Own server optional; Flux = third-party preset operator (opt-in).

### Test status (2026-09-18)

- [x] Databag up; admin password works via **/#/admin** (cog)
- [x] User account created; UX sparse — pilot only without domain/HTTPS
- [x] Databag via secondary proxy HTTP OK; direct primary:7000 from secondary-VPN unreliable
- [x] SMP on primary works from secondary-VPN
- [x] XFTP on primary fixes “no file/media server”
- [ ] Longer family trial / mobile apps
- [ ] Domain + TLS (Snikket still blocked by :443 VLESS without SNI split)
- [ ] Optional: close secondary socat to world; only VPN

### Ops

```bash
# primary
systemctl status smp-server xftp-server
docker ps --filter name=databag
# secondary
systemctl status databag-proxy simplex-5223-proxy simplex-5224-proxy
```

## NVR / Tapo cameras (draft)

See [PLAN-NVR-TAPO.md](PLAN-NVR-TAPO.md) — leases on OpenWrt agent, RTSP to primary, record/UI/encryption options.

### NVR locked (2026-09-18)

- **VPN only** (no public RTSP/NVR).
- Record on **primary** first; storage backend **pluggable** → home later.
- **Encryption at rest** required for primary NVR data (LUKS/gocryptfs).

Full text: [PLAN-NVR-TAPO.md](PLAN-NVR-TAPO.md).

- NVR addendum: **multi-site cameras**, motion schedules/zones, PTZ/night, live via go2rtc **VPN-only**; TG = alerts + one-time links (no TG live stream).

### NVR CLI (0.7.16-dev)

```
netductor nvr config show|set retention_days=7 max_gb=40 min_free_gb=5
netductor nvr cameras list|add name=… site=… mac=… ip=… password=…
netductor nvr leases <device_id>
netductor nvr retention
netductor nvr prepare-storage
```

API: `/api/nvr/config`, `/api/nvr/retention/run`, `/api/nvr/cameras`, …  
Plan: [PLAN-NVR-TAPO.md](PLAN-NVR-TAPO.md). Background retention on `serve`.


### NVR 0.7.17-dev

- TG: Tools → **NVR**
- CLI leases wait for agent result (default)
- doctor shows NVR path/retention/segment count

### NVR 0.7.18-dev

- TG: **From leases** → edge site → async wait → add camera (password prompt) + optional static DHCP
- `nvr storage` / `GET /api/nvr/storage`

### NVR 0.7.19-dev

- Agent: `rtsp_probe` (TCP + optional ffprobe)
- CLI: `netductor nvr probe <camera_id>`
- Ingest: `POST /api/nvr/ingest` (multipart `file` + `camera_id`) for site→primary segment push

### NVR 0.7.20-dev

- Agent records on LAN (`ffmpeg` segments in `/tmp/netductor-nvr`) and uploads to primary ingest
- CLI: `netductor nvr record start|stop <id>`
- TG: Cameras → 🔍/⏺/⏹ per camera
- Doctor: mountpoint / encryption hint

### Cudy / edge load (0.7.21)

- **Preferred:** record on site with `-c copy` (no re-encode). CPU stays low when stream is healthy.
- **Offline camera:** agent does **TCP probe first**; exponential backoff 5s→5m — no tight ffmpeg restart loop.
- **Limits:** max **2** concurrent cameras per agent; **/tmp** NVR cap **~200MB** (oldest segments dropped).
- **ffprobe:** 8s timeout kill.
- Cudy TR1200-class devices are fine for 1–2 substreams copy; avoid full HD×N + encode on-router.
- Primary `StartRecorder` does **not** auto-restart (prevents CPU spin if URL unreachable from VPS).
### NVR 0.7.22-dev

- Motion schedule (not CV yet): `nvr motion set enabled=true timezone=Europe/Moscow`
- Events JSONL on segment ingest; TG Events/Motion
- API `/api/nvr/motion`, `/api/nvr/events`

### NVR storage (0.7.23)

- **Archive on primary only.** Cudy: `/tmp` buffer **≤64MB**, upload→delete.
- TG alert on segment if motion schedule allows (`AlertOnSegment`).

### NVR + Cudy storage (0.7.24-dev) — handoff

**Where video lives**

1. **Primary** — permanent archive (`/var/lib/netductor/nvr/…`), retention, encrypt.
2. **OpenWrt agent** — short buffer → `POST /api/nvr/ingest` → delete local file.

**Cudy constraints**

- Flash **16MB**: never NVR path.
- RAM **128MB**: default buffer **`NVR_MAX_MB=24`** on `/tmp` (tmpfs).
- Optional **USB**: mount + `NVR_DIR=/mnt/…/netductor-nvr`, raise `NVR_MAX_MB` (e.g. 512).
- **LTE modem SD**: only if `ls /dev/sd*` shows it; many modems hide SD from OpenWrt.

**Agent config keys:** `NVR_DIR`, `NVR_MAX_MB` (also env `NETDUCTOR_NVR_DIR`, `NETDUCTOR_NVR_MAX_MB`).

**Safety:** max **2** concurrent cameras; TCP pre-check + backoff; no tight ffmpeg restart.

**CLI (primary):** `nvr cameras|leases|probe|record|motion|events|status|prepare-storage`  
**TG:** Tools → NVR (leases, cameras P/R/S, motion, events)  
**Plan:** [PLAN-NVR-TAPO.md](PLAN-NVR-TAPO.md) · [EDGE-AGENT.md](EDGE-AGENT.md)

### NVR backlog vs done (0.7.25-dev)

**Done:** inventory, leases→camera, agent record→ingest, retention, motion schedule, events, clip one-shot tokens, NVR_DIR/MAX_MB, TG hub.

**Not done (no deploy required to code later):** go2rtc live UI, PTZ/night API, CV zones, auto LUKS unlock, NFS home backend automation, Admin web NVR page, TUI NVR wizard.

**Cudy:** flash 16MB unused for video; RAM tmpfs buffer default 24MB; USB via NVR_DIR.

## Tapo C200 — locked integration notes

**Do not depend on Home Assistant Tapo-Control / arbitrary HA plugins.** C200 support varies by firmware; HA ONVIF PTZ is often slow or incomplete.

| Capability | Approach |
|------------|----------|
| Video | **RTSP** Camera Account: `rtsp://USER:PASS@IP:554/stream1` (HD) / `stream2` (SD for edge record) |
| Auth | Tapo app → Advanced → **Camera Account** (not TP-Link cloud email) |
| ONVIF | Profile **S**, port **2020** when enabled/supported by FW; PTZ ContinuousMove best-effort |
| PTZ | Agent `camera_ptz` → ONVIF SOAP; if FW has RTSP-only, use Tapo app for PTZ |
| Night/IR | On-camera automatic; NVR records whatever RTSP delivers |
| Audio | One-way may work; **no** two-way over Profile S |
| Live | Optional **go2rtc** config: `netductor nvr go2rtc` → bind 127.0.0.1 only |

Flash 16MB / RAM 128MB on Cudy: buffer on tmpfs max **24MB** or **USB** `NVR_DIR`.

## PTZ for Tapo C200 — preferred path (HA-compatible)

Working stack in the wild: **[HomeAssistant-Tapo-Control](https://github.com/JurajNyiri/HomeAssistant-Tapo-Control)** → library **[pytapo](https://github.com/JurajNyiri/pytapo)**.

| Method | Reliability on C200 |
|--------|---------------------|
| **pytapo `motorMove`** | Primary — same as HA |
| ONVIF ContinuousMove :2020 | Fallback only; FW-dependent |

**Prerequisites (camera):**
1. Tapo app → **Me → Tapo Lab → Third-Party Compatibility → On**
2. **Camera Account** (Advanced → Camera Account), not TP-Link cloud login

**On site (agent host):**
```
pip3 install pytapo   # or python3-pytapo if packaged
# ship scripts/tapo_control.py to /opt/netductor/scripts/
```

Agent `camera_ptz` tries **pytapo script first**, then ONVIF.

Commands: `move left|right|up|down`, `night:on|off|auto`, `privacy:on|off`

## Native Go port of pytapo (0.7.28-dev) — DONE

Package `internal/tapo` ports the **secure local control** path used by
[pytapo](https://github.com/JurajNyiri/pytapo) / HA Tapo-Control:

- probe encrypt_type 3
- device_confirm (MD5/SHA256 password hash)
- digest login → stok + AES-CBC lsk/ivb
- `securePassthrough` + Seq / Tapo_tag
- `motorMove`, `setDayNightModeConfig`, `setLensMaskConfig`
- legacy hashed-password login fallback

Agent `camera_ptz` order: **tapo-go → python pytapo → ONVIF**.

Still required on camera: **Third-Party Compatibility On** + Camera Account.

Ported: secure+KLAP, presets, motion, alarm, child wrap. Media stream: RTSP. Hub list via children.

### Tapo Go port progress (0.7.29-dev)

| Feature | Status |
|---------|--------|
| Secure login + AES passthrough | Done |
| motorMove left/right/up/down | Done |
| relativeMove step:angle | Done |
| calibrate, cruise_stop | Done |
| day/night, privacy, LED | Done |
| presets list/save/goto/del | Done |
| getBasicInfo | Done |
| KLAP transport | **Done** (v1+v2 handshake, /app/request) |
| Hub child devices | **Partial** — ChildID + controlChild + children list |
| Media/direct stream | Not yet (use RTSP) |

Agent order: **tapo-go → python → ONVIF**.

### 0.7.30-dev
- KLAP v1/v2 in `internal/tapo/klap.go` (python-kasa compatible)
- Login: probe KLAP → classic secure/legacy → KLAP fallback
- alarm + reboot actions

### 0.7.31-dev
- motion get/set, privacy get, alarm set, smart_track, children
- Client.ChildID + controlChild; Perform for raw `set`
- CLI: `netductor nvr tapo <host> <user> <pass> <action>`

### NVR MVP 0.7.32-dev — closed
- go2rtc yaml + example unit; storage backends; motion zones schema; TG night/privacy/calibrate
- Remaining: live hardware validation only

### 0.7.33-dev
- PathUnderRoot for clips; motion zones API; TUI NVR ops; docs/NVR-CODE-REVIEW.md

### Client RU-direct (Gosuslugi / banks)
- Shared list: `internal/vpn/ru_direct.go` (suffixes + keywords)
- Client sing-box + secondary relay use it; GEOIP RU rule-set on client
- `nd-oc.conf` + generated `shadowrocket-routing.conf` — DIRECT for RU/gov
- Docs: `docs/ru/SHADOWROCKET.md`

---

## Snapshot 2026-09-19 (0.7.34-dev)

- **RU-direct closed:** shared `internal/vpn/ru_direct.go`; client sing-box + secondary + `nd-oc.conf` / `shadowrocket-routing.conf`.
- **Deployed primary:** netductor + netductor-tg updated; `vpn client-config` for Pavel, Nelya, Mama.
- **Review:** [REVIEW-2026-09-19.md](REVIEW-2026-09-19.md) — code, security, refactor plan.
- **Open:** hardware e2e (OpenWrt/Tapo/MikroTik), durable domain/HTTPS redirect, release-asset discipline.
- **Not regressing:** do not remove client-side RU DIRECT in favor of “relay-only split”.

### 0.7.35-dev
- `paths.SecondaryDir` / DevicesFile; install writes secondary; TUI no lab IP; tests clientcfg; `scripts/build-release-local.sh`; GH release workflow_dispatch.

### 0.7.36-dev — relay name removed
- State: only `secondary/` (no `relay/` fallback).
- CLI: `netductor secondary` only (no `relay` alias).
- API: `/api/secondary/*` only.
- Node role/id prefix: `secondary` / `secondary-…`.

### Edge after primary reinstall
See [EDGE-REINSTALL.md](EDGE-REINSTALL.md) — pending queue only after enroll with valid bootstrap; old device_token does not auto-approve.

### 0.7.38-dev
- Edge LAN recovery page + recovery codes; site attach; CONTROL_ONLY on agent.

### 0.7.39-dev
- Edge recovery/register/set-site in TG, Admin UI, TUI; API /api/edge/recovery|register|set-site|export|import.

## 0.8.0 lock (2026-09-20)

- Version **0.8.0** tag `v0.8.0`
- Architecture: primary + secondary (VPN entry only); no full RU mirror
- Edge recovery: LAN page + recovery codes; CONTROL_ONLY; pending→approve
- Updates: primary from GitHub **Release** via TG; agents **not** auto-updated
- Docs: EN + `docs/ru/` (missing long plans have RU stubs pointing to EN)
- Review: docs/REVIEW-2026-09-20.md

### Post-0.8.0 review
See docs/REVIEW-2026-09-20-POST.md (security P0 recovery bind, SHA256 self-update, mTLS 8788).

### Hardening follow-up
Recovery LAN bind + SERVER_PIN; update SHA256; TG edge approve handlers + token mask.

---

## 0.8.1 lock (2026-09-20)

| Field | Value |
|-------|--------|
| Version | **0.8.1** · tags `v0.8.0`, `v0.8.1` |
| Repo | https://github.com/PavelNeyman/netductor |
| Primary (test) | may change; use keys/env not hardcode |
| Architecture | **primary** (control + VPN core + API/TG) + **secondary** (RU VLESS entry only) |
| Edge re-attach | LAN `http://<router>:7879/netductor-recovery` + recovery code → pending → approve · `CONTROL_ONLY` |
| Updates | Primary: TG Tools → Updates (GitHub Release + **SHA256SUMS**). Agents: **manual** `agent_update` only |
| Security post-review | [REVIEW-2026-09-20-POST](REVIEW-2026-09-20-POST.md) |
| Open work | [OPEN_ITEMS](OPEN_ITEMS.md) — mostly hardware/domain |

### Do not

- Auto-rollout agents on every primary release  
- Expose Admin/API to WAN without explicit operator decision  
- Treat secondary as full Lampac/bot mirror (reverted by design)  

### Next chat checklist

1. `git pull` · read OPEN_ITEMS + this lock  
2. Prefer release binaries for prod VPS; `main` for development  
3. Edge: recovery code + approve; never full site re-provision for re-bind  
4. Docs: EN + `docs/ru/` for user-facing changes


---

## 2026-09-20 — Workstation SSH + transport notes (v0.8.3)

### Operator SSH
- After bootstrap, **Mac** `~/.ssh/netductor_primary` is the operator key for primary / secondary / OpenWrt / MikroTik (best-effort).
- Primary **does not** generate `/root/.ssh/id_ed25519` when `authorized_keys` already has a key (DeployPrimary path).
- Secondary provision accepts `--operator-pubkey`; workstation deploy always passes Mac `.pub`.
- Devices do **not** maintain an SSH mesh for day-2 ops.

### Agent → primary transport
- **Admin** `:8787` stays localhost; use SSH local-forward from Mac.
- **Secondary agent:** prefer **mTLS** `:8789`; plain `:8788` is legacy/token-only on the wire.
- **Edge agent:** HTTP(S) + device token. Prefer HTTPS or **VPN path** to primary — do not treat plain public HTTP as confidential.
- Runtime control plane = agent heartbeat/poll, not primary→device SSH.

### Releases
- Tag **v0.8.3** includes deploy harden + docs; brew formula tracks release assets / `HEAD`.


---

## Plan: close plaintext control plane (2026-09-20)

### Done in v0.8.4 (secondary)
- [x] mTLS certs **auto-generated** (`mtls.EnsureAll` on install + serve; `EnsureClientFor` per node)
- [x] Secondary provision installs client material **over same SSH session** (no post-hoc primary→secondary scp needed after Mac-key-only)
- [x] `CoreAgentURL` = `https://primary:8789`
- [x] Plain `:8788` **off** by default; emergency only `NETDUCTOR_PLAIN_AGENT=1`
- [x] UFW: deny 8788; allow 8789; after secondary provision → **from secondary IP only**
- [x] Doctor: FAIL if 8788 exposed or mTLS missing/not listening

### Edge (next — do not block on VPN for control)
**Decision:** edge should reach primary on **public mTLS/HTTPS always**, not only via site VPN.
Reason: if VPN dies, control-plane via VPN-only would black-hole the router (no enroll refresh, no cmds, no recovery path except LAN recovery UI).

| Option | Pros | Cons |
|--------|------|------|
| Control only over VPN | No public agent port | Router lost to control when VPN down |
| **mTLS :8789 always (chosen)** | Works if VPN down; encrypted + client cert | 8789 public (restrict by IP/rate later) |
| Plain HTTP public | Simple | Rejected |

**TODO edge (phase 2):**
- [x] Edge-agent: load client certs from /etc/netductor-agent/mtls (v0.8.5)
- [x] Enroll + heartbeat on `https://primary:8789` (mTLS agent plane hosts edge+nvr APIs)
- [x] Workstation seeds `https://PRIMARY:8789`; plain :8787 rewritten to mTLS
- [x] Edge does not need `API_PUBLIC` — uses agent plane :8789
- [x] **Default:** per-agent IP allowlist on :8789 (`agent_allowlist` + ufw)

### Cert automation (code, not manual VPS)
- `internal/mtls.EnsureAll(ip)` — CA + server + default client if missing
- `EnsureClientFor(nodeID)` — per-secondary client cert
- Install calls EnsureAll; serve calls EnsureAll; provision pushes material in-band


### Done in v0.8.5 (edge mTLS)
- Agent plane :8789 serves edge + nvr device APIs (same mTLS as secondary)
- `netductor-agent` uses client certs; normalizes SERVER → https://host:8789
- DeployEdge issues per-device cert on primary and installs on router
- Control plane independent of site VPN

### v0.8.6
- Agent plane allowlist is **default** (not optional).


### Agent plane firewall

- Deny plain `:8788`
- Allow mTLS `:8789` (auth = client certificate; no IP allowlist)


---

## Current baseline — v0.8.19 (2026-09-20)

### Release
- Tag: **v0.8.19** · https://github.com/PavelNeyman/netductor/releases/tag/v0.8.19
- Assets: darwin/linux CLI, agent (amd64/arm/arm64/mipsle), tg, SHA256SUMS
- Homebrew Formula `0.8.19` · `brew reinstall netductor`

### Operator model

### Mac TUI deploy scenario (verified)

Intended flow: `brew install netductor` → `netductor tui --mode workstation` → Setup wizard:

1. **Primary** → `deploy.DeployPrimary` (password → Mac key in authorized_keys → binary from Releases → secrets → install/harden → SNI → fleet bootstrap) → saves `remote_host`/`remote_key` in `~/.config/netductor/tui.yaml`
2. **Secondary** → SSH to primary → `fleet provision-secondary --operator-pubkey` (Mac pub) + mTLS client material
3. **OpenWrt** → `DeployEdge` (agent binary + mTLS certs + `SERVER=https://primary:8789` + Mac pub harden)
4. **NVR/cameras** → CLI via primary after edge online
5. **MikroTik** → site wizard / RSC push + optional SSH harden

**Fixed in tree:** bubbletea tab wizard previously ran *local* `install` / `fleet provision-secondary` without operator pubkey and defaulted edge to `http://127.0.0.1:8787`. Now tab wizard **delegates** to the same `wizardPrimary/Secondary/OpenWrt` as the menu Setup wizard.

**Requirements on Mac:** `ssh`, `scp`, `ssh-keygen`, `curl`, **`sshpass`** (first password login).


- **Deployment centre** = Mac TUI (`netductor tui --mode workstation`)
- First SSH: password → install **Mac** `~/.ssh/netductor_primary` pub → password off (primary/secondary/OpenWrt; MikroTik best-effort)
- Day-2: **no device↔device SSH mesh**; agents → primary over **mTLS :8789**
- Admin UI: **localhost :8787** only (`ssh -L 8787:127.0.0.1:8787 primary`)

### Transport security (locked)
| Surface | Policy |
|---------|--------|
| Admin `:8787` | `127.0.0.1` · need `API_PUBLIC=1` + TLS to bind public |
| Agent plane `:8789` | mTLS (TLS1.3 + client cert) · ufw allow · **no IP allowlist** |
| Plain `:8788` | **off** unless `NETDUCTOR_PLAIN_AGENT=1` |
| Edge | same mTLS; works behind ISP NAT; independent of site VPN |
| SSH | key-only after bootstrap |

### UI i18n
- Admin web, TUI (deploy/forms/sites/sshhosts/menus), TG dict + NVR buttons: **EN/RU**

### Certs
- `mtls.EnsureAll` on install/serve · `EnsureClientFor(id)` on secondary/edge provision · material on router under `/etc/netductor-agent/mtls/`

---

## Security review snapshot (v0.8.19)

### OK
- Admin not world-open by default
- Agent plane encrypted + mutual TLS
- Bootstrap password path short-lived; operator key only after
- Recovery HTTP LAN-bound
- Camera `InsecureSkipVerify` limited to LAN Tapo/probes (documented)

### Residual risks
1. **`:8789` reachable from internet** — intentional for NAT edge; without client cert handshake fails, but port is probeable (rate-limit / fail2ban optional)
2. **Long-lived `edge_bootstrap_token`** — protect like a secret; prefer recovery codes for re-attach
3. **`NETDUCTOR_PLAIN_AGENT=1` / `API_PUBLIC=1`** — foot-guns if left on prod
4. **Hardcoded old download URLs** in some legacy docs/scripts (secondary self-update pin fixed to v0.8.19 in agent)
5. **UFW vs cloud SG** — host ufw does not replace provider security groups
6. **Admin session token** strength depends on `vpn session` issuance

### Not bugs
- VPN ports 443/4443/8443 public (product requirement)
- Lampac on localhost only

---

## Refactor plan (priority)

| P | Item | Why |
|---|------|-----|
| P1 | ~~Single version pin~~ **done** `internal/deploy.Release` + agent update URL | remaining: TG one-liner / docs/ru |
| P1 | Unify agent-plane mux construction (secondary+edge+nvr) in one `StartAgentPlane()` | serve.go / api_secondary split |
| P1 | ~~Tab wizard vs deploy wizard split~~ **done** (tab delegates to deploy wizards) | |
| P2 | ~~FormT dict~~ **done** for deploy wizard + shared keys | remaining long status lines may use TT |
| P2 | Admin i18n: remaining placeholders + dynamic JS strings | Polish |
| P2 | Drop residual “relay” naming in admin one-liner / docs → secondary | Consistency |
| P3 | ~~rate-limit :8789~~ **done** `httpx.PlaneLimiter` 180/min/IP | |
| P3 | Per-edge client cert rotation + revoke list | Long-term PKI |
| P3 | ~~Split agent main~~ **done** (nvr_cmds / openwrt_net / agent_ops) | |
| P3 | ~~version grep + CI~~ **done** (connector updated workflow) | |

Do **not** reintroduce IP allowlist for edge.

### v0.8.19
- mTLS revoke/rotate/list; doctor cert expiry WARN; plane IP ban after repeated 429
- Docs: MTLS.md · OPEN_ITEMS hardware-only remaining
