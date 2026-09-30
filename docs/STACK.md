# Stack orchestrator (simplified)

## Rules

1. **One writer** for primary `netductor` + `netductor-tg`: `netductor stack apply [vX]`.
2. API / TG only **schedule** apply (`stack schedule` → systemd-run). Never apply inside api/bot process.
3. **`prev/`** = snapshot after **successful** apply only. Manual `stack rollback` target.
4. **No `attempt/`**, no auto-rollback, no auto-promote on status view or watchdog.
5. **Watchdog** = restart failed units only.
6. **`stack promote`** = explicit; refuses dirty prev (VERSION ≠ binary).
7. **`stack pin`** = freeze apply/promote while stabilizing.

## Operator flow

```bash
netductor stack status
netductor stack apply v0.9.138    # or: stack schedule v0.9.138
netductor stack pin "stabilize"
# … wait …
netductor stack unpin
netductor stack watchdog-install  # optional
```

Force binary install without stack logic: see `docs/RECOVER-BOT.md`.
