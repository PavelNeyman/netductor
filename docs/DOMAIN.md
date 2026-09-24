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
