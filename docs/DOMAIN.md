# Domain & TLS (explicit hosts)

All public hostnames are **operator-supplied**. There is **no** invent of `p.` / `s.` / `i.` from a base domain.

## Required for production

| Role | Flag / field | Example |
|------|----------------|---------|
| CORE / primary | `--primary` / `domain_primary` | `p2.nd.example.com` |
| VPN entry | `--vpn` / `domain_vpn` | `s.nd.example.com` |
| Import / redirect | `--redirect` / `domain_redirect` | `https://i2.nd.example.com:8443` |
| Org label (optional) | `--base` / `domain_base` | `nd.example.com` → conf `DOMAIN=` only |
| LE email | `--le --email` / `le_email` | needed with primary (+ redirect host for dual cert) |

```bash
netductor domain set \
  --primary p2.nd.example.com \
  --vpn s.nd.example.com \
  --redirect https://i2.nd.example.com:8443 \
  --base nd.example.com \
  --le --email admin@example.com
```

Deploy (Mac):

```bash
netductor-op deploy primary … \
  --domain-primary p2.nd.example.com \
  --domain-vpn s.nd.example.com \
  --domain-redirect https://i2.nd.example.com:8443 \
  --domain-base nd.example.com \
  --le-email admin@example.com
```

## Ports

| Port | Service | Certificate |
|------|---------|-------------|
| **443** | sing-box Reality | Camouflage SNI — **not** LE |
| **8443** | `netductor-redirect` | **Let's Encrypt** (when enabled) |
| **80** | **off** after LE | Free for `certbot renew` |
| **52222** | SSH | key-only |
| **8789** | agent mTLS | internal CA |

## Let's Encrypt

- Pass **explicit** hostnames (`--domains` or via `domain set --primary` + host from `--redirect`).
- **`--base` alone does not issue certs** and does not invent names.
- Alternate names (`p2`/`i2` vs `p`/`i`) avoid LE duplicate-certificate rate limits on the same apex.

```bash
netductor tls le --email admin@example.com --domains p2.nd.example.com,i2.nd.example.com --agree-tos
```

## Cloudflare (optional)

Orange proxy only on the **redirect** hostname if you want HTTPS without `:8443`. Keep CORE and VPN **DNS-only** (grey) so Reality on 443 is not broken.
