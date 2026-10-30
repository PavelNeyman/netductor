# Work summary for review — Unified update system (2026-10-29+)

**Living document.** Update this file on every phase commit so the next reviewer has a single entry point.

## 1. Goal

One safe, observable update path for Netductor components (node / tg / agent / op) across CLI (and later API/Web/TG), with clear identity, integrity, health, and recovery — without breaking live Primary/Secondary on **v0.9.596** mid-migration.

## 2. Commit chain (review from here)

```text
e79e87b  Merge PR #1: update plan + audit policy
d00c78c  Phase 0 baseline audit
357a09d  Phase 1 contract + tests
c98ccf3  work summary + Phase 2 resolve/jobreg
9d6d77c  DownloadReleaseAsset → ResolveReleaseAsset
a16f7b1  docs pin chain
8affa9e  ApplyHostComponent + health unit
855a27a  docs ApplyHost
d1e00b6  Phase 2 atomic install + version probe (0.9.597)
ec1eebe  Phase 2 failure matrix + node job + recovery (0.9.598)
66411c6  Phase 3 version semantics + asset matrix + UI label (0.9.599)
56f69f1  Phase 4 Edge identity (0.9.600)
d5d3ec4  Phase 5 TG Versions jobs+confirm (0.9.601)
aebeea6  Phase 6 Web jobs+ApplyHost (0.9.602)
fcaee96  Phase 7 i18n (0.9.603)
```

```bash
git log --oneline e79e87b^..HEAD
go test ./internal/update/ -count=1
bash scripts/check-versions.sh
bash scripts/check-update-asset-matrix.sh
```

**Production operator baseline:** release **v0.9.596** may still be what hosts run. Tips **0.9.597+** are mainline engine work; publish a GH release before RemoteJoin/stack apply of those versions.

## 3. Phases status

| Phase | Status | Evidence |
|-------|--------|----------|
| 0 Audit | **done** | `docs/audits/2026-10-29-phase0-update-baseline-e79e87b.md` |
| 1 Contract | **done** | `docs/UPDATE-CONTRACT.md`, `internal/update/contract.go` |
| 2 Engine | **done** | ApplyHost, Resolve, JobRegistry, failures, recovery doc, node Begin/Finish |
| 3 Release/version | **done** | `VERSION-SEMANTICS.md`, `manifest.go`, CI scripts, `UserFacingVersionLabel` |
| 4 Edge identity | **done** | `EDGE-AGENT-IDENTITY.md`, `edge_identity.go`, NormalizeEdgeGOARCH |
| 5 TG UI | **done** | Versions hub: jobs snippet, confirm apply, conflict gate |
| 6 Web UI | **done** | status.jobs, ApplyHost API, Updates card |
| 7 i18n/docs | **done** | UPDATE-I18N, ver_*/upd_* keys, check-update-i18n.sh |
| 8 CLI/runbooks | **done** | UPDATE-CLI, UPDATE-RUNBOOK, jobs subcommand |
| 6 Web UI | open | |
| 7+ Rollout | open | |

## 4. Code map

| Path | Role |
|------|------|
| `internal/update/contract.go` | ComponentID, JobState graph, AssetName |
| `internal/update/resolve.go` | ResolveReleaseAsset |
| `internal/update/jobreg.go` | one active job per (component, instance) |
| `internal/update/apply_host.go` | tg/agent: download `.new`, `.prev`, health, version probe |
| `internal/update/failures.go` | ClassifyFailure |
| `internal/update/node_job.go` | stack.Apply observability in registry |
| `internal/update/manifest.go` | required assets, SUMS cover, UI version label |
| `internal/stack/stack.go` | node still downloads node+tg; BeginNodeApply/FinishNodeApply |
| `internal/version` | Release, Commit, Running() |
| `docs/UPDATE-RECOVERY.md` | auto vs operator rollback |
| `docs/VERSION-SEMANTICS.md` | desired vs running rules |

## 5. Seamlessness (operator rules)

- CLI flags **may** change if engine needs it; hosts must still update cleanly.
- Config **migrate** when formats change.
- UI: do not break; **add** surfaces for catalog/job when API exists.
- Engine under the hood for tg/agent; node via stack + registry.

## 6. Ops context (same day, not this phase)

- Secondary **A** UFW: was inactive; fixed with `firewall apply secondary`; baseline refreshed.
- Isolated **B** / cutover: still **not** declared ready; Promote=false only when planned.
- OpenWrt safety commits earlier (sysupgrade, uciBatch, USB) — separate from update-engine phases.

## 7. How to review

1. Read this summary + `UPDATE-SYSTEM-IMPLEMENTATION-PLAN.md` checkboxes.
2. `go test ./internal/update/ -count=1`
3. Spot-check transitions in `contract.go` and apply path in `apply_host.go`.
4. Confirm `UserFacingVersionLabel` never returns `running` without `confirmedRunning`.
5. Do **not** treat unit tests as R5 / live secondary cutover proof.

## 8. Open for next phases

- Phase 5–6: TG/Web update UI on shared job API
- Edge device e2e upgrade matrix (lab)
- Independent release signature trust
- Independent signature trust (beyond SHA256SUMS from same host)
- Publish GH release when operators should run tip binaries


## Phase 9 (2026-10-30)

- Added `internal/update/phase9_test.go` (tag parse, unsupported arch, concurrent Begin, jobs snapshot, safety asset names, failure classes).
- Audit: `docs/audits/2026-10-30-update-phase9-verification.md` (+ RU copy).
- Plan: unit/lifecycle/failure/safety/checkboxes partially [x] with OPEN notes for dual-VPS, Edge, live UI.
- **Not claimed:** full Phase 9 parent done; integration + canary remain.



## Phase 9 status (honest)

| Slice | Status |
|-------|--------|
| Unit / contract / host apply / failure classes | **Closed in code** (`go test ./internal/update/`) |
| Integration Primary→Secondary / Edge / live UI | **OPEN** |
| Live/canary validation | **OPEN** |

Phase 9 **parent acceptance is not fully met** until integration + live evidence exists.

## Phase 10 (2026-10-30)

- Documented staged order + rollback: `docs/UPDATE-STAGED-ROLLOUT.md` (+ RU)
- Code-side checklist items marked where engines already landed (CLI/API/TG/Web share path; `.prev` / stack rollback)
- Edge canary, fleet evidence for legacy removal, live go/no-go remain **OPEN**



## Code close-out (0.9.616)

- `RollbackHostComponent` for tg/agent (`.prev`); tests
- CLI/API parity: jobs + rollback
- Contract allows `queued → rollback_in_progress`
- Still OPEN without stand: dual-VPS integration, Edge canary, live go/no-go

