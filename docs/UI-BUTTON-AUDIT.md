# Web / op UI — button audit plan

**Status:** active checklist for operator review (human walkthrough).  
**Scope:** `netductor-op serve` web UI (sidebar, Deploy, Operate, Settings, Result, Jobs).  
**Related:** Result dock UX, jobs → Deploy log (Log tab in Result still pending).  
**Version note:** catalog buttons may also load from `/v1/catalog`; static fallback is `BTN` in `internal/operator/web/app.js`.

---

## How to review

1. Walk **left → right** in the sidebar.
2. For each action: compare **Expected** below with what you see (ok / 401 / empty / unclear).
3. Mark each item: **keep / remove / rename / move**.
4. Note where output went: **Result** vs **Deploy log** vs **Jobs**.

**Before Operate:** Settings → primary host + SSH key → **Tunnel Start / ensure** → **Issue session** (if 401).

Feedback format:

```text
section → button → actual result / unclear / keep|remove|rename|move
```

---

## Global chrome

| UI | Purpose | Expected |
|----|---------|----------|
| Collapse sidebar | Icons-only sidebar | Narrow rail |
| Result (header / status bar) | Toggle Result dock | Dock shows/hides |
| Jobs | Operator jobs (deploy, offline-prep, …) | Right panel; Refresh / Running only |
| Status bar: API / secondary / paths / channel | Health summary | Dots/text; not full API calls |
| Theme / language | Light-dark, EN\|RU | Switch without full reload |

---

## Servers

| Action | Expected |
|--------|----------|
| Primary/secondary cards (with tunnel + session) | Hostname, role, short metrics |
| Click card | Detail / node actions |
| Back | Return to list |

Without session: empty or offline placeholder.

---

## Sites

| Action | Expected |
|--------|----------|
| Refresh | Pull edge devices from primary |
| + Add site | Opens OpenWrt wizard (Deploy → edge) |
| Site card | Topology / drill-in |

---

## Deploy (Installer)

### Fleet

| Button | Expected |
|--------|----------|
| **Fleet deploy** | Job: primary (+ optional secondary/build); log in **Deploy log / Jobs** |

### Primary / Secondary / Build-host

| Button | Expected |
|--------|----------|
| **Deploy …** | Long job, SSH from Mac; log in Deploy log |

### OpenWrt (edge)

| Button | Expected |
|--------|----------|
| **Prepare offline pack** | Job `edge.offline_prep`; cache agents/token; PARTIAL → retry missing |
| **Probe device** | SSH to router → device card (model, LAN, radios, …) |
| Wizard steps | Modules → Dry-run → **Provision edge** |
| **Dry-run plan** | Plan only, no apply |
| **Provision edge** | Provision job + reboot; log in Deploy log |
| Storage: List / Backup / Restore / Format+restore / Mute / Unmute / Offline pack on primary | Day-2 USB/extroot via agent/API |

### MikroTik (site)

| Button | Expected |
|--------|----------|
| **Save site / push** | RSC / site shell push |

### NVR

| Button | Expected |
|--------|----------|
| Add camera / Run / related | Cameras, leases (session required) |

### Credentials

| Button | Expected |
|--------|----------|
| **Collect** | Collect credentials/keys for host |

---

## Operate

### Always on top (tunnel / session)

| Button | Purpose | Expected |
|--------|---------|----------|
| **Start / ensure** | Path to primary API | `direct` or SSH `-L`; badge update |
| **Stop** | Kill SSH local forward | Tunnel down |
| **Issue session** | API token → `~/.netductor/node_session` | ok + file written |
| **Load session file** | Load existing token | Session filled |

**Note:** VLESS may make API reachable as `direct`; session is still required for most `/api/*` (401 without it).

### Overview

| Group | Purpose | Expected in Result |
|-------|---------|-------------------|
| health / status / overview / doctor / domain / bot / metrics* | API alive, summary, doctor | JSON/tables |
| stack status / apply / rollback / watchdog | Stack version on primary | apply = node update path |
| addons / lampac | Containers | List |
| sni / presets | Reality SNI | Current + presets |
| sessions | Operator sessions | List |
| update status / latest / gh-token | Updates | Versions |
| release local / tags / mirror / build / status / prune | Local releases on primary | Build/mirror (use carefully) |

### VPN / users

| Action | Expected |
|--------|----------|
| Users list / refresh links | VLESS users + links |
| Add / enable / disable / revoke / link | User CRUD |
| Policy load/save + presets media/full/none | `allow_internet` + services |
| Services / Policy apply | Service catalog; sing-box rewrite |

### Nodes / fleet plane

| Action | Expected |
|--------|----------|
| nodes / self / secondary status+links | Nodes, secondary |
| ssh-hosts / clear | Known SSH hosts |
| mTLS certs / rotate / revoke | Agent-plane certs |
| sites list / push / rsc | Sites |
| sec-cmd / sync / exit | Secondary commands |
| hostname / journal / restart | Node diagnostics |

### Edge (day-2)

| Actions | Expected |
|---------|----------|
| Pending / devices / metrics / act / bind / cmd / guest / template / recovery / backup / export / set-site / vpn get·set / policy | Enroll, day-2 edge, guest, policy |

### NVR / media

| Actions | Expected |
|---------|----------|
| cams, leases, live, PTZ, rec start/stop, clip token, wifi, wizard | Cameras if NVR enabled |

### Git / registry / build

| Actions | Expected |
|---------|----------|
| projects list/add/sync/build/ship, ship jobs, migrate, mirror, vps-queue, build-host status, artifacts | CI on primary/build-host |

### Backup / DNS / probes / incidents

| Actions | Expected |
|---------|----------|
| backup schedule, DNS set, cleanup, probes, audit, devices locator, canary, smoke, incident | Schedules, DNS, incident packs |

### SSH allowlist / reboot

| Action | Expected |
|--------|----------|
| Inventory / probe secondary / allowlist get·save·merge·enforce | SSH keys |
| Reboot challenge → confirm | Two-step reboot with code |

### Rooms

| Action | Expected |
|--------|----------|
| List / upsert / delete / photo | Site rooms |

### Advanced

| Action | Expected |
|--------|----------|
| Method + path + body **Send** | Raw call to any `/api/...` |

---

## Settings

| Control | Purpose |
|---------|---------|
| Primary host / VPN host / prefer VPN | SSH target and `prefer_vpn` |
| Key, user, API port/base | Tunnel + session |
| Secondary host | Form prefills |
| **Save** | `operator.conf` + localStorage |
| GitHub token Save / Clear / Refresh | Token on primary for releases |

---

## Result dock

| UI | Expected |
|----|----------|
| Output | Tables/summary of last API call |
| History | Previous calls |
| Position bottom / right / top | Right: **⋮ only** |
| Mode panel / card / both | Dock and/or inline under button |
| JSON / Clear / Hide | Raw JSON; clear; hide dock |

**Pending product work:** third tab **Log** for job stdout (offline-prep, provision); today those streams go to **Deploy log** + **Jobs**, not Result Output.

---

## Suggested walk order (minimal)

1. Settings + Tunnel + Session  
2. Servers / Sites (layout only)  
3. Operate → Overview: health, doctor, status, metrics  
4. VPN: users + policy load  
5. Nodes: secondary status  
6. One long Deploy job: offline-prep → **Jobs + Deploy log**  
7. Everything else only if you use it; mark candidates for remove / Advanced  

---

## Code anchors

| Item | Location |
|------|----------|
| Static Operate buttons | `internal/operator/web/app.js` → `BTN` |
| Special form actions | `special[...]` in same file |
| Optional live catalog | `GET /v1/catalog` → `loadCatalogButtons` |
| Long jobs | `streamPost` → `/v1/jobs` (`edge.offline_prep`, `edge.provision`, …) |
| Deploy log DOM | `#log`, `#deployLogSec` |
| Result dock | `#resultDock`, `showControl` |

---

## Outcomes to record after audit

- Buttons to **remove** or hide behind Advanced  
- Labels that need rename (RU/EN)  
- Groups to merge (e.g. release/* under Updates only)  
- Layout breakage (cards overflow) — screenshots + section name  
- Confirm Log-in-Result design (see handoff / OPEN_ITEMS when implemented)
