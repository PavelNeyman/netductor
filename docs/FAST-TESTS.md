# Fast test profile (~2 GiB hosts)

**Status:** current (2026-10-12).  
Use when the machine has ~2 GiB RAM and no swap (full `go test ./internal/...` OOMs).

## Recommended env

```bash
export GOTOOLCHAIN=local
export GOPROXY=https://proxy.golang.org,direct
export GOMAXPROCS=1
export NETDUCTOR_DISABLE_VPS_WORKER=1
```

## What to run

Prefer package-by-package (or small batches):

```bash
go test ./internal/notify -count=1 -timeout 60s
go test ./internal/incident -count=1 -timeout 60s
go test ./internal/session -count=1 -timeout 60s
go test ./internal/audit -count=1 -timeout 60s
go test ./internal/fleet -count=1 -timeout 60s
# ... other light packages
```

## Skip / watch

- `./internal/incident` collect-tests can timeout — not a blocker.
- `./internal/install` can OOM on low-memory hosts.
- Full suite in one process is not supported on ~2 GiB.

## Notes

- Worker infinite-loop fix is already in main (`NETDUCTOR_DISABLE_VPS_WORKER=1`).
- After a successful package batch, a green doctor + VERSION tag check is sufficient for most changes.
- For CI or larger hosts, the normal `go test ./...` remains fine.

## TG UX invariants

```bash
go test ./cmd/netductor-tg -count=1 -run 'TestMenu|TestWait|TestUsers|TestNoObvious'
```

See [REVIEW-SUITE §1.4](REVIEW-SUITE.md).
