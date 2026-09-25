# Plan: Mac client = UI · Node = API only

**Status:** locked · **Baseline:** v0.9.2+  
**Decision:** Primary/secondary are **servers** (API + vpn + agents). Human UI runs on **Mac** via `netductor-op` (localhost WebUI / TUI).  
**Not a product goal:** rich admin UI hosted on the VPS (optional legacy only).

## Target

```text
┌─────────────────────────────┐
│  Mac: netductor-op          │
│  localhost WebUI + TUI      │
│  ├── Installer (deploy)     │
│  ├── Control (day-2 admin)  │
│  └── Settings (nodes/keys)  │
└─────────────┬───────────────┘
              │ SSH and/or VPN + API
              ▼
┌─────────────────────────────┐
│  Node: netductor            │
│  JSON API :8787 (loopback)  │
│  mTLS agent :8789           │
│  vpn / backup / recovery    │
│  NO product web admin       │
└─────────────────────────────┘
```

Transport to `:8787`:

- Prefer **SSH local forward** from op (`ssh -L 8787:127.0.0.1:8787`) or built-in tunnel helper.
- Or access while operator VPN is up if API is exposed only on VPN path (future).

## Migration phases

### P0 — Lock & docs

- [x] This plan + ARCHITECTURE / HANDOFF / OPERATOR-PLAN
- [x] ADMIN.md marked legacy (VPS static UI not the product path)

### P1 — Mac client shell

- [x] WebUI tabs: **Installer | Control | Settings**
- [x] Settings: node profiles (localStorage)
- [x] Control: fetch `/health` + bot-status via API base
- [x] `netductor-op tunnel --host` SSH `-L` to node :8787

### P2 — Control parity (API-driven)

- [x] VPN users list / create (Control + /v1/node proxy)
- [x] Nodes list + self
- [x] Edge pending / devices / approve / deny
- [x] Credentials collect stays Installer-adjacent

### P3 — Node pure API

- [x] Stop installing/shipping admin static by default (`NETDUCTOR_LEGACY_ADMIN_UI=1` only)
- [x] `serve` API-only; `/admin` → 410 unless LEGACY=1
- [x] TG bot remains (not a VPS web UI)

### P4 — Polish

- [x] `POST /v1/tunnel/start|stop`, `GET /v1/tunnel/status`
- [x] `POST /v1/session/issue` from WebUI (stores token in Settings)
### P4 — Polish (detail)

- [x] Shared WebUI shell Installer ↔ Control ↔ Settings
- [x] Auto-tunnel when Control opens (+ Start/Stop + badge)
- [ ] Mobile day-2 later (same APIs, not bootstrap) — deferred

## Rules

1. New features: **API on node first**, then Mac client screens — never new VPS HTML admin.
2. Deploy/secrets only on Mac (`internal/operator`).
3. Do not bind node admin/API to `0.0.0.0` without existing public-API guards.

## Out of scope

- Hosting installer on primary
- Public VPS web UI
- Replacing Telegram bot


## P5 — Control API coverage + tables (0.9.6)

- [x] Control subsections: Overview, VPN, Nodes, Edge (+guest), NVR, Git/Registry, Backup, Probes
- [x] Table renderer for array/object API responses (raw JSON toggle)
- [x] Deeper POST forms (NVR PTZ, recorder, git pipeline, registry ensure/stop)
- [ ] Mobile day-2 — deferred

### Tunnel without key / without BatchMode

Auto-tunnel uses **SSH with a private key** (`BatchMode=yes` = no interactive password/prompt).  
Without a key on the Mac there is nothing non-interactive to open `-L` safely from a headless `op` process.

Options later (not default):

| Approach | Notes |
|----------|--------|
| **ssh-agent** | Key unlocked once; BatchMode still works |
| **Password SSH** | Needs sshpass/expect; weaker; not product default |
| **VPN-only path** | When operator VPN is up, API on a private path — no tunnel |
| **Manual tunnel** | CLI still works |

### In-process GUI

Today: **localhost WebUI** in the browser + **TUI** in the terminal — both talk to the same `internal/operator` / node API.  
“In-process GUI” would mean embedding a native window (WebView/Wails/Fyne) **inside** the binary so no external browser is needed. Same backend; different shell. **Not required** while `operator serve` + browser works; optional polish later.


## Path selection (0.9.7)

1. If `api_base` `/health` already OK → **direct** (existing tunnel or API via VPN path) — no new SSH.
2. Else SSH tunnel: **VPN host first** if `prefer_vpn` and TCP to SSH port works, else public `primary_host`.
3. Auth always **SSH key** (+ agent for passphrase).

## Auth: password vs key

| Phase | Auth |
|-------|------|
| **First deploy** (Installer primary/secondary) | **Password** on VPS once — inject operator SSH key, then harden (password off) |
| **Day-2** tunnel / session / Control | **SSH key only** (`BatchMode`); passphrase via ssh-agent |
| Password for day-2 tunnel | **Not used** — not needed after first deploy |

Domain + LE: set in Installer / `domain` flags; certbot on primary as before (see DOMAIN.md).


## P8 — Max API coverage + i18n (0.9.9)

- [x] Control buttons for operator-session APIs (GET/POST forms)
- [x] Advanced generic method+path+JSON
- [x] EN/RU chrome (tabs, settings labels)
- [x] Docs archive of obsolete reviews/plans
