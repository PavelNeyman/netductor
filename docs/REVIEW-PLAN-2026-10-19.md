**EN** · [RU](ru/REVIEW-PLAN-2026-10-19.md)

# Full-complex review plan — run of 2026-10-19/20 (tip v0.9.427)

Canonical checklist: [REVIEW-SUITE.md](REVIEW-SUITE.md). Results and findings: [REVIEW-REPORT-2026-10-19.md](REVIEW-REPORT-2026-10-19.md).
Close boxes as chunks finish; a chunk that needs the owner or a live fleet stays open with the reason.

**Owner constraints:** automated tests not run; docs-only changes; no live fleet access.

## Chunks

- [x] **C0 Context and access** — read AGENTS.md, handoff, OPEN_ITEMS, REVIEW-SUITE; read/write access to `main` verified (API permissions + `git push --dry-run`, nothing pushed)
- [x] **C1 Static build health** (replaces §1 for this run) — `go build ./...`, `go vet ./...`, `gofmt -l`, cross-build of the agent for arm/mips/mipsle/386 → findings Q-2, Q-3, Q-4
- [ ] **C1b Automated tests (§1.1)** — **deferred by owner**; run `go test ./internal/... ./cmd/... -count=1` before the next release tag
- [x] **C2 Tip / release / formula (§2)** — VERSION, pins, Formula SHA OK; v0.9.427 not yet released
- [x] **C3 Handoff and plans (§3)** — AGENT_HANDOFF EN+RU restructured, OPEN_ITEMS EN+RU, AGENTS.md baseline + Project State aligned to 0.9.427
- [x] **C4 Docs EN↔RU (§4)** — parity audit done (finding D-1); new docs added as EN+RU pairs and indexed
- [ ] **C4b Translate the 19 EN-only docs** — see D-1; needs owner go-ahead (large)
- [x] **C5 UI parity (§5)** — `check-ui-parity.sh` OK
- [x] **C6 TG cards / menu (§6)** — static callback audit: 299 emitted, 0 orphans (tests not run)
- [x] **C7 Bilingual UI (§7)** — `check-web-ui.sh` → Q-1 (4 missing keys)
- [ ] **C8 Security / product smoke on the fleet (§8)** — needs live access (owner): `netductor doctor`, `host-audit`, `channel status`, secondary agent version
- [x] **C9 Deep code review (§9)** — read: edge store/auth, vpn sub tokens + subscription handler, guest store/portal, API session gate, serve bind, recovery TLS client, shell-out sites
- [x] **C10 Security review (§10)** — findings S-1…S-10
- [x] **C11 Refactoring necessity (§11)** — R-1…R-8 (not "no refactor required")
- [x] **C12 Dead code** — `deadcode` (RTA, two OS targets) + identifier scan + manual verification → DC-A/B/C
- [x] **C13 Report** — [REVIEW-REPORT-2026-10-19.md](REVIEW-REPORT-2026-10-19.md)

## Follow-up work (needs owner decision — all code changes)

- [ ] **F1** S-1: call `RevokeSubToken` in `RevokeNative`, reject disabled users in `SubscriptionBody`, add a test
- [ ] **F2** Q-1: add `edge_st_g_bak`, `rd_mode`, `rd_pos`, `result_hint` to `I18N.en` and `I18N.ru`
- [ ] **F3** Q-2: fix the self-assignment in `internal/git/projects.go:108`
- [ ] **F4** Q-3: one mechanical `gofmt -w` commit; add `gofmt -l` to the suite
- [ ] **F5** Remove DC-A dead code in one commit per package group; decide per item for DC-B
- [ ] **F6** S-2: recovery TLS fail-closed on bad pin + explicit first-contact flag
- [ ] **F7** S-5 + R-4: hardened `httpx.NewServer` with timeouts
- [ ] **F8** S-6, S-7, R-3: remove shell composition (`ufw delete`, `wg pubkey`), replace `curl | sh`
- [ ] **F9** S-8: propagate `saveDevices` errors in `internal/edge`
- [ ] **F10** R-1…R-8 as separate small refactors, each with its own tests

## Reproducing the static run

Go is not installed on a clean sandbox; the toolchain used here was `apt install golang-go` (1.22.2). Modules hosted on vanity domains were replaced with GitHub mirrors in a **scratch copy** only (`go mod edit -replace golang.org/x/<m>=github.com/golang/<m>@<ver>`, likewise `gopkg.in/yaml.v3` → `github.com/go-yaml/yaml/v3`, `gopkg.in/check.v1` → `github.com/go-check/check`). `deadcode` was built from `github.com/golang/tools` at `v0.28.0`. Run `deadcode ./...` for every `GOOS` you ship.
