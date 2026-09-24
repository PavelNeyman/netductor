# Domain hostnames (DNS external)

DNS (Cloudflare etc.) is configured **outside** netductor. This only maps names into conf and VPN advertise hosts.

```bash
netductor domain set --base netductor.neyman.top --http
# → primary.netductor.neyman.top, vpn.netductor.neyman.top, http://i.netductor.neyman.top

netductor domain set --base netductor.neyman.top --enable-redirect
netductor domain show
```

| Flag | Meaning |
|------|---------|
| `--base` | Preset suffix |
| `--primary` | Core host in links |
| `--vpn` | VPN entry host |
| `--redirect` | Full REDIRECT_BASE |
| `--http` | `http://i.<base>` instead of https |
| `--enable-redirect` | enable unit if installed |

Reality SNI is separate (`vpn set-sni`).

## Let's Encrypt

```bash
# after DNS A records for primary.<base> and i.<base> point at this VPS:
netductor domain set --base netductor.neyman.top --le --email admin@example.com

# or only certs:
netductor tls le --email admin@example.com --base netductor.neyman.top
netductor tls show
```

Uses **certbot standalone** (needs :80 free briefly). Opens ufw 80/443. Writes certs, sets `REDIRECT_BASE=https://i.<base>`, restarts redirect on :80+:443.

Renewal: certbot timer (distro default); after renew restart `netductor-redirect`.


Redirect HTTPS listens on **:8443** (port 443 is Reality/sing-box). HTTP :80 for ACME/CF flexible.
