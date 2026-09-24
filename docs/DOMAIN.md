# Domain hostnames (DNS external)

DNS (Cloudflare etc.) is configured **outside** netductor. This only maps names into conf and VPN advertise hosts.

```bash
netductor domain set --base netductor.neyman.top --le --email you@example.com
netductor domain show
```

| Flag | Meaning |
|------|---------|
| `--base` | Preset: `primary.` / `vpn.` / `i.` |
| `--primary` / `--vpn` / `--redirect` | Explicit hosts/URL |
| `--le --email` | certbot HTTP-01 for `primary.` + `i.`; then HTTPS redirect |
| `--cf-proxy` | `REDIRECT_BASE=https://i.<base>` (no port); CF orange on **only** `i.` |
| `--http` | HTTP redirect only (no LE) |
| `--enable-redirect` | enable unit |

## Ports (locked)

| Port | Service | Certificate |
|------|---------|-------------|
| **443** | sing-box Reality | Camouflage (e.g. VK) — **not** LE |
| **8443** | `netductor-redirect` | **Let's Encrypt** |
| **80** | **off** after LE | Free for `certbot renew` standalone |
| **52222** | SSH (primary) | key-only |
| **8789** | agent mTLS | internal CA |

`REDIRECT_BASE` after LE = `https://i.<base>:8443`.

**Optional cleaner URL:** Cloudflare orange-cloud **only** for `i.` → CF terminates HTTPS on 443; origin can use :8443 or HTTP. VPN/Reality names stay DNS-only (grey).

## Let's Encrypt

```bash
netductor domain set --base netductor.neyman.top --le --email admin@example.com
# or
netductor tls le --email admin@example.com --base netductor.neyman.top
```

Uses **certbot standalone** (needs free :80 briefly). Opens ufw 80/443/8443. After success, redirect listens **only on :8443**. Renew: certbot timer + deploy-hook restarts redirect.

Reality SNI is separate (`vpn set-sni` / primary wizard).


## Cloudflare orange cloud for `i.` — optional

**Not required** if clients use `https://i.<base>:8443` (LE on primary).

| Setup | Result |
|-------|--------|
| DNS only (grey) for `i.` | Browser/TG must use **:8443** for valid LE cert. Port 443 = Reality (wrong cert). |
| Orange proxy **only** for `i.` | CF terminates HTTPS on 443 with CF cert; origin can be `:8443` or HTTP. URL can be `https://i.<base>` without port. |
| Orange on `primary.` / VPN hosts | **Avoid** — breaks Reality fingerprint on 443. |

Do orange **only** for the import hostname (`i.`), keep `primary.` and `vpn.` DNS-only.


## Cloudflare setup (when using --cf-proxy)

1. DNS: `i.<base>` **Proxied** (orange). `primary.` / `vpn.` **DNS only** (grey).
2. SSL/TLS mode: **Full** (or Full strict).
3. **Origin Rule** (required): hostname equals `i.<base>` → destination port **8443**  
   (default CF→origin:443 would hit Reality, not LE redirect).
4. `netductor domain set --base <base> --le --email … --cf-proxy`

Without Origin Rule, leave grey cloud and use `https://i.:8443`.


## Cloudflare and multi-level names

`i.netductor.neyman.top` is **two** labels under `neyman.top`. Free Universal SSL covers `*.neyman.top` only, **not** `*.netductor.neyman.top`.

**Default / recommended:** DNS **only** (grey) for `i.` / `primary.` / `vpn.` and:

`REDIRECT_BASE=https://i.<base>:8443`

`--cf-proxy` only if the import host is a **single** level under the zone apex (e.g. `i.neyman.top`) or you have Advanced Certificate Manager.
