# Netductor review suite (mandatory)

**Purpose:** one canonical checklist for “full complex” reviews after feature batches, security passes, or before production apply.  
**Do not** invent a shorter ad-hoc list in chat — follow this document (or [RU](ru/REVIEW-SUITE.md)).  
**Language rule:** every durable doc change ships **EN + RU** with the **same meaning** (not a one-line stub). Short notes belong only in `POST-REVIEW-*.md` release crumbs, not as replacements for this suite.

**Related:** [AGENT_HANDOFF.md](AGENT_HANDOFF.md) · [CLEANUP-PASSES-2026-10.md](CLEANUP-PASSES-2026-10.md) (hygiene journal post-0.9.403) · [OPEN_ITEMS.md](OPEN_ITEMS.md) · [OPCATALOG.md](OPCATALOG.md) · [I18N.md](I18N.md) · [PLAN-TG-MENU.md](PLAN-TG-MENU.md) · [AGENTS.md](../AGENTS.md)

---

## When to run

| Trigger | Suite |
|--|--|
| After a multi-PR feature wave (security, UI, DNS, SSH, …) | **Full** (all sections) |
| After a single small fix | **Mini**: §1 tests + §3 tip/stop line + affected UI only |
| Before production `stack apply` of a tip you did not just ship | **Full** at least §1–§5 + §8 |
| User says «весь комплекс» / «полный ревью» | **Full**, no skipping §7 tests debt |

Record the stop line and date in the PR/commit message or a new `docs/POST-REVIEW-<version>.md` only as a pointer; **results live in the reply to the operator**, not as a substitute for updating handoff.

---

## 1. Automated tests (blocking)

### 1.1 Run

```bash
export GOPROXY=https://proxy.golang.org,direct
go test ./internal/... -count=1
go test ./cmd/... -count=1
```

- **Pass required** before release tag.
- On proxy 502, retry with public `GOPROXY`; do not skip the suite.

### 1.2 Coverage obligation

| Rule | Detail |
|--|--|
| New production logic | Same change **must** add or extend unit tests where feasible without root/VPS |
| Packages with **no** `*_test.go` | Listed in §1.3; closing a feature in that package **requires** at least one focused test or an explicit OPEN_ITEMS deferral with reason |
| Security-sensitive paths | challenge codes, allowlist parse, download SHA, session gates, path containment — **tests preferred over manual-only** |

### 1.3 Packages currently without tests (debt tracker)

Update this table when adding tests (checkboxes → done in OPEN_ITEMS or here).

| Package | Priority | Notes |
|--|--|--|
| `internal/integrity` | **P0** | baseline + SSH parse — **tests added 0.9.329** |
| `internal/notify` reboot challenge | **P0** | one-shot codes — **tests added 0.9.329** |
| `internal/firewall` | P1 | truncate/firstLines/detectRole — **tests 0.9.330** |
| `internal/channels` | P1 | FormatPathE2E/FormatText — **tests 0.9.330** |
| `internal/logs` | P2 | schedule format — **tests 0.9.331** |
| `internal/servicenet` | P2 | Enabled/Collect smoke — **tests 0.9.331** |
| `internal/svcpaths` | P2 | peerOf / DefaultPolicy — **tests 0.9.331** |
| `internal/devices` | P2 | List empty — **tests 0.9.331** |
| `internal/incident` | P2 | Collect smoke — **tests 0.9.331** |
| `internal/hardening` | P1 | DropInConf/RemoteHardenScript/SSHPort — **tests 0.9.330** |
| `internal/dnsblock` | P2 | defaults/catalog — **tests 0.9.331** |
| `internal/cli18n` | P3 | T/Lang — **tests 0.9.331** |
| `internal/version` | P3 | non-empty Release |
| `internal/fleet` | P2 | empty/defaultPolicy — **tests 0.9.331** |
| `internal/onboard` | P2 | defaultItems — **tests 0.9.331** |
| `internal/cleanup` | P2 | humanBytes/FormatText — **tests 0.9.331** |
| `internal/upstream` | P2 | defaultPins — **tests 0.9.331** |
| `internal/addons` | P3 | str helper — **tests 0.9.331** |
| `internal/tlsle` | P3 | findLiveCert empty — **tests 0.9.331** |
| `cmd/netductor-op` | P2 | catalog smoke — **tests 0.9.330** |

**Obligation:** each full suite run must either (a) add tests for at least one P0/P1 debt row still open, or (b) state that no P0/P1 debt remains for the changed packages.


### 1.4 TG UX invariants (blocking — `cmd/netductor-tg`)

These catch **chat UX regressions** that pure API/i18n review misses. Run with §1.1 (`go test ./cmd/...`).

| Invariant | Why | Test |
|--|--|--|
| `/menu` and `/start` call **`forceHub`** (not `reply` on stored id) | After “clear history” / delete-for-me, edit targets an **invisible** message — bot looks dead | `TestMenuCommandsUseForceHub` |
| `wait_*` text paths use **`completeInput`** (not `sendHTML`) | `sendHTML` stacks prompt + user text + new menu | `TestWaitHandlersPreferCompleteInput` |
| Users list exposes **`m:vpn_devices`** + `vpn_devices_btn` | kind=device profiles must not be confused with online-IP locator (`m:devices`) | `TestUsersListWiresVpnDevicesCallback` |
| Critical menu callbacks are handled | Dead buttons after refactors | `TestNoObviousOrphanMenuCallbacks` (critical subset) |

**Manual smoke (still useful):** add VPN user from TG once per major TG release; `/menu` after clearing chat.


---

## 2. Tip, releases, formula

| Check | Pass criteria |
|--|--|
| `VERSION` == `internal/version.Release` | exact match |
| GitHub release `v$VERSION` | node, tg, agent (multi-arch), op darwin/linux, `SHA256SUMS` |
| Public download | HTTP 200 for `netductor-linux-amd64` |
| `Formula/netductor-op.rb` | version + real SHA256 (never `:no_check`); **no** `Formula/netductor.rb` |
| Tag body | mentions user-visible changes |

---

## 3. Handoff and plans

| Check | Pass criteria |
|--|--|
| `docs/AGENT_HANDOFF.md` stop line | **equals tip** |
| `docs/ru/AGENT_HANDOFF.md` stop line | **equals tip**, same meaning as EN |
| `docs/OPEN_ITEMS.md` (+ RU if present) | stop line current; closed work checked |
| No conflicting historical “stop” at top | hist may exist deeper; top of file is tip only |
| Feature plans | relevant PLAN-* checkboxes updated in the **same** change when a phase completes ([AGENTS.md](../AGENTS.md) discipline) |

---

## 4. Documentation quality (EN ↔ RU)

| Check | Pass criteria |
|--|--|
| New durable doc | EN under `docs/` **and** RU under `docs/ru/` |
| Meaning parity | Same sections, obligations, commands; not a 5-line “see EN” stub for mandatory docs |
| This suite | Both `REVIEW-SUITE.md` and `ru/REVIEW-SUITE.md` maintained together |
| Index | Linked from `docs/README.md` / `docs/ru/README.md` |

---

## 5. UI parity (API is source of truth)

Surfaces: **API** · **TG** · **Web (op control)** · **op TUI / opcatalog** · **CLI** where applicable.

| Capability class | Expectation |
|--|--|
| Day-2 ops in opcatalog | Row in `internal/opcatalog` **before** UI-only code |
| Destructive ops (reboot, SSH enforce, stack apply) | Challenge or explicit confirm; session required on API |
| Status/metrics | No raw dump where tables/cards were agreed |
| New security features | TG + API minimum; Web/opcatalog in the same wave or tracked OPEN_ITEMS with owner |

### 5.1 Current security/DNS track (reference)

| Feature | API | TG | Web | opcatalog |
|--|--|--|--|--|
| SSH allowlist | ✅ | ✅ `m:sshkeys:` | ✅ Advanced | ✅ |
| Reboot challenge | ✅ | ✅ | ✅ Advanced | ✅ |
| Integrity / doctor | collect/doctor | alerts | — | — |
| Canary | ✅ | ✅ Users | catalog | ✅ |
| Channel tables | status API | ✅ | metrics | — |

---

## 6. Telegram cards and menu

| Check | Pass criteria |
|--|--|
| Hubs | Status / Users / Fleet / Tools / Operator only as primary IA ([PLAN-TG-MENU](PLAN-TG-MENU.md)) |
| No Status clone under Tools | Tools ≠ second Status |
| Callbacks | No ambiguous prefixes (e.g. allowlist `m:sshkeys:` vs hosts `m:ssh:rm:`) |
| Formatting | Tables/lists for status; no long unformatted JSON in user-facing status |
| Duplicates | Same action not under two hubs without reason |

---


| TG UX invariants (§1.4) | **must pass** on the same tip |
| Singleton hub | Callbacks **edit** one hub; `/menu` **forceHub**; text waits **completeInput** |
| Device profiles vs online IPs | `m:vpn_devices` (kind) ≠ `m:devices` (log IPs) — both labeled in Users |

## 7. Bilingual UI (EN/RU)

| Check | Pass criteria |
|--|--|
| TG | User-visible strings via `T()` / i18n maps; no hardcoded EN-only button labels on new flows |
| Web | Keys in both `I18N.en` and `I18N.ru` |
| opcatalog | `label_en` + `label_ru` |
| Alerts recovery | EN and RU messages for negative keys |
| Docs | §4 |

---

## 8. Security / product smoke (manual or scripted on fleet)

Not a substitute for §1. Run when touching VPN, agent, firewall, or SSH.

| Check | Notes |
|--|--|
| `netductor doctor` primary | fail=0 preferred; WARN explained |
| `netductor host-audit` | CLEAN or known residual |
| `netductor channel status` | uplink/path coherent with reality |
| Secondary agent version | matches tip after upgrade |
| SSH allowlist detect | inventory without enforce unless intentional |
| Client egress | ifconfig / service VIP only if that wave touched policy |

---

## 9. Deep code review (recent diff + touched packages)

Mandatory on **Full** suite. Scope: commits since last stop line (or explicit range).

| Check | Pass criteria |
|--|--|
| Correctness | Logic matches product rules; edge cases (empty lists, stale role, offline secondary) handled |
| API contracts | JSON fields stable or versioned; errors use consistent `{ok,error}` / HTTP codes |
| Concurrency | No data races on shared maps/files without lock; agent registry and session stores safe |
| Idempotency | Apply/firewall/allowlist/baseline safe to re-run |
| Observability | Failures surface to doctor/alerts/logs; no silent ignore on security-critical paths |
| Dependencies | No new heavy deps without reason; `go.mod` tidy |
| Tests | New logic covered per §1.2 |

**How:** read changed files fully in touched packages; do not mark "reviewed" from commit message alone.

---

## 10. Security review

| Check | Pass criteria |
|--|--|
| Authn/Authz | Operator session on mutating API; TG operator-only; no unauthenticated destructive routes |
| Agent plane | Cmd allowlist only; payloads size-bounded where practical; mTLS on 8789 |
| Trust boundaries | Secondary naming on :9443 documented; client cannot escalate to host root via API |
| Secrets | Files 0600 under etc/state; tokens not in docs/git; reboot challenge one-shot + TTL |
| Supply chain | Release downloads verify SHA256; tag characters restricted; `NETDUCTOR_UPDATE_SKIP_VERIFY` is footgun |
| Injection | No shell from user-controlled strings without allowlist; paths cleaned / contained |
| SSH | Allowlist detect vs enforce clear; enforce warns about hoster keys |
| Network exposure | API :8787 localhost; agent plain :8788 not on WAN; doctor/host-audit clean of hoster agents |

**Severity labels for findings:** Critical (exploit/data loss) · High · Medium · Low · Info.

---

## 11. Refactoring necessity

| Check | Pass criteria |
|--|--|
| Duplication | Same logic in TG/Web/CLI without shared helper → note R-item or fix if small |
| File size / god functions | New oversized functions or packages that mix unrelated concerns → backlog with reason |
| Dead code | Unreachable handlers, unused flags after feature removal → delete or ticket |
| Naming | Confusing prefixes (`m:ssh` vs allowlist) already fixed pattern — do not reintroduce |
| Layering | UI must not embed policy that belongs in `internal/` (API source of truth) |
| Prior R-items | Check `docs/REVIEW-PROGRESS.md` / OPEN_ITEMS; do not reopen closed R* without new evidence |

**Rule:** Full suite **must** either (a) list concrete refactor follow-ups with severity, or (b) state **no refactor required** for the diff. "Looks fine" without this sentence is incomplete.

---

## 12. Output template (operator reply)

1. **Tip / release** — version, assets OK?
2. **Tests** — `go test` result; new tests added? debt table?
3. **Handoff / OPEN_ITEMS / docs EN+RU**
4. **UI parity** (deltas)
5. **TG cards / i18n**
6. **Code review** (§9) — issues or clean
7. **Security review** (§10) — issues or clean
8. **Refactoring** (§11) — follow-ups or "no refactor required"
9. **Findings** — severity + fix-now vs defer
10. **Apply commands** if tip moved

---

## Changelog of this suite

- **0.9.344+:** §1.4 TG UX invariants (`forceHub`, `completeInput`, `m:vpn_devices`) + tests in `cmd/netductor-tg`.

| Version | Change |
|--|--|
| 0.9.427 (docs 2026-10-20) | First run with static dead-code analysis (`deadcode` on every shipped GOOS) and `gofmt -l` / `go vet` as part of §9/§11; see [REVIEW-REPORT-2026-10-19.md](REVIEW-REPORT-2026-10-19.md). Recommended additions: `gofmt -l` gate; EN-only docs audit (§4) |
| 0.9.331 | Expanded §9–§11: deep code review, security review, refactoring necessity; remaining package unit tests |
| 0.9.330 | P1/P2 tests: firewall, channels, hardening, op catalog |
| 0.9.329 | Initial suite doc EN+RU; integrity + reboot challenge tests; mandatory full-complex reference |



## Web UI + version gates (mandatory)

Before any release that includes `netductor-op` or `internal/operator/web/`:

```bash
./scripts/check-versions.sh   # VERSION == Release; no hardcoded cmd version
./scripts/check-web-ui.sh     # .hidden, ND_TOKEN, i18n heuristics
```

Browser smoke: [SMOKE-WEB-OP.md](SMOKE-WEB-OP.md) (RU: [ru/SMOKE-WEB-OP.md](ru/SMOKE-WEB-OP.md)).

**Rule:** Diff touching web UI without static scripts + smoke note is **incomplete suite**.

**Known failure class:** `var version = "x.y.z"` in `cmd/*/main.go` — always `ndver.Release`.

## Web UI static guards

```bash
./scripts/check-web-ui.sh
```

Catches `.hidden !important` vs `style.display`, `window.ND_TOKEN`, basic i18n keys.
