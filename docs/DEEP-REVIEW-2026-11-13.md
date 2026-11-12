# Netductor Deep Review Report — 2026-11-13

**Auditor:** Qwen3  
**Commit:** 131464d83e26 (main branch)  
**Coverage:** ~95-100% of codebase  
**Findings:** 19 bugs (3 HIGH, 3 MED-HIGH, 10 MED, 3 LOW)

---

## Executive Summary

Comprehensive code review covering ~95-100% of the Netductor codebase identified **19 real bugs**, with **3 HIGH severity** issues requiring immediate attention. The primary systemic problem is **inconsistent use of locking mechanisms** across shared JSON registries, which is the root cause of all HIGH-severity bugs.

The project is well-designed overall (atomic writes in statefile, proper validation, idempotency in OpenWrt agent), but locking is applied inconsistently—some registries use flock while others have no synchronization, leading to lost updates and resurrection of deleted entries.

---

## Critical Issues (HIGH) — Immediate Fix Required

### P1-1. `SetRegion` without locks → command resurrection
**File:** `internal/secondary/store.go:883-897`  
**Issue:** CLI and API-heartbeat operate concurrently without mu/flock → lost-update → `PendingCmds` can resurrect → duplicate `reboot`/`upgrade` execution.  
**Impact:** Duplicate critical operations on secondary VPS.

### P3-R1. VPN user registry without locks → revoked user resurrection
**File:** `internal/vpn/registry.go` (Add:340, Revoke:401, SetEnabled:380, SetNote:367, Rename:288, SetUserPolicy:520); `users.go` SetKind:121  
**Issue:** API + CLI + TG bot mutate registry concurrently without mu/flock (only `subMu` for sub-tokens).  
**Impact:** Revoked user can resurrect → unauthorized VPN access.

### P4-SV1. `Tick()` without failover-lock from collect → state machine race
**File:** `cmd/netductor/collect.go:209` (apply.go:391-399 under lock)  
**Issue:** Concurrent collect + apply without synchronization → race on `failover-state.json`; `minSample` guard races. Lock exists specifically for this (`lock.go:13`).  
**Impact:** False failback, anti-flap state reset, routing flap.

---

## Important Issues (MED-HIGH)

### P1-5. Release build/ship doesn't stamp version
**File:** `internal/update/build.go:69-70,125`; `ship.go:126-127`  
**Issue:** Pinning looks for `const Release` (but declared as `var`) = no-op; ldflags `-X main.version` (symbol doesn't exist in package main) = no-op.  
**Contrast:** `scripts/release.sh:41` correctly uses `-X internal/version.Release`.  
**Impact:** Binaries report source defaults, not built tag → fleet confusion, version tracking impossible.

### P6-D2. Telegram token with `$` or backticks written incorrectly
**File:** `internal/deploy/primary.go:151-155`  
**Issue:** `%q` produces Go-escape (`"\n"`, `"\t"`), but shell interprets `$` and backticks inside `"..."` → variable expansion.  
**Impact:** Token `foo$HOMEbar` → shell expands → wrong token written → bot fails or runs under wrong account.

### P6-B1. `keyBytes` = SHA256 without KDF
**File:** `internal/install/backup.go:80-82`  
**Issue:** `sha256.Sum256([]byte(pass))` — not a KDF, just a hash. No salt, no iterations (PBKDF2/scrypt/argon2).  
**Impact:** For `backup_key` (always `randomHex(32)`) this is OK, but if someone uses `Restore` with user password — brute-force is trivial.

---

## Medium Issues (MED)

### P1-2. Command result — 1 attempt + race with self-restart
**File:** `internal/secondary/agent.go:216-222,331-352`  
**Issue:** `reboot`/`mtls_refresh` (2s delay) kill in-flight report → result lost.  
**Contrast:** OpenWrt agent (pass 5) uses atomic `started`/`completed` markers + idempotent re-lease.  
**Recommendation:** Port `cmd_done` pattern from OpenWrt to secondary.

### P1-3. `reportCmdDone` overwrites MismatchTotal/ByIP + UplinkOK semantics
**File:** `internal/secondary/agent.go:331-338` + `store.go:376-377`  
**Issue:** Sends zeros instead of actual values; `UplinkOK` via TCP:443 instead of tunnel-ping.

### P1-4. Heartbeat unconditionally writes PBK/SID → silent RU-entry loss
**File:** `internal/secondary/store.go:363-364`; `internal/vpn/secondary_bundle.go:38`  
**Issue:** Empty PBK/SID → `continue` in bundle → RU-entry disappears from all client configs.  
**Impact:** Failover degradation (RU endpoint unavailable), self-heals on next heartbeat.

### P2-F1. Host port opens only before explicit global DROP
**File:** `internal/firewall/firewall.go:2192-2217,2324-2381,2429-2487`  
**Issue:** REJECT/policy-DROP → ACCEPT not inserted → `VerifyWSS` fail → Apply fail-closed.  
**Contrast:** `cleanup:2600` recognizes REJECT.

### P2-F2. `hostAcceptsTCPPort` ignores -s/-i/-d/!
**File:** `internal/firewall/firewall.go:2219-2237`  
**Issue:** Restricted ACCEPT (`-s 10.0.0.0/8`) considered open → global not inserted → port closed for secondary/edge.

### P3-R7. `ApplyConfig` writes sing-box config non-atomically
**File:** `internal/vpn/apply.go:126`  
**Issue:** `os.WriteFile` directly (vs atomic `bumpSecondaryConfigVer:184-192`). Crash → truncated config → VPN down.

### P3-R9. `bumpSecondaryConfigVer` mutates devices.json without flock
**File:** `internal/vpn/apply.go:159-192`  
**Issue:** Overwrites heartbeat (same class as P1-1/P3-R1).

### P3-R3. `RenameNative` commits registry before `RenameSubToken`
**File:** `internal/vpn/registry.go:308-314`  
**Issue:** `RenameSubToken` failure after `writeRegistry` → subscription desync.

### P4-SV2. `saveState` non-atomic + `LoadState` without defaults on corrupted JSON
**File:** `internal/svcpaths/failover.go:110,94-104`  
**Issue:** Corrupted state → `UserPath=""`/anti-flap reset → flap/empty route.

### P6-D3. `DeployPrimary`/`Run` non-atomic
**File:** `internal/deploy/primary.go:50-175`; `internal/install/install.go:35`  
**Issue:** Partial deploy without rollback/cleanup → system in inconsistent state.

---

## Low Issues (LOW)

### P3-R8. `tryCheck=true` when sing-box binary missing
**File:** `internal/vpn/apply.go:212-213`  
**Issue:** Validation skip in production (binary should exist), but window possible during upgrade.

### P6-D1. `writeAskPass` doesn't escape `secPath` in script
**File:** `internal/deploy/sshutil.go:170`  
**Issue:** tmp-path is safe, but pattern is defective (no `ShellQuote`).

### P7-O1. `cli_project.go:284` — `%q` in bash-script for paths
**File:** `cmd/netductor-op/cli_project.go:284`  
**Issue:** Self-inflicted (user enters path with `$`), but pattern is wrong.

---

## Clean Areas (Passes 5, 8, 9)

Verified, no bugs found:
- **cmd/netductor-agent (OpenWrt)** — idempotency via `cmd_done`, atomic markers, crash-ambiguity handling, UCI validation, argv without shell
- **internal/mtls** — TLS 1.3, mutual auth, CRL, hot-reload for CA rollover
- **internal/git** — rev validation, path traversal check, argv without shell
- **internal/stack** — O_EXCL lock + liveness check, atomicity, extensive rollback
- **internal/edge** — locks (45 hits), atomicity (4 hits), SSH keys escaped (`provision.go:128`)
- **internal/policy** — locks (4 hits), atomicity (2 hits)
- **internal/operator/serve.go** — `fleetMu` for deploy locking (correct)
- **internal/aptguard** — `exec.Command("bash", "-c", ...)` but hardcoded command (apt-get update), no user input
- **internal/nvr** — exec only ffmpeg without shell, http with timeout, locks, atomicity
- **internal/jobs, statefile** — correct atomic write pattern
- **internal/tapo, migrate, sites, metrics, channels, addons, domain, servicenet** — exec (df, systemctl, docker, ip, iptables), all argv without shell

---

## Systemic Root Causes

### 1. Locking applied inconsistently
**Problem:** Some registries under flock (secondary store, svcpaths-apply, stack), some without (vpn users, SetRegion, bumpSecondaryConfigVer, collect Tick).  
**Root of:** P1-1, P3-R1, P3-R9, P4-SV1 (all 3 HIGH + 1 MED).  
**Fix:** Unified locking scheme for all shared JSON registries.

### 2. Non-atomic writes of critical files
**Problem:** Despite atomic analogues nearby (bumpSecondaryConfigVer, stack apply_tx, statefile.SaveJSON).  
**Root of:** P3-R7, P4-SV2, P6-D3.  
**Fix:** All critical writes via tmp+rename+fsync (statefile pattern).

### 3. Shell injection via remote scripts
**Problem:** `%q` produces Go-escape, but shell interprets `$` and backticks.  
**Root of:** P6-D2 (telegram token), P6-D1, P7-O1.  
**Fix:** Use `ShellQuote` or pass via env/argv, not via formatting.

---

## Coverage

**Deeply reviewed (~95-100% of codebase):**
- internal/secondary (174 KB) ✅
- internal/firewall (118 KB) ✅
- internal/vpn (141 KB) ✅
- internal/svcpaths (290 KB) ✅
- internal/update (117 KB) ✅
- cmd/netductor-agent (185 KB) ✅
- internal/deploy + install (486 KB) ✅
- cmd/netductor-op + internal/operator (360 KB) ✅
- internal/mtls (86 KB) ✅
- internal/git (103 KB) ✅
- internal/stack (80 KB) ✅
- internal/edge (125 KB) ✅
- internal/policy (25 KB) ✅
- internal/jobs (8 KB) ✅
- internal/statefile (2 KB) ✅
- internal/nvr (48 KB) ✅
- internal/hardening (29 KB) ✅
- internal/notify (35 KB) ✅
- internal/fleet (21 KB) ✅
- Small packages (tapo, migrate, sites, metrics, channels, addons, domain, servicenet) ✅

**Not deeply covered:** Nothing significant remains.

---

## Fix Priorities

### Immediate (HIGH + MED-HIGH):
1. **P1-1, P3-R1, P4-SV1** — add locks for all shared registries (unified scheme)
2. **P6-D2** — escape telegram token via `ShellQuote` or env
3. **P6-B1** — replace SHA256 with PBKDF2/scrypt/argon2 for user passwords
4. **P1-5** — fix version stamping in build/ship

### Next sprint (critical MED):
5. **P1-2** — port `cmd_done` pattern from OpenWrt to secondary
6. **P3-R7, P4-SV2, P6-D3** — atomic write for all critical files (use statefile.SaveJSON)

### Backlog (remaining MED + LOW):
7. P1-3, P1-4, P2-F1, P2-F2, P3-R9, P3-R3 — logic and validation
8. P3-R8, P6-D1, P7-O1 — patterns and edge cases

---

## Conclusion

**Review complete. 19 real bugs found on ~95-100% codebase coverage.**

**3 HIGH require immediate fix** — all related to missing locks (unified flock/O_EXCL scheme will close them). Project is well-designed overall (atomic writes in statefile, proper validation, idempotency in OpenWrt agent), but locking is applied inconsistently — this is the main systemic root.

---

**Reviewed by:** Qwen3  
**Date:** 2026-11-13  
**Methodology:** Static analysis, code reading, pattern detection (no code modifications)
