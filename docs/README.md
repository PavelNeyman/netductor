# Documentation

**EN** canonical · **RU** same meaning: [ru/](ru/) (every product doc has a pair).

## Must-read
| EN | RU |
|----|-----|
| [AGENT_HANDOFF.md](AGENT_HANDOFF.md) | [ru/AGENT_HANDOFF.md](ru/AGENT_HANDOFF.md) |

| [REVIEW-SUITE.md](REVIEW-SUITE.md) | [ru/REVIEW-SUITE.md](ru/REVIEW-SUITE.md) | Full review suite (tests, UI, i18n, docs) |
| [REVIEW-PLAN-2026-10-19.md](REVIEW-PLAN-2026-10-19.md) | [ru/REVIEW-PLAN-2026-10-19.md](ru/REVIEW-PLAN-2026-10-19.md) | Review run plan with chunk checkboxes and follow-ups F1–F10 |
| [REVIEW-REPORT-2026-10-19.md](REVIEW-REPORT-2026-10-19.md) | [ru/REVIEW-REPORT-2026-10-19.md](ru/REVIEW-REPORT-2026-10-19.md) | Review findings: security, quality, dead code, refactoring |
| [UPDATE-SYSTEM-IMPLEMENTATION-PLAN.md](UPDATE-SYSTEM-IMPLEMENTATION-PLAN.md) | [ru/UPDATE-SYSTEM-IMPLEMENTATION-PLAN.md](ru/UPDATE-SYSTEM-IMPLEMENTATION-PLAN.md) | Unified component update plan with progress checklist |
- [Update contract (Phase 1)](UPDATE-CONTRACT.md) · [RU](ru/UPDATE-CONTRACT.md)
| [AUDIT-ARTIFACTS.md](AUDIT-ARTIFACTS.md) | [ru/AUDIT-ARTIFACTS.md](ru/AUDIT-ARTIFACTS.md) | Audit evidence, secret handling, and status rules |
| [ARCHITECTURE-FREEZE.md](ARCHITECTURE-FREEZE.md) | [ru/](ru/ARCHITECTURE-FREEZE.md) |
| [OPEN_ITEMS.md](OPEN_ITEMS.md) | [ru/](ru/OPEN_ITEMS.md) |
| [RECOVER-DRILL.md](RECOVER-DRILL.md) | [ru/](ru/RECOVER-DRILL.md) |
| [RUNBOOK-INSTALL-RECOVER.md](RUNBOOK-INSTALL-RECOVER.md) | [ru/](ru/RUNBOOK-INSTALL-RECOVER.md) |
| [DEPLOY-MAC.md](DEPLOY-MAC.md) | [ru/](ru/DEPLOY-MAC.md) |
| [SECONDARY-DEPLOY.md](SECONDARY-DEPLOY.md) | [ru/SECONDARY-DEPLOY.md](ru/SECONDARY-DEPLOY.md) | Secondary trust, end-state, R22 failover checklist |
| [BREW.md](BREW.md) | [ru/](ru/BREW.md) |

## Product / plans
Full set under `docs/` and `docs/ru/` — architecture, ports, security, backup, mTLS, VPN, edge, MikroTik, git, UI rules, plans (NVR, rooms, MT API).

**Bilingualism (owner rule — mandatory):**

- EN under `docs/`, RU under `docs/ru/` — **full semantic parity**, not a summary.
- Not required to be word-for-word; required to carry **the same meaning completely** (all sections, steps, warnings, parameters).
- **Never** treat RU as “condensed” or “shorter by design”. If EN has detail, RU must have that detail in meaning.
- Updating only one language = unfinished work.

## Security & plans

- [HOST-AUDIT.md](HOST-AUDIT.md) — hoster / listen audit
- [PLAN-SERVICE-ACCESS-POLICY.md](PLAN-SERVICE-ACCESS-POLICY.md) — service access policies

## Audit archive

- [Audit artifact policy](AUDIT-ARTIFACTS.md)
- [Reusable audit template](audits/TEMPLATE.md)
- Store each substantive audit as a dated report under `docs/audits/` and its equivalent under `docs/ru/audits/`; link reports from the relevant plan and issue/PR.
