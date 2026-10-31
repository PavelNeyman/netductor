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



## Security-critical: Secondary SSH password → key-only (OPEN → in progress)

**TZ (2026-10-30):** password only for first connect; after pubkey install + independent key-only verify, all deploy uses key-only; harden with sshd -t + independent rollback; no password fallback; stage errors; remove `pkill -f`.

### Audit map (pre-change)

| Function | Auth today | When | Required |
|----------|------------|------|----------|
| `secondary.sshClient` | Password **and/or** PublicKeys (both offered) | ProvisionFromCore dial | Bootstrap: password only; after gate: key only |
| `InstallOperatorKeys` | same client session | After dial | Bootstrap session |
| Key-only verify | **missing** | — | New dial, PublicKeys only, `true` |
| install binary / RemoteJoin / units | **same bootstrap client** | Mid provision | KeyVerified |
| `DisablePasswordAuth` → `RemoteHardenScript` | same client | End | KeyVerified; harden needs sshd -t + rollback |
| `runSSHOnPort` (deploy after provision) | key path, empty password | DeploySecondary post | Key-only (already) |

Root issue: initial path clears `SecondarySSHKey` and keeps one password-capable session through join+harden; password stays in Auth methods if key also present.

See `docs/PLAN-SSH-KEYONLY-PROVISION.md`.


### Implementation progress (0.9.617)

| Item | Status |
|------|--------|
| Audit map + PLAN-SSH-KEYONLY-PROVISION | done |
| sshClientBootstrap password-only when password set | done |
| sshClientKeyOnly / verifyKeyOnly | done |
| ProvisionFromCore key gate + zero password | done |
| DeploySecondary passes PrimaryKey for gate | done |
| Harden: no pkill; sshd -t | done (minimal) |
| Independent systemd harden rollback | OPEN |
| Integration sshd lab | OPEN |

### 0.9.618 — two-phase SSH harden + independent rollback

| Item | Status |
|------|--------|
| `RemoteHardenPrepareScript` | Port 52222, keep :22, backup, `sshd -t`, arm 8m rollback timer |
| `RemoteHardenFinalizeScript` | Password off, deny :22, cancel timer, unit purge (no pkill) |
| ProvisionFromCore | prepare → close → key-only :52222 → finalize → key-only again |
| Integration sshd lab | still OPEN |


### 0.9.619 — SSHTransport model + stage IDs

| Item | Status |
|------|--------|
| `SSHTransport` Bootstrap → KeyVerified → Hardened | done; password zeroed on promote |
| Stage constants (`secondary-*`) | done |
| InstallOperatorKeys verifies lines present | done |
| Integration sshd lab / live Secondary | still OPEN |


### 0.9.620 — P0 harden rollback lifecycle (security review)

| Item | Status |
|------|--------|
| Finalize does **not** cancel rollback timer | done |
| Cancel only after post-finalize verifyKeyOnly | done |
| systemctl restart errors fail-closed | done |
| Mandatory SSH backup + fresh backup dir | done |
| UFW snapshot + restore in rollback unit | done |
| PromoteToHardened after proven access | done |

Still OPEN: live Secondary e2e, full sshd integration lab.


### 0.9.621 — harden P0 follow-up (review 0.9.620)

| Item | Status |
|------|--------|
| Arm rollback **before** UFW/SSH mutations | done |
| UFW policy absent/inactive/active + restore | done |
| Finalize: SSH only (no package purge) | done |
| Host agent purge **after** cancel + proven access | done |
| Cancel verifies timer inactive | done |
| PromoteToHardened error handled | done |


### 0.9.622 — cancel race + UFW/SSH rollback hardening

| Item | Status |
|------|--------|
| CANCELLED stamp before stop timer/service | done (rollback no-op if stamped) |
| Cancel checks timer **and** unit inactive | done |
| Rollback: SSH restore first; UFW best-effort after | done |
| UFW active requires user.rules snapshot | done |
| Socket enabled/active snapshot + restore | done |
| Legacy RemoteHardenScript / DisablePasswordAuth fail closed | done |


### 0.9.623 — flock mutual exclusion for cancel/rollback

| Item | Status |
|------|--------|
| Shared flock on backup/lock for rollback + cancel | done |
| CANCELLED re-checked under lock before restart | done |
| Cancel waits for unit inactive after stop | done |
| UFW restore = file copy + reload only | done |
| Socket masked/enabled/disabled restore | done |


### 0.9.624 — Prepare re-entry P0 + APT Repository Guard start

SSH:
- Stop/verify previous rollback timer+unit before rm -rf backup
- Prepare holds flock for mutation phase
- Socket snapshot preserves masked (stdout before exit code)
- Cancel: timeout on systemctl stop

APT Guard (stages 0–2):
- docs/PLAN-APT-REPOSITORY-GUARD.md integration map
- internal/aptguard policy + scanner (.list + .sources)

