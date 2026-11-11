# Plan: Dual Telegram HTML Widgets (info + nav)

**Status:** D1–D5 in main behind `NETDUCTOR_TG_DUAL=1` (default off); D6 = operator smoke  
**Depends on:** SPA app shell, multi-shell hub/spark (`docs/PLAN-TG-WIDGET-UI-V2.md`)  
**Non-goals:** replacing classic keyboard; single-message mode remains default until dual is stable.

## Goal

Two coordinated messages per admin chat:

| Slot | Message | Content | Interaction |
|------|---------|---------|-------------|
| **info** | upper | status, metrics, bars, sparkline, context summary | refresh only (optional) |
| **nav** | lower | tabs, segments, lists, toggles, destructive confirms | full widget queries |

Both share one logical “session”; navigation in **nav** drives what **info** shows.

## Why

- Separate zlib budgets (2 × ≤4 KiB compressed HTML).
- Separate chat_data budgets (2 × ≤1024 B).
- Metrics stay visible while operator walks menus.
- Heavier info shell (canvas) without bloating nav shell.

## Lifecycle (strict)

See **`docs/AI-RULES-TG-WIDGETS.md`** — binding for all agents.

Summary:

1. **Pair identity:** `(chat_id → {infoMsgID, navMsgID, infoShell, navShell})`.
2. **Open:** create **both** or neither; never leave a half-pair without recovery.
3. **Update:** prefer `editMessageWidgetData` on each slot; shell change → swap that slot only (send + delete old for that slot).
4. **Close /start /clear:** delete **both** messages; forget both IDs and shells.
5. **Failure:** if one edit fails, retry once; if still fail, recreate that slot only, then re-sync the other.

## Flag matrix

| Env | Effect |
|-----|--------|
| `NETDUCTOR_TG_WIDGETS=1` | widgets on (existing) |
| `NETDUCTOR_TG_DUAL=1` | dual pair mode |
| dual off | single SPA message (current behaviour) |

## Data model

```text
type dualPair struct {
  InfoID, NavID int
  InfoShell, NavShell string // hub|spark|...
}
lastDual[chat] dualPair
```

**info chat_data (sketch):** `{s,t,rows,pct,spk,h,L,ts}` — no nav/acts/ids (or only `r`).  
**nav chat_data:** current appState without heavy `spk` (spark lives on info).

## Navigation flow

1. User taps act on **nav** → `widget_query` with nav `message_id`.
2. Handler builds `navCD` + `infoCD` for new screen.
3. `answerWidgetQuery` with **nav** payload (required).
4. `editMessageWidgetData` nav; `editMessageWidgetData` info (order: answer → nav → info).
5. If screen needs spark shell on info and current info shell is hub → **swap info only**.

## /start behaviour

1. Forget previous pair (delete messages if IDs known).
2. Send info widget (status summary).
3. Send nav widget (home).
4. Store both IDs.

## Implementation phases

| Phase | Deliverable |
|-------|-------------|
| **D0** | Done — plan + AI rules |
| **D1** | Done — dualPair, flag, open/close/edit/swap |
| **D2** | Done — `buildInfoData(screen)` per hub |
| **D3** | Done — `/start` dual open; `appPushState` dual path |
| **D4** | Done — info metrics shell; spark on home/status |
| **D5** | Done — dual openAppScreen, lifecycle tests, info all parity screens |
| **D6** | Manual QA on primary: WIDGETS=1 DUAL=1 |

## Tests

- Dual flag off → existing SPA tests green.
- Dual on (unit): open records two IDs; forget clears both; shell swap updates one slot.
- chat_data size ≤1024 for info and nav samples.

## Rollout

No release until D3–D5 green and operator smoke on primary with flag.

## Multi-shell relationship

Dual **supersedes** SPA multi-shell for the interactive surface:

- **Nav** = always `hub` HTML (tabs/acts).
- **Info** = metrics HTML (rows/bars/canvas data in chat_data).
- Single-mode `appPushStateSmart` remains for `NETDUCTOR_TG_DUAL!=1` only.

Do not reintroduce status→spark swaps on the nav message while dual is enabled.
