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

- [ ] B.1 `internal/deploy` + `edgeagent` (OpenWrt path 0.9.186–191)  
- [ ] B.2 `cmd/netductor-agent` guest + luci + vpn client  
- [ ] B.3 `internal/edge` templates / peers / enroll  
- [ ] B.4 Primary install + harden + LE  
- [ ] B.5 Secondary provision + svc-paths + backup_pull  
- [ ] B.6 Stack/update policy (no autonomous version churn)  
- [ ] B.7 TG navigation + version display + card templates  
- [ ] B.8 op Web/TUI parity vs opcatalog  
- [ ] B.9 NVR/tapo/mikrotik stubs vs claimed UX  
- [ ] B.10 Security pass: ports, mTLS, recovery, tokens  

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
| | | OpenWrt live test deferred (router returned) |

---

## 7. Session checklist for agents

Before coding from this plan:

1. Read stop line in handoff.  
2. Pick **one** unchecked Phase B/C/… item.  
3. Review or test; write Progress log row.  
4. Tick box; bump docs RU if needed.  
5. Do **not** mark hardware items done without device evidence.
