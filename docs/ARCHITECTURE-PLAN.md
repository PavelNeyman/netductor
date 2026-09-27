# Plan until architecture is feature-stable

## A — Close before freeze is “done in practice”

| ID | Item | Why it hurts if skipped |
|----|------|-------------------------|
| A2 | Deploy parity TUI ↔ Web (one fleet checklist) | Mac reinstall diverges |
| A6 | Unified update path (node + agent + op from Release) | Manual scp forever |
| A5 | Doctor: footgun env gone; cert expiry WARN | Silent rot |
| Recover drill | Backup → wipe primary → recover from secondary | Fear of failure |

## B — Missing for calm “features only”

| ID | Item | Notes |
|----|------|-------|
| B1 | Version pins + self-update agent/op | Heartbeat / brew |
| B2 | Idempotent install/recover documented + tested | RUNBOOK one path |
| B3 | Desired state only via agent plane | No primary→secondary SSH |
| B4 | Operator secrets stay on Mac | Never put private key on primary for secondary |
| B5 | UI↔API matrix (opcatalog) | Avoid API-only features |

## C — Explicitly not architecture (later features)

- Full HA bot/API on secondary
- Users always on WSS (failover only)
- Mandatory self-hosted git/registry
- SMTP when mailbox exists
- Hardware e2e OpenWrt/Tapo

## D — Security hardening done in freeze pass

- Env footguns removed/refused in code
- Redirect no :80; :8443 TLS only
- No VPS admin panel product path
