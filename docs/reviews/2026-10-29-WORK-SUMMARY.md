**EN** · [RU](../ru/reviews/2026-10-29-WORK-SUMMARY.md)

# Work summary for review — 2026-10-28 … 2026-10-29

> Purpose: single handoff document for external/code review.  
> Repo: `PavelNeyman/netductor`  
> Tip at write time: see latest `main` (includes Phase 0–1 commits below).  
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

## 2. Git commits (documentation / update system)

| SHA | Summary |
|-----|---------|
| `e79e87b` | Merge PR #1: update implementation plan, audit templates, EN/RU policy |
| `d00c78c` | Phase 0 audit pair + plan Phase 0 checkboxes |
| `357a09d` | Phase 1 `internal/update/contract.go` + tests + UPDATE-CONTRACT EN/RU |
| *(this)* | Work summary + Phase 2 resolve/job primitives |

Release **v0.9.596** assets were published earlier to GitHub Releases (tag `v0.9.596`, commit `bf54587` era builds); tip `main` may be ahead of that tag with docs/contract only.

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
