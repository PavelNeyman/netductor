# Code & security review — v0.9.41

**Date:** 2026-09-26 · **Baseline code:** v0.9.41 · **This doc revision:** handoff for next chat.

## Executive summary

| Area | Verdict |
|------|---------|
| Critical security issues | **None** |
| Mac deploy TUI ↔ Web ↔ CLI | **Parity OK** — shared `internal/operator` → `internal/deploy` |
| Operator Web bind | Loopback only + SHA-256 token (constant-time) |
| Node API | Session auth; non-local bind needs `NETDUCTOR_API_PUBLIC=1` |
| Agent plane | mTLS :8789 default; plain only with `PLAIN_AGENT=1` |
| Day-2 UI (CLI/TUI/Web/TG) | Functional parity for core ops |
| TG UI pattern | Body actions complete (0.9.41); keyboard = nav |
| EN/RU | TUI, Web, TG core; residual free-text prompts only |
| Must-fix before next feature | **None** |

## Architecture (locked)

```
Mac: netductor-op
  ├─ TUI wizard / CLI deploy  ─┐
  └─ operator serve (Web)     ─┴─→ operator.Deploy* → deploy (SSH/SCP from Mac)
VPS: netductor (node) — :8787 session API, VPN, agents, TG addon
```

**No** provision hop primary→secondary. Secondary is Mac-direct.

### Deploy entry points (same backend)

| | Primary | Secondary | Fleet | Edge | Site / MikroTik |
|--|---------|-----------|-------|------|-----------------|
| TUI | `DeployPrimary` | `DeploySecondary` | `FleetDeployWithReport` | `DeployEdge` | `DeploySite` / `MikroTikAction` |
| Web | `POST /v1/primary` | `POST /v1/secondary` | `POST /v1/fleet` | `POST /v1/edge` | `/v1/site`, `/v1/mikrotik` |
| CLI | `netductor-op deploy …` | same packages | | | |

Field mapping: `PrimaryFromFields` / `FleetFromFields` / specs in `internal/operator`.

## Security review

### Strengths

- `ValidHost` / `ValidUser` / `ValidDeviceID` on operator deploy paths (+ tests)
- SSH/SCP: `--` separator; password via `SSHPASS` / ASKPASS (not argv)
- `shellQuote` on remote fragments
- Operator token: SHA-256 + `subtle.ConstantTimeCompare`
- Secondary / recovery tokens: constant-time compares
- Node API: refuse non-local bind without `NETDUCTOR_API_PUBLIC=1`
- XFF in rate-limit / clientIP only if `NETDUCTOR_TRUST_PROXY=1`
- Recovery HTTP: off until **arm**, TTL ≤2h; bind `0.0.0.0` while armed (product intent) + optional `ALLOW_CIDR`
- Content-Disposition / BackupPath sanitization (prior passes)

### Residual risks (ops, not code bugs)

Documented in [RESIDUAL_RISKS.md](RESIDUAL_RISKS.md):

| Risk | Mitigation |
|------|------------|
| Recovery WAN while armed | Short TTL, strong token, optional CIDR, doctor WARN |
| `PLAIN_AGENT` / `CLAIM_FIRST` / `API_PUBLIC` | Explicit env; doctor FAIL/WARN |
| Local camera / SNI probe TLS skip | Intentional scoped use |
| Homebrew Formula still 0.8.98 pin | Until first v0.9 brew cut |

### No product change this pass

Recovery default bind while armed remains `0.0.0.0` (user-confirmed design).

## API coverage

| Layer | Count / notes |
|-------|----------------|
| Node routes (`/api/*`, `/vpn/*`) | **~114** unique HandleFunc paths |
| Web Control buttons / special forms | **~90** path references + Advanced generic |
| opcatalog actions | **~49** shared TG/Web labels (GET + simple POST) |
| Operator `/v1/*` | fleet, primary, secondary, edge, site, mikrotik, tunnel, session, catalog, node proxy |

**Rule:** rare edge/NVR paths → Advanced Control or CLI; not every route needs a catalog button.

## UI parity (day-2)

| Capability | CLI | TUI | Web | TG |
|------------|-----|-----|-----|-----|
| Deploy fleet/primary/secondary/edge/site | ✅ op | ✅ | ✅ | ❌ by design |
| Doctor / domain / metrics | ✅ | ✅ | ✅ | ✅ catalog |
| VPN users | ✅ | ✅ | ✅ | ✅ compact |
| DNS lists | ✅ | ✅ Ops | ✅ | ✅ body |
| Backup schedule/list | ✅ | ✅ Ops | ✅ | ✅ body |
| Git / registry | ✅ | ✅ Ops | ✅ | ✅ body |
| NVR | ✅ | ✅ | ✅ | ✅ body |
| Guest / Edge guest / mTLS / Sessions | ✅ | partial Ops | Control | ✅ body (0.9.41) |
| SSH hosts TOFU | ✅ | — | — | ✅ body |

## Bilingual EN/RU

| Surface | Status |
|---------|--------|
| TUI | `langEN` / `langRU`, FormT / TT |
| Web | `I18N.en` / `I18N.ru`, `data-i18n`, catalog labels |
| TG | Dual maps in `i18n.go` + bilingual branches on hubs |
| Gaps | Occasional free-text wait prompts; Advanced raw API paths |

## TG UI pattern (0.9.41)

Keyboard under message = **navigation only**. Actions in `<tg-button-row>` body. Applied to DNS, Users, Nodes, Backup, Locations, Git, Registry, SSH, NVR, Catalog, Guest, Edge Guest, mTLS, Updates, Sessions, Secondary.

## Build / tests (this pass)

- `go test` operator, edge, httpx, secondary, session — OK  
- `go build` netductor-op, netductor, netductor-tg — OK  
- Live VPS deploy not re-executed in this session  

## Next chat

1. Read: AGENTS.md → AGENT_HANDOFF → **this file** → ARCHITECTURE-OPERATOR → OPEN_ITEMS  
2. Do **not** reopen closed day-2 UI unless regression  
3. Optional later: hardware e2e, SMTP, mobile, CDN-XHTTP idea, Formula v0.9 pin after release tag  
