# TG Widget UI v2

## Shells (multi-widget) — **implemented**

HTML is fixed after `sendWidget`. Layouts that need canvas (CPU/channel sparkline) use a **second shell**:

| Kind | When | HTML |
|------|------|------|
| `hub` | lists, cards, tools, most screens | SPA without canvas JS |
| `spark` | **status**, **channel** | hub + sparkline canvas |

**Transition (`appPushStateSingle` / `openAppScreen`):**
- same shell → `editMessageWidgetData` (in-place)
- different shell or edit fail → `sendWidget` + `deleteMessage` (one card in chat)

Shell kind is persisted in `widget_ui.json` (`app_shell`) with the SPA message id.

## Dual pair (`NETDUCTOR_TG_DUAL=1`)

| Slot | Message | Content |
|------|---------|---------|
| **info** | upper | rows / lines / metrics (`appInfoHTML`) |
| **nav** | lower | controls — always **hub** SPA shell |

On every navigation both slots are refreshed for the target screen. Nav shell mismatch → `swapDualSlot` (send + delete). Info always edits `appInfoHTML` chat_data (one info document).

## Formatters
- `appLines` / `formatCmdOut` — JSON → key lines
- Secondary list from `fleet.Build` rows
- Git status → rows when possible

## Limits
- zlib ≤ 4096 **per shell** (tested hub + spark)
- chat_data ≤ 1024

## Related
- `docs/PLAN-TG-DUAL-WIDGET.md`
- `docs/AI-RULES-TG-WIDGETS.md`

## Dual info HTML

Info slot uses **one** `appInfoHTML` document (rows/lines/spark via chat_data). Multi-shell is for the **nav/SPA** path under zlib pressure. Info does not need hub/spark split: the document is already small; content switches with screen.
