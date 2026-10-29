# Update recovery (Phase 2)

## Automatic rollback — when safe

| Situation | Automatic? | Mechanism |
|-----------|------------|-----------|
| tg/agent health fail after replace | **Yes** | Restore `dest.prev` → state `rolled_back` or `failed` if restore fails |
| tg/agent version probe mismatch | **Yes** | Same as health fail |
| Concurrent second apply | **No install** | JobRegistry conflict |
| Bad checksum / resolve | **No install** | Fail before replace |
| **node** `stack apply` API health fail | **No binary rollback** | Binaries kept; operator may `stack rollback` to `prev/` |
| Updater process killed mid-download | **Partial** | `.new` may remain; next apply removes and retries |
| Updater killed after rename, before health | **Operator** | Check unit + `bin version`; restore `.prev` or re-apply |

## Operator intervention

1. **Stuck job (process registry only):** restart CLI/process clears in-memory JobRegistry; on-disk binaries decide truth.
2. **`.prev` present, service broken:** `mv dest.prev dest` and `systemctl restart <unit>`.
3. **node:** `netductor stack rollback` uses last-good `prev/` snapshot after a successful apply.
4. **Never** set `NETDUCTOR_UPDATE_SKIP_VERIFY` on production except documented emergency.

## Interrupted updater

- Download target is `dest.new` (tg/agent). Incomplete `.new` is deleted on next apply start.
- `WriteVERSION` runs only after CLI reports success — do not trust VERSION alone after a crash.
- stack lock file: if apply was interrupted, clear only after confirming no other apply (`stack` lock helpers).
