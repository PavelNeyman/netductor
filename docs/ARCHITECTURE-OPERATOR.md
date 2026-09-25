# Operator architecture (Mac client · Node API)

**Status:** locked · **Baseline:** v0.9.2+

## Product split

| Binary | Role |
|--------|------|
| **netductor-op** (Mac) | **Only human UI** + deploy orchestration: WebUI, TUI, SSH, credentials |
| **netductor** (VPS) | **Server only**: JSON API, vpn, agents, backup, recovery — **not** a product web console |

Migration plan: [PLAN-MAC-CLIENT.md](PLAN-MAC-CLIENT.md).

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

Same Go module/repo. **Binary split (v0.9+):**

| Binary | Asset | Role |
|--------|--------|------|
| **netductor-op** | `netductor-op-darwin-*` / `netductor-op-linux-*` | Workstation: `deploy`, `operator serve`, `credentials`, Setup TUI |
| **netductor** | `netductor-linux-*` | VPS node: `install`, `serve`, vpn, doctor, secondary, edge plane |

Packages: `./cmd/netductor-op` (workstation) and `./cmd/netductor` (node).  
**Deploy must install the node asset on VPS** — never copy netductor-op onto a server.

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
| **4** | Binary split + optional mobile day-2 | netductor-op vs netductor-linux node assets |

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


## Operator HTTP security (0.8.98)

- Bind loopback only
- Random or `--token` / `NETDUCTOR_OPERATOR_TOKEN`
- Header `X-Netductor-Token` required on `POST /v1/fleet`
- Single in-flight deploy (409 if busy)
- Hostnames validated (no shell metacharacters)


## Operator WebUI (0.9.1)

`netductor-op operator serve` → http://127.0.0.1:7373/

| Endpoint | Purpose |
|----------|---------|
| GET /v1/meta | version, credentials_dir |
| POST /v1/fleet | full fleet (stream steps) |
| POST /v1/primary | primary only |
| POST /v1/secondary | secondary only |
| POST /v1/credentials | collect to ~/.netductor/credentials |

Auth: `X-Netductor-Token`. UI tabs + EN/RU + step chips. Not a public installer.
