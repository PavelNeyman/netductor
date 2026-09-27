**RU** · [EN](../DOMAIN.md)

# Домен и LE

Поддомены вида `p.nd.` / `s.nd.` / `i.nd.` (пример neyman.top).  
DNS grey cloud для origin LE. Redirect base `https://i…:8443`.  
CAA: letsencrypt.org для прямого certbot.  
Мастер op запрашивает домен; `domain set --le` пишет conf + certbot + redirect unit **без :80**.
