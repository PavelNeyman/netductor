# TG screen inventory (v0.9.48)

Patterns: **A** list · **B** card · **C** status (see [TG-UI-PATTERN.md](TG-UI-PATTERN.md)).

Keyboard rule: under-message = **nav only** (Back / Main / parent). Actions in body.

## Main menu

| Screen | Callback | Pattern | Status | Notes |
|--------|----------|---------|--------|-------|
| Main hub | `m:menu` | nav | OK | Status, Users, Fleet, Tools, Operator, Lang, Help |
| **Status** | `m:status` | **C** | OK | **Single** overview: metrics table + units + nodes + VPN + mismatch. **No separate Metrics.** |
| Users list | `m:users` | A | OK | Reference quality |
| User hub | `u:open:` | B | OK | |
| User access / QR | `u:access:` | B+photo | OK | |
| Fleet hub | `m:fleet` | nav | OK | |
| Tools hub | `m:tools` | nav | OK | opcatalog sections + Guest/Loc/Updates/mTLS |
| Operator hub | `m:operator` | nav | OK | |
| Lang | `m:lang` | nav | OK | |
| Help | `m:help` | C-ish | OK | |

### Dedup (done 0.9.46)

| Was | Now |
|-----|-----|
| Main **Status** + Tools **Metrics** (`m:metrics` / catalog metrics) | **Status only**; `m:metrics` → same `formatStatusPretty`; catalog metrics = **web-only** |
| Tools catalog **DNS** + hard-coded **DNS** button | Hard-coded removed; use section **DNS** or keep deep link `m:dns` from section |
| Health + Status in catalog | Health **web-only** on TG surface |

## Fleet

| Screen | Callback | Pattern | Debt |
|--------|----------|---------|------|
| Nodes list | `m:nodes_list` / `m:cat:nodes` | A | Low |
| Node card | `m:nd:*` | B | Keep 📊 = **node** metrics (not global) |
| Routers / pending | `m:pending` … | A/B | Check actions in body |
| Sites | `m:sites:*` | A/B | |
| Addons / Lampac | `m:addons` | B | |

## Tools (product hubs)

| Screen | Callback | Pattern | Debt |
|--------|----------|---------|------|
| Guest VPN | `m:guest` | B | |
| Guest Wi‑Fi | `m:edgeguest` | B | EN strings |
| DNS | `m:dns` | A | Was fragile rich; verify |
| Locations | `m:loc` | A/B | |
| Updates | `m:updates` | B | |
| mTLS | `m:mtls` | B | |
| NVR | `m:nvr` | A/B | Large surface |
| Backup | `m:backup` | A/B | |
| Git / Registry | `m:git` / `m:registry` | A/B | |
| Catalog `m:ops:` / `m:op:` | section / action | A-like / C | Prefer format.API; JSON in body only |

## Operator

| Screen | Debt |
|--------|------|
| Session / sessions / audit / refresh links | Body actions preferred |

## Explicit non-goals on TG

- VPS deploy / fleet install  
- Full metrics history charts (Web)  
- Rooms/photos inventory (Web + optional TG later — [PLAN-SITE-ROOMS.md](PLAN-SITE-ROOMS.md))

## Fix order (remaining)

1. ~~Metrics vs Status~~ **done**  
2. Grep remaining `formatSmart` for known day-2 IDs on TG path  
3. Guest Wi‑Fi EN → i18n  
4. NVR keyboard actions → body where still on keyboard  
5. Operator revoke/refresh on keyboard → body  
6. Walk bot checklist before each TG release  

## Audit commands

```bash
rg 'formatSmart|m:metrics|inline_keyboard' cmd/netductor-tg
rg 'Metrics history|m:metrics' cmd/netductor-tg internal/opcatalog
```

### 0.9.49
Fleet/Nodes/Routers/Sites/Tools hubs → body buttons.
