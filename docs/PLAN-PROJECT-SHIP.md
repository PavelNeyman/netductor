# PLAN: Unified project ship (CI → pause → publish)

**Status:** agreed design · **not implemented** (as of 0.9.354)  
**EN** · related: [PLAN-INTERNAL-RELEASES.md](PLAN-INTERNAL-RELEASES.md), [PLAN-GIT-CI-OPS.md](PLAN-GIT-CI-OPS.md), [SELFHOST-GIT.md](SELFHOST-GIT.md)


## Decisions (locked 2026-10-12)

| # | Decision |
|--|--|
| D1 | **No job timeout.** Jobs may wait indefinitely. Operator deletes explicitly. |
| D2 | **`project job rm <id>`** (CLI + UI): drop snapshot + job-bound artifacts. Doctor may INFO count/age of waiting jobs (not alert spam). |
| D3 | **Resume = continue, do not redo ok steps.** Cursor advances only over pending steps. Completed step outputs kept under job id when needed for publish. |
| D4 | **Mac / external:** publish needs uploaded artifacts (`complete --from`); no silent rebuild on VPS for host=mac. |
| D5 | **VPS build steps:** if output files still present after pause, reuse; only re-run step if outputs missing (corruption/rm). Prefer not to re-test/re-build when status=ok. |
| D6 | **One product command** for the full cycle: tests → build → version bump → publish (+ optional middle steps). Name: **`project ship`** (primary). No parallel “build without publish” as a first-class flow. |
| D7 | **No long-lived aliases.** Do not keep `release ship` / `git project build` as wrappers. Replace call sites (TG/Web/docs) with `project ship` in the same change set. Old subcommands may print one-line “use project ship” and exit 2 for a short period only if needed for safety — prefer hard cut in implementing release. |
| D8 | **Auto version** patch+1 per project; `--tag` override. |
| D9 | **VPS slot max 1** for running/publishing; unlimited paused/waiting_external. |
| D10 | Implementation order **S0→S6** after this lock; no parallel feature work in this track. |

## Goal

One operator command / UI action for **any** registered project:

```bash
netductor project ship <name> [--tag vX.Y.Z] [--ref REF]
```

Pipeline is driven by **project meta + workflow YAML**. Same UX for netductor, bots, and Mac-only apps. Failed tests **never** publish. Pause does **not** hold the VPS compute slot.

## Non-goals

- Full GitHub Actions compatibility (`uses:`, hosted runners, matrix fan-out on GH).
- CRIU / freeze of a live container mid-step.
- Unlimited parallel **running** builds on primary (RAM).
- Replacing `stack apply` (runtime install stays separate; ship only **produces** artifacts).

---

## 1. Two resource axes (locked)

| Axis | Limit | Examples |
|--|--|--|
| **VPS compute slot** | **max 1** `running` / `publishing` job | `go test`, matrix build, docker |
| **Wait / pause** | **N ≥ 0**, no CPU slot | `paused`, `waiting_external` (Mac, approval) |

A job in `waiting_mac` or `paused` sets `vps_slot=false`. Another project (e.g. netductor) may take the slot immediately. Ten iOS jobs waiting on Mac must not block VPS ship.

---

## 2. Job state machine

```text
queued
  → running          # holds vps_slot
  → paused           # after step boundary; vps_slot=false; snapshot only
  → waiting_external # needs Mac/builder/approval; vps_slot=false
  → publishing       # short; holds vps_slot
  → done | failed
```

**Pause request** (user or system): do **not** kill the current step. Finish current step → persist snapshot → remove containers/worktree → `paused` or `waiting_external`.

**Resume / external complete**: acquire VPS slot → **do not re-run steps with status=ok** (D3) → use cached outputs when present → run only pending steps from `cursor`.

### Snapshot (minimal disk)

```json
{
  "id": "niimbot-ios-1734…",
  "project": "niimbot-ios",
  "ref": "abc123",
  "tag": "v0.2.1",
  "state": "waiting_external",
  "reason": "mac",
  "vps_slot": false,
  "cursor": 2,
  "steps": [
    {"name": "test", "status": "ok", "log": "git-artifacts/…/test.log"},
    {"name": "build", "status": "pending"},
    {"name": "publish", "status": "pending"}
  ],
  "created_at": "…",
  "wait_since": "…",
  "timeout_at": "…"
}
```

Store: `/var/lib/netductor/ship-jobs/<id>.json` (or single queue file with indices).  
Logs: under `git-artifacts/` with prune. **No** long-lived docker, **no** full worktree while waiting (optional small output cache for `dist/`).

---

## 3. Unified `ship` flow

```text
project ship NAME
  → resolve tag (auto patch+1 per project, or --tag)
  → resolve ref (default HEAD/main)
  → load workflow + project.Host / publish target
  → enqueue job
  → VPS worker (if slot free):
       for step in steps:
         if pause_requested: checkpoint; release slot; return
         run step (container, sequential)
         if fail: state=failed; no publish
         if step needs external builder: state=waiting_external; release slot
       publish; state=done
```

### Step sources

| Source | Role |
|--|--|
| `.github/workflows/<name>.yml` or `ci/netductor.yml` | `run:` steps (existing GHA subset) |
| `git-projects.json` | `host`: `vps` \| `mac`; `publish`: `releases` \| `artifacts` \| `registry` |
| Built-in for **netductor** | after tests → multi-arch matrix → `/var/lib/netductor/releases/<tag>/` (today’s `release ship`) |

### Publish targets

| `publish` | Output |
|--|--|
| `releases` | `/var/lib/netductor/releases/<tag>/` + SHA256SUMS (netductor product) |
| `artifacts` | `/var/lib/netductor/git-artifacts/<project>/<tag>/` |
| `registry` | local OCI `127.0.0.1:5000/...` when Dockerfile present |

GitHub Release upload remains **optional**, not required for prod apply.

---

## 4. External builder (Mac and beyond)

Not Mac-only: any `waiting_external` with `reason`.

| reason | Continue when |
|--|--|
| `mac` | op builds; `project complete <job_id>` or import bound to job |
| `approval` | operator confirms in TG/Web |
| `user` | explicit resume after manual pause |

**Mac queue:** many pending jobs allowed. Operator may complete in any order.  
CLI sketch:

```bash
netductor project ship niimbot-ios
netductor project jobs                    # running / paused / waiting_external
netductor project pause <job_id>          # after current step
netductor project resume <job_id>
netductor-op project build niimbot-ios --job <id>
netductor project complete <job_id> --from ./dist
```

---

## 5. Versioning

Per project (not only netductor):

- Baseline: highest of local published tags for that project, project `VERSION` file, optional upstream tags.
- Default: **patch +1**.
- Override: `--tag v1.0.0` (skip/jump).

`netductor project next <name>` prints next tag (mirror of `release next` for product).

---

## 6. Concurrency rules (locked)

1. At most **one** job with `vps_slot=true`.
2. Unlimited `paused` / `waiting_external` (disk policy applies).
3. Pause/wait **never** holds docker containers.
4. Cancel running: kill container by name (`NETDUCTOR_CI_CONTAINER` / tracked id), mark failed; already partially exists (PLAN-GIT-CI-OPS P6).
5. Queued VPS jobs FIFO; waiting jobs do not enter FIFO until resume/complete.

---

## 7. Retention & cleanup

| Path | Policy |
|--|--|
| `ship-jobs/*.json` | keep done/failed N days or last K per project |
| `git-artifacts/` | `release prune-artifacts --keep 20` (exists); ship auto-prune after job |
| `releases/` | `release prune --keep 5` (exists) |
| Job timeout | **none** (D1). Delete via `project job rm`. |

Wire into `netductor cleanup` dry-run/apply.

---

## 8. UI parity

| UI | |
|--|--|
| CLI | `project ship\|jobs\|pause\|resume\|complete\|next` |
| TG | Tools → Git/Projects: Ship, job list (Running vs Waiting), pause/resume |
| Web | same actions via API |
| API | session-protected `/api/git/projects/ship`, `/jobs`, … |

One backend; thin UIs (existing architecture rule).

---

## 9. Implementation phases

| Phase | Deliverable |
|--|--|
| **S0** | Job JSON schema + store; states; `vps_slot`; list/status CLI |
| **S1** | VPS worker respects slot; `project ship` for `host=vps` + `publish=artifacts` (generic) |
| **S2** | Fold **netductor** `release ship` into `project ship netductor` (`publish=releases`) |
| **S3** | Pause after step + resume; no container while paused |
| **S4** | `waiting_external` + Mac complete/import by job id; multi-wait OK |
| **S5** | Per-project auto tag; TG/Web job board |
| **S6** | Timeouts, cleanup integration, metrics (`ship_running`, `ship_waiting`) |

Dependency: existing `gha.Run` step loop, `vps_queue`, `mac_queue`, `release ship` (0.9.354).

---

## 10. Gaps to agree (not in earlier chat)

| Topic | Proposal | Need decision? |
|--|--|--|
| **Worktree on resume** | Re-checkout same `ref`; re-run only pending steps. Cache `dist/` only if external builder uploaded it | Default OK? |
| **Secret exposure on Mac** | Op build uses local code; no primary secrets in Mac job payload unless explicit | OK |
| **Partial test fail** | Whole ship fails; no publish | OK |
| **Change ref while paused** | Invalidate job or forbid | Prefer **forbid** |
| **Priority** | Optional `priority` field later; FIFO first | Defer |
| **Concurrent Mac completes** | Serialize `publishing` via VPS slot only | OK |
| **Metrics/alerts** | TG on fail, timeout, and “N waiting Mac” digest daily | Agree thresholds |
| **Disk quota** | Soft cap on `git-artifacts` total size in doctor | Yes later |
| **Idempotent publish** | Re-ship same tag overwrites store with confirm flag | Agree |
| **netductor VERSION in git** | Ship pins VERSION in artifact only; optional commit/tag push to bare+GH | Defer (mirror-push exists) |
| **Windows/other builders** | Same as Mac via `reason=external` + different op CLI later | Future |
| **Security** | Job complete API requires operator session; artifact paths contained under state dir | Required |

---

## 11. Acceptance criteria

1. `project ship netductor` (or alias) tests → matrix → local releases; fail tests ⇒ no tag dir.
2. `project ship other-go-app` tests → build → artifacts; auto tag.
3. Start `host=mac` project → `waiting_external`; meanwhile `ship netductor` runs on VPS.
4. Two+ Mac waits pending; complete out of order; each publishes once.
5. User pause during long test: current test finishes; containers gone; resume continues next step.
6. Doctor/cleanup reports ship job + artifact sizes; prune does not delete `waiting_*` outputs still referenced.
7. `project job rm` removes snapshot and job-bound artifacts; no automatic timeout failure.

---

## 12. CLI naming (hard cut — D6, D7)

| Command | Role |
|--|--|
| **`project ship <name>`** | Full cycle: test → build → version → publish |
| **`project next <name>`** | Print next auto tag |
| **`project jobs`** | List running / paused / waiting |
| **`project pause\|resume\|rm <id>`** | Control |
| **`project complete <id> --from <dir>`** | External/Mac artifact handoff |

`release ship`, `git project build` as product entry points are **removed** in the implementing release (or exit 2 with message once). Mac queue merges into ship jobs (`reason=mac`). Docs/TG/Web updated in the same PR.
