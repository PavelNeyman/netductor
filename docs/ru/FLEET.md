# Флот: primary + secondary

| Нода | Роль |
|------|------|
| **primary** (abroad) | Control plane: users, API, бот, Blocky, edge, бэкапы, опционально Lampac |
| **secondary** (RU) | **Только вход VPN** (VLESS/Reality + тонкий agent) |

Зеркалирование сервисов / Lampac на RU / bot failover / hourly fleet sync — **убраны**.  
Пользователи VPN на secondary: `ApplyConfig` → `config_ver` → agent. Форс: `netductor relay sync`.

План: [../PLAN-SECONDARY-VPN-ONLY.md](../PLAN-SECONDARY-VPN-ONLY.md).
