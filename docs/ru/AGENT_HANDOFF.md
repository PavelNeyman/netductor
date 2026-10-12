# AGENT Handoff (RU)

**Стоп-линия: v0.9.340** (2026-10-12). Комплекс ревью: [REVIEW-SUITE](REVIEW-SUITE.md). Новый чат: AGENTS.md → [EN AGENT_HANDOFF](../AGENT_HANDOFF.md) → OPEN_ITEMS.

### Сводка 0.9.320–0.9.331

- Metrics UI, recovery-алерты EN/RU, таблицы Channel
- unattended-upgrades, integrity (+SSH), backup age, SERVFAIL
- Reboot: TG + **API/Web/opcatalog** (код 3 мин, one-shot)
- Blocky DoT на primary
- SSH allowlist: TG (`m:sshkeys:`) / API / agent / Web / opcatalog

Apply:

```bash
netductor stack apply v0.9.339
netductor secondary cmd <id> upgrade:v0.9.339
```

### Открыто (не код)

- dual-node smoke (оператор)
- железо e2e: OpenWrt / Tapo / MikroTik
- lab checklist

Полная EN-история и детали: [AGENT_HANDOFF.md](../AGENT_HANDOFF.md).
