# Audit: Apply + Startup Migration (Stage 1)

**Base SHA:** tip at time of audit (post 0.9.640 / update API fixes).  
**Scope:** existing Apply paths only — no parallel update system.

## 1. Where backup is created

| Path | Function | When |
|------|----------|------|
| `internal/stack/stack.go` `ApplyOpts` | `install.Backup()` | Before node+tg download when `!noBackup` |
| Failure of backup | **warn only** (`fmt.Fprintln(os.Stderr, "warn backup:", err)`) — **does not stop Apply** |

**Gap (P1):** backup failure must fail Apply (TZ §3.1).

## 2. Binary download order (node stack)

1. Download `node` → `/usr/local/bin/netductor`
2. Download `tg` → `/usr/local/bin/netductor-tg`
3. Restart units (errors often ignored)
4. Health check API; limited auto-rollback

**Gap:** second download failure can leave first binary replaced (partial pair).

## 3. Host component Apply (`internal/update/apply_host.go`)

- In-process job registry only (no cross-process flock)
- `atomicReplaceBinary` / `copyFile` without full durable fsync protocol
- Restart via `systemctl try-restart` with ignored error in some paths
- Version probe runs **disk binary**, not running process

## 4. API `/api/update/apply` (fixed in 0.9.641)

| Before | After |
|--------|--------|
| `Restart` not passed | `Restart: !NoRestart && unit != ""` |
| `agent` → edge binary + secondary unit | Split: `edge-agent` vs `secondary-agent` |
| `WriteVERSION` on every component | Only for `node` |

## 5. Startup migration

`RepairStateOnUpgrade` / baseline exist but **no** Migration Registry, durable migration journal, or Ready gate.

## 6. Ordered work (from TZ)

1. ~~Audit~~ (this doc)
2. Apply: fail-closed backup, pair atomicity node+tg, restart errors, process version
3. Cross-process lock for Host Apply
4. Migration Runner before Ready
5. 0.9.612 fixture upgrade tests
6. Secondary SSH lock / join rollback (separate P1 track)

## Evidence

Commands reviewed: `stack.ApplyOpts`, `ndupdate.ApplyHostComponent`, `api_update.go`, `SecondaryUnits`, `InstallRelay` ExecStart.

## Stage 2 progress (0.9.642)

| Item | Status |
|------|--------|
| Backup fail-closed | Done — error returns; `--no-backup` / `noBackup` still available |
| Stage both downloads before live replace | Done |
| Restore node if tg install fails | Done (`installStagedPairTo`) |
| Restart error logging | Done (API/TG/redirect) |
| Process version probe | Open |
| Host Apply cross-process flock | Open |
| Startup Migration Runner | Open |

## Stage 2b progress (0.9.643)

| Item | Status |
|------|--------|
| Process version probe | Done — MainPID → /proc/PID/exe |
| Host Apply cross-process flock | Done |
| Startup Migration Runner | Open |

## Stage 3 progress (0.9.644)

| Item | Status |
|------|--------|
| Migration interface + registry | Done |
| Durable journal | Done |
| Flock | Done |
| runServe gate | Done |
| 0.9.612 fixture suite | Open |
| More role-aware migrations | Open |

## Stage 3b (0.9.645)

| Item | Status |
|------|--------|
| Role file migration | Done |
| PostCheck | Done |
| Secondary agent hook | Done |
| CLI migrate status/run | Done |
| Empty-state fixture | Done (minimal) |
| Full 0.9.612 snapshot fixture | Open |
