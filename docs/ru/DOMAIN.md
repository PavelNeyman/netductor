# Домен

**RU** · [EN](../DOMAIN.md)

- `netductor domain set --base …` → primary/vpn/redirect hostnames
- Reality **SNI** (например api.vk.me) ≠ ваш домен
- Redirect: `https://i…:8443`, Cloudflare DNS вручную (без CF API в netductor)
- LE — по возможности автоматически при деплое/настройке

DNS у провайдера/CF настраивает оператор.
