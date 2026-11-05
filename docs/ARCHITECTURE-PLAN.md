# Architecture plan

**EN** · Status: **close-out complete** (code + live recover drill).

| ID | Status |
|----|--------|
| A2 Deploy parity | **[x]** |
| A5 Doctor footgun + cert WARN | **[x]** |
| A6 Update path | **[x]** |
| Recover drill doc + **live unattended** | **[x]** 2026--09-27 v0.9.71 |
| B1 agent + op update | **[x]** |
| B2 runbook | **[x]** |
| B3/B4 posture | **[x]** |
| B5 opcatalog | **[x]** |

**Owner next:** dual-node smoke · hardware e2e  
**Brew:** [BREW.md](BREW.md) — never `sha256 :no_check`

## Deferred product work

### Telegram HTMLBubbles operator UI

**Status:** deferred — revisit when Telegram publishes a stable, public developer API for HTMLBubbles.

Goal: evaluate replacing the current Telegram bot menu with an interactive in-chat UI, without introducing a Telegram Mini App.

Architecture constraints:
- Keep Netductor backend/API as the source of truth for state and operations.
- Telegram bot remains the authenticated command/authorization boundary and invokes backend operations.
- HTMLBubbles, if/when publicly supported, is presentation only; no direct infrastructure access.
- Never expose SSH keys, API tokens, or other secrets to client-side HTML/JavaScript.
- Keep UI and backend contracts separate so the current bot menu remains a fallback and the UI can be replaced independently.

Re-entry checklist:
1. Confirm public API, message publishing mechanism, supported clients, and production availability.
2. Build a minimal capability prototype (render, buttons/toggles, bot round-trip, state refresh).
3. Verify authorization, replay protection, input validation, and failure handling on the bot/backend boundary.
4. Compare the prototype with the existing menu and decide whether to migrate.

Do not start implementation while HTMLBubbles remains experimental and unavailable through a documented public developer API.
