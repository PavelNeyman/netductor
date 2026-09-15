# Роли fleet (core + relay)

## Решение

| Задача | Нода | Почему |
|--------|------|--------|
| **Control-plane primary** (users, policies, TG-бот оператора, админка, авторитет бэкапов) | **Зарубежный core** | Секреты безопаснее; API Telegram стабильнее; уже источник правды |
| **Вход VPN (data plane)** | **RU relay** | Белые списки / мобильный интернет |
| **Lampac (по умолчанию)** | **RU** (prefer) | Меньше задержка; тёплая реплика через sync |
| **TG-бот** | **За рубежом** (active), RU = standby позже | Один токен; только active/passive |

VPN **не** балансируем. Размещение сервисов и синк данных — отдельно.

## Команды

```bash
netductor fleet bootstrap
netductor fleet status
netductor fleet set-primary <id>
netductor fleet set-secondary <id>
netductor fleet set-service lampac <id>
netductor fleet set-service bot <id>
netductor fleet sync
```

## Sync

Копирует `/opt/netductor/lampac`, `components.json`, policy `fleet/`. Хост peer — из `backup peer-set`, если не указан.
