# Code fix backlog after Secondary B deploy (2026-11)

**Status:** open — **CF-1…CF-5, CF-7, CF-9…CF-13 through 0.9.691; CF-14 naming recorded; CF-15 upgrade floors planned**; remaining CF-6, CF-8, CF-14, CF-15  
**Scope:** code changes only (not lab/hardware e2e, not pure operator steps)  
**Context:** second Secondary `195.140.147.20` (peer `peer-195-140-147-20`, slot 1) was deployed and **promoted** 2026-11-06; A remains `ready` for observation.

Related evidence:

- [audits/PROBE-TCP-FALSE-POSITIVE-2026-11-06.md](audits/PROBE-TCP-FALSE-POSITIVE-2026-11-06.md)
- Operator notes in [SECONDARY-FIRST-DEPLOY.md](SECONDARY-FIRST-DEPLOY.md)
- Deploy friction captured during B provisioning (journal, redaction, UFW, join, SP/PS)

---

## Priority order

| P | ID | Title | Area |
|---|-----|--------|------|
| P0 | **CF-1** | Remote TCP probe must not succeed via local `ss` | `internal/probes` |
| P0 | **CF-2** | Uplink watchdog must not restart sing-box solely on SP/PS fail streak | secondary agent |
| P1 | **CF-3** | Slot-aware agent / heartbeat / uplink VIP (no slot-0 addresses on slot-1 peer) | svc-paths + agent |
| P1 | **CF-4** | Prepare ↔ host-baseline UFW policy contradiction on `:22` mid-transition | hardening + firewall |
| P1 | **CF-5** | First-deploy hoster firewall cleanup (optional automated preflight remediation) | deploy / baseline |
| P2 | **CF-6** | Residual SSH harden edge cases on re-deploy of already-hardened host | hardening |
| P2 | **CF-7** | Integrity drift after manual binary replace (VERSION vs binary) | stack / doctor |
| P2 | **CF-8** | Secondary safety plan residual test debt (focused tests never run in CI notes) | tests / CI |
| P1 | **CF-9** | Secondary `pullAndApply` ignores sing-box restart failure | `internal/secondary/agent.go` |
| P1 | **CF-10** | Primary `ApplyConfig` ignores local sing-box restart + `bumpSecondaryConfigVer` errors | `internal/vpn/apply.go` |
| P2 | **CF-11** | `bumpSecondaryConfigVer` silent no-op / dual path `secondary`+`relay` | `internal/vpn/apply.go` |
| P1 | **CF-12** | APT Guard default policy for Debian 13 (Trixie) | `internal/aptguard` |
| P1 | **CF-13** | `stack apply` / host update path after manual binary replace | `internal/stack`, update engine |
| P2 | **CF-14** | Naming: role control/egress + region (user-facing geo in sub) | docs, API, UI, role file |
| P1 | **CF-15** | Upgrade floors: auto `min_from` from code + CI (step only on break) | migrate, update/stack, release CI |
| P1 | **CF-16** | PeerRemove UFW 8789 ownership comment mismatch (`netductor-api-peer`) | `svcpaths` peer_ops |

Items already shipped during B firefight (do **not** re-open): opaque SSH errors (0.9.664), mTLS without SSH stdin tar (0.9.665), join HTTP/1.1 + error tails (0.9.666–672), UFW routed=disabled + keep :22 (0.9.668), harden UFW idempotency / mawk / pipefail (0.9.674–683), prepared `/sub` public IP (0.9.682–683).

---

## CF-1 — Remote TCP probe false positive (P0) — **done 0.9.685**

**Evidence:** [PROBE-TCP-FALSE-POSITIVE-2026-11-06.md](audits/PROBE-TCP-FALSE-POSITIVE-2026-11-06.md)

**Code:** `internal/probes/run.go` `probeTCP` / `probeUDP`

**Bug:** On dial failure to a **remote** host, success is inferred if **any** local process listens on the same port (`ssListening`). Primary’s `*:443` (sing-box) marks Secondary B `:443` as `ok=true` with `ms≈5000` and `via=ss`.

**Fix:**

1. `probeTCP`/`probeUDP` return only the remote dial (or explicit remote UDP semantics) result.
2. Local listener health is a **separate** probe type/name (e.g. `local_listen`), never mixed into remote targets.
3. Regression test: local listener on 443 + failed dial to non-local IP → `ok=false`.

**Acceptance:** history for `secondary-*:443` never shows `ok=true` + timeout-length `ms` solely because Primary listens on 443.

---

## CF-2 — Watchdog restarts user plane on service-plane flap (P0) — **done 0.9.685**

**Evidence:** B agent logs 2026-11-05 and 2026-11-06 ~07:04–07:32 UTC — `service-plane tunnel probe (SP/PS) failed; fail streak=3` → `uplink watchdog: restarting sing-box` → client freezes.

**Code:** `internal/secondary/agent.go` (and related svc-paths apply)

**Bug:** SP/PS control-plane probe failures restart **sing-box** (user VLESS path), dropping client sessions even when public `:443` / uplink may still be fine.

**Fix direction:**

1. Separate **control-plane health** (agent↔primary via WG) from **user-plane health** (sing-box / uplink / Reality).
2. Restart sing-box only on user-plane failure criteria (e.g. local :443 down, uplink dial fail), not on SP/PS streak alone.
3. SP/PS failures: switch agent URL / alert / optional WG bounce — **not** wholesale sing-box restart.
4. Tests for streak thresholds and action matrix.

**Acceptance:** simulated SP down does not restart sing-box; user sessions stay up; alerts still fire.

---

## CF-3 — Slot-aware paths on multi-peer (P1) — **mostly done; residual 0.9.686**

**Evidence:** On slot-1 B, logs showed heartbeat/uplink targets including **slot-0** addresses (`10.87.11.1`, `10.87.10.1`) interleaved with slot-1 (`10.87.20.1` / `10.87.21.1`).

**Fix direction:**

1. Material + agent apply must pin agent URL and service VIP to **this peer’s slot** only.
2. Fail closed if material slot ≠ runtime iface/addr.
3. Doctor/peer validate: flag cross-slot URL.

**Acceptance:** on slot-1 peer, no steady-state traffic to 10.87.10/11 except explicit migration windows documented in code.

---

## CF-4 — Prepare vs host-baseline UFW on `:22` (P1) — **done 0.9.688**

**Evidence:** [SECONDARY_DEPLOYMENT_SAFETY_REMEDIATION_PLAN.md](SECONDARY_DEPLOYMENT_SAFETY_REMEDIATION_PLAN.md) residual notes; live B deploy required manual `ufw` comment / order surgery.

**Symptom:** prepare path and `host-baseline apply secondary` disagree on first WAN action for port 22 during transition to key-only `:52222`. Re-deploy / partial harden leaves operator without a policy that satisfies both.

**Fix direction:**

1. Single transition state machine: factory → key-verified → hardened, with one owner of UFW mutations.
2. After key-only `:52222` proven, baseline must accept hardened policy (deny 22 first) and must not require bootstrap ALLOW.
3. Mid-transition: baseline either no-ops SSH rules or runs only after finalize success.

**Acceptance:** clean and retry deploy on a host that already has 52222 key-only does not require manual `ufw` comment renames.

---

## CF-5 — First-deploy hoster shell (P1) — **done 0.9.691** (purge-hoster CLI + preflight hints)

**Evidence:** [SECONDARY-FIRST-DEPLOY.md](SECONDARY-FIRST-DEPLOY.md); live B had provider ALLOW 80 / ISPManager 1500 blocking baseline.

**Code today:** preflight **fails closed** if 80/8787–8789 already ALLOW — correct, but operator must clean manually.

**Optional product fix:**

1. Deploy preflight lists offending rules with exact `ufw delete` lines.
2. Optional explicit flag `--purge-hoster-allows` (or wizard confirm) that removes **only** documented hoster patterns (80, panel ports), never blind `ufw reset`.
3. Never auto-delete without confirmation in UI/CLI.

**Acceptance:** first deploy on stock hoster image either auto-clears known noise with confirm, or prints a one-shot remediation script; no silent reset.

---

## CF-6 — Harden re-deploy residuals (P2)

Cross-check after 0.9.674–683:

- [ ] Any remaining `netductor-ssh-bootstrap` **comment-only** requirements in prepare/cancel paths (grep `ssh_dropin.go`).
- [ ] Cancel/rollback vs second Prepare on same host (abandoned timer / bak dir) — regression tests on real systemd preferred over string tests only.
- [ ] Document which residual items from the safety remediation plan are **verified closed** vs still open; close the checklist in that file by reference to this plan.

---

## CF-7 — Integrity drift noise (P2) — **done 0.9.691**

**Evidence:** primary `incident_timeline` flooded with `integrity:drift` (VERSION vs `/usr/local/bin/netductor`) after manual binary replace.

**Fix direction:** doctor/stack should either update VERSION on forced install, or rate-limit / one-shot the alert after intentional offline replace; document recovery: `stack apply` or write VERSION to match binary.

---

## CF-8 — Test / CI debt from safety plan (P2)

Many `[ ] Focused tests have not been executed` lines in the secondary safety remediation plan. Either:

1. Run and mark evidence, or  
2. Move “must automate” cases into `go test` / CI and delete the stale checklist noise.

Do not treat unrun checkbox text as open product features.

---



## CF-9 — Secondary pullAndApply ignores sing-box restart (P1) — **done 0.9.687**

**Code:** `internal/secondary/agent.go` `pullAndApply()`

```go
if err := vpn.WriteSecondarySingBox(&b, priv, sid); err != nil {
    return err
}
_ = exec.Command("systemctl", "restart", "sing-box").Run()
_ = os.WriteFile(filepath.Join(paths.SecondaryDir(), "bundle.json"), raw, 0o600)
return nil
```

**Why real (not over-defense):** config file and `bundle.json` advance while the running process may still serve the previous config. Heartbeat/`config_ver` then claim the node is in sync. Other paths in the same package already fail closed on restart (`restorePrevious` / join). Same bar for agent pull.

**Fix:**

1. Capture `systemctl restart sing-box` error (and optionally `is-active` / `sing-box check`).
2. On failure: do **not** write success `bundle.json` / do not treat pull as applied; return error so the agent retries.
3. Prefer: write config → `sing-box check` → restart → confirm active → then persist bundle marker.
4. Regression test with fake runner / injectable exec.

**Acceptance:** failed restart leaves last-good process + no advanced “applied” marker; next tick retries.

---

## CF-10 — Primary ApplyConfig swallows restart and bump errors (P1) — **done 0.9.687**

**Code:** `internal/vpn/apply.go` `ApplyConfig()`

```go
_ = exec.Command("systemctl", "restart", "sing-box").Run()
fmt.Fprintf(os.Stderr, "sing-box config variant=%s ...")
_ = bumpSecondaryConfigVer()
return nil
```

**Why real:** disk config updated, CLI reports success, local sing-box may still run the old JSON; secondaries may or may not be notified. Operators and UI assume apply is atomic enough for production.

**Fix:**

1. Surface restart failure as `ApplyConfig` error (after config is on disk, state is “written but not running” — document recovery: restart / re-apply).
2. Surface `bumpSecondaryConfigVer` failure (no silent `_ =`).
3. Optionally: if restart fails, still return error but leave files for inspection (no silent success).
4. Tests for error propagation.

**Acceptance:** `netductor vpn apply` / API apply non-zero when local restart or config_ver bump fails; stderr names the stage.

---

## CF-11 — bumpSecondaryConfigVer registry selection / silent success (P2) — **done 0.9.687**

**Code:** `internal/vpn/apply.go` `bumpSecondaryConfigVer()`

- Canonical consumer path is `paths.SecondaryDevicesFile()` → `state/secondary/devices.json` only.
- Function tries `secondary` then `relay`, **returns nil after the first successful write**, and also **returns nil if neither file was updated** (all `continue`).

**Verdict:** Not primarily “wrong registry on dual-B” in the current tree (only `secondary/devices.json` is authoritative). Real issues:

1. Silent no-op when `devices.json` is missing/unreadable — primary thinks secondaries will pull, they will not.
2. Legacy `relay/` name is a migration leftover; if both files existed they could diverge (only first is bumped).

**Fix:**

1. Prefer `SecondaryDevicesFile()` only; optional one-shot migration copy from `relay/` if secondary missing.
2. Return error if no registry was successfully bumped.
3. If both files must stay for a transition window, bump **both** or document single writer.

**Acceptance:** bump failure is visible to `ApplyConfig`; doctor can WARN when `config_ver` cannot be read.

---


## CF-12 — **done 0.9.689** — APT Guard policy for Debian Trixie (P1)

**Evidence (primary 2026-11-06):** after host-baseline, doctor still:

`FAIL aptguard … policy_v=1 fail=8` with findings:

- `POLICY_MISMATCH` on `https://deb.debian.org/debian/` suites `trixie` / `trixie-updates` and security `trixie-security` (URI matches policy, suite/components/type/signed-by differ).
- `MISSING_REQUIRED` for `debian-main` and `debian-security` (required entries not recognized as present).
- Disabled cdrom line in `sources.list` → INFO only.

**Verdict:** live sources are normal Debian 13 (Trixie). Default **policy v1** is written for an older suite (or stricter shape) and marks a healthy primary as FAIL. Not a broken APT install; not a reason for emergency `aptguard apply`.

**Fix:**

1. Ship default / migration policy that accepts Trixie (`trixie`, `trixie-updates`, `trixie-security`) with current components (`main`, `contrib`, `non-free`, `non-free-firmware`) and HTTPS mirrors.
2. Map required `debian-main` / `debian-security` to those suites (or rename required keys and document).
3. Optional: codename detection (`/etc/os-release`) so bookworm hosts keep bookworm policy.
4. Doctor: after policy fix, `aptguard scan` → status OK or WARN only for true unknowns; no FAIL on stock Trixie sources.list.
5. **Do not** require operators to `aptguard apply` solely to silence doctor on a correct Trixie mirror set.

**Acceptance:** fresh `netductor aptguard scan` on stock Trixie primary with only official mirrors → no `POLICY_MISMATCH` / `MISSING_REQUIRED` for those lines; doctor aptguard not FAIL.

**Ops until fixed:** leave FAIL; do **not** apply remediation to force policy v1 shape onto Trixie.

---

## CF-13 — Stack apply / normal update path (P1) — **done 0.9.690** (idempotent + preflight UX; full GH asset pipeline still ops)

**Evidence:** Primary was updated by **manual binary replace** because `netductor stack apply` did not complete successfully in the B-deploy window. Operator stayed on mixed versions (e.g. node 0.9.682, agents 0.9.656 / 0.9.681). Integrity WARN cleared only via `host-baseline apply` (re-snapshot), not via a successful transactional apply.

**Scope:** restore a **safe, usable** apply path so production does not depend on scp/curl of binaries.

**Investigate and fix (code + tests):**

1. **Root cause audit** of recent stack apply failures (journal, bootstrap-transactions, download/SHA, installStagedPair, TG/node restart, post-check). Separate operator mistakes from hard bugs.
2. **Fail closed with actionable errors** (no silent partial install); surface stage name + next recovery command.
3. **Transaction / recovery** consistent with migrate + stack design: interrupted apply must be detectable at next start; no mixed Node/TG without recovery path.
4. **VERSION + integrity:** successful apply must write `/etc/netductor/VERSION` and refresh integrity manifest (overlap with CF-7).
5. **Secondary agents:** documented path to upgrade B (and A while retained) without manual file drops; align with update-system plan if already in tree.
6. Regression: dry-run or unit tests around stages that failed in field notes; no requirement to weaken SHA verify.

**Acceptance:**

- On a healthy primary matching current release assets, `stack apply <tag>` completes or fails with a clear stage error and safe rollback/recovery instructions.
- After success, `netductor version` / VERSION file / doctor integrity agree; no routine need for manual binary replace.
- Documented operator runbook one-pager linked from this CF when closed.

**Ops until fixed:** prefer host-baseline for firewall/snapshot noise only; do not force stack apply on production solely to quiet doctor.

---

## CF-14 — Naming: control / egress + region (P2)

**Decision (2026-11-06):** do **not** use in/out as the primary model. Split by **role** and **region**.

| Axis | Values | Notes |
|------|--------|--------|
| **Role** | `control` (ex-primary), `egress` (ex-secondary), `build`, `edge` | Stable function of the node |
| **Region** | user-facing geo labels e.g. `eu`, `ru` (also internal DC tags if useful) | Shown in **subscription / client profiles**, not only operator UI |
| **Instance** | peer id + slot | Blue-green within a region |

**Aliases:** keep `primary`≡`control`, `secondary`≡`egress` in CLI/API for a long transition; do not break role files in one shot.

**Sub:** profiles and display names may include region (`nd-egress-eu`, human label “EU”) so users pick exit by geography; active peer still drives which IP/PBK/SID.

**Implementation slices (later):**

1. Document ADR + glossary EN/RU.
2. Optional `region` field on peer / device registry; set at deploy.
3. Subscription body includes region in profile name/tag when set.
4. Web/TG/TUI labels; doctor role string.
5. Migrate `/etc/netductor/role` values with dual-read.

**Not in this CF:** renaming every internal package or WG iface.

---
## Skip-version policy

Operators often jump many tags (e.g. 0.9.650 → 0.9.690). Baseline requirements (partly in 0.9.691):

1. Startup migrations remain **idempotent** and re-check `Required` even after journal `completed`.
2. Policy/defaults that change (aptguard v2, role files, state dirs) must migrate without intermediate releases.
3. Integrity must not spam on VERSION/binary drift after intentional multi-hop updates (auto-heal metadata only).
4. `stack apply` already-at-tag refreshes metadata without re-download.

**Not every release needs an intermediate binary step.** Stepping is required only when code cannot safely jump — see **CF-15**.

## CF-15 — Upgrade floors from code (P1) — **in progress (0.9.692: registry + preflight gate)**

### Goal

When updating toward latest, the engine may need a **short chain** of intermediate releases — **only** when something actually breaks across versions. A manual “set `min_from` on the release” step will be forgotten (humans and coding agents). **Source of truth = code + CI**, not a release-notes checkbox.

### When a floor is required (breaking)

Register a floor in the **same commit** that introduces the break:

1. **On-disk format** without a forward-compatible reader in older binaries.
2. **Apply / SSH / harden / stack transaction order** such that applying the new binary from too-old local state is unsafe.
3. **Peer / slot material** without a compatible recover path for older agents.

Ordinary features and **idempotent** migrations that a new binary can run after any jump → **no floor**.

### Code contract

- API such as `migrate.DeclareBinaryFloor(id, minBinary, reason)` (final name TBD) next to the breaking change, migration, or format owner.
- Optional format table: path → `MinBinary` + reason.
- Contributor / agent rule: **breaking change without a floor in the same commit is incomplete**.

### CI (so floors are not forgotten)

1. Scan the repo for declared floors (registry / AST / grep).
2. On release/tag, **generate** `upgrade.json` (or release sidecar): `version`, `min_from` (aggregate of floors for that jump), optional `reason[]`.
3. Fail the release job if floors and generated metadata disagree.
4. Prefer generated artifacts over hand-edited release text.

### Update engine (phased)

| Phase | Behavior |
|-------|----------|
| **A — gate** | `stack apply` / `update`: if `local < min_from` → **hard refuse** with a clear message (“install intermediate vX first”); no silent skip |
| **B — path** | `ResolveUpgradePath(local, target)` using published `upgrade.json` history → ordered tag list |
| **C — apply chain** | Download + verify + apply each step; migrate/health between steps; journal/resume on partial failure |

Phase A can ship before a full auto-chain. Phase C must be **role-aware** (control node vs egress agent vs edge).

### Tests

- Old state + new binary where a floor exists → apply without intermediate **fails closed**.
- Same with the required intermediate in path → succeeds (unit/integration as feasible).
- CI: diff touches known breaking packages without a new/updated floor → fail (default; no silent omit).

### Explicit non-goals

- One intermediate step **per** tag by default.
- Relying on memory or PR checkbox instead of CI.
- Auto-guessing floors from arbitrary diffs without a declaration (unsafe).

### Acceptance

1. Breaking on-disk / apply-order / peer-material changes land with an in-tree floor.
2. Release `min_from` is **generated**, not hand-waved.
3. Jumping many **non-breaking** tags remains **one** apply + startup migrate.
4. `local < min_from` → clear refuse (A) or safe automatic chain (C).

### Implementation order

1. ~~Floor registry + Declare API + unit tests~~ — **0.9.692** (`internal/upgrade`).
2. Generator script `scripts/gen-upgrade-json.go` (wire into release CI next).
3. ~~Phase A refuse when target `upgrade.json` has min_from~~ — **0.9.692** preflight critical check.
4. Phase B/C path planner + multi-step apply with resume — **open**.
5. CI fail if floors ≠ generated metadata — **open**.


## CF-17 — svc-paths Status/doctor slot-aware (P1) — **done 0.9.695**

After active peer is slot N, `MaterialIfaces()` / legacy material may still report slot-0 `nd-svc-sp` (down after A remove). Doctor WARN SP/PS peer was false-positive while slot1 ifaces were UP.

**Fix:** `Status()` uses `active_id` peer slot ifaces + matching WSS unit names (`nd-wss-*-nd-svc-sp1`).


## CF-18 — UX: conditional migrate TG buttons + bash completion + unix-style CLI (P2)

### 18.1 TG migrate buttons only when needed
Startup migrations already run on API start (`migrate.RunAll`). Buttons status/run/clear on Versions are repair tools.
**Change:** show migrate keyboard rows only when journal has `failed`/`running` or pending required migrations; otherwise hide. Optional one-line status if all ok.

### 18.2 Bash/zsh completion
- `netductor completion bash|zsh` (and `netductor-op`)
- install under `/usr/share/bash-completion/completions/` on stack apply / host-baseline (node)
- brew path for op on Mac
- v1: top-level + depth-2 subcommands

### 18.3 Unix-style flags (compat layer)
Canonical: `noun verb [options]` with `-h/--help`, `-v/--version`, `--json`, `--dry-run` where mutations.
Keep git-style aliases 1–2 releases (`secondary cmd ID upgrade:vX` stays).
Group help by plane / host / edge. Completion follows canon.

### 18.4 HTMLBubbles / in-chat widgets — **scaffold in tree**
Gated (`NETDUCTOR_TG_WIDGETS=1`): status bubble + Refresh/Versions/Channel/Menu; migrate attention line.
Bot API 10.4 beta methods; soft-fail → classic status. No upgrade floor (UX only, AGENT_HANDOFF CF-15).

---
## CF-16 — PeerRemove UFW 8789 comment mismatch (P1) — **done 0.9.693**

**Symptom:** `netductor svc-paths peer remove legacy-…` → `refuse to delete unowned UFW rule 8789/tcp from 10.87.10.2`.

**Cause:** `applyPrimaryWSSFirewall` creates `8789` with comment `netductor-api-peer`; `removePeerUFW` expected `nd-svc-sp`.

**Fix:** accept `netductor-api-peer` (canonical) and legacy `nd-svc-sp`; leave 9443 / wrong comments alone.

**Tests:** API-peer delete, legacy comment, wrong comment refuse, 9443 untouched.

## Explicitly out of scope (not code in this backlog)

| Item | Track |
|------|--------|
| R5 observation window / retire A | operator — [BLUE-GREEN-SECONDARY.md](BLUE-GREEN-SECONDARY.md) |
| Edge / NVR / MikroTik lab e2e | [OPEN_ITEMS.md](OPEN_ITEMS.md) |
| APT Guard **live apply** canary (after CF-12 policy) | ops evidence |
| Update-system live canary on VPS | ops — blocked on **CF-13** |
| Telegram HTMLBubbles live canary | ops — Bot API 10.4 send rights |
| OIDC / read-only guest | future |

---

## B deploy friction → code map (chat 2026-11-04…06)

| Symptom during B | Resolution | Backlog |
|------------------|------------|---------|
| `ssh failed (error)` empty | 0.9.664 surface redacted errors | done |
| Journal `provision_started` ambiguous refuse | recovery checkpoint / clear journal ops | done + ops |
| mTLS install no transaction marker | 0.9.665 base64 pipe | done |
| `remote join failed` truncated | 0.9.666–667 tails, HTTP/1.1 | done |
| host-baseline / UFW 80, routed, deny 22 lockout | 0.9.668 + ops cleanup | **CF-5**, **CF-4** residual |
| Harden finalize ufw skip / pipefail / mawk | 0.9.674–681 | mostly done; **CF-6** audit |
| Metrics :443 ok while B down | probe `ss` fallback | **CF-1** |
| Agent config pull / primary apply silent restart | ignored `systemctl restart` / bump | **CF-9**, **CF-10**, **CF-11** |
| Client freezes, sing-box restarts | SP probe → watchdog | **CF-2**, **CF-3** |
| Doctor aptguard FAIL on stock Trixie | policy v1 vs trixie suites | **CF-12** |
| stack apply blocked → manual binary | update/stack transaction | **CF-13** |
| Multi-release jump needs intermediate binary | upgrade floors / min_from | **CF-15** |
| Sub hostname pointed at A for prepared B | 0.9.682–683 | done |

---

## Implementation notes

- Thin UI: any new flags/status must hit API first; Web/TG/TUI only present.
- No weakening SSH host-key trust or auto-accept unknown keys.
- Prefer small commits per CF-id with regression tests.

When a CF item ships: mark `[x]` here, link commit/tag, and trim duplicate wording from evidence docs to a one-line pointer.

---

## Diagnostic script (deploy correctness)

`scripts/diag-secondary-b.sh` — after promote / when investigating B.

```bash
# Primary (download from main, run in-place)
ROLE=primary PEER_IP=195.140.147.20 PEER_ID=peer-195-140-147-20 \
  AGENT_ID=secondary-bb9952c66617ca91 \
  curl -fsSL https://raw.githubusercontent.com/PavelNeyman/netductor/main/scripts/diag-secondary-b.sh | bash

# Private repo
ROLE=primary PEER_IP=195.140.147.20 PEER_ID=peer-195-140-147-20 \
  AGENT_ID=secondary-bb9952c66617ca91 \
  curl -fsSL -H "Authorization: Bearer ${GH_TOKEN}" \
    https://raw.githubusercontent.com/PavelNeyman/netductor/main/scripts/diag-secondary-b.sh | bash

# On Secondary B
ROLE=secondary curl -fsSL https://raw.githubusercontent.com/PavelNeyman/netductor/main/scripts/diag-secondary-b.sh | bash
# or with token: ROLE=secondary curl -fsSL -H "Authorization: Bearer ${GH_TOKEN}" ... | bash
```

Checks: active/preferred peer, slot1 units + ping, devices entry, TCP :443 to B; on B — sing-box/agent/WG, mtls, slot0 VIP leakage, recent watchdog restarts.
