**EN** · [RU](ru/PLAN-TG-HTML-WIDGETS-MENU.md)

# Plan: migrate TG operator menu to HTML Widgets (Bot API 10.4)

**Status:** planned · **Gate:** `NETDUCTOR_TG_WIDGETS=1` (off by default until soak)  
**Basis:** CF-18.4 status widget (`cmd/netductor-tg/widget_status.go`, commit `cfe6535`)  
**Related:** [TG-UI-PATTERN.md](TG-UI-PATTERN.md) (rich templates A/B/C remain fallback)

## Goal

When widgets are enabled, **operator menu screens** open as in-message **HTML widgets** (live UI, `widget_query`, data refresh without new bubbles) instead of `sendRichMessage` tables + `tg-button`.  
When disabled, behavior stays **exactly** as today (rich HTML).

Default remains **off** until operator soak on production private chat. Then flip default to on (or drop the gate).

## Non-goals

- Mini Apps / WebApp URL hosts  
- Group / channel widgets (prod Bot API 10.4: **private only**)  
- TON wallet flows  
- Replacing alert channel payloads (channel stays classic / rich notify)  
- Topics / guest mode

## Constraints (Telegram 10.4)

| Constraint | Implication |
|------------|-------------|
| Private chats only | Matches admin-DM-only bot policy |
| Protected content | No forward; OK for ops |
| HTML ≤ ~32 KiB | One shared shell + small `chat_data` JSON |
| No external fetch in widget | All data via `chat_data` / `answerWidgetQuery` |
| ~600×600, limited JS | Compact panels; no heavy charts |
| Client ≥ 13.2 | Document; fallback to rich if `sendWidget` fails |

## Architecture

```
widgetsEnabled()  // NETDUCTOR_TG_WIDGETS=1
        │
        ├─ true  → tryWidget(screen) → sendWidget | editMessageWidgetData
        │              └─ widget_query → handleWidgetQuery → answer / edit data / nav
        └─ false → existing reply() + format*HTML + keyboards (unchanged)
```

**Shared shell (recommended):** one base HTML (nav chrome + content host) + per-screen `chat_data` JSON, not N full HTML copies. Status widget is the prototype; extract `widget_shell.go` + `widget_data_*.go`.

**Nav model:**
- In-widget buttons: `refresh`, screen-local actions, `menu` / parent  
- Heavy / multi-step flows (rename prompts, enroll paste, confirm revoke): keep **classic message** after `widget_query` nav (same as status → Versions today)

**ACL:** every `widget_query` checks admin id (already in `handleWidgetQuery`).

## Phases

### Phase 0 — Foundation (1–2 days)

1. Extract from `widget_status.go`:
   - `widgetsEnabled()`, `apiPostWidget`, shell HTML constants, size tests  
   - `WidgetScreen` registry: id → buildChatData + optional actions  
2. Generic `sendWidgetScreen(token, chat, screenID)`, `editWidgetData`, `handleWidgetQuery` dispatch table  
3. Keep status as first registered screen  
4. Docs: this plan; note in AGENT_HANDOFF / OPEN_ITEMS  
5. Tests: HTML size, chat_data size, gate off by default, admin deny  

**Exit:** status widget still works; no behavior change when gate off.

### Phase 1 — Read-only / status family (soak candidate)

Screens (map from handlers / TG-UI matrix):

| Screen | Callback today | Widget actions |
|--------|----------------|----------------|
| Status | `m:status` | refresh, → versions, channel, menu |
| Channel detail | `m:channel` | refresh, back menu |
| Secondary status | `m:secondary:status` | refresh |
| Netmap | `m:netmap` | refresh |
| Doctor / probes | `m:probes` | refresh (summary only) |
| Metrics snapshot | `m:metrics` | refresh (compact) |
| Versions / digest | `m:digest` / versions | refresh, → apply stays confirm via classic |

**Exit:** gate on → these open as widgets; gate off → rich. Operator smoke in DM.

### Phase 2 — Lists (template A → widget)

| Screen | Notes |
|--------|--------|
| Users / VPN list | `m:users`, `m:vpn_list` — numbered open → user card |
| VPN devices | `m:vpn_devices` — paginate in data |
| Nodes list | `m:nodes_list` / `m:cat:nodes` |
| Fleet | `m:fleet` |
| Sites | `m:sites:list` |
| SSH hosts | `m:sshhosts` |
| Backup list | `m:backup` |
| Secondary list | `m:relay:list` |
| Pending | `m:pending` |

List UX: rows + index buttons in widget; page next/prev in `chat_data`. Open item → card widget or classic card if prompt-heavy.

**Exit:** lists usable under gate; pagination stable; no 32 KiB overflow (page size ≤ 12 as rich).

### Phase 3 — Cards / hubs (template B)

| Screen | Notes |
|--------|--------|
| User hub | enable/disable/revoke/quota — confirm destructive via classic reply |
| Node card | upgrade/journal — queue cmd then classic ack |
| Operator hub | toggles |
| Addons hub | lampac status |
| Svc-paths | status + apply confirm classic |
| Edge / sites cards | as needed |

Destructive actions: widget button → `answerWidgetQuery` + **classic** confirm message (existing challenge codes), not silent in-widget delete.

**Exit:** parity with rich cards for daily ops.

### Phase 4 — Catalog / tools residual

Catalog sections (`m:cat:*`), tools that today use `formatCatalogHTML` / `formatSmart`: port high-traffic only; low-traffic may stay rich indefinitely.

Main menu (`m:menu`): optional compact widget launcher **or** keep rich keyboard (keyboards work well offline of widget). Prefer **keep classic main menu**; widgets for content screens.

### Phase 5 — Default on

1. Soak ≥ 3–7 days with `NETDUCTOR_TG_WIDGETS=1` on primary  
2. Journal: no `sendWidget` error rate spike; client 13.2+  
3. Change default: `widgetsEnabled()` true unless `NETDUCTOR_TG_WIDGETS=0`  
4. Release note; optional unit drop-in remove  

## Implementation rules

1. **No gate on → zero path change** (existing tests / UX invariants stay green).  
2. Every widget path: if `sendWidget` fails → **fallback `reply()` rich** (log once).  
3. EN/RU: strings in `chat_data` via existing `T()` / lang from admin prefs.  
4. Do not put secrets in widget HTML or `chat_data` (tokens, full UUIDs, raw keys).  
5. Size tests mandatory per new HTML blob and per max chat_data.  
6. `allowed_updates` already includes `widget_query`; keep it.  
7. Align naming: `widget_*.go`, tests `widget_*_test.go`.

## Test plan

| Level | What |
|-------|------|
| Unit | size gates, JSON build, enable flag matrix |
| Bot DM | gate off: Status = rich; gate on: Status = widget, Refresh works |
| Regression | users open/revoke, node upgrade queue, backup run confirm still classic where required |
| Client | Telegram iOS/Android/Desktop ≥ 13.2; one older client → fallback |

## Rollout

```bash
# test enable
mkdir -p /etc/systemd/system/netductor-telegram-bot.service.d
printf '%s\n' '[Service]' 'Environment=NETDUCTOR_TG_WIDGETS=1' \
  > /etc/systemd/system/netductor-telegram-bot.service.d/widgets.conf
systemctl daemon-reload && systemctl restart netductor-telegram-bot
```

Disable: set `=0` or remove drop-in.

## Order of work (suggested PRs)

1. Foundation extract + registry (Phase 0)  
2. Phase 1 screens  
3. Phase 2 lists  
4. Phase 3 cards  
5. Default-on + docs  

No release required until Phase 1 smoke; ship behind flag in ordinary TG binary releases.

## Done when

- [ ] All Phase 1–3 screens widget-capable under gate  
- [ ] Fallback rich on API/client failure  
- [ ] Soak with flag on  
- [ ] Default enabled (or documented permanent opt-in)  
- [ ] TG-UI-PATTERN updated: “Widgets = optional layer over A/B/C”

