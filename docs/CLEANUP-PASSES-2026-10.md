# Cleanup & security review passes (2026-10)

**Audience:** any AI or human continuing netductor work.  
**Stop line at doc write:** tip after **v0.9.403** + cleanup commits (see `git log v0.9.403..HEAD`). Product release still **v0.9.403** until operator asks 0.9.404+.  
**Do not** treat this as a substitute for [REVIEW-SUITE.md](REVIEW-SUITE.md) — it is a **changelog of hygiene passes** and a **remaining backlog**.

Related: [AGENT_HANDOFF.md](AGENT_HANDOFF.md) · [OPEN_ITEMS.md](OPEN_ITEMS.md) · [REVIEW-SUITE.md](REVIEW-SUITE.md) · [ru/CLEANUP-PASSES-2026-10.md](ru/CLEANUP-PASSES-2026-10.md)

---

## 1. Why these passes existed

After feature work (USB edge storage, build-host, channel monitoring, SSH allowlist, policy, SR profiles), the operator asked for:

1. Full-project review (not only last diff).
2. Remove **unused TG menus** and **dead code** (only if call-graph proved dead).
3. Security hardening on agent/secondary/build-host **command allowlists** and API session gates.
4. Document results so the next agent does not re-discover the same debt.

Autotests were **skipped on live VPS** by operator request during some suite runs; package unit tests were still run in the sandbox where noted.

---

## 2. What was removed or cleaned (dead code)

| Area | Action | Notes |
|------|--------|--------|
| **TG** | Removed legacy release menu body; `m:release` → Versions | Deep-link / update button must open **Versions**, not old primary-only menu |
| **TG** | Pruned unused keyboards/handlers after `rg` call-site = 0 | Verify before delete: callbacks ≤64 bytes, forceHub `/menu` |
| **op** | Dropped unused wizard wrappers, dead SSH helpers, `guest_qr` | Operator Web is product UI |
| **packages** | Removed unused `internal/singboxconfig` | VPN config lives under `internal/vpn` |
| **agent / fleet / notify / CLI** | Dead helpers removed | See commits `5b8f9ee`, `7dbf82a`, `489232c` |
| **build-host** | Dropped dead primary→B SSH helpers | Agent plane only |

**Rule used:** a symbol is dead only if zero references outside its definition (and not required for `//go:linkname`, tests, or external API stability). Prefer leaving exported stubs that return a clear error (e.g. `SetBackupPeer`) over silent no-ops.

---


### Pass (post-journal) — dead symbols proved by call-graph

| Removed | Why |
|---------|-----|
| `internal/fleet/bot_failover.go` | All funcs only returned errors; CLI never called them |
| `internal/fleet/apply.go` (`ApplyLampac`) | Zero callers; lampac via `netductor install lampac` |
| `internal/fleet/sync.go` (`SyncPaths`) | Deprecated no-op; zero callers |
| `internal/vpn/qr.go` | `WriteQR`/`ASCIIQR`/`PrintASCIIQR` unused (QR via registry `qrcode.WriteFile`) |
| `ListGuests` / `GuestSummary` | Only each other |
| `TelegramCmd` | Zero callers |
| `FactsToJSON` / `FactsFromJSON` | Zero callers |
| `TouchBuildHostSeen` | Zero callers |
| `EnqueueSyncAll` | Dead; live path is `SyncAll` |
| `sshIdentityArgs` / `scpToSecondary` / `sshOnSecondary` | Primary→secondary SSH leftover |
| `cli_git` unreachable after `git project build` | Dead after `return 2` |

**Left intentionally:** `RotateToken` / `StatusOf` (store API, not wired to UI yet — product surface, not trash).



### Pass continued — more zero-ref exports

| Removed | Why |
|---------|-----|
| `formatFlatMap` | never called |
| `ShadowrocketGeneralBlock` | superseded by `ShadowrocketGeneralBlockFor` |
| `update.ApplyTag` | zero callers (`DownloadReleaseAsset` used) |
| `MarkShipPaused` | zero callers |
| `mtls.WriteClientMaterial` | zero callers |
| `ListHumanNames` | zero callers |
| `DeployPlan.Applying` | zero callers |
| `ApplyRoutesOrError` | zero callers |
| `edge/health.go` (`CheckRelayTCP`) | agent never called |
| `ndconfig.APIPort` / `SvcSPPrimary` / `SvcSPSecondary` | accessors unused (CIDR helpers kept) |



### Pass layer 3 — more zero-ref (including previously “intentional”)

| Removed | Why |
|---------|-----|
| `RotateToken`, `StatusOf`, `FindResult` | edge store, never wired |
| `CopyClientMaterial` | mtls, zero callers |
| `GroupSectionsCSV` | opcatalog docs helper unused |
| `PolicySnapshotPath` | never read |
| `secondary/catalog.go` | `RunCatalogHTTP`/`FormatCatalog` zero callers |
| `SaveMaterial`/`LoadMaterial` | Bootstrap writes material.json inline |
| `LoadBuildMaterial` | zero callers (`SaveBuildMaterial` kept) |
| `ClearToken` | only `ClearTokenKind` used |
| `SelfReplace`, `AgentDownloadURL` | zero callers |
| `deploy.NodeAsset`/`OperatorAsset` | zero callers; keep `Release` |
| `BuildShadowrocketRoutingConf` | alias; profile path used |
| `AddCustomURL` | zero callers (UI backlog was never wired) |



### Pass layer 4

| Removed | Why |
|---------|-----|
| `HostMonitoringPresent` | doctor/cli use Findings/Residual directly |
| `TokenConfigured` | zero callers |
| `SetToken` | only `SetTokenKind` used |
| `GithubTokenRepos` / `GithubTokenForAPI` | git uses `githubTokenLocal`; package uses unexported `githubToken` |
| `TryLocalReleaseAsset` | only unexported `tryLocalNamedAsset` used |
| `netmap.HintByID` | zero callers |



### Pass layer 5 — docs/scripts surface + CLI stubs

| Action | Detail |
|--------|--------|
| Deleted | all `docs/POST-REVIEW-0.9.*.md` (EN+RU) — one-shot suite crumbs |
| Deleted | `docs/DEEP-CODE-REVIEW-2026-10-02.md` (+ RU) — historical; not live handoff |
| Fixed | `VERIFICATION-PLAN` links to deep review removed |
| `InstallSyncTimer` | removed; `DisableLegacyFleetUnits` only |
| CLI fleet | `sync`/`bot-failover`/`apply-lampac` → exit 2 + disable legacy units |
| CLI git `build-all` | message matches real `ShipAllVPS` behavior |
| CLI `release ship` | deprecated alias wording (still runs matrix) |

Kept: `scripts/audit-hoster-*`, `tapo_v4` (protocol reference), agent python fallback (day-2 NVR).



### Pass layer 6 — TG leftovers (no behavior change for live menus)

| Action | Detail |
|--------|--------|
| Removed | `topicsLastReconcile` unused var |
| Removed | i18n `btn_topics` (no `T("btn_topics")` call sites) |
| Merged | `m:topics*` cases with `m:alerts-chat` (same reply body; deep-links still work) |
| Script | `build-tg-bot.sh` no longer creates `freshvps-tg-*` symlink |

**Not removed:** `MigrateAwayFromTopics` (startup one-shot), `MessageThreadID` JSON field, `m:release`→Versions bridge.


## 3. Security / correctness fixes from the same campaign

| Topic | Problem | Fix |
|-------|---------|-----|
| **Edge agent cmds** | `apply_rsc` in allowlist (OpenWrt-only agent) | Removed; agent errors explicitly |
| **Secondary cmds** | Enqueue without allowlist match | `allowedSecondaryCmd` + tests |
| **Build-host cmds** | Unrestricted enqueue | Allowlist: `backup_pull`, `recovery_arm` |
| **API session gate :8787** | Agent-token routes (build-host jobs, edge storage push) blocked if hit on session plane | Exempt only token-auth paths; operator routes still `requireSession` |
| **Stack apply** | Empty `version` failed `ValidReleaseTag` | Empty → `LatestReleaseTag` then validate |
| **Smoke** | Unrestricted mode string | `dual\|secondary\|primary\|local` only |
| **dnsblock** | `AddCustomURL` called `SetEnabled` with URL but only catalog **ids** worked | Resolve id or `https://`; reject http, localhost, private IP, credentials |
| **Recovery client** | Any scheme accepted for `:8790` URL | **https only** (`RecoverFromSecondary`, `PushBackupRecovery`) |
| **Ship extract/build** | `cd 'path'` shell-quoted `work` | `cmd.Dir = work` (no path in shell quote) |
| **Formula** | Scripts still pointed at `Formula/netductor.rb` | `netductor-op.rb` only |
| **Op version pin** | Hardcoded stale version string | Pin to release line (0.9.403) |

Already solid (verified, not rewritten): mTLS `ClientDir` sanitization, recovery `ConstantTimeCompare` + rate limit, NVR `PathUnderRoot` + clip tokens, openssl backup pass via **env** (not argv), SR work CIDRs in skip-proxy not tun-excluded.

---

## 4. Commits (post v0.9.403 hygiene series)

Approximate order (newest last on main may differ; use `git log`):

- `489232c` — TG dead menus; SSH show wired; release → Versions  
- `7dbf82a` — op leftovers, singboxconfig, Web policy form wiring  
- `5b8f9ee` — agent/install/fleet/vpn/notify/cli dead helpers  
- `55490fb` — Formula path scripts  
- `ff14b04` — edge/secondary cmd allowlists  
- `8574c31` — build-host cmd allowlist  
- `12dd1c0` — API session exemptions + test  
- `0823824` — stack empty→latest; smoke modes  
- `3a54537` — dnsblock custom URL safety  
- `1245563` — recovery https client  
- *(this doc + ship `cmd.Dir`)*  

**Release:** last published product stop was **v0.9.403**. Tip may be **ahead** of the GitHub Release until the operator asks to cut **0.9.404+**.

---

## 5. What remains for the next AI

### 5.1 Must do before claiming “clean prod tip = release”

1. **Cut a release** (0.9.404 or next) including all tip commits: node, tg, agent multi-arch, op darwin/linux, `SHA256SUMS`, Formula sha256, VERSION == `internal/version.Release` == op pin.  
2. Run **REVIEW-SUITE** sections that were skipped (especially full `go test` if operator allows; never invent a shorter suite).  
3. Update **AGENT_HANDOFF** / **OPEN_ITEMS** stop line to the new tag (EN+RU).

### 5.2 Product / hardware (not closed by cleanup)

| Item | Notes |
|------|--------|
| **OpenWrt / Cudy e2e** | USB overlay plan implemented in tree; lab hardware still operator-driven |
| **Edge VLESS day-2** | Full client after enroll/template; confirm first-boot vs day-2 contract in PLAN |
| **Build-host VPS e2e** | Code path ready; no host → no false “paths down” alerts when registry empty (verify live) |
| **Dual-node smoke on prod** | Operator checklist, not only unit tests |
| **MikroTik API edge** | Plan exists; not the focus of this cleanup |
| **NVR / Tapo two-way** | Research / partial |

### 5.3 Optional hardening (backlog, not blockers)

- Further **R1** split of secondary sing-box builders (partially done).  
- Wire **AddCustomURL** to Web/TG UI if product wants custom lists (backend ready).  
- Recovery client still uses `InsecureSkipVerify` (self-signed :8790) — pin CA if product wants.  
- CrowdSec / advanced scanner noise — deferred historically.  
- IPv6 dual-stack, multi-primary HA, IdP — explicitly **not now**.

### 5.4 How to continue review without redoing everything

1. Start from **this file** + `git log v0.9.403..HEAD`.  
2. Next unreviewed packages if still hungry: `internal/notify` batching edge cases, `cmd/netductor-op` Web static assets, OpenWrt shell under `openwrt/`.  
3. Dead code: **prove** zero refs; do not delete “unused-looking” allowlist entries that agents still send.  
4. Any API change: session plane vs agent plane (`:8787` vs `:8789`); update `apiPathExemptFromSession` + `httputil_exempt_test.go`.  
5. Docs: **EN + RU same meaning**; no stub RU.

---

## 6. Operator apply (when release exists)

```bash
# primary
netductor stack apply v0.9.404   # or tip tag
netductor secondary cmd <id> upgrade:v0.9.404
# Mac
brew update && brew upgrade netductor-op
# prefer brew over netductor-op update when installed via Formula
```

---

*Last updated: dead-fleet/QR/guest/notify/secondary helpers pass (post e0e1859). Extend §2/§3 on next pass.*
