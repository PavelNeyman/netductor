# AGENT Handoff (RU)

**Стоп-линия: v0.9.347** (2026-10-12). Комплекс ревью: [REVIEW-SUITE](REVIEW-SUITE.md). Новый чат: AGENTS.md → [EN AGENT_HANDOFF](../AGENT_HANDOFF.md) → OPEN_ITEMS.

### Сводка 0.9.320–0.9.347

- Metrics UI, recovery-алерты EN/RU, таблицы Channel
- unattended-upgrades, integrity (+SSH), backup age, SERVFAIL
- Reboot: TG + **API/Web/opcatalog** (код 3 мин, one-shot)
- Blocky DoT на primary; SSH allowlist (`m:sshkeys:`)
- **0.9.341–342:** Action required на Status; `stack preflight` + SHA; VPN kind user/device; auto recover-drill mark; backup-verify age в Attention
- SR family/operator уже в TG (`btn_sr_config` / `btn_sr_family`)

Apply (после публикации assets):

```bash
netductor stack preflight v0.9.347
netductor stack apply v0.9.347
netductor secondary cmd <id> upgrade:v0.9.347
```

### Открыто (не код-блокеры)

- dual-node smoke (оператор)
- железо e2e: OpenWrt / Tapo / MikroTik
- **later:** read-only guest session; OIDC ([OIDC-DESIGN](../OIDC-DESIGN.md))

Полная EN-история: [AGENT_HANDOFF.md](../AGENT_HANDOFF.md).
