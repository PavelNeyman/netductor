# Клиентские профили VLESS+Reality

## flow mismatch

На сервере **только** `xtls-rprx-vision`. Сообщение `flow mismatch: got none` значит: до VLESS дошли (UUID/Reality ок), но **в этом TCP-сеансе** flow пустой.

### Это не всегда «чужой»

На relay часто один IP (ваш ISP) даёт:
- успешные сессии (например `gateway.icloud.com` → uplink) — Vision ок;
- параллельно пачки ERROR flow mismatch с того же IP.

Типично:
1. **Несколько клиентов/устройств** за одним NAT (телефон + ПК + старый профиль).
2. Один клиент: **основной туннель с Vision** + фоновые/битые попытки без flow (второй профиль, «тест», устаревший URL в другом приложении).
3. Реже — сканеры 443 (обычно другой IP и без UUID).

Проверка: на 1–2 минуты выключить VPN на всех устройствах — если mismatch пропал, это ваши клиенты. Оставить один Happ с одной ссылкой.

## Ссылка
`...&flow=xtls-rprx-vision&security=reality&...`

## Клиенты
Happ (LTE), Shadowrocket, INCY, sing-box — везде явно Vision, не XTLS-none.
