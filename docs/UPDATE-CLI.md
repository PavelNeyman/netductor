# Update CLI (Phase 8)

Node binary (`netductor` on VPS). Same engine as TG/Web for host components.

## Commands

```bash
netductor update check|status [--refresh]   # local vs latest
netductor update list [--limit N]           # GH releases
netductor update jobs                       # in-process job registry snapshot
netductor update rollback --component tg|agent  # restore binary from .prev (+ unit restart)
netductor update [apply] [vX.Y.Z] \
  [--component node|tg|agent] \
  [--no-restart] [--skip-verify] [--no-backup]
netductor update github-token status|set <tok>|clear
```

## Components

| ID | Binary / unit | Notes |
|----|---------------|--------|
| `node` | `/usr/local/bin/netductor`, stack apply | Primary path; uses `stack.ApplyOpts` |
| `tg` | `netductor-tg` + telegram unit | `ApplyHostComponent` |
| `agent` | `netductor-agent` + **secondary-agent** unit | **Not** OpenWrt edge |
| edge | — | `agent_update` / edge control |
| `op` | Mac | `netductor-op update` / Homebrew |

## Engine

- Integrity: GH release asset + SHA256SUMS (unless skip-verify confirmed).
- Jobs: `internal/update` registry; `jobs` shows local snapshot (process memory).
- Health: post-restart probe for host apply; desired ≠ running until health succeeds.

See [UPDATE-CONTRACT.md](UPDATE-CONTRACT.md), [UPDATE-RECOVERY.md](UPDATE-RECOVERY.md), [SSH-HOST-TRUST.md](SSH-HOST-TRUST.md) for SSH before remote apply.
