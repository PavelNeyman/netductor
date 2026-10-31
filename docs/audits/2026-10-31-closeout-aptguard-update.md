# Close-out: APT Guard + Update system (2026-10-31)

## APT Repository Guard

| Stage | Code | Live stand |
|-------|------|------------|
| 0–8 | Done | Operator run once on P/S recommended |
| 9 automated e2e | `go test ./internal/aptguard/ -run E2E` | N/A |
| 9 live canary | — | **OPEN** |

## Update system

| Phase | Code | Live stand |
|-------|------|------------|
| 0–8 | Done | — |
| 9 unit matrix | Done | — |
| 9 dual-VPS / Edge / live UI | — | **OPEN** |
| 10 Edge canary, fleet go/no-go | — | **OPEN** |

**Conclusion:** update-system is **code-complete** for the designed engine; remaining acceptance is **operational evidence**.

## SSH key-only secondary

Code through harden/flock path is in tree. **OPEN:** live Secondary provision e2e.
