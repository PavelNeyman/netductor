**RU** · [EN](../OPEN_ITEMS.md)

# Открытые пункты

План: [ARCHITECTURE-PLAN](ARCHITECTURE-PLAN.md) · freeze: [ARCHITECTURE-FREEZE](ARCHITECTURE-FREEZE.md)

## Планы (документы)

- [ ] [PLAN-SERVICE-ACCESS-POLICY](PLAN-SERVICE-ACCESS-POLICY.md) — каталог сервисов + политики VPN-user и edge (галочки во всех UI); enforcement sing-box
- [ ] [HOST-AUDIT](HOST-AUDIT.md) — полный аудит хостера; `collect-host-audit.sh`; baseline; doctor FAIL (частично 0.9.192–194)
- [ ] Расширить denylist host-audit (RMM/otel/avahi) — синхрон script ↔ Go
- [ ] Статус firewall во всех UI + алерты
- [ ] Baseline snapshot в конце install


## Закрыто


## Закрыто недавно (код 0.9.164–172)

- [x] Edge template VPN UI + policy hardening
- [x] Удалена legacy VPS admin UI
- [x] Unit-тесты + уникальные ID opcatalog
- [ ] Force-update живых нод до **≥0.9.181** при отставании
- [ ] Dual-node smoke после обновления
- [x] Freeze архитектуры
- [x] Recover 0.9.70–0.9.75
- [x] Live recover drill
- [x] Доки EN/RU (42)

## У владельца

- [ ] Force-update VPS до **0.9.181** (если ещё 111–115) — [RUNBOOK-FORCE-UPDATE.md](RUNBOOK-FORCE-UPDATE.md)
- [ ] Smoke после обновления

- [x] **Показать пароль** (Web + TUI Ctrl+P, 0.9.103) в Web/TUI (одноразовые пароли деплоя) — [WEB-UI-FIXES.md](WEB-UI-FIXES.md). Только план.
- [x] Web Updates; полное упрощение Day-2 отложено — много вкладок/кнопок; [WEB-UI-FIXES.md](WEB-UI-FIXES.md) § Day-2. Только план.
- [x] Dual-node smoke
- [ ] Hardware e2e
- [ ] CI SHA Formula (опционально)

Ревью: [REVIEW-0.9.75](REVIEW-0.9.75.md).

## Web UI / Fleet (код отложен)

См. [WEB-UI-FIXES.md](WEB-UI-FIXES.md) — подписи, галочка Lampac, CF proxy, TOFU secondary на Mac. **Не реализовывать, пока не приоритет.**
- ~~OpenWrt opkg feed / ipk for agent~~ — **rejected** (0.9.155): first-boot binary+SCP; day-2 stack/agent_update. Revisit only if fleet needs offline opkg without primary.
