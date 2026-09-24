# Operator architecture (Mac / workstation)

**Status:** approved direction · **Baseline:** v0.8.94+  
**Audience:** humans + AI agents continuing implementation.

## Problem

CLI and TUI were synchronized feature-by-feature. Logic leaked into UI layers → permanent drift (domain/LE/add-ons present in one path, missing in another).

## Target model

```text
  CLI · TUI · WebUI (embed) · future Desktop/Mobile
              │
              ▼
     Operator core (use-cases + Spec types)
              │
    internal/deploy · domain · credentials · SSH
              │
              ▼
     primary / secondary / edge  (nodes)
```

| Plane | Where | Role |
|-------|--------|------|
| **Operator** | Mac / workstation | Bootstrap fleet, domain/LE, collect secrets to `~/.netductor/` |
| **Node** | VPS / OpenWrt | `install`, `serve`, vpn, agent, day-2 admin/TG |

Same Go module/repo for now. **Logical** split is mandatory; **binary** split (`netductor` vs `netductor-op`) is optional later.

### Rules

1. **No deploy logic in TUI/CLI** beyond mapping form/flags → `Spec` and calling one use-case.
2. **New deploy field** = field on Spec + applied once in operator/deploy core — not per UI.
3. **Secrets** after deploy land on **Mac** (`credentials collect` / deploy hooks). Do not rely on primary as the only copy.
4. **Operator HTTP API** (when added) binds **127.0.0.1 only** — not a public installer.
5. **Primary admin API** (`:8787`) is day-2 runtime, not bootstrap. Do not merge trust boundaries.
6. **Legacy huh deploy** is removed; framed Wizard / Fleet / CLI only.

### Use-cases (in  since 0.8.95)

DeployPrimary, DeploySecondary, FleetDeploy, CollectCredentials, *FromFields helpers.

### Remaining

Structured events; operator serve + WebUI (phase 3).

### Use-cases (historical contract)

| Use-case | Spec (approx.) | Notes |
|----------|----------------|--------|
| `DeployPrimary` | host, key, sni, domain, LE, add-ons, TG | Exists as `deploy.DeployPrimary` |
| `DeploySecondary` | primary + secondary hosts, keys/pass | Mac-direct; exists |
| `FleetDeploy` | primary + secondary + domain/LE + add-ons | TUI Fleet maps fields; should become **one** core entry |
| `ApplyDomain` | base / primary / vpn / redirect / LE | `domain.Apply` |
| `CollectCredentials` | role, host, key | `deploy.CollectOperatorSecrets` |
| `DeployEdge` | router + primary mTLS | exists |

**Events:** prefer structured step events from core (for TUI/Web progress). Parsing stderr is transitional only.

### Localhost Operator API (phase 3)

```text
netductor operator serve -bind 127.0.0.1:<port>
POST /v1/fleet
POST /v1/deploy/primary
GET  /v1/events   (SSE or similar)
```

Embed WebUI via `go:embed`. CLI/TUI may keep in-process Go calls (no HTTP required).

---

## Implementation plan

Mark progress **only** in [OPERATOR-PLAN.md](OPERATOR-PLAN.md).  
Every completed checkbox **must** update that file + relevant docs in the **same change** (see AGENTS.md).

| Phase | Goal | Exit criteria |
|-------|------|----------------|
| **0** | Docs + handoff | This file + compressed handoff + AGENT rules |
| **1** | `internal/operator` (or equivalent) use-cases | CLI + TUI call **only** use-cases; no duplicate opts assembly |
| **2** | `FleetDeploy` single entry | One Spec → primary→secondary→credentials; progress from events |
| **3** | `operator serve` + embed WebUI | Fleet form + log on localhost |
| **4** | Optional binary split / mobile day-2 | Only if needed |

### Explicitly out of scope until phase 1 done

- New parallel deploy paths in TUI
- Cloudflare API from netductor
- Public installer on primary
- Reviving huh deploy wizards

### Related docs

- [DOMAIN.md](DOMAIN.md) — DNS external; domain set / LE  
- [DEPLOY-MAC.md](DEPLOY-MAC.md) — Mac two-VPS flow  
- [FLEET.md](FLEET.md) — primary/secondary roles  
- [UI-PARITY.md](UI-PARITY.md) — historical parity notes  
- [PORTS.md](PORTS.md) — exposure rules  
