**EN** · [RU](../ru/audits/2026-10-29-phase0-update-baseline-e79e87b.md)

# Phase 0 — Unified update system baseline audit

> Status: **final** (source inspection only; no production apply exercised in this audit)  
> Report date: 2026-10-29  
> Repository: `PavelNeyman/netductor`  
> Base commit: `e79e87b07999f9e2de3d129de4489bbbe531e2ff`  
> Release/tag: **v0.9.596** (`VERSION` file = `0.9.596`; GitHub release tag `v0.9.596` exists)  
> Scope: map of all component-update entry points, artifact matrix, trust/rollback notes, TG/Web conventions, open questions  
> Language pair: [RU](../ru/audits/2026-10-29-phase0-update-baseline-e79e87b.md)

## 1. Executive summary

- **Overall assessment:** Netductor already has **several independent update paths** (node self-update via GitHub releases, operator self-update, Edge `agent_update`, secondary install/join ship, addons, local build/ship). There is **no single component catalog or shared job state machine** as required by the unified plan.
- **Critical/high-risk conclusion:** Trust is primarily **SHA-256 from the same GitHub Release as the binary** (not an independent signed provenance). Cross-component updates are operator-driven and uneven (Edge has transactional binary replace + confirm; node uses `internal/update` + optional restart).
- **Confirmed:** entry-point map below; release asset matrix for v0.9.596; agent update confirm=yes + ELF + prev rollback pattern; release scripts (`release.sh`, `verify-release-binaries.sh`, `check-versions.sh`).
- **Unknown / live-fleet:** end-to-end `netductor update apply` on production Primary was not re-run in this audit session; Secondary host `netductor-agent` PATH version skew observed earlier in ops (0.9.151 vs reported 0.9.596) remains an ops finding outside this static map.

## 2. Scope and method

- **Questions:** What initiates updates? Which binary/asset? What integrity checks? Atomic install? Restart? Health? Rollback? UI?
- **Inspected:** `cmd/netductor/cli_misc.go` (`runUpdate`), `cmd/netductor/api_update.go`, `internal/update/*`, `cmd/netductor-agent/agent_ops.go` (`agentUpdate`), `cmd/netductor/cli_edge.go` (`buildAgentUpdateArg`), `cmd/netductor-op/cli_update.go`, `cmd/netductor-tg/handlers_versions.go`, `handlers_extra.go` (`handleUpdatesCB`), `scripts/release.sh`, `scripts/verify-release-binaries.sh`, `scripts/check-versions.sh`, `.github/workflows/release-netductor.yml`, `docs/UPDATE-SYSTEM-IMPLEMENTATION-PLAN.md`.
- **Commands:** `git rev-parse HEAD`, `cat VERSION`, `rg` over tree (static). **`go test` / live update apply: NOT RUN** in this Phase 0.
- **Exclusions:** full Web UI JS inventory line-by-line; production SSH apply; Blue-Green secondary deploy (ops track).

## 3. Environment and verification status

| Check | Exact command | Environment | Result | Evidence |
|---|---|---|---|---|
| HEAD SHA | `git rev-parse HEAD` | clone main | PASS | `e79e87b07999f9e2de3d129de4489bbbe531e2ff` |
| VERSION file | `cat VERSION` | same | PASS | `0.9.596` |
| Tag present | `git tag -l 'v0.9.59*'` | same | PASS | `v0.9.596` listed |
| Unit tests | `go test ./...` | — | **NOT RUN** | Phase 0 is source map only |
| Live `update apply` | — | production | **NOT RUN** | prior ops confirmed hosts on 0.9.596 outside this report |

## 4. Baseline identity

| Field | Value |
|---|---|
| main HEAD | `e79e87b07999f9e2de3d129de4489bbbe531e2ff` |
| Merge | PR #1 docs update plan |
| VERSION / Release pin | 0.9.596 |
| Published release | https://github.com/PavelNeyman/netductor/releases/tag/v0.9.596 |
| Operator | macOS only (`netductor-op-darwin-*`); no linux op assets in release matrix |

## 5. Update entry-point map

| ID | Initiator | Target | Asset selection | Trust / integrity | Install atomicity | Restart / health | Rollback | Notes |
|---|---|---|---|---|---|---|---|---|
| U-CLI-NODE | `netductor update [apply] [tag] --component node\|tg\|agent` | Primary (or host) node/tg/agent binary | GitHub release asset by component + GOARCH; tag or latest | Download + SHA256SUMS verify unless `--skip-verify` | Via `internal/update` ship/apply path | Optional service restart (`--no-restart`) | Backup unless `--no-backup` | See `cli_misc.go` `runUpdate` |
| U-CLI-CHECK | `netductor update check\|list` | — (read-only) | Lists releases | GitHub API (+ optional token) | n/a | n/a | n/a | JSON status |
| U-API | `POST/GET /api/update/*` | Same engine as CLI | Same | Session auth | Same as CLI apply | As implemented in handler | As CLI | `api_update.go`: status, releases, apply, github-token |
| U-TG | Telegram **Updates** / versions menus | Operator-driven toward same backend patterns | Latest tag compare local/secondary | Token status shown; apply paths via callbacks | Delegates; not a second engine | UI status | Limited | `handlers_versions.go`, `handleUpdatesCB` |
| U-WEB | Web Control update surfaces | Same `/api/update/*` | Same | Session | Same | UI polling as existing | Same | Must reuse design-system; not fully line-audited |
| U-EDGE-CMD | `netductor edge cmd <id> agent_update` | Edge agent binary on device | `buildAgentUpdateArg`: arch from device + SHA256SUMS URL | **confirm=yes** required; SHA; ELF magic | Stage file, `.prev`, rename; rollback on failure | Agent process picks up binary (device-specific) | Restores `.prev` on failure | `agent_ops.go` `agentUpdate` |
| U-EDGE-AGENT | Device executes `agent_update` | Self | URL\|sha\|confirm | Same | Same | Same | Same | Fail-closed without confirm |
| U-OP | `netductor-op` update / brew | Operator binary on Mac | GitHub darwin assets or Homebrew | Release checksums / brew | Self-replace / Cellar | Operator restart | Brew path special-cases | `cmd/netductor-op/cli_update.go` |
| U-SEC-DEPLOY | `deploy` secondary / join ship | Secondary **node** install | Ship from primary/release | SSH + ship artifact | Install path in deploy | Units start | Re-deploy / snapshot | Not day-2 unified job |
| U-SEC-AGENT | Secondary may run agent unit | Agent on secondary | Same family as edge or node agent asset | As provisioned | As install | systemd | Manual | Ops saw PATH `netductor-agent` 0.9.151 vs node 0.9.596 — clarify which binary is authoritative |
| U-ADDONS | `addons update` / API | lampac, registry, git mirrors | Addon-specific | Varies | Varies | Varies | Varies | `internal/addons/update.go` — out of core node matrix |
| U-CI-REL | `scripts/release.sh`, GHA `release-netductor.yml` | Publish assets | Multi-arch build matrix | SHA256SUMS; post-upload SHA verify in release.sh; remote tag==HEAD | n/a publish | n/a | Re-publish / delete assets | Op = darwin only |
| U-LOCAL | `internal/update` local releases dir / build host | Offline/mirror tags | Local SHA256SUMS | File checksums | Import to local dir | n/a | Prune local | `LocalReleasesDir`, `Ship`, `BuildLocal` |

## 6. Component / artifact matrix (v0.9.596 release)

| Component logical ID (proposed) | Binary / asset pattern | Roles | Arch |
|---|---|---|---|
| `node` | `netductor-linux-{amd64,arm64}` | Primary control plane; Secondary node binary | linux amd64/arm64 |
| `tg` | `netductor-tg-linux-{amd64,arm64}` | Telegram bot on primary | linux amd64/arm64 |
| `agent` | `netductor-agent-linux-{amd64,arm64,arm,mipsle,riscv64}` | Edge OpenWrt / secondary agent | multi |
| `op` | `netductor-op-darwin-{arm64,amd64}` | Mac operator only | darwin |
| manifest | `SHA256SUMS` | All of the above | — |

**Gap vs unified plan:** no stable public “component ID + instance ID + job ID” API spanning node/tg/agent/edge; clients still know asset naming conventions.

## 7. Trust and integrity (confirmed from source)

| Mechanism | Present | Limitation |
|---|---|---|
| SHA256SUMS next to assets | Yes | Same publisher as binary (R-05 style residual) |
| `--skip-verify` | Yes | Dangerous escape hatch |
| Edge `confirm=yes` | Yes | Prevents accidental update |
| ELF check on agent_update | Yes | Format only |
| release.sh clean tree + tag==HEAD + post-download SHA | Yes (0.9.596 scripts) | Signed manifest still absent |
| VERSION vs `internal/version.Release` gate | `check-versions.sh` | Must match before release |

## 8. Telegram / Web conventions (audit notes)

### Telegram

- Entry: **Updates** / versions (`m:versions`, `m:updates`, nested `m:updates:*`).
- Reuses shared btn helpers and back to `m:menu`.
- Shows local vs latest, secondary version string, GitHub token status.
- Apply/edge/token flows in `handleUpdatesCB` — **must keep** existing card/keyboard patterns in Phase 5 (no parallel template system).
- Stale callback / auth: follow existing server-side patterns (Phase 5 must re-verify).

### Web UI

- Update API under `/api/update/*` with session.
- Localization: project i18n for web (existing `check-web-ui.sh` warns on missing keys — seen in release runs).
- Phase 6 must use design-system components only.

## 9. Findings

### P0-01 Multiple update engines without shared job model

- **Status:** confirmed  
- **Severity:** high (architecture debt for unified plan)  
- **Path:** `internal/update` vs `agentUpdate` vs op self-update vs deploy ship  
- **Impact:** inconsistent observability, permissions, and rollback UX across components  
- **Remediation:** Phase 1–2 contract + engine wrapping existing paths  
- **Regression test:** contract tests when Phase 1 lands  

### P0-02 Provenance = SHA from same release channel

- **Status:** confirmed  
- **Severity:** medium (known; plan Phase 2 calls out signed manifest)  
- **Evidence:** verify flow uses SHA256SUMS from GitHub release  
- **Remediation:** Phase 2 / R-05 style independent signing (deferred)  

### P0-03 `--skip-verify` on node update

- **Status:** confirmed  
- **Severity:** medium  
- **Path:** `cli_misc.go` sets `NETDUCTOR_UPDATE_SKIP_VERIFY`  
- **Remediation:** require explicit dual confirm; audit log; disallow from TG by default  

### P0-04 Secondary/Edge agent version identity ambiguity

- **Status:** hypothesis (ops evidence) / needs live clarification  
- **Severity:** medium  
- **Impact:** operator may update wrong binary or misread fleet version  
- **Remediation:** Phase 1 instance identity; document which unit/binary is source of truth on secondary  

### P0-05 Addons update path outside core matrix

- **Status:** confirmed  
- **Severity:** low / informational  
- **Remediation:** either fold into catalog as optional components or explicitly exclude from unified UI v1  

## 10. Open questions

1. Should Secondary **node** day-2 update always go through `netductor update --component node` on the secondary host, or only via primary-orchestrated job?
2. Is `netductor-agent` on secondary a required long-lived unit or legacy alongside node binary?
3. Web UI: exact pages/components that call `/api/update/apply` (line-level inventory for Phase 6).
4. Minimum supported from-version for Edge agent_update in production (ops currently targeting 0.9.596 assets).
5. Whether TG may ever call apply without operator dual-confirm on Primary.

## 11. Phase 0 acceptance checklist

| Deliverable | Status |
|---|---|
| Source-linked flow map | **Yes** §5 |
| Component/artifact matrix | **Yes** §6 |
| UI/i18n conventions map | **Yes** §8 (TG detailed; Web API-level) |
| Explicit open questions | **Yes** §10 |
| Code changes required | **None** for Phase 0 |

## 12. Next phase

Proceed to **Phase 1 — component identity and update contract** using this map; do not mark Phase 1 items complete without schema/tests and EN/RU doc updates in the same change.
