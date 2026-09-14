# Открытые темы

## Сделано (пакет ops)
- [x] Алерты: probe, service down, relay offline, mismatch spike, backup offsite fail
- [x] `vpn refresh-links`
- [x] Audit / sessions / doctor / apply --dry-run / runbook
- [x] `vpn sni-import`, multi-relay helper URIs
- [x] **Subscription убран** из продукта (только отдельные ссылки VLESS primary / core / HY2)

## Future (не в работе, только зафиксировано)

### User Telegram bot — path B (отдельный бот)
Отдельный bot token + binary (`netductor-tg-user`), **без** admin-handlers.

- Оператор в admin-боте выдаёт one-time bind / deep-link → `telegram_id` ↔ VPN user в registry.
- User-бот: только «мой VLESS + QR» и (опционально) своя статистика трафика.
- Нельзя: список users, nodes, enroll, session, audit, чужие данные.
- Зачем отдельный бот: компромисс user-токена не открывает флот; на каждый update роль не путается с admin.

**Статус:** идея на полку. Самообслуживание ссылок пользователям может не понадобиться — решение отложить.

## Нужно железо / внешние условия
- [ ] Site wizard e2e (MikroTik + RPi OpenWrt)
- [ ] Реальный L3-WL entry (RU IP из белого списка / подходящий SNI path)
- [ ] HTTPS / Mini App — только если появится домен и явное желание

## Имеет смысл добить без железа (кандидаты)
- [ ] Паритет TG ↔ Admin по всем действиям над user/node (кнопки без дублей, единые карточки)
- [ ] Трафик per-user из sing-box stats → registry (даже без user-бота полезно оператору)
- [ ] Сгладить UX Access (один primary по умолчанию, core/HY2 как «ещё…»)
- [ ] Cross-backup peer: проверить e2e restore на чистой VPS
- [ ] WAN-fallback / policy на edge-агенте — проверка на реальном OpenWrt
- [ ] Документация: CLIENT / SHADOWROCKET без subscription; day-1 runbook актуализировать
