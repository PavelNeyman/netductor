# Маршрутизация secondary (RU / non-RU)

| Трафик | Куда |
|--------|------|
| Private IP | **direct** |
| domain_suffix (.ru, .su, .рф, крупные сервисы) | **direct** |
| **geoip-ru** (rule-set) | **direct** |
| Остальное с relay-in | **uplink** → primary |

Скачивание geoip: через uplink, если GitHub с RU недоступен.
