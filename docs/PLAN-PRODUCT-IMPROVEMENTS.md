# PLAN: Product improvements (post-0.9.339)

**Status:** proposed → **closed** (recorded 2026-10-12).  
Hardware e2e (OpenWrt / Tapo / MikroTik / NVR lab) intentionally excluded — remains in OPEN_ITEMS as lab-only.

Related: [REVIEW-SUITE.md](REVIEW-SUITE.md) · [PLAN-TG-MENU.md](PLAN-TG-MENU.md) · [OPEN_ITEMS.md](OPEN_ITEMS.md).

---

## 1. Observability & operations

- [ ] Prometheus/OpenMetrics endpoint (`/metrics` or dedicated port) with vpn, channel, backup age, firewall, agent heartbeats.
- [ ] Unified incident timeline card (Web + TG) with 1h journal download in one place.
- [ ] Single fleet overview landing in Web (primary/secondary/edge + channel + backup age + last update).

## 2. Git / CI / internal releases

- [ ] Web project card with last build status + Cancel button (parity with TG `m:git:proj:vps-cancel:`).
- [ ] Clearer UX for per-repo `build_on_fetch` (default off stays; make enable obvious).
- [ ] Visible CI artifact / container cleanup policy in UI (timers already exist).

## 3. Security & access

- [ ] Active operator sessions list with explicit TTL (Web + TG).
- [ ] Audit log viewer (stack apply / SSH enforce / reboot) in TG/Web.
- [ ] Optional OIDC/SSO hook for Web (future multi-operator; design only for now).

## 4. UX & surface parity

- [ ] Update README to current tip (v0.9.339+) and accurate commands.
- [ ] Finish remaining Web i18n hardcodes (migrate owner/mode labels and similar).
- [ ] More compact TG status cards; fewer long HTML dumps.

## 5. Backup / DR

- [ ] Backup verify status in fleet digest and Web (CLI/timer already exist).
- [ ] Recover-drill reminder (“last successful recover-test N days ago”).
- [ ] Restore preview before real recover.

## 6. Product clarity

- [ ] Doctor/UI explicit DNS path note: client → VPN → blocky vs direct (YouTube/ads case).
- [ ] Mark NVR/Tapo as lab-only in main roadmap if not prioritized (already open in OPEN_ITEMS).

## 7. DX / docs

- [ ] Document “fast” test profile and how to run suite on ~2 GiB (skip heavy packages).
- [ ] `doctor` or pre-check: VERSION matches latest GitHub tag.
- [ ] Policy: new PLAN docs ship EN+RU or are marked EN-only draft.

## 8. Telegram menu structure & logic (new)

Walk every TG hub and nested screen. Make navigation structured and logical.

Rules:
- Different buttons must not lead to the same destination without a clear reason.
- No duplicate actions under two hubs (Status clone under Tools, etc.).
- Callback prefixes unambiguous (`m:sshkeys:` vs hosts, `m:git:proj:*` vs others).
- Hub set stays Status / Users / Fleet / Tools / Operator.
- Tools = day-2 only (no Status/Users/Fleet clones).
- Card body short; actions on buttons.
- After changes, update [PLAN-TG-MENU.md](PLAN-TG-MENU.md) callback map if prefixes move.

Checklist:
- [ ] Inventory all `m:` callbacks and their handlers.
- [ ] Map button → screen; flag duplicates and dead ends.
- [ ] Remove or merge redundant paths.
- [ ] Verify back/menu keys always present and correct.
- [ ] EN/RU labels via `T()` where user-visible.

---

**Closed:** proposals recorded. Implementation tracked in OPEN_ITEMS / follow-up PRs. Hardware e2e stays lab-only.
