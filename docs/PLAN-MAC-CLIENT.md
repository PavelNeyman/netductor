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

- [ ] Stop installing/shipping `runtime/api/admin` by default (or `NETDUCTOR_LEGACY_ADMIN_UI=1` only)
- [ ] `serve` log: API-only; no admin path in happy path
- [ ] TG bot remains (not a VPS web UI)

### P4 — Polish

- [ ] Single design language Installer ↔ Control
- [ ] Auto-tunnel when Control opens
- [ ] Mobile day-2 later (same APIs, not bootstrap)

## Rules

1. New features: **API on node first**, then Mac client screens — never new VPS HTML admin.
2. Deploy/secrets only on Mac (`internal/operator`).
3. Do not bind node admin/API to `0.0.0.0` without existing public-API guards.

## Out of scope

- Hosting installer on primary
- Public VPS web UI
- Replacing Telegram bot
