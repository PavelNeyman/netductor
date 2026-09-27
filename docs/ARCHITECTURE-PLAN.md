# Plan until architecture is feature-stable

## A — Close before freeze is “done in practice”

| ID | Item | Status |
|----|------|--------|
| A2 | Deploy parity TUI ↔ Web (one fleet checklist) | **[x]** [DEPLOY-PARITY.md](DEPLOY-PARITY.md) |
| A6 | Unified update path (node + agent + op from Release) | **[x]** [UPDATE.md](UPDATE.md), `netductor update` |
| A5 | Doctor: footgun env gone; cert expiry WARN | **[x]** 0.9.68 (footgun + cert ≤30d) |
| Recover drill | Backup → wipe primary → recover from secondary | **[x] doc**; live = owner |

## B — Missing for calm “features only”

| ID | Item | Status |
|----|------|--------|
| B1 | Version pins + self-update agent/op | **[x] agent** desired_release; Mac op/brew — rest |
| B2 | Idempotent install/recover documented + tested | open |
| B3 | Desired state only via agent plane | **[x] posture** [B3-B4.md](B3-B4.md) |
| B4 | Operator secrets stay on Mac | **[x] posture** [B3-B4.md](B3-B4.md) |
| B5 | UI↔API matrix (opcatalog) | **[x]** [OPCATALOG.md](OPCATALOG.md) |

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

## Cross-cutting

- i18n EN+RU: [I18N.md](I18N.md)
