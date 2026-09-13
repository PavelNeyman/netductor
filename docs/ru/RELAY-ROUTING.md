# Маршрутизация на relay (RU / non-RU)

## Политика
| Трафик | Куда |
|--------|------|
| Домены РФ (.ru, .su, .рф, крупные сервисы) | **direct** с IP relay (РФ) |
| Private IP | **direct** |
| Всё остальное с `relay-in` (клиенты) | **uplink** → core (зарубежный выход) |
| `exit-in` :4443 (если RU-exit включён) | **direct** (для оператора за границей) |

## DNS
- RU-суффиксы → 77.88.8.8
- Остальное → 9.9.9.9 (final)

## Клиенты
- Мобильный / БС: VLESS на **relay**
- Дома (скорость): опционально VLESS на **core**
- OpenWrt: primary relay; при fail TCP:443 → WAN (ISP) без VPN

ГеоIP-база sing-box rule-set можно добавить позже; сейчас явный domain_suffix список в `WriteRelaySingBox`.
