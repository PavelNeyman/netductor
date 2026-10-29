**EN** · [RU](../ru/reviews/2026-10-29-WORK-SUMMARY.md)

# Work summary for review — 2026-10-28 … 2026-10-29

> Purpose: single handoff document for external/code review.  
> Repo: `PavelNeyman/netductor`  
> Tip at write time: track `main`; commit chain in §2.  
> **No secrets** in this file.

## 1. Timeline (high level)

| When | Theme | Result |
|------|--------|--------|
| Ops | Production Primary/Secondary on **v0.9.596** | Update applied; service plane OK; user VPN via A throughout |
| Ops | Secondary UFW inactive | Restored via `netductor firewall apply secondary`; baseline refreshed |
| Ops | Peer registry | Only peer `legacy-…` **slot 0** **active** — safe premise for future B |
| Ops | Stale edge mTLS clients (debug leftovers) | Revoked + removed `cudy-tr1200-izv11`, `tr1200-izvilistaya11`; kept `secondary-*` + plane certs |
| Docs | PR #1 update-system plan + audit policy | Merged to `main` (`e79e87b`) |
| Docs/Code | **Phase 0** baseline audit | `d00c78c` |
| Docs/Code | **Phase 1** update contract | `357a09d` |
| Docs/Code | This summary + **Phase 2** engine primitives | current commit |

Blue-Green **Secondary B** and full **Edge Cudy rollout** were **deferred** by operator priority: unified **update system** first.

## 2. Commit chain for reviewers (oldest → newest)

Review this range on `main`:

```text
e79e87b  Merge PR #1: update plan, audit policy/templates (EN/RU)
d00c78c  docs: Phase 0 update-system baseline audit (EN/RU)
357a09d  feat(update): Phase 1 component identity and update contract
c98ccf3  docs+feat(update): review work summary; Phase 2 resolve and job registry
(next)   seamless DownloadReleaseAsset → ResolveReleaseAsset; migration notes in this file
```

```bash
git log --oneline e79e87b^..HEAD
go test ./internal/update/ -count=1 -run 'Contract|AssetName|Resolve|JobRegistry|Legacy'
```

**Production baseline the operator already runs:** GitHub release **v0.9.596** (binaries built around `bf54587`).  
**Commits after that tag** on `main` are mostly docs + update-contract/engine primitives; they must remain **safe to pull** without a forced config rewrite (see §9).


## 3. Files added or changed (by area)

### 3.1 Unified update system (in-repo)

| Path | Action | What |
|------|--------|------|
| `docs/UPDATE-SYSTEM-IMPLEMENTATION-PLAN.md` | updated | Living checklist; Phase 0–1 `[x]` |
| `docs/ru/UPDATE-SYSTEM-IMPLEMENTATION-PLAN.md` | updated | RU parity |
| `docs/AUDIT-ARTIFACTS.md` / `docs/ru/…` | added (PR #1) | How to store audits |
| `docs/audits/TEMPLATE.md` / `docs/ru/audits/TEMPLATE.md` | added (PR #1) | Report template |
| `docs/audits/2026-10-29-phase0-update-baseline-e79e87b.md` | added | Phase 0 EN audit |
| `docs/ru/audits/2026-10-29-phase0-update-baseline-e79e87b.md` | added | Phase 0 RU audit |
| `docs/UPDATE-CONTRACT.md` | added | Phase 1 contract EN |
| `docs/ru/UPDATE-CONTRACT.md` | added | Phase 1 contract RU |
| `internal/update/contract.go` | added | ComponentID, JobState, AssetName, transitions |
| `internal/update/contract_test.go` | added | Transition + asset tests |
| `docs/README.md` / `docs/ru/README.md` | updated | Index links |
| `docs/reviews/2026-10-29-WORK-SUMMARY.md` | added | This file |
| `docs/ru/reviews/2026-10-29-WORK-SUMMARY.md` | added | RU summary |
| Phase 2 files | see §5 | ResolveAsset, job registry, path audit |

### 3.2 Release path (historical, already on main / release tag)

Relevant scripts (hardened across 0.9.593–0.9.596 workstream):

| Path | Role |
|------|------|
| `scripts/release.sh` | Clean tree, HEAD bind, multi-arch build, fail-closed upload, post SHA verify |
| `scripts/verify-release-binaries.sh` | Required asset set; version embed check via `grep -aF` |
| `scripts/check-versions.sh` | VERSION vs `internal/version.Release` |
| `.github/workflows/release-netductor.yml` | CI matrix; op **darwin-only** |

### 3.3 Diagnostics (optional)

| Path | Role |
|------|------|
| `scripts/nd-post-update-report.sh` | Sanitized P/S health report (still may need tighter redaction for VPN user dumps) |

### 3.4 Explicitly **not** changed in product runtime for Phases 0–1

- No change to live Primary/Secondary VPN data plane required by Phase 0–1 docs.
- Edge OpenWrt **not** provisioned (certs cleaned only).
- Secondary **B** not deployed.

## 4. Phase 0–1 outcomes (for reviewers)

**Phase 0:** Map of update entry points (CLI/API/TG/Web/Edge/op/deploy/CI). Findings P0-01…05 (multiple engines, SHA trust boundary, skip-verify, agent version ambiguity, addons out of scope).

**Phase 1:** Stable IDs `node|tg|agent|op`; job state machine; `AssetName`; clients must not invent assets. Tests: `go test ./internal/update/`.

## 5. Phase 2 (in this same change set)

Primitives toward a secure engine:

- Reconfirm existing paths in `docs/audits/…-phase2-…` (or section below).
- `ResolveReleaseAsset(component, platform, tag)` — uses contract `AssetName` + `FetchSHA256SUMS`; rejects empty/unsupported.
- In-process job registry: one active job per `(component, instance)`.
- Document: SHA256SUMS is **not** independent provenance (same channel as binary).

Full failure-matrix integration against every live apply path may continue in follow-up commits; checkboxes only marked when evidence exists.

## 6. Ops evidence (sanitized)

- Primary/Secondary **0.9.596**; SP/PS peer ping OK; A slot 0.
- Secondary UFW: was inactive with rules on disk → `firewall apply secondary` → active, doctor firewall OK.
- mTLS clients dir after cleanup: only `secondary-04c55cee…` under `/etc/netductor/secrets/mtls/clients/`.

## 7. How to review

```bash
git log --oneline e79e87b^..HEAD
go test ./internal/update/ -count=1
# read
#   docs/UPDATE-SYSTEM-IMPLEMENTATION-PLAN.md
#   docs/UPDATE-CONTRACT.md
#   docs/audits/2026-10-29-phase0-update-baseline-e79e87b.md
#   docs/reviews/2026-10-29-WORK-SUMMARY.md
```

## 8. Suggested review focus

1. Is the component/instance split correct for Secondary node vs Edge agent?  
2. Is the job state graph missing transitions needed by real apply?  
3. Phase 2: should `DownloadNamedAsset` be forced to go only through `ResolveReleaseAsset`?  
4. Residual: `--skip-verify`, unsigned manifest, TG apply confirm policy.

## 9. Seamless upgrade from v0.9.596 (operator requirements)

Operator production is already on **v0.9.596**. Work on the unified update system must not require a painful cutover. Tracked **here** (not by rewriting the implementation-plan checkboxes).

| ID | Requirement | Status | Notes |
|----|-------------|--------|-------|
| **M1** | Day-2 binary update from **v0.9.596** keeps working with existing CLI: `netductor update check\|list\|apply [--component node\|tg\|agent]` | **Done (compat path)** | Same flags; `DownloadReleaseAsset` resolves assets via Phase 1 `AssetName` + SHA256SUMS |
| **M2** | Config / state migration on Primary & Secondary when installing post-0.9.596 builds | **Done (none required)** | Phases 0–2 add docs, `contract` types, resolve/job registry — **no** new mandatory on-disk schema for existing installs |
| **M3** | Existing TG **Updates** / Web update UI keep working; no forced new menu | **Done (no break)** | Handlers still call existing update APIs; Phase 5–6 may *add* catalog UX later without removing old entry points first |
| **M4** | Automatic adoption of the new update **engine** under the hood without a separate “switch” step | **In progress** | CLI/API download path uses `ResolveReleaseAsset`; Edge remains `agent_update` + confirm until Phase 2 wiring completes; JobRegistry not yet mandatory for all apply paths |
| **M5** | Document rollback if a post-0.9.596 binary misbehaves | **Done (ops)** | Keep v0.9.596 release assets; snapshot VPS before apply; `--no-backup` remains opt-in on CLI |
| **M6** | No silent change of Secondary ↔ Edge identity during node update | **Done (contract)** | Job targets `(component, instance)`; updating `node` must not restart unrelated `agent` instances (enforced in design; runtime still per existing unit scripts) |

### Migration narrative for reviewers

1. Hosts on **v0.9.596** continue to use published release assets and current unit layout.  
2. Pulling newer `main` **source** or installing a newer **node** build that only contains contract/resolve changes should not require editing `/etc/netductor` by hand.  
3. When a future release **fully** replaces the apply pipeline, it must still accept the same CLI/API surface so operators are not taught a second tool on day one.  
4. UI “transition” = progressive enhancement (Phase 5–6), not a flag day.

### Explicit non-goals of this migration slice

- Requiring re-enrollment of Edge or Secondary.  
- Rewriting `vpn-users.json` / svc-paths material.  
- Mandatory GitHub token for checks that already work anonymously (rate limits may still apply).
