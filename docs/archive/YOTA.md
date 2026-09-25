# Yota (Megafon MVNO) and whitelist

Yota uses the same TSPU stack as Megafon.

## What works (2025–2026 field reports)

- **VLESS + Reality + Vision**, TCP **443**, destination IP in a **whitelist CIDR** (Yandex Cloud, some Timeweb/VK ranges), SNI of an allowed site (`vk.com`, `api.vk.me`, `ya.ru`).
- Random “cheap RU VPS” IP is often **not** in the mobile whitelist → connection fails on LTE while home Wi‑Fi works.
- WireGuard / plain OpenVPN: generally blocked under whitelist mode.

## Practical checklist

1. From phone **without VPN**: can you open TCP to `RELAY_IP:443`?
2. If no → IP not useful for Yota WL; try another hoster (Yandex Cloud preferred in community guides).
3. If yes but VPN fails → try SNI `api.vk.me` / `www.cloudflare.com` and fingerprint `firefox`.
4. Home Wi‑Fi: prefer **core** link (faster); mobile: **relay** only.

## netductor

- Primary client links → online relay.
- Extra **core** link for home speed.
- Relay does RU-direct split; rest uplinks to core.
