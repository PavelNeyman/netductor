# Security: Telegram HTML Widgets (HTML Bubbles)

**Status:** active advisory · 2026-11-11  
**Audience:** operators and AI agents changing `cmd/netductor-tg` widget code.

## What happened

On 2026-11-11 researchers (V12 and others) demonstrated **remote code execution in the Telegram client** via HTML widgets (Bot API 10.4 `sendWidget`). Public write-ups describe a **QuickJS** engine issue (use-after-free / parser class of bugs) inside the app process, not a bug in a specific bot’s business logic.

- Demo: widget message → client executes malicious JS → native impact (e.g. spawn Calculator on Android as RCE proof).
- Details vary by client build; some builds moved widgets to a more isolated process, older/in-process builds are worse.
- **Any bot that can `sendWidget` to a user can deliver the payload** if the user’s Telegram app is vulnerable.

This is a **client-side** vulnerability. Netductor cannot “patch Telegram”; we can only **not send widgets** or send them only under strict policy.

## Netductor posture

| Control | Default | Note |
|---------|---------|------|
| `NETDUCTOR_TG_WIDGETS` | **off** (must be `1`) | No `sendWidget` unless explicitly enabled |
| `NETDUCTOR_TG_DUAL` | **off** | Extra messages only if widgets already on |
| Classic keyboard | always available | Safe fallback UX |
| Recipient | admin private chat only (our design) | Still RCE risk **on the admin’s phone** if widgets on |

**Recommendation until Telegram clients are widely patched and you verified your app build:**

1. Keep **`NETDUCTOR_TG_WIDGETS` unset / not `1`**.
2. Update Telegram on **all** devices used for the admin account (Android / iOS / Desktop).
3. Do not enable widgets on shared or untrusted devices.
4. Treat widget enablement as a temporary ops experiment, not production default.

## What our HTML does (and does not)

- Our widget HTML is first-party, fixed strings + `chat_data` JSON from the bot.
- We do **not** load remote scripts (Telegram widgets have no network in the documented model).
- That does **not** protect against **engine** bugs: a malicious or crafted widget from *any* bot can still hit QuickJS flaws; our simpler JS does not make the interpreter safe.

## Agent rules (binding)

1. Do **not** default `NETDUCTOR_TG_WIDGETS=1` in unit files, install docs, or “make it easy” scripts without an explicit operator decision and this advisory link.
2. Do **not** remove the env gate “because classic is worse UX”.
3. Prefer classic Telegram keyboards for production control plane until advisory is lifted by the operator.
4. When changing widget JS, avoid exotic language features (async dispose, heavy typed arrays, experimental syntax) that stress QuickJS — defense in depth only.
5. Document any intentional enablement in operator notes.

## Lift criteria (operator)

Advisory can be relaxed when:

- Admin clients are on builds known to isolate or patch the disclosed issues, **and**
- Operator accepts residual QuickJS risk, **and**
- Widgets remain optional behind the env flag.

