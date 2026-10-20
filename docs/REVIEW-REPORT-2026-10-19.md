
## Fixed in 0.9.428 (2026-10-20)

| ID | Fix |
|--|--|
| S-1 | `RevokeSubToken` on delete and disable; `/sub/` rejects disabled users |
| S-2 | Recovery pin present but not PEM → fail closed (no silent SkipVerify) |
| S-8 | `Approve` propagates `saveDevices` error |
| Q-1 | i18n keys `edge_st_g_bak`, `rd_mode`, `rd_pos`, `result_hint` EN/RU |
| Q-2 | Removed self-assignment in `internal/git/projects.go` |
| incident | `Collect` no longer execs `*.test` (fork-bomb under `go test`) |

**EN** · [RU](ru/REVIEW-REPORT-2026-10-19.md)

# Full-complex review report — tip v0.9.427 (2026-10-20)

Run per [REVIEW-SUITE.md](REVIEW-SUITE.md). Work plan and chunk status: [REVIEW-PLAN-2026-10-19.md](REVIEW-PLAN-2026-10-19.md).
Scope: whole repository at `main` = `451a83bd` (v0.9.427), 533 Go files, ~80.8k production LoC, 141 test files.

**Owner constraints for this run:** automated tests (§1) were **not executed** by request; no code was changed (documentation only); no live fleet access (§8 not run). Everything below is static analysis + reading code.

## 0. Method (reproducible)

| Tool | Purpose | Notes |
|--|--|--|
| `gofmt -l`, `go vet ./...`, `go build ./...` | format / vet / compile | Go 1.22.2; deps fetched from GitHub mirrors (`golang.org/x/*` → `github.com/golang/*`, `gopkg.in/yaml.v3` → `go-yaml/yaml`) in a scratch copy, repo `go.mod` untouched |
| `deadcode` (x/tools v0.28, RTA) | functions unreachable from any `main` | run for linux/amd64 and darwin/arm64 (identical result); no build tags / GOOS files exist, so reachability is arch-independent |
| `python` identifier scan | unused package-level types/vars/consts | counts references inside the package and `pkg.Name` elsewhere; every hit re-checked with `git grep` |
| `scripts/check-*.sh` | repo static guards | versions, version pins, web UI, UI parity |
| manual reading | auth, tokens, recovery TLS, guest portal, edge store | see §3 |

Dead-code rule: a symbol is listed as **dead** only if it has no non-test reference, is not reached through reflection/templates (none are used in the repo), is not an interface-satisfying method, and `deadcode` agrees on both OS targets.

## 1. Tip / release / formula (§2)

- `VERSION` = `internal/version.Release` = 0.9.427 ✅; version pins ✅.
- GitHub: latest published release is **v0.9.426**; **v0.9.427 is not released yet** (tip ahead of release, known from handoff). `Formula/netductor-op.rb` = 0.9.426 with real SHA256 and no `:no_check` ✅; only `netductor-op.rb` exists ✅.

## 2. Tests (§1)

Not run (owner request). Static facts: `go build ./...` passes; all four binaries and the agent cross-build (arm, mips, mipsle, 386) compile. §1.3 debt table: no P0/P1 package without tests remains. §1.4 TG invariants are tests and were not run.

## 3. Findings

Severity: Critical · High · Medium · Low · Info. **Status** = what was done in this run (docs only) / proposal.

### 3.1 Security (§10)

| ID | Sev | Finding | Proposal |
|--|--|--|--|
| S-1 | **Medium** | `vpn.RevokeSubToken` is never called. Deleting a VPN user (`RevokeNative`) leaves its row in `vpn_sub_tokens.json`; **creating a user with the same name later makes the old subscription URL valid again** for whoever still holds it. Also `handleSubscription` does not check `Enabled`, so a *disabled* user's subscription body is still served (the UUID no longer works in sing-box, but the server list leaks). | Call `RevokeSubToken(name)` in `RevokeNative`; reject disabled users in `SubscriptionBody`; add a test |
| S-2 | **Medium** | Recovery HTTP client (`install/backup.go` `recoveryHTTPClient`) falls back to `InsecureSkipVerify` when no CA pin exists **or when the pin file is present but not valid PEM** (silent downgrade). The channel carries encrypted backups and the key sidecar. | Fail closed on unparsable pin; for first contact require explicit flag or TOFU-record the fingerprint |
| S-3 | Low | Secondary stores `BACKUP_KEY.txt` next to the `.ndenc` archives in `peers/core/` — at-rest encryption on that host is effectively void. | Document as accepted risk, or split key custody (operator-only) |
| S-4 | Low | Non-constant-time secret comparisons: guest `Code`/`Token`, `ResolveSubToken` (linear scan with `==`), reboot challenge code. Mitigated by TTL, one-shot and rate limit on `/sub/`. Edge tokens already use `constEq` (sha256 + `ConstantTimeCompare`). | Promote `constEq` to a shared helper and use it everywhere |
| S-5 | Low | `http.ListenAndServe*` without `http.Server` timeouts at 6 sites (guest captive portal on untrusted guest Wi-Fi, agent `:8790`, secondary recovery, serve/operator on loopback). Only 4 timeout settings in the whole repo → slow-client exhaustion. | One constructor in `internal/httpx` with Read/Write/Idle timeouts |
| S-6 | Low | Shell composition: `echo y \| ufw delete <n>` where `n` is parsed from `ufw` output; WireGuard **private key passed in argv** (`bash -c "echo '<priv>' \| wg pubkey"`, `svcpaths/bootstrap.go:44`) is visible in `/proc/<pid>/cmdline`. | argv + stdin (`cmd.Stdin`), no shell |
| S-7 | Low | `curl … get.docker.com \| sh` in `install/lampac.go:71` contradicts the "verify SHA256" supply-chain rule (§10). | Use distro `docker.io` or a pinned/verified script |
| S-8 | Low | `edge/store.go`: `_ = saveDevices(m)` at 8 sites; `Approve` returns a token that may not have been persisted. Silent failure on a security-critical path (§9 observability). | Propagate the error |
| S-9 | Info | `InsecureSkipVerify` also in `probes/run.go`, `tapo/client.go`, `tapo/klap.go`, `vpn/sni_health.go` — local self-signed targets / SNI probe, by design; two lack the `//nolint:gosec` rationale comment. | Annotate |
| S-10 | Info | **Verified OK:** global `apiSessionGate` (defence-in-depth for `/api/*`, explicit exempt list); `serve` refuses non-loopback bind; edge `Deny`/`Revoke` clear the device token so `DeviceIDFromAuth` cannot match; secrets written `0600`; no `math/rand`; no reflection/templates; `0644` writes found are non-secret config. | — |

### 3.2 Correctness / quality (§9)

| ID | Sev | Finding | Proposal |
|--|--|--|--|
| Q-1 | Medium | Web UI i18n: `check-web-ui.sh` reports **4 `data-i18n` keys missing from the JS dictionary**: `edge_st_g_bak`, `rd_mode`, `rd_pos`, `result_hint` (Result dock, 0.9.422–0.9.426). Violates §7 (both `I18N.en` and `I18N.ru`). | Add keys EN+RU |
| Q-2 | Low | `go vet`: `internal/git/projects.go:108` self-assignment `list[i].CreatedAt = list[i].CreatedAt` (no-op; intended source field unknown). | Fix or delete |
| Q-3 | Low | **129 of 533 Go files are not gofmt-clean** (mostly `cmd/*`); no format gate. | One mechanical `gofmt -w` commit + add `gofmt -l` to the suite |
| Q-4 | Info | `install/backup_verify.go:52` — constant 4 GiB overflows `int` on 32-bit; the package is not part of any 32-bit binary (agent cross-builds fine). | Use `int64` |
| Q-5 | Info | `check-web-ui.sh` WARN: two `/v1` fetches without `X-Netductor-Token` (`app.js:779`, `:2130`); `/v1/meta` (version line) looks intentionally open, `:779` builds headers earlier — likely false positives. | Confirm in browser smoke |

### 3.3 Documentation (§3, §4)

| ID | Sev | Finding | Status |
|--|--|--|--|
| D-1 | Medium | **19 EN docs have no RU twin**, against the AGENTS.md rule: CI-WORKFLOWS, CLEANUP, DAY2-GROUPS, DISASTER-RECOVERY, GH-PRIVATE-SETUP, OPENWRT-LAB, PLAN-CI-NEXT, PLAN-EDGE-DEPLOY-HYBRID, PLAN-GIT-CI-OPS, PLAN-INTERNAL-RELEASES, PLAN-NVR-IMPLEMENTATION, PLAN-NVR-UI, PLAN-REMOVE-TG-TOPICS, PLAN-TG-MENU, RECOVER-BOT, RELEASE, REVIEW-PROGRESS, TG-ALERTS-CHANNEL, YOUTUBE-ADS. | Open — tracked in OPEN_ITEMS |
| D-2 | Medium | Stale tip markers: `AGENTS.md` baseline v0.9.261 and Project State 0.8.30; handoff top said v0.9.414/0.9.421; OPEN_ITEMS stop v0.9.422/0.9.414; RU handoff top contradicted itself; EN handoff had several concatenated `# AGENT Handoff` blocks. | **Fixed in this run** (EN+RU) |
| D-3 | Info | `PLAN-EDGE-DEPLOY-HYBRID.md` says `ssh_harden` reuses `edge.Harden`, but `edge.Harden` has no caller (see DC-A). | Reconcile when dead code is removed |
| D-4 | Low | Broken relative links: `ru/OPEN_ITEMS.md` pointed to five docs that have no RU twin (consequence of D-1); `AGENT_HANDOFF.md` linked a removed `TG-MEDIA-TOPICS.md`. | **Fixed in this run** |

## 4. Dead code (verified)

### DC-A. Dead everywhere (no production, test, or doc-as-API reference)

| Package | Symbols |
|--|--|
| `cmd/netductor` | `withSession` (superseded by global `apiSessionGate`) |
| `cmd/netductor-op` | `capture`; type `menuItem` + methods `Title/Description/FilterValue`; model field `list list.Model` (+ its `bubbles/list` import); styles `docStyle`, `helpStyle`, `chipStyle` |
| `cmd/netductor-tg` | `shareURIFrom`; `var lastAccessMsg`; const `statusSh` |
| `internal/cli18n` | `Register` |
| `internal/deploy` | `scpTo`; `PrepareEdgeOffline` (wrapper — callers use `PrepareEdgeOfflineOpts`) |
| `internal/git` | `BuildProject` and, transitively, `WaitVpsJob`; `ExtractAndBuild`; `shouldPauseAfterStep` |
| `internal/install` | `PrunePeerBackups` (duplicate of live `secondary.prunePeerArchives`); `AllowAgentMTLSFromIP` (comment says it must not be called); `var unwantedPackages` (superseded by substring detection — verify purge coverage) |
| `internal/logs` | `runOut` (the live one is `install.runOut`) |
| `internal/mikrotik` | `Policy` |
| `internal/notify` | `htmlEsc` (the agent has its own) |
| `internal/nvr` | `EnsureStorageReady` |
| `internal/opcatalog` | `GroupLabel`, `Sections()` function (the `Group.Sections` field is live) |
| `internal/operator` | `FleetDeploy` (live: `FleetDeployWithReport`) |
| `internal/registry` | `Catalog` (live: `CatalogDetail`) |
| `internal/secondary` | `HardenSSH` (legacy; `mikrotik.HardenSSH` is a different live function) |
| `internal/svcpaths` | `WaitPing` |
| `internal/tapo` | `Client.GetDeviceInfo` (alias) |
| `internal/upstream` | `Save` (no caller; pins are read-only today) |
| `internal/edge` | `Harden` |
| `internal/vpn` | `RevokeSubToken` — **not harmless dead code, see S-1** |
| constants / vars | `devices.reFrom`; `edgestorage.ExpectUSBFlag`, `FactUSBPresent`, `FactExtrootActive`, `FactExpectUSB` (agent/UI use string literals — duplication); `fleet.LabelLampac`, `LabelBot`; `guest.TokenTTL`; `ndconfig.DefSvcSPPrimary/SPSecondary/PSPrimary/PSSecondary` (duplicates of `svcpaths.AddrP_*`); `servicenet.LegacyCIDR`, `Prefix`; `sites.Location` (alias) |

### DC-B. Alive only because tests call them (production-unreferenced)

`tg.formatVPNList`, `deploy.LoadCachedTemplate`, `dnsblock.DefaultBlockyDenyURLs`, `edge.DeleteTemplate`, **`edge.ValidBearer`** (tests assert a stricter check than the production path `DeviceIDFromAuth`, so the tests give false confidence — see R-5), `edgeagent.ShellApply`, `edgeagent.ParseUCIBatch`, `edgeagent.VLESSClientConfig`, `git.ReleaseVpsSlot`, `metrics.LatestProbes`, `nvr.LeaseRemainingSec`, `opcatalog.ByID`, `opcatalog.Destructive`, `vpn.ParseMismatchCount`; and a test helper living in production code: `edge.resetUSBPrevForTest`.

### DC-C. Checked and **not** dead (false positives of naive tools)

`FleetDeployWithReport`, `PrepareEdgeOfflineOpts`, `CatalogDetail`, `mikrotik.HardenSSH`, `install.runOut`, `addons.Register`, `jobs.Store.Register`, `mtls.Dir`, agent `htmlEsc`. Reachability was identical on linux/amd64 and darwin/arm64.

Deleting DC-A removes roughly 45 functions plus ~25 constants/vars; for DC-B decide per item: delete together with its test, or keep and document as public API.

## 5. Refactoring (§11)

| ID | Priority | Item | Why |
|--|--|--|--|
| R-1 | High | God functions: `registerEdgeAPI` 937 lines, `handleCallback` (TG) 779, `handleGitCB` 666, `runDoctorNative` 637, `registerSessionAPI` 597, `registerGitAPI` 567, `handleMessage` 540, `registerNVRAPI` 533, `runDeploy` 480, `DeployEdge` 385 | Untestable, merge-conflict magnets; split into one handler per route/callback and a table-driven dispatch |
| R-2 | Medium | Duplicated helpers: `htmlEsc`×2, `shellQuote`×2, `runOut`×2, `randomToken`×3, `truncate`×4, `esc`×2 | Move to one small internal package (`internal/textutil`, `internal/sysexec`) |
| R-3 | Medium | 32 `bash -c`/`sh -c` call sites, many `_ = exec.Command(...).Run()` | One `sysexec` helper: argv only, context timeout, logged errors |
| R-4 | Medium | `internal/httpx` exists but servers are built ad hoc (S-5) | Single hardened `NewServer` |
| R-5 | Medium | Edge auth has three near-duplicates (`ValidBearer`, `DeviceIDFromAuth`, `RequireApproved`) | One function returning `(deviceID, approved)`; point tests at the production path |
| R-6 | Low | Constants duplicated across packages (DC-A last row) | Single source of truth in `ndconfig` |
| R-7 | Low | `internal/operator/serve.go` 1208 lines mixes routing, jobs, deploy | Split by concern |
| R-8 | Low | Test helper in production file; legacy wrappers kept for tests (DC-B) | `export_test.go`; delete wrappers |

Prior closed R* items in `REVIEW-PROGRESS.md` were not reopened.

## 6. Layering / UI parity (§5–§7)

`check-ui-parity.sh`: OK for doctor, vpn, edge, secondary, mtls, nodes, sites, backup, nvr, deploy, fleet, ssh-hosts. TG: 299 distinct `m:*` callbacks are emitted, **0 without a handler** (heuristic; the real guard is `TestNoObviousOrphanMenuCallbacks`). Web i18n: see Q-1.

## 7. Operator reply template (§12)

1. Tip: 0.9.427 local, release 0.9.426 published. 2. Tests: not run (owner). 3. Docs: handoff / OPEN_ITEMS / AGENTS updated EN+RU; D-1 open. 4. UI parity: OK, Q-1. 5. TG: no orphans. 6. Code review: Q-1…Q-5. 7. Security: S-1…S-10. 8. Refactoring: R-1…R-8 (not "no refactor required"). 9. Fix-now candidates (need owner approval, all small): S-1, Q-1, Q-2, gofmt commit, DC-A removal. 10. No apply commands (tip not moved).
