# Netductor review suite (mandatory)

**Purpose:** one canonical checklist for “full complex” reviews after feature batches, security passes, or before production apply.  
**Do not** invent a shorter ad-hoc list in chat — follow this document (or [RU](ru/REVIEW-SUITE.md)).  
**Language rule:** every durable doc change ships **EN + RU** with the **same meaning** (not a one-line stub). Short notes belong only in `POST-REVIEW-*.md` release crumbs, not as replacements for this suite.

**Related:** [AGENT_HANDOFF.md](AGENT_HANDOFF.md) · [OPEN_ITEMS.md](OPEN_ITEMS.md) · [OPCATALOG.md](OPCATALOG.md) · [I18N.md](I18N.md) · [PLAN-TG-MENU.md](PLAN-TG-MENU.md) · [AGENTS.md](../AGENTS.md)

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
| `internal/logs` | P2 | schedule parse |
| `internal/servicenet` | P2 | VIP helpers |
| `internal/svcpaths` | P2 | health file parse |
| `internal/devices` | P2 | device registry |
| `internal/incident` | P2 | pack assembly |
| `internal/hardening` | P1 | DropInConf/RemoteHardenScript/SSHPort — **tests 0.9.330** |
| `internal/dnsblock` | P2 | list URL allow |
| `internal/cli18n` | P3 | key presence EN/RU |
| `internal/version` | P3 | non-empty Release |
| `cmd/netductor-op` | P2 | catalog smoke — **tests 0.9.330** |

**Obligation:** each full suite run must either (a) add tests for at least one P0/P1 debt row still open, or (b) state that no P0/P1 debt remains for the changed packages.

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

## 9. Code review focus (recent diff)

| Focus | Pass criteria |
|--|--|
| Trust boundaries | session, mTLS, agent cmd allowlist |
| Download / update | SHA required unless explicit env footgun documented |
| Path containment | clips, git artifacts, NVR paths stay under roots |
| Secrets | 0600 paths; no tokens in handoff |
| Error handling | no silent swallow on security checks |

---

## 10. Output template (operator reply)

Use this structure when reporting a full suite:

1. **Tip / release** — version, assets OK?  
2. **Tests** — `go test` result; new tests added?  
3. **Handoff / OPEN_ITEMS / docs EN+RU**  
4. **UI parity table** (only deltas)  
5. **TG cards / i18n**  
6. **Findings** — severity + fix-now vs defer  
7. **Apply commands** if tip moved  

---

## Changelog of this suite

| Version | Change |
|--|--|
| 0.9.329 | Initial suite doc EN+RU; integrity + reboot challenge tests; mandatory full-complex reference |

