**RU** · [EN](../RECOVER-DRILL.md)

# Recover drill

## Статус
Unattended recover **пройден** 2026-09-27 / **v0.9.71+**; Reality/uplink checklist **v0.9.73**.

## Цель
Снести primary → поднять с secondary backup → стек primary + VPN path без ручного «докручивания».

## Фазы лога
| Фаза | Что |
|------|-----|
| Pre-restore install | COMPONENTS; `vpn-users` может soft-fail (нет secrets) → **continuing** |
| Tar restore | secrets, users, state |
| Post-restore pass | api / telegram / backup / vpn-users снова |
| ensure-relay-uplink + vpn apply | conf из secrets, UUID uplink в inbound |
| UFW | secondary IP → api-allow |

Ожидаемый хвост: `(continuing)` → `post-restore component pass` → `vpn apply OK` → `recover: done`.

## Чеклист Reality / uplink (обязательно)

После wipe+recover **не считать VPN рабочим**, пока:

1. `netductor vpn apply` **успешен**
2. Reality inbound primary = `/etc/netductor/secrets/singbox_*` (private, sid, sni)
3. Secondary uplink pbk/sid/SNI = public secrets primary
4. **Multiplex ON**: inbound primary только `enabled`+`padding`; outbound secondary — полный object
5. Клиентские ссылки — актуальный secondary pbk/sid из `devices.json`

**Инцидент 2026-09-27:** conf≠secrets, users=[], apply падал на `max_connections` в inbound mux → unknown UUID / x509. Фикс 0.9.73.

## SSH
Day-2: `-p 52222` + Mac key. Recovery arm/disarm на secondary.

## Secondary после drill
Disarm recovery; :8790 закрыт.


## LE после recover (0.9.74+)

Сертификаты **не** в `.ndenc`. Если в conf есть `DOMAIN` + `LE_EMAIL`, recover сам вызывает `domain`/`tls le` и `InstallRedirect`. Иначе один раз: `netductor domain set --base … --le --email …`.

Post-restore также **sanitize conf** — удаляет footgun-ключи (PLAIN_AGENT, ALLOW_PUBLIC, …).
