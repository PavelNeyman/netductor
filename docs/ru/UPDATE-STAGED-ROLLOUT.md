# Staged rollout checklist (Phase 10)

Do **not** treat unit green or tip SHA as go-live. Requires operator approval and live health.

## Preconditions

- [ ] Inventory: Primary, Secondary, Edge devices, Mac op version
- [ ] Backup / snapshot available; `netductor stack rollback` known on Primary
- [ ] TOFU SSH pins for Primary (and Secondary if Mac deploys)
- [ ] Target release **published** on GH (or internal store) with SHA256SUMS
- [ ] Explicit go/no-go owner named

## Stage order (bounded)

1. **Mac operator** — `brew upgrade netductor-op` / install `netductor-op-darwin-*` for target tag only.
2. **Primary node** — `netductor update check` → `update apply vX.Y.Z --component node` (or `stack apply`). Confirm version + `doctor`.
3. **Primary tg** (if used) — `--component tg`; confirm bot responds.
4. **Secondary host agent** (on secondary VPS) — operator-driven upgrade of secondary binary/agent; **Promote=false** remains until traffic R5.
5. **Edge canary** — one device via `agent_update` + confirm; fleet only after canary healthy.
6. **Stop** on failed health, unexplained heartbeat/version mismatch, or VPN/user-path regression.

## Rollback

| Component | Path |
|-----------|------|
| node | `netductor stack rollback` (prev snapshot) |
| tg/agent host | restore `.prev` binary or re-apply previous tag |
| op | previous Homebrew bottle / prior darwin asset |
| edge | previous agent binary via control plane / offline pack |

## Evidence to record (no secrets)

- Before/after `version` strings per host
- `update jobs` / apply result
- doctor summary fail/warn counts
- Decision: **GO** or **NO-GO** with date and operator

## Explicit non-goals this stage

- Automatic Secondary traffic promote
- Deleting previous Primary/Secondary
- Fleet-wide Edge without canary
