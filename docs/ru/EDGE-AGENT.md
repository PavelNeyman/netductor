**RU** · [EN](../EDGE-AGENT.md)

# Edge agent (OpenWrt)

## Назначение
Агент на OpenWrt (Cudy и др.): enroll к primary, применение шаблонов, heartbeat, guest Wi‑Fi, NVR helpers, remote commands.

## Жизненный цикл
1. Install с Mac (SSH на чистый роутер) — бинарь agent
2. Agent пишет local config, поднимает сервисы сайта
3. Enroll на `https://PRIMARY:8789` (mTLS после выдачи cert)
4. Пока нет интернета — backoff enroll (не ограничение «5 минут навсегда»)
5. Pending на primary → оператор approve/reject
6. После approve — pull template/config_ver, применение

## Идентичность
Device id, token, client cert. Re-bind: recovery code / страница в LAN без полного re-flash.

## Шаблоны
Серверные templates (не backup другого железа). Agent не должен затирать уникальный site state вслепую.

## Guest Wi‑Fi
Отдельный SSID (не band-split по умолчанию); isolation; WAN direct; grant PIN+QR; TTL (default 10m, max 24h).

## NVR / камеры
Leases list, static DHCP, onboard Tapo — [PLAN-NVR-TAPO](PLAN-NVR-TAPO.md).

## Обновления
UI показывает доступную версию agent; update per-device или everywhere; self-replace binary + restart unit.

## Безопасность
mTLS; rate-limit enroll; revoke/rotate certs — ops. Нет требования VPN для enroll (agent за NAT).

## Команды / API
Полные пути HTTP и поля JSON — в EN; CLI `netductor edge …` на primary.
