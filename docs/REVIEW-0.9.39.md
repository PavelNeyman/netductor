# Code & security review — v0.9.39

**Date:** 2026-09-26 · **Scope:** full tree with focus on operator/deploy, node API, TG/Web day-2, Mac deploy parity.

## Executive summary

| Area | Verdict |
|------|---------|
| Mac deploy TUI ↔ Web | **Parity OK** — both call `internal/operator` → `internal/deploy` (Mac-direct SSH) |
| CLI deploy | Same packages (`netductor-op deploy …`) |
| Operator Web bind | **Loopback + token** (SHA-256 constant-time) |
| Node API | Session required; public bind needs `NETDUCTOR_API_PUBLIC=1` |
| Agent plane | mTLS :8789; plain :8788 only `PLAIN_AGENT=1` |
| Day-2 surfaces | CLI / TUI Ops / Web Control / TG — functional parity for core ops |
| EN/RU | TUI + Web Control + TG; residual hard-coded English only in Advanced/edge edge-cases |
| Critical vulns | **None new** in this pass |

## Architecture (locked)

```
Mac: netductor-op
  ├─ TUI wizard / CLI deploy  ─┐
  └─ operator serve WebUI     ─┴─→ internal/operator.*Deploy* → internal/deploy (SSH/SCP)
VPS: netductor (node) — API :8787, VPN, agents, TG addon
```

Deploy **never** hops primary→secondary for provision; secondary is Mac-direct.

### Deploy entry points

| Path | Primary | Secondary | Fleet | Edge | Site/MikroTik |
|------|---------|-----------|-------|------|---------------|
| TUI Wizard | `DeployPrimary` | `DeploySecondary` | `FleetDeployWithReport` | `DeployEdge` | `DeploySite` / `MikroTikAction` |
| Web `/v1/*` | POST primary | POST secondary | POST fleet | POST edge | POST site / mikrotik |
| CLI `netductor-op deploy` | ✅ | ✅ | ✅ | ✅ | ✅ |

Shared field mapping: `PrimaryFromFields` / `FleetFromFields` / specs in `internal/operator`.

## Security

### Strengths

- `ValidHost` / `ValidUser` / `ValidDeviceID` on operator deploy paths
- SSH/SCP use `--` separator; password via `SSHPASS` / ASKPASS (not argv)
- `shellQuote` on remote argv fragments
- Operator token: SHA-256 + `subtle.ConstantTimeCompare`
- Secondary/edge tokens: SHA-based compares
- Content-Disposition / BackupPath hardened
- Rate-limit clientIP trusts XFF only if `NETDUCTOR_TRUST_PROXY=1`
- Recovery HTTP: off until **arm**, TTL ≤2h; default bind `0.0.0.0` **while armed** (by design); optional `ALLOW_CIDR`
- Redirect import: scheme allowlist

### Residual risks (ops, not code bugs)

See [RESIDUAL_RISKS.md](RESIDUAL_RISKS.md):

- Recovery WAN while armed (token + TTL + optional CIDR)
- `PLAIN_AGENT` / `CLAIM_FIRST` / `API_PUBLIC` footguns (doctor WARN/FAIL)
- Local camera / SNI probe `InsecureSkipVerify` (intentional)
- Formula still pins 0.8.98 release assets until first v0.9 brew cut

### No change required this pass

Recovery `0.0.0.0` default while armed remains product intent (wiped primary recover).

## API coverage

### Node session API (`netductor serve`)

~110 routes under `/api/*` and `/vpn/*`. Day-2 clients:

| Client | Coverage |
|--------|----------|
| **Web Control** | Broad: catalog GET/POST + special forms (DNS set, backup schedule, VPN, edge, NVR, git, nodes…) + Advanced generic path |
| **opcatalog** | Shared registry for TG Tools + Web button labels; GETs + simple POSTs; forms stay UI-specific |
| **TG** | Product hubs (Users, DNS, Backup, NVR…) + catalog sections (`m:ops:` / `m:op:`) |
| **TUI Ops** | `netductor …` local/remote for doctor, dns, backup, git, registry, domain, vpn, edge, nvr… |
| **CLI** | Full node surface |

Not every rare edge/NVR path is a catalog button; Advanced Control or CLI covers remainder.

### Operator API (`netductor-op operator serve`)

`/v1/fleet|primary|secondary|edge|site|mikrotik|credentials|tunnel/*|session/*|catalog|node/*` — sufficient for Installer + Control proxy.

## Bilingual (EN/RU)

| Surface | Status |
|---------|--------|
| TUI | `langEN` / `langRU`, FormT / TT |
| Web | `I18N.en` / `I18N.ru`, `data-i18n`, catalog `label_en`/`label_ru` |
| TG | `i18n.go` dual maps |
| Gaps | Some Advanced/raw API paths; rare installer edge labels (improved 0.9.39–0.9.40) |

## UI parity matrix (day-2)

| Capability | CLI | TUI | Web | TG |
|------------|-----|-----|-----|-----|
| Deploy fleet/primary/secondary/edge/site | ✅ op | ✅ | ✅ | ❌ by design |
| Doctor / domain / metrics | ✅ | ✅ | ✅ | ✅ catalog |
| VPN users | ✅ | ✅ | ✅ | ✅ compact |
| DNS lists | ✅ | ✅ Ops | ✅ | ✅ |
| Backup schedule/list | ✅ | ✅ Ops | ✅ | ✅ |
| Git / registry | ✅ | ✅ Ops | ✅ | ✅ body |
| NVR | ✅ | ✅ | ✅ | ✅ |
| SSH hosts TOFU | ✅ | — | — | ✅ body |

## Testing notes (this pass)

- `go test` operator, edge, httpx, secondary, session — OK
- Static review of deploy/serve/token paths — OK
- Live VPS deploy not re-run in this session

## Recommendations for next chat

1. Optional: brew Formula → `netductor-op` + v0.9.x SHA after release tag  
2. Hardware e2e (OpenWrt / Tapo / MikroTik) — owner  
3. CDN-XHTTP — idea only ([OPEN_ITEMS](OPEN_ITEMS.md))  
4. Keep parity rule: new deploy fields → Spec + operator + thin TUI/Web together  
