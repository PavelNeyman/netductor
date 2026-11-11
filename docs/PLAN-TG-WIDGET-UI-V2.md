# TG Widget UI v2

## Shells (multi-widget)

HTML is fixed after `sendWidget`. Layouts that need extra JS (CPU canvas) use a **second shell**:

| Kind | When | HTML |
|------|------|------|
| `hub` | home, lists, cards, tools… | tabs/seg/bars, no canvas |
| `spark` | **status** only | hub + sparkline canvas |

Transition: `appPushStateSmart` — same shell → `editMessageWidgetData`; different shell → `sendWidget` + `deleteMessage` (one card in chat).

## Formatters
- `appLines` / `formatCmdOut` — JSON → key lines
- Secondary list from `fleet.Build` rows
- Git status → rows when possible

## Limits
- zlib ≤ 4096 per shell (tested hub + spark)
- chat_data ≤ 1024

## Dual pair (next)

See `docs/PLAN-TG-DUAL-WIDGET.md` and binding rules `docs/AI-RULES-TG-WIDGETS.md`.
