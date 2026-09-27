# PLAN: TG menu global refactor

**Status:** plan only (no implementation in this commit)  
**Baseline at write:** v0.9.23  
**Goal:** one UX model for all Telegram menus: rich body (tables + in-body actions), navigation-only under-message keyboard, consistent back stack, EN/RU, shared catalog where appropriate.

---

## 1. Locked product rules (must enforce everywhere)

| Rule | Detail |
|------|--------|
| **R1 Body = content + actions** | Tables, lists, status cards, and **functional** controls live in the **message body** via Bot API 10.x rich HTML: `<table>`, `<tg-button>`, `<tg-button-row>`, `<details>`. |
| **R2 Keyboard = navigation only** | `reply_markup` / inline keyboard under the message: **Back (parent)**, **Main menu**, optional **section hub** (Tools / Fleet / …). **No** Enable/Disable/Run/Revoke/Delete on the keyboard. |
| **R3 One step back** | Every screen has **⬅️ parent** (not only 🏠 Main). Parent is the menu that opened this screen. |
| **R4 Always sendRich** | Prefer `sendRichMessage` / `editMessageText` + `rich_message.html`. If edit fails, **new** rich message. Classic `parse_mode=HTML` strips `<tg-button>` / tables → **actions die**. |
| **R5 EN/RU** | All user-visible strings via `T()` / `i18n` or `opcatalog` labels. No hard-coded English in handlers (except proper names: Happ, Shadowrocket). |
| **R6 JSON secondary** | Catalog / API dumps: human card first; raw JSON only via in-body control or one nav-adjacent **📄 JSON** that still respects R2 (JSON is a *view mode*, not a random action pile). Prefer **📄 JSON as in-body** `<tg-button>` to keep keyboard pure nav. |
| **R7 Ideal reference** | **Users list** (`formatUsersListHTML` + `usersListKeyboard`) and **intended DNS** (`dnsblock.FormatCatalogHTML`): table/cards + row actions in body, keyboard = Main only (must gain Back). |

---

## 2. Why DNS (and similar) “buttons don’t work”

Code path today:

1. `FormatCatalogHTML()` builds `<table>…<tg-button data="m:dns:on:…">` + Reload row — **correct intent**.  
2. `showDNSMenu` → `reply()` → `editRich` / fallback classic HTML.  
3. If **edit falls back to classic** `parse_mode=HTML`, Telegram **strips** unknown tags → toggles vanish; only text remains.  
4. Keyboard under message is **only Main menu** — no fallback action buttons → user cannot toggle.

**Fix direction:** force rich path for any screen with `<tg-button>`; on edit failure delete+`sendRichMessage` (already partially in `replyCatalog`); never ship action-only-in-keyboard as compensation without fixing rich send.

Also verify Bot API 10.3 table-cell buttons vs row-below-table pattern (Users list uses **paragraph + `tg-button-row` under each item**, not button inside `<td>`). DNS puts buttons **inside `<td>`** — may be less reliable on some clients than the Users pattern.

**Preferred visual (owner):** keep **buttons inside table cells** (DNS layout). Users row-under-item is alternate.

**Fix (0.9.24):** rich transport + no classic strip — not layout change.

**Optional later:** re-render DNS like Users only if table-cell buttons still fail on some clients: one block per list (title, state) + `tg-button-row` under it; keep table optional for read-only columns.

---

## 3. Menu inventory (step-by-step)

### 3.1 Root — `mainKeyboard`

| Item | Callback | Type | Notes |
|------|----------|------|--------|
| Status | `m:status` | action-ish hub | Opens status card |
| Users | `m:users` | hub | |
| Fleet | `m:fleet` | hub | |
| Tools | `m:tools` | hub | |
| Operator | `m:operator` | hub | |
| Lang | `m:lang` | hub | |
| Help | `m:help` | content | |

**Issue:** root keyboard mixing hubs is OK (navigation). Hard-coded `"🧰 Tools"` not in i18n.

### 3.2 Users / VPN — **reference quality**

| Screen | Body | Keyboard | Back |
|--------|------|----------|------|
| Users list | `formatUsersListHTML` + in-body Open | Main only | ❌ no Back to… (is root) |
| User hub | `formatUserHubHTML` + in-body actions | Users + Main | ✅ Users |
| Access / link / QR | rich + photo | nav | mostly OK |

**Debt:** align other menus to this pattern; ensure `editRich` always used.

### 3.3 Fleet

| Screen | Notes |
|--------|--------|
| Fleet hub | nodes / routers / sites / addons — nav OK |
| Nodes list/card | `format_nodes` + in-body actions; keyboard nav |
| Routers / pending | approve/deny often inline; check R1/R2 |
| Sites | format_sites |
| Addons / Lampac | format_status helpers; bilingual OK |

### 3.4 Tools — **split brain**

| Part | Source | Issues |
|------|--------|--------|
| Catalog sections | `opcatalog` + `m:ops:` / `m:op:` | Human `internal/format`; **📄 JSON on keyboard** (R2 gray area); section titles half-hardcoded; back = section/Tools/Main |
| Guest VPN | `m:guest` | scenario UI |
| Guest Wi‑Fi | `m:edgeguest` | EN hard-coded messages |
| DNS | `m:dns` | **tg-button in table cells; rich path fragile; no Back to Tools** |
| Probes | catalog + old | |
| Backup | `m:backup` | **Run/Set on keyboard** (R2 violation); table+Restore in body for list |
| Locations | `m:loc` | **Rename/Delete on keyboard** |
| NVR | `m:nvr` | many actions on keyboard |
| Metrics | catalog | format OK after 0.9.22 |
| Updates | `m:updates` | |
| mTLS | `m:mtls` | actions on keyboard |
| Git | `m:git` | pipeline **Run on keyboard** |
| Registry | `m:registry` | **Ensure/Crane on keyboard**, EN labels |
| Secondary status | catalog / secondary | |
| Audit | catalog | |

### 3.5 Operator

| Screen | Issues |
|--------|--------|
| Session / admin / sessions | **Revoke all on keyboard** |
| Audit | catalog |
| Refresh links | action on keyboard |

### 3.6 Catalog results (`m:op:*`)

- Body: `format.API` (improving).  
- Keyboard: JSON + section + Tools + Main — **JSON should move in-body**; add explicit Back to section.

---

## 4. Violation matrix (priority)

| ID | Violation | Where | Severity |
|----|-----------|-------|----------|
| V1 | Actions on `reply_markup` | backup, loc, registry, git run, nvr pick, sessions revoke, operator refresh | High |
| V2 | In-body `tg-button` lost after classic fallback | DNS, any rich edit fail | High (user-reported) |
| V3 | Buttons inside `<td>` vs row-under-item | DNS | High |
| V4 | No parent Back (only Main) | DNS, many Tools leaves, catalog section partially | High |
| V5 | Raw / semi-raw JSON as primary view | residual catalog actions, errors | Med |
| V6 | Hard-coded EN (or RU-only) | dnsblock HTML, registry, git empty, edgeguest, catalog section names, main Tools | Med |
| V7 | Duplicate backup entrypoints | handlers_cb + handlers_extra | Med |
| V8 | Handoff baseline stale (0.9.20 vs 0.9.23) | docs | Low (doc debt) |
| V9 | Release 0.9.23 incomplete multi-arch assets | GitHub release | Low |

---

## 5. Target architecture

```text
cmd/netductor-tg/
  nav.go          — NavStack / parent callbacks: Back(parent), Main
  render.go       — replyRich(chat, msgID, html, parentID) always rich; keyboard=NavOnly(parent)
  menus/*.go      — one file per hub (optional later)
internal/format   — API cards (done, extend)
internal/opcatalog — read-only day-2 actions
internal/dnsblock — FormatCatalogHTML → Users-style rows + i18n
```

### Navigation helper (to implement)

```text
navKeyboard(parentCallback string) → [ Back → parent | Main ]
parent of DNS = m:tools
parent of m:ops:overview = m:tools
parent of m:op:doctor = m:ops:overview
parent of user card = m:users
```

### Action placement checklist

Before any new button: **Is it navigation?** → keyboard. **Else** → body `<tg-button>`.

---

## 6. Phased implementation plan

### Phase 0 — Doc debt (same release as first code drop)

- [ ] `AGENT_HANDOFF.md` baseline **v0.9.23+**  
- [ ] `OPEN_ITEMS.md` link this plan  
- [ ] Ensure latest release has full multi-arch assets if missing  

### Phase 1 — Transport / rich reliability

- [ ] Single `replyScreen(token, chat, msgID, html, parent)` used by all menus  
- [ ] Always rich; on edit fail → sendRich (no classic strip)  
- [ ] Log sendRichMessage errors with body snippet  

### Phase 2 — DNS fix (user-reported ideal)

- [ ] Rewrite `FormatCatalogHTML` to Users-style: block per list + `tg-button-row` (Enable/Disable), not button-in-`<td>`  
- [ ] i18n titles/actions  
- [ ] Keyboard: Back→Tools, Main  
- [ ] Verify toggle + reload callbacks after edit  

### Phase 3 — R2 sweep (move actions into body)

Priority order:

1. Backup (list already has Restore in table — move Run/schedule into body)  
2. Registry  
3. Git pipeline run  
4. Locations rename/delete  
5. NVR device/camera actions  
6. Sessions revoke / operator refresh links  
7. Catalog **📄 JSON** → in-body button  

### Phase 4 — Back stack everywhere

- [ ] Every `reply`/`show*` gets `parent` callback  
- [ ] Replace “Main only” keyboards  

### Phase 5 — i18n completion

- [ ] Audit `cmd/netductor-tg` + `internal/dnsblock` for user strings  
- [ ] Catalog section titles via `T("cat_overview")` etc.  
- [ ] Main menu “Tools” via `T("tools")`  

### Phase 6 — Format polish

- [ ] Remaining catalog actions still generic → dedicated cards as needed  
- [ ] Error responses never dump raw JSON as sole body  

### Phase 7 — Regression checklist (manual)

- [ ] Users list: open card, link, QR  
- [ ] DNS: enable/disable/reload, Back to Tools  
- [ ] Backup: run, list restore  
- [ ] Catalog: doctor, metrics, health  
- [ ] Lang switch EN/RU on all touched screens  
- [ ] No action-only-on-keyboard left on touched screens  

---

## 7. Explicit non-goals (this refactor)

- Deploy from Telegram  
- Replacing scenario menus (Guest grant dialogs) with pure catalog  
- Full TG feature parity with Web Installer  

---

## 8. Ideal card template (copy-paste target)

```html
<h3>🛡 DNS</h3>
<p>Status line…</p>
<!-- per item -->
<p>🟢 <b>List name</b> · ON</p>
<tg-button-row align="left">
  <tg-button type="callback_data" style="danger" data="m:dns:off:id">Disable</tg-button>
</tg-button-row>
<tg-button-row>
  <tg-button type="callback_data" style="primary" data="m:dns:reload">Reload</tg-button>
</tg-button-row>
```

Keyboard:

```text
[ ⬅️ Tools ] [ 🏠 Main ]
```

---

## 9. Next step after this doc

Implement **Phase 0 + 1 + 2** first (doc debt, rich transport, DNS), then Phase 3–5 in order. Do not mix large registry/git moves until DNS is verified on device.


### DNS lesson (0.9.26)

**Telegram does not deliver `callback_query` for `<tg-button>` placed inside `<td>`.**
Working pattern: read-only `<table>` + `<tg-button-row>` under each item (same as VPN Users).
