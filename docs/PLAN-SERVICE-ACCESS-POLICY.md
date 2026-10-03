# Plan: Service access policy (VPN users + OpenWrt routers)

**Status:** design / implementation plan (not fully coded)  
**Audience:** any agent or developer implementing the feature end-to-end  
**Related:** [PORTS.md](PORTS.md), [CLIENT-PROFILES.md](CLIENT-PROFILES.md), [EDGE-AGENT.md](EDGE-AGENT.md), [ARCHITECTURE-FREEZE.md](ARCHITECTURE-FREEZE.md)  
**RU:** [ru/PLAN-SERVICE-ACCESS-POLICY.md](ru/PLAN-SERVICE-ACCESS-POLICY.md)

Semantic parity with the Russian document is mandatory (same meaning, not word-for-word).

---

## 1. Goal

Operators must be able to grant **per-subject** access to **internal services** reached over the existing VLESS path (and future paths), using **checkboxes in all UIs**, without mixing router management into the human VPN user list.

| Subject type | Examples | UI home |
|--------------|----------|---------|
| **VPN user** | operator, family phone, Apple TV profile | Users → card → Access / Policy |
| **Edge router** | Cudy / OpenWrt site | Routers / Fleet → device card → Policy (separate section) |

Do **not** put router peers (`edge-*`) into the normal Users list. Do **not** drive router policy only via opaque JSON templates forever — templates remain the *backend* merge target; the *operator surface* is the catalog + checkboxes.

---

## 2. Non-goals (v1)

- Per-device ACL inside one shared UUID (one UUID = one policy).
- Trusting only client-side split configs (Shadowrocket rules) as enforcement.
- Publishing Lampac/git/registry on the public WAN.
- Merging “guest Wi‑Fi TTL grant” with this catalog (guest stays its own feature).

---

## 3. Core model

### 3.1 Service catalog (global, on primary)

Stored under something like `/var/lib/netductor/service-catalog.json` (exact path: follow `PATHS.md` / `ndconfig`).

```json
{
  "version": 1,
  "services": [
    {
      "id": "internet",
      "title": "Internet egress",
      "kind": "egress",
      "description": "General non-RU / default proxy path"
    },
    {
      "id": "lampac",
      "title": "Lampac",
      "kind": "internal",
      "endpoints": [{"network": "service", "addr": "10.88.0.10", "port": 9118, "proto": "tcp"}],
      "publish": {"mode": "service-net"}
    },
    {
      "id": "git",
      "title": "Git",
      "kind": "internal",
      "endpoints": [{"network": "service", "addr": "10.88.0.11", "port": 2222, "proto": "tcp"}]
    },
    {
      "id": "registry",
      "title": "OCI registry",
      "kind": "internal",
      "endpoints": [{"network": "service", "addr": "10.88.0.12", "port": 5000, "proto": "tcp"}]
    },
    {
      "id": "nvr",
      "title": "NVR / go2rtc",
      "kind": "internal",
      "endpoints": []
    }
  ]
}
```

Rules:

- `id` is stable, lowercase, `[a-z0-9_-]{1,32}`.
- Adding a product component via `netductor install <x>` **should register** a catalog entry (or ship a default entry in code).
- Removing a component **disables** the entry (soft) so UI still shows history but policy apply ignores missing endpoints.
- `kind=egress` is special: controls default internet path, not a fixed VIP.

**Service-net:** preferred enforcement plane is a small internal net on primary (e.g. `10.88.0.0/24`) where internal apps bind or are DNATed from loopback. Until service-net exists, v1 may use `127.0.0.1` + sing-box `direct` only for allowed users via tun/loopback rules — document the interim clearly in code comments.

### 3.2 Policy object

Shared shape for users and routers:

```json
{
  "subject_type": "vpn_user" | "edge",
  "subject_id": "operator" | "appletv-salon" | "edge-cudy-hall",
  "allow_internet": true,
  "services": ["lampac", "git"],
  "services_mode": "list" | "all",
  "updated_at": "RFC3339",
  "updated_by": "tg|web|cli|system"
}
```

- `services_mode=all` ⇒ all catalog `kind=internal` (operator preset).
- Empty `services` + `allow_internet=false` ⇒ deny useful access (break-glass only).
- Default for **new human users**: `allow_internet=true`, `services=[]` (no internal until checked).
- Default for **new edge devices** after approve: policy from edge template / site preset (often `allow_internet=true` via VLESS uplink + soft WAN fallback — see edge VPN plan), internal services opt-in.

### 3.3 Presets (optional but recommended)

| Preset id | Meaning |
|-----------|---------|
| `full` | internet + all internal |
| `media` | internet optional + `lampac` (+ later `nvr`) |
| `dev` | internet + git + registry |
| `edge-default` | internet via secondary VLESS + soft WAN fallback; no internal admin services |

UI: “Apply preset” then fine-tune checkboxes.

---

## 4. Enforcement (must implement for checkboxes to be real)

Client apps cannot be trusted. Enforcement lives on **primary** (and secondary only if a service is published there — freeze says services stay on primary).

### 4.1 sing-box

1. Map VLESS user UUID / name → `subject_id`.
2. Route rules (order matters):
   - Private / service-net destinations: allow only if policy lists that service; else **reject**.
   - `allow_internet=false`: reject default egress (still allow primary↔secondary service paths used by the stack itself — never break SP/PS).
3. Prefer match by **auth user** (sing-box route `auth_user` / equivalent on current sing-box version). If unstable, fallback: dedicated inbound tag per policy group (fewer groups than users).
4. Edge routers: traffic identified by their dedicated VLESS peer (`edge-<device_id>`), **not** shown in Users; same route machinery.

### 4.2 Publish path for internal apps

- Today many bind `127.0.0.1` (see PORTS.md). Policy access requires either:
  - **A:** bind/publish on service-net VIP, or
  - **B:** sing-box `direct` + `route` to 127.0.0.1 from tun for allowed users only.
- Prefer **A** long-term. Document chosen approach in `ARCHITECTURE.md` when implemented.

### 4.3 Second line (optional v1.1)

nft/iptables on primary: default drop to service ports from tunnel interface unless mark set by sing-box — only if sing-box alone is insufficient.

---

## 5. API surface (node + op)

All actions must appear in **opcatalog** and be reachable from Web, TUI, CLI; TG gets operator-safe subset.

| Method | Path / CLI | Purpose |
|--------|------------|---------|
| GET | `/api/services` | list catalog |
| POST | `/api/services` | add/update service (admin) |
| DELETE | `/api/services/{id}` | soft-disable |
| GET | `/api/vpn/users/{id}/policy` | get user policy |
| PUT | `/api/vpn/users/{id}/policy` | set checkboxes |
| GET | `/api/edge/devices/{id}/policy` | get router policy |
| PUT | `/api/edge/devices/{id}/policy` | set router checkboxes |
| POST | `/api/policy/apply` | rebuild sing-box routes + reload (or automatic on PUT) |
| CLI | `netductor services list\|set` | catalog |
| CLI | `netductor vpn policy get\|set <user>` | user policy |
| CLI | `netductor edge policy get\|set <device>` | router policy |

Idempotent apply: writing policy always converges sing-box config; no manual “don’t forget reload”.

---

## 6. UI requirements

### 6.1 Separation

- **Users** section: human/device VPN profiles only (no `edge-*`).
- **Routers / Edge** section: devices + **Policy** tab with the same checkbox component fed by the same catalog.
- Do not reuse “Users → Access → VLESS QR” screen for router policy.

### 6.2 Checkbox UX

- One row per catalog service: title + short description + toggle.
- Separate toggle: **Internet egress**.
- Show effective preset name if any.
- After save: short confirmation + “routes applied” or error from apply.
- TG: follow [TG-UI-PATTERN](TG-UI-PATTERN.md) templates A/B/C; toggles as in-message controls where possible; nav under message.

### 6.3 Parity

Web (op embed), TUI, CLI, TG must all be able to:

- list catalog  
- edit user policy  
- edit edge policy  

If TG is too narrow for catalog admin, catalog CRUD may be Web/TUI/CLI only; **policy toggles** still required on TG for users and routers.

---

## 7. OpenWrt / edge specifics

| Topic | Rule |
|-------|------|
| Identity | Dedicated VLESS user `edge-<device_id>` (or stable id), hidden from Users list |
| Default traffic | Private Wi‑Fi clients → VLESS to secondary + **soft WAN fallback** (existing edge VPN direction) |
| Internal services | Opt-in checkboxes (e.g. allow LAN to reach Lampac via tunnel only if checked) |
| Templates | Keep JSON templates for network/wifi/guest; **policy** is separate object merged at apply time |
| UI | Device card → Policy (catalog checkboxes), not only “edit template JSON” |
| LuCI / guest Wi‑Fi | Unrelated controls; do not fold into service catalog |

Implementation order for edge policy:

1. Ensure peer exists and is filtered from Users formatters (already partially done via `edge-` prefix).
2. Attach `policy` blob on device record.
3. Generate route rules for that auth user.
4. Replace pure-template DNS/VPN knobs in UI with checkboxes where they map to catalog (`internet`, future `dns-filter`, etc.).

---

## 8. Implementation phases (checklist for agents)

Mark each item done in PR + OPEN_ITEMS / this doc.

### Phase P0 — data model

- [x] Catalog file + load/save + validation  
- [x] Default catalog seed (`internet`, `lampac`, `git`, `registry`, `nvr` stubs)  
- [x] Policy fields on vpn user + edge device records  
- [x] Migration: existing users → allow_internet=true, services=[]  
- [x] Unit tests for validate/merge  

### Phase P1 — enforcement

- [ ] Service-net design note + minimal VIP publish for Lampac (or interim loopback route)  
- [ ] sing-box route generation from policies  
- [ ] Apply on policy change + on `vpn apply` / stack apply  
- [ ] Tests: golden config snippets for full / media / deny-internal  

### Phase P2 — API + CLI

- [x] REST endpoints above (catalog + policy; apply stub)  
- [x] CLI commands (services, policy)  
- [x] opcatalog entries  

### Phase P3 — UI

- [ ] Web: user policy + edge policy checkbox panels  
- [ ] TUI: same  
- [ ] TG: toggles + templates  
- [ ] i18n EN/RU full semantic parity  

### Phase P4 — product polish

- [ ] Presets  
- [ ] Doctor: warn if policy references missing service endpoints  
- [ ] Docs: CLIENT-PROFILES + EDGE-AGENT cross-links  
- [ ] Release notes  

---

## 9. Security notes

- Policies are operator-only (same auth as vpn user admin).  
- Catalog admin is higher privilege (can point endpoints at arbitrary addrs — validate private/service ranges).  
- Never expose internal endpoints on `0.0.0.0` as a shortcut for policy.  
- Reject is safer than silent direct for denied internal ports.

---

## 10. Acceptance criteria

1. Operator can uncheck `lampac` for user `appletv` and that UUID cannot open Lampac; `operator` still can.  
2. Edge device policy screen is **not** under Users.  
3. Adding catalog service `foo` shows a new checkbox in all policy UIs without code change to each screen (UI reads catalog).  
4. `vpn apply` / policy PUT rebuilds routes; reboot not required.  
5. EN and RU docs and UI strings match in meaning.

---

## 11. Handoff line for next agent

Start at **Phase P0**. Do not ship UI checkboxes before P1 enforcement. Keep edge peers out of Users. Update this checklist and `AGENT_HANDOFF.md` stop line when a phase completes.
