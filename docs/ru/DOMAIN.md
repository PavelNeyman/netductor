# Домен

**RU** · [EN](../DOMAIN.md)

## Роли имён
| Имя | Назначение |
|-----|------------|
| base (например netductor.neyman.top / nd.neyman.top) | пресет |
| primary / p.… | CORE / public hostname |
| vpn / s.… | VPN entry (часто secondary) |
| i.… | redirect base для TG-кнопок |

Reality **SNI** (api.vk.me и т.п.) — **не** ваш домен.

## CLI
```bash
netductor domain set --base netductor.neyman.top --http
netductor domain set --base … --enable-redirect
netductor domain set --primary p.nd.neyman.top --vpn s.nd.neyman.top \
  --redirect https://i.nd.neyman.top:8443 --enable-redirect
netductor domain show
```

Пишет `netductor.conf`, `public_hostname`, `vpn_hostname`.

## DNS
Cloudflare/провайдер вручную. netductor **не** дергает CF API.  
Записи A на primary/secondary IP; CAA на letsencrypt.org при необходимости.

## TLS
```bash
netductor tls le --email you@mail --domains i.nd.neyman.top,p.nd.neyman.top
```
HTTP-01 standalone :80 (кратко). Redirect слушает **:8443** с LE.  
После recover LE часто нужно **заново** — certs не всегда в `.ndenc`.
