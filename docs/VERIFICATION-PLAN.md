# Netductor — phased verification plan

**Purpose:** systematic scenario + module review against real hardware constraints.  
**Progress:** check boxes as phases complete; do not skip marking.  
**Stop line at creation:** v0.9.191 (2026-10-02).  
**Rule:** EN ↔ RU **full semantic parity** (AGENTS §7).

Related: [AGENT_HANDOFF.md](AGENT_HANDOFF.md) · [OPEN_ITEMS.md](OPEN_ITEMS.md) · [AGENTS.md](../AGENTS.md)

---

## 0. How to use this file

1. Work **one phase at a time** (P0 → P1 → …).  
2. For each item: **review (code)** and/or **live test** as marked.  
3. Tick `[x]` only when done; add short note under **Progress log**.  
4. Hardware-blocked items stay `[ ]` with note `blocked: hardware`.  
5. New bugs → fix in code + release; reference tag in Progress log.

---

## 1. Hardware & environment matrix

| ID | Device / env | Role | Constraints (must not ignore) |
|----|----------------|------|-------------------------------|
| **H-P** | Primary VPS (abroad) | Control plane, API, TG, LE, git/CI/registry, Lampac optional | SSH **52222** key-only after harden; mTLS **:8789** not public by default; LE rate limits |
| **H-S** | Secondary VPS (RU) | VPN entry (VLESS Reality), agent, backup_pull peer | Thin node — **not** full mirror; agent → primary :8789 over SP; limited RAM/CPU |
| **H-Mac** | Operator Mac | `netductor-op` TUI/Web/CLI deploy | brew Formula; SSH keys in `~/.ssh`; credentials under `~/.netductor/` |
| **H-OW** | OpenWrt (Cudy etc.) | Edge agent, guest Wi‑Fi, optional LAN VPN | **mipsle** softfloat; **Dropbear** (no SFTP); **16 MB flash / ~128 MB RAM**; ash not bash; no `base64` on some builds; factory SSH **:22** password then harden |
| **H-Cam** | Tapo C200 | RTSP / NVR | LAN only; Tapo-specific API (not generic ONVIF-only); Wi‑Fi power |
| **H-MT** | MikroTik (+ optional RPi OpenWrt) | Site routing | No container on some ROS; prefer RSC/scripts + edge agent on RPi; not full OpenWrt parity |
| **H-Phone** | iOS (SR / Happ / INCY) | User VPN clients | White-list DPI; prefer **secondary** entry; QR/url buttons need redirect base |
| **H-Corp** | Mac/iPhone + corporate OpenConnect | Split with home VPN | Private routes via OC; internet via our tunnel — client profile sensitive |

**Live inventory (operator-maintained):** fill after each wipe.

| Host | IP / name | Notes |
|------|-----------|--------|
| Primary | `2.27.118.70` / `p*.nd.neyman.top` | |
| Secondary | `92.255.77.253` / `s*.nd.neyman.top` | |
| Redirect | `i*.nd.neyman.top:8443` | LE; not proxied CF |
| OpenWrt | (pending) | Cudy mipsle — edge deploy interrupted 0.9.190; fixed echo 0.9.191 |

---

## 2. Usage scenarios (end-to-end)

Each scenario lists **stages**. Verification maps modules to these stages.

### S1 — Greenfield fleet (Mac operator)

| Stage | Action | Success criteria |
|-------|--------|------------------|
| S1.1 | Install/update `netductor-op` (brew) | `version` matches release |
| S1.2 | Deploy **primary** (password first login → key, harden 52222) | doctor ok; API local; units active |
| S1.3 | Domain + LE (`p`/`i` hosts) | cert live; redirect `/healthz` |
| S1.4 | TG bot + admin id | `/menu` works; language RU/EN |
| S1.5 | Deploy **secondary** + svc-paths SP/PS | agent online; VLESS listen; peer ping |
| S1.6 | Optional addons (git, CI, registry, Lampac) | only if selected; no silent install |
| S1.7 | Credentials saved on Mac | `~/.netductor/credentials/*` |

**UIs:** Web Fleet wizard, TUI master, CLI `deploy primary|secondary`.

### S2 — Day-2 operator (no wipe)

| Stage | Action | Success |
|-------|--------|---------|
| S2.1 | Status / fleet health | primary + secondary versions truthful |
| S2.2 | VPN users: add / rename / access QR | QR + links; edge-* peers **hidden** from Users |
| S2.3 | Stack/update apply (manual confirm only) | no auto-rollback loop; prev/local consistent |
| S2.4 | Backup schedule + pull to secondary | age OK; peer has artifact |
| S2.5 | API public arm (TG) | temporary :8789; auto disarm |
| S2.6 | DNS block lists / tools | apply without false doctor alarms |

### S3 — Disaster recovery

| Stage | Action | Success |
|-------|--------|---------|
| S3.1 | Primary wipe | secondary still holds backup_pull |
| S3.2 | `recover --from-secondary` (or Mac restore path) | components reinstall by list; data restored |
| S3.3 | Post-recover LE re-issue (certs not in .ndenc) | redirect up |
| S3.4 | Secondary agent re-enroll / heartbeat | online; VPN works |
| S3.5 | Operator key available | SSH without password trap |

### S4 — OpenWrt edge first-boot (on-LAN)

| Stage | Action | Success |
|-------|--------|---------|
| S4.1 | Factory OpenWrt; Mac same L2/L3 | SSH :22 password (empty or known) |
| S4.2 | Probe arch → download agent (mipsle/…) | correct asset; cache ok |
| S4.3 | Provision agent + mTLS **without** harden | binary on router; material written |
| S4.4 | Network UCI staged (no reload) | ash-safe script; commit only |
| S4.5 | Guest staged (`--stage`) | UCI+nft; no mid-deploy reload |
| S4.6 | Harden (pubkey, SSH password off) | LuCI password = New root (unchanged by harden) |
| S4.7 | Reboot; new LAN IP if set | agent enroll → pending approval |
| S4.8 | Approve device; template apply (VPN/DNS) | private Wi‑Fi via secondary VLESS; soft WAN fallback |
| S4.9 | Guest flow | captive/desk grant; TTL; QR WIFI |

**Hardware risks:** Dropbear no SFTP; ash; flash size; radio0-only guest AP; half-state if abort mid-path.

### S5 — OpenWrt day-2 / offline

| Stage | Action | Success |
|-------|--------|---------|
| S5.1 | LuCI enable/disable/extend (agent + SSH LAN) | TTL auto-off; TG notify optional |
| S5.2 | agent_update from primary release | version matches; no full wipe |
| S5.3 | Recovery token page on LAN | enroll without re-flash |
| S5.4 | Offline pack / bootstrap-token | install without WAN on router |

### S6 — Guest Wi‑Fi commerce pattern

| Stage | Action | Success |
|-------|--------|---------|
| S6.1 | Guest SSID (visible default / optional hidden) | isolated zone; no LAN forward |
| S6.2 | Client joins → captive code | no internet until grant |
| S6.3 | Staff desk PIN / QR grant (default ~10 min, max 24 h) | nft MAC allow |
| S6.4 | Expiry | client blocked; re-grant possible |

### S7 — NVR / cameras

| Stage | Action | Success |
|-------|--------|---------|
| S7.1 | Discover leases on edge | find Tapo MAC/IP |
| S7.2 | Static lease + path to primary | RTSP reachable over VPN/LAN design |
| S7.3 | Record segments + rotation | disk not filled unbounded |
| S7.4 | UI list/download (Web/TG) | VPN-only access |
| S7.5 | Motion / PTZ / night (Tapo port) | hardware e2e |

### S8 — Site MikroTik + RPi

| Stage | Action | Success |
|-------|--------|---------|
| S8.1 | RPi OpenWrt agent as edge | same enroll model |
| S8.2 | MT routing scripts from operator | no unsupported containers |
| S8.3 | Site/location inventory | devices grouped |

### S9 — Client VPN UX

| Stage | Action | Success |
|-------|--------|---------|
| S9.1 | Default entry **secondary** | VLESS Reality works home + LTE when lists allow |
| S9.2 | Operator primary profile / sub toggle | RU exit when abroad (selected users) |
| S9.3 | SR work profile (OC + our VPN) | corp LAN + internet |
| S9.4 | Redirect url-buttons | https base set; no CF orange on i.* |

### S10 — Thin git / CI / registry (optional)

| Stage | Action | Success |
|-------|--------|---------|
| S10.1 | Install only if chosen | no silent fleet install |
| S10.2 | Repo list / pipeline trigger | op UI |
| S10.3 | Registry push/pull | auth scoped |

---

## 3. Module inventory → scenarios

| Module / plane | Path (approx.) | Scenarios | Review focus |
|----------------|----------------|-----------|--------------|
| **op** deploy | `internal/deploy`, `cmd/netductor-op` | S1, S4–S5 | SSH order, ash, Dropbear, abort vs warn |
| **op** Web/TUI | `internal/operator` | S1–S2, S4 | parity, prefill, no silent addons |
| **node** install | `internal/install` | S1, S3 | components list, harden, LE |
| **node** API | `cmd/netductor`, edge plane | S2–S5 | mTLS, rate limit, arm |
| **tg** | `cmd/netductor-tg` | S2, S9 | templates A/B/C, versions truth, topics |
| **agent** | `cmd/netductor-agent` | S4–S7 | guest stage, VPN soft fallback, luci TTL |
| **edgeagent** UCI | `internal/edgeagent` | S4 | DesiredUCI, ShellApplyStaged ash-safe |
| **edge** templates | `internal/edge` | S4.8 | peer `edge-*`, dns vpn, fallback |
| **vpn** users | `internal/vpn` | S2, S9 | hide edge peers; Reality |
| **secondary** / svcpaths | `internal/secondary`, `svcpaths` | S1.5, S3 | SP/PS only service traffic |
| **stack** / update | `internal/stack`, `update` | S2.3 | **manual only** apply/rollback |
| **backup** | install/backup paths | S2.4, S3 | COMPONENTS-driven; no binaries in archive |
| **guest** | agent guest + nft | S6 | TTL grants |
| **nvr** / tapo | `internal/nvr`, `tapo` | S7 | C200 specifics |
| **mikrotik** | `internal/mikrotik` | S8 | RSC generation |
| **dnsblock** | blocky | S2.6 | lists UI |
| **tlsle** / domain | `internal/tlsle`, `domain` | S1.3, S3.3 | re-issue after recover |
| **git/ci/registry** | optional | S10 | resource use on primary |

---

## 4. Phased work (execute in order)

### Phase A — Document & inventory (this file)

- [x] A.1 Create EN verification plan with scenarios + matrix  
- [x] A.2 RU semantic twin `docs/ru/VERIFICATION-PLAN.md`  
- [x] A.3 Link from handoff + OPEN_ITEMS  
- [ ] A.4 Fill live inventory table after next wipe  

### Phase B — Code review by module (no hardware)

Do **one module group per session**; mark when review notes written under Progress log.

- [x] B.1 `internal/deploy` (code review 2026-10-02; live open) + `edgeagent` (OpenWrt path 0.9.186–191)  
- [x] B.2 `cmd/netductor-agent` guest + luci + vpn client (code 2026-10-02)  
- [x] B.3 `internal/edge` templates / peers / enroll (code 2026-10-02)  
- [x] B.4 Primary install + harden + LE (code 2026-10-02)  
- [x] B.5 Secondary provision + svc-paths + backup_pull (code 2026-10-02)  
- [x] B.6 Stack/update policy (no autonomous version churn) (code 2026-10-02)  
- [x] B.7 TG navigation + version display + card templates (code skim 2026-10-02)  
- [x] B.8 op Web/TUI parity vs opcatalog (wiring 2026-10-02; live open)  
- [x] B.9 NVR/tapo/mikrotik stubs vs claimed UX (code present 2026-10-02; live F)  
- [x] B.10 Security pass: ports, mTLS, recovery, tokens (skim 2026-10-02; formal live C)  

### Phase C — Live dual-VPS smoke (no OpenWrt)

- [ ] C.1 Primary doctor + units  
- [ ] C.2 Secondary agent + VLESS  
- [ ] C.3 SP/PS health  
- [ ] C.4 TG menu + one VPN user QR  
- [ ] C.5 Backup + list  
- [ ] C.6 Manual stack status vs `VERSION` file  

### Phase D — OpenWrt e2e (Cudy / mipsle)

- [ ] D.1 Factory + deploy 0.9.191+ full path S4  
- [ ] D.2 Enroll approve + template VPN  
- [ ] D.3 Guest grant 10 min  
- [ ] D.4 LuCI on 1 h / off / SSH LAN  
- [ ] D.5 agent_update  
- [ ] D.6 Multi-radio guest if needed (code gap)  

### Phase E — Clients & split

- [ ] E.1 Phone secondary VLESS  
- [ ] E.2 Operator primary / sub policy  
- [ ] E.3 SR + OpenConnect profile  

### Phase F — NVR / site (when hardware ready)

- [ ] F.1 Tapo discover + record  
- [ ] F.2 MT+RPi site  

### Phase G — DR drill

- [ ] G.1 Secondary backup present  
- [ ] G.2 Primary recover path  
- [ ] G.3 LE + agent recovery  

---

## 5. Known gaps (track here)

| Gap | Impact | Phase |
|-----|--------|-------|
| Guest AP fixed **radio0** | SSID may miss if only radio1 | D / B.2 |
| No auto SSH to new LAN IP after stage | Operator must reconnect | D |
| LE not in backup | Post-recover certbot | G |
| OpenWrt half-state on mid-fail | Prefer factory retry | D |
| Version UI historically lied (stack prev) | Trust `netductor version` + file | C.6 / B.6 |
| Hardware e2e still open | OPEN_ITEMS | D–F |

---

## 6. Progress log

| Date | Phase item | Result |
|------|------------|--------|
| 2026-10-02 | A.1 | Plan created (EN); stop **0.9.191**; OpenWrt network stage ash paren fixed |
| 2026-10-02 | A.2 | RU plan added (semantic parity) |
| 2026-10-02 | B.1 | Code review deploy+edgeagent; residual radio0/default_radio |
| | | OpenWrt live test deferred (router returned) |

---

## 7. Session checklist for agents

Before coding from this plan:

1. Read stop line in handoff.  
2. Pick **one** unchecked Phase B/C/… item.  
3. Review or test; write Progress log row.  
4. Tick box; bump docs RU if needed.  
5. Do **not** mark hardware items done without device evidence.


---

## 8. Full product coverage matrix (must all be exercised)

Every row must eventually have a **review** and a **live** outcome (or explicit N/A).  
This is the complete functional surface of the project — primary, secondary, edge, cameras, operator UIs.

| Area | Capabilities | Scenarios | Live env | Review | Live |
|------|--------------|-----------|----------|--------|------|
| **Primary install** | dirs, harden SSH 52222, sing-box, blocky, API, redirect, units | S1 | H-P | [ ] | [ ] |
| **Primary domain/LE** | p/i hosts, certbot, redirect :8443 healthz | S1.3, S3.3 | H-P | [ ] | [ ] |
| **Primary TG bot** | menu, i18n, cards A/B/C, topics/alerts, arm API | S2 | H-P | [ ] | [ ] |
| **VPN users** | add/rename/disable, VLESS links/QR, hide edge-* | S2.2, S9 | H-P/S | [ ] | [ ] |
| **Redirect / profiles** | url buttons SR/Happ/INCY, nd-oc.conf, REDIRECT_BASE | S9.4 | H-P | [ ] | [ ] |
| **Secondary provision** | Reality, agent mTLS, harden, no full mirror | S1.5 | H-S | [ ] | [ ] |
| **Service paths SP/PS** | WG-over-WSS, health, agent on SP, recovery on PS | S1.5, S3 | H-P/S | [ ] | [ ] |
| **Backup primary** | COMPONENTS list, schedule, encryption .ndenc | S2.4 | H-P | [ ] | [ ] |
| **backup_pull secondary** | agent pull, peer storage, restore source | S2.4, S3 | H-S | [ ] | [ ] |
| **Recover primary** | from-secondary, two-pass components, key inject | S3 | H-P | [ ] | [ ] |
| **Stack / updates** | manual apply only, prev snapshot, secondary queue | S2.3 | H-P/S | [ ] | [ ] |
| **Doctor / probes** | health units, truthful versions | S2.1 | H-P | [ ] | [ ] |
| **DNS blocky** | lists UI, AdGuard registry URLs, reload | S2.6 | H-P | [ ] | [ ] |
| **Fleet / nodes** | registry, rename, roles primary/secondary/edge | S2 | H-P | [ ] | [ ] |
| **Locations / sites** | groups, inventory | S8 | H-P | [ ] | [ ] |
| **mTLS agent plane** | :8789 CIDR, enroll pending/approve, revoke | S4–S5 | H-P | [ ] | [ ] |
| **OpenWrt first-boot** | probe arch, agent put, net/guest stage, harden, reboot | S4 | H-OW | [ ] B.1 partial | [ ] |
| **OpenWrt template VPN** | edge peer, soft WAN fallback, dns vpn, LAN direct | S4.8 | H-OW | [ ] | [ ] |
| **Guest Wi‑Fi** | SSID/PSK/captive/desk, TTL grant, isolation | S6 | H-OW | [ ] | [ ] |
| **LuCI control** | enable 1h default, extend, SSH LAN path | S5.1 | H-OW | [ ] | [ ] |
| **Edge recovery HTTP** | LAN token page re-bind to new primary | S5.3 | H-OW | [ ] | [ ] |
| **Offline edge pack** | bootstrap without router WAN | S5.4 | H-Mac/OW | [ ] | [ ] |
| **NVR core** | cameras CRUD, segments, retention, storage | S7 | H-P | [ ] | [ ] |
| **NVR edge buffer** | leases, optional record on edge, upload | S7 | H-OW | [ ] | [ ] |
| **Tapo C200** | RTSP, PTZ, motion/night as implemented | S7 | H-Cam | [ ] | [ ] |
| **go2rtc / live** | paths, VPN-only view | S7 | H-P | [ ] | [ ] |
| **MikroTik** | RSC/scripts, site with RPi agent | S8 | H-MT | [ ] | [ ] |
| **Git / CI / registry** | optional install, list, pipeline, images | S10 | H-P | [ ] | [ ] |
| **Lampac addon** | install, status in addons UI | S1.6 | H-P | [ ] | [ ] |
| **Operator Web** | Fleet wizard, day-2 groups, Control | S1–S2 | H-Mac | [ ] | [ ] |
| **Operator TUI** | master, remote, i18n, settings yaml | S1–S2 | H-Mac | [ ] | [ ] |
| **Operator CLI** | deploy, edge, stack, recover parity | S1–S3 | H-Mac | [ ] | [ ] |
| **Client iOS** | secondary default, sub/primary policy | S9 | H-Phone | [ ] | [ ] |
| **Corp OC + VPN** | SR work profile | S9.3 | H-Corp | [ ] | [ ] |
| **Security posture** | no public admin, no footgun env, ports | B.10 | all | [ ] | [ ] |
| **i18n EN/RU** | TG, TUI, docs semantic parity | all | — | [ ] | [ ] |

**Rule:** “полный функционал” = все строки выше закрыты review+live (или N/A с обоснованием), не только OpenWrt.

---

## 9. Phase B.1 notes — `deploy` + `edgeagent` (2026-10-02)

**Reviewed:** `internal/deploy/edge.go`, `sshutil.go`, `internal/edgeagent/apply.go` (+ tests), guest stage hooks in agent (cross-ref).

### Aligned with design

| Topic | Finding |
|-------|---------|
| Order | Provision (no harden) → network stage → guest `--stage` → harden → reboot |
| SSH auth | Prefer password while set (`usePass`); after New root, `sshPass` updated |
| Staged UCI | `ShellApplyStaged` commits only; no network/wifi reload mid-SSH |
| Ash safety | Echo without bare `()` (0.9.191); values single-quoted for `uci set` |
| Dropbear | Historical: ssh stdin / no SFTP (0.9.186–188) |
| Abort | Network/guest errors fail deploy (not soft `done`) |
| LuCI vs harden | Harden is SSH-only; root pass set at provision for LuCI |

### Residual risks / gaps

| Risk | Severity | Follow-up |
|------|----------|-----------|
| Guest wireless forced `device=radio0` | High on multi-radio / odd board | B.2 / D.6 probe radios |
| `wireless.default_radio0/1` names | Med if device uses non-default iface names | Probe `uci show wireless` |
| Multi-line remote script via `ssh … cmd` | Low after ash fix; SSID with `'` handled via quote | Keep quoting discipline |
| `EnqueueCmd(apply_template)` at first-boot | Low — may no-op until approve | S4.8 day-2 explicit apply |
| Harden failure only **warn** | Med — password SSH may remain | Consider abort option |
| No auto reconnect to new LAN IP | Ops | Document only |
| Flash/RAM on Cudy | High if agent+ffmpeg+guest | NVR buffer policy on edge |

### Verdict B.1

Code path **matches** frozen first-boot story for OpenWrt **after 0.9.191**. Not a substitute for **D.** live e2e.  
**B.1 review (code): done.** Live still open.


---

## 10. Phase B.2–B.6 + security skim (2026-10-02)

### B.2 Agent — guest, LuCI, VPN client

| Topic | Status | Notes |
|-------|--------|-------|
| Guest zone | OK design | `forward=REJECT`, no guest→wan section; internet only via nft MAC set |
| Guest VPN bypass | OK | separate routing so guest stays on ISP WAN |
| Guest radio | **GAP** | still `wireless.guest24.device=radio0` |
| LuCI TTL | OK | default 1h parse; enable/extend/disable; until file + tick expected in main loop |
| VPN template apply | OK | writes vless; `enabled=false` / empty vless → stop netductor-vpn |
| Soft fallback | OK | urltest proxy+direct; `fallback=block` disables soft |
| TUN fail → socks | OK | retry path in agent_ops |
| DNS mode vpn | OK | in ClientOpts / sing-box JSON |
| Private direct | OK | private CIDRs + primary host direct in vless_client |

**B.2 code review: done.** Live guest/LuCI/VPN on Cudy open.

### B.3 Edge templates / peers / enroll

| Topic | Status | Notes |
|-------|--------|-------|
| Enroll pending→approve | OK | tests `TestEnrollApproveFlow`; rate limit AllowEnroll |
| Peer name `edge-<id>` | OK | `vpnclient.go`; Users filter `IsEdge` / prefix |
| TemplateWithVPN merge | OK | does not clobber fallback/dns/mode (policy tests) |
| Recovery token bind | Present | `recovery.go` pending device path |

**B.3 code review: done.**

### B.4 Primary install / harden / LE

| Topic | Status | Notes |
|-------|--------|-------|
| COMPONENT install order | OK | hardening, sing-box, blocky, vpn-users, api, tg, backup, redirect |
| SSH harden | OK | port 52222, pubkey; keys before password off |
| Redirect | OK | LE paths; EnsureRedirectRunning; post-recover InstallRedirect / reissue_le |
| LE not in .ndenc | By design | recover must re-issue (matrix row) |
| API bind | OK | non-local API refused; API_PUBLIC removed from serve |

**B.4 code review: done.**

### B.5 Secondary / svc-paths / backup_pull

| Topic | Status | Notes |
|-------|--------|-------|
| Offsite SCP removed | OK | backup_peer errors; only agent backup_pull |
| Queue backup_pull | OK | on backup; WaitForBackupPull soft timeout |
| Agent plane :8789 | OK | mTLS; ufw service CIDRs; api-public arm separate |
| Plain :8788 | Removed | product path mTLS only |
| Secondary upgrade | Manual | stack does **not** auto-upgrade secondary |

**B.5 code review: done.**

### B.6 Stack / update policy

| Topic | Status | Notes |
|-------|--------|-------|
| Apply lock | OK | TryAcquireApplyLock; ScheduleApply via systemd-run |
| Auto-rollback on health | **Disabled for downgrade** | failed units → alert, binaries **kept**; comment vs ancient prev 0.9.121 |
| Manual rollback | OK | explicit restore last-good |
| Watchdog | OK | restart failed units only — **does not** change version |
| Secondary | Not auto | operator enqueue |

**B.6 code review: done** (matches “manual version changes only”).

### B.10 Security skim (partial)

| Control | Status |
|---------|--------|
| Operator API not public by default | OK |
| Agent :8789 mTLS + ufw CIDR | OK |
| Arm API public TTL | OK (install/api_public.go) |
| Guest isolation nft | OK design |
| Footgun env | Reduced (API_PUBLIC removed from serve) |

Full port audit still **B.10** formal pass.

---

### B.7 TG (code skim)

| Topic | Status | Notes |
|-------|--------|-------|
| Stack HTML | Present | `stack.FormatHTML` in handlers |
| Versions legend | Present | handlers_versions legend line |
| Card templates A/B/C | Partial product debt | historical UX polish; not re-audited screen-by-screen this pass |
| Version truth | Depends on stack Collect + binary | B.6 keeps binaries on failed health |

**B.7:** structural OK; full screen audit still optional polish (not blocking architecture).

### B.8 Web/TUI opcatalog

| Topic | Status |
|-------|--------|
| Web exposes `opcatalog.Groups` / ForSurface("web") | OK in `operator/serve.go` |
| Day-2 groups | Catalog-driven |
| Full action parity every screen | Tracked in matrix §8; no exhaustive UI crawl this pass |

**B.8:** wiring OK; live parity checks remain C/Web.

### B.9 NVR / Tapo / MikroTik

| Topic | Status | Notes |
|-------|--------|-------|
| NVR recorder + retention | Code present | tests retention age/max GB |
| Clip tokens | Present | Issue/Redeem |
| Motion config | Present | window helper |
| ONVIF PTZ | Present | generic SOAP |
| Tapo KLAP | Package `internal/tapo` | C200 e2e hardware |
| Agent nvr cmds | record/upload/leases | edge path |
| MikroTik | RSC + SSH PushRSC + harden | site model; no containers required |

**B.9 code: present.** Live F-phase required for cameras/MT.


