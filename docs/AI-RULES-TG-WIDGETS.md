# AI rules: Telegram HTML Widgets (Netductor)

**Audience:** every human or AI agent changing `cmd/netductor-tg` widget code, dual UI, or related plans.  
**Precedence:** these rules override casual “just send another widget” instincts.  
**Related:** `docs/PLAN-TG-DUAL-WIDGET.md`, `docs/PLAN-TG-WIDGET-UI-V2.md`.

---

## 0. Hard platform limits (never invent around them)

1. Widget **HTML is immutable** after `sendWidget`. New layout = new message (+ delete old).
2. **chat_data ≤ 1024 bytes** per message.
3. **HTML ≤ 32 KiB raw; zlib-compressed ≤ 4 KiB** per message.
4. **request / act string ≤ 64 bytes**.
5. No network, no keyboard, limited height (~600px), no scroll assumptions.
6. Widget queries are tied to the **message that was tapped**.

Violating limits is a product bug, not a “temporary experiment”.

---

## 1. Dual-pair mode (`NETDUCTOR_TG_DUAL=1`)

When dual is enabled, the operator UI is a **pair**, not two independent widgets.

### 1.1 Pair is atomic

- State key: `chat_id` → `{infoMsgID, navMsgID, infoShell, navShell}`.
- **Never** update or delete one half without considering the other.
- **Never** send a third widget for the same menu session while a pair exists (except intentional slot **swap** of one side).
- **Open:** create info + nav (or roll back / delete orphans on failure).
- **Close** (`/start` reset, disable widgets, chat clear recovery): delete **both** messages; clear **both** IDs and shell kinds.

### 1.2 Roles (do not mix)

| Slot | Allowed | Forbidden |
|------|---------|-----------|
| **info** | rows, lines, pct, spark, short hint, optional refresh act | full nav tabs as primary UX, long act grids, list open-by-index as main path |
| **nav** | tabs, segments, lists, acts, toggles, confirms | heavy sparkline series as primary payload (keep spark on info) |

If a change needs metrics on the interactive card, **put metrics on info** and keep nav for actions.

### 1.3 Updates are paired

On every successful nav `widget_query` that changes screen or context:

1. `answerWidgetQuery` (required; payload = **nav** chat_data unless API constraints force otherwise).
2. Edit **nav** message chat_data.
3. Edit **info** message chat_data to match the same logical screen/context.

Skipping info update after a context change is a **lifecycle bug**.

Refresh-only on info may update info alone; still must not desync stored shell/IDs.

### 1.4 Shell swap (per slot)

- Track shell kind **per slot** (`hub` / `spark` / future `card`).
- If HTML kind must change for **one** slot: `sendWidget` new → store new ID → `deleteMessage` old **for that slot only**.
- Do not delete the other slot during a one-sided swap.
- After swap, re-apply current chat_data if send did not include it correctly.

### 1.5 Failure and recovery

- Edit failed once → retry once.
- Still failed → recreate **that slot** with correct shell + data; do not leave `msgID` pointing at a deleted message.
- If pair is half-broken (one ID zero/missing): recreate missing slot; do not open a second full pair.
- Log clearly: `dual:`, slot name, chat, old/new msg id.

### 1.6 Single-message mode

When `NETDUCTOR_TG_DUAL` is not `1`, **do not** apply dual pair logic. Existing single SPA + multi-shell rules apply. Do not half-enable dual (one dual message + classic).

---

## 2b. Dual vs multi-shell

| Mode | Messages | Where is spark/canvas? | Multi-shell swap |
|------|----------|------------------------|------------------|
| Dual **off** | 1 SPA | SPA spark shell on status | `appPushStateSmart` hub↔spark |
| Dual **on** | 2 (info+nav) | **info only** | **Do not** spark-swap nav; nav stays hub |

When dual is on, treating the pair like single-SPA multi-shell (swapping nav HTML for status) is wrong.

## 2. Single SPA / multi-shell (dual off)

1. Prefer **one** live app message per chat (`lastAppMsgID`).
2. Same shell → `editMessageWidgetData` only.
3. Different shell → send + delete previous app message (see UI v2 plan).
4. `/start` or chat wipe: forget id **and** shell; do not edit unknown message ids blindly without recovery path.

---

## 3. Feature flags and defaults

1. Widgets stay behind `NETDUCTOR_TG_WIDGETS=1`.
2. Dual stays behind `NETDUCTOR_TG_DUAL=1` until explicitly productized.
3. Do not default dual on without operator confirmation and a release note.
4. Do not remove classic keyboard paths while widgets are optional.

---

## 4. Code change discipline

1. Any new screen must define **what info shows** and **what nav shows** if dual is on (even if info reuses status summary).
2. Do not dump raw multi-KB JSON into chat_data; use formatters / rows.
3. zlib size tests must cover **each** HTML shell variant you register.
4. Prefer extending `docs/PLAN-TG-DUAL-WIDGET.md` when changing lifecycle; do not invent a third parallel state map without documenting it.
5. No release solely for dual until open/close/update/swap paths are tested.

---

## 5. Explicit prohibitions

- Spamming a new widget message per menu tap “because edit failed once”.
- Leaving orphan info or nav messages after `/start`.
- Putting primary navigation only on the info slot in dual mode.
- Sharing one `msgID` variable for both slots.
- Assuming HTML can be patched in place.

---

## 6. Checklist before merge (widgets)

- [ ] Flag behaviour documented  
- [ ] Pair open/close clears both IDs  
- [ ] Context change updates both slots in dual mode  
- [ ] Shell swap is per-slot  
- [ ] Size tests for new HTML  
- [ ] Dual default still off unless agreed  

