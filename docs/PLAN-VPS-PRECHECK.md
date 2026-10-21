# VPS pre-check (candidate secondary / primary)

**Status:** implemented **0.9.465** — `internal/vpscheck`, job `vps.precheck`, CLI/Web/TUI (MVP scorecard, not full YABS clone).

## Intent

Before buying or switching a secondary (or primary) VPS, run a **read-only** battery:

| Check | Goal |
|--|--|
| IP reputation / cleanliness | spamhaus / abuse / open proxy signals |
| Bandwidth / latency | to primary and to major CDNs |
| CPU / disk micro-bench | hoster oversubscription |
| Network path | packet loss, MTU, DPI symptoms |

Older shell scripts (if you still have them) were the source; they were **not** fully ported into `netductor` as a first-class command.

## What exists today

| Tool | Use |
|--|--|
| `netductor host-audit` / `host-audit --purge` | **Day-2 on an already installed node**: hoster agents, residual packages, listen surface |
| `netductor doctor` | stack health after install |
| `netductor channel status` | secondary uplink after join |
| Deploy harden | purges hoster agents **during** install |

## Gap

There is **no** `netductor-op vps-check --host …` that runs the full pre-purchase suite from Mac against a **fresh** VPS (password SSH) and prints a scorecard.

If you still have the original scripts — attach them; we can map 1:1 into `internal/vpscheck` + job `vps.precheck` + TUI/Web.

## Host-audit as day-2

```bash
# on the node (primary or secondary), as root
netductor host-audit           # report
netductor host-audit --dry-run # list purge candidates
netductor host-audit --purge   # remove hoster residual

# offline bundle
bash scripts/run-host-audit-bundle.sh
```

See [HOST-AUDIT.md](HOST-AUDIT.md). Not yet a full TG/Web card parity item (CLI + scripts are the path).
