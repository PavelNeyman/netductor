# Shadowrocket и netductor

## Вывод по логам (Happ vs SR)
При **Happ** mismatch по flow пропал (0 за 20 мин при сотнях uplink).  
При смешанном/внешнем конфиге Shadowrocket с того же IP шли пачки `flow mismatch`.

## Рекомендация
1. Импортировать **только** URI из бота / `shadowrocket-uris.txt` (там всегда `flow=xtls-rprx-vision`).
2. Не подмешивать внешние JSON с balancer/loopback на тот же server:443.
3. RU/non-RU уже на **relay** — локальный split в SR для нашего узла не обязателен.

## Генерация на сервере
```bash
netductor vpn client-config operator
# → /var/lib/netductor/vpn/clients/operator/
#    shadowrocket-uris.txt
#    shadowrocket-helper.json
#    sing-box-client.json
```

## Если нужен split в SR
Держите **один** VLESS outbound на relay с Vision; правила RU → DIRECT, остальное → этот proxy.  
Не создавайте второй VLESS на тот же host без flow.
