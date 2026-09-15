# Роли fleet (core + relay)

## Решение

| Задача | Нода | Почему |
|--------|------|--------|
| **Control-plane primary** | **Зарубежный core** | Users, policies, TG-бот, админка, бэкапы |
| **Вход VPN** | **RU relay** | Белые списки / LTE |
| **Lampac** | **RU** | Задержка; данные синкаются раз в час |
| **TG-бот** | **Abroad active**; RU **standby** через SOCKS→core | В РФ Telegram часто недоступен; Bot API нужен выход с core |

## TG failover (без MTProxy)

Bot API = HTTPS на `api.telegram.org`. MTProxy для этого не нужен.

На **secondary (RU)**:
1. `ssh -D 127.0.0.1:1089` на primary → SOCKS с IP **core**
2. Standby: `ALL_PROXY=socks5://127.0.0.1:1089`
3. Таймер 2 мин: primary мёртв → promote standby; primary жив → demote

```bash
# на primary
netductor fleet sync-timer
netductor fleet apply-lampac

# на secondary
netductor fleet bot-standby-install root@CORE_IP
netductor fleet bot-failover timer
```
