# Admin UI (legacy on VPS)

> **Not the product path.** Human UI is **netductor-op on Mac** — see [PLAN-MAC-CLIENT.md](PLAN-MAC-CLIENT.md).

## Default (0.9.4+)

- Node **`serve`** is **API-only**.
- `GET /admin` returns **410** with a JSON pointer to Mac client.
- Install does **not** copy `runtime/api/admin` unless `NETDUCTOR_LEGACY_ADMIN_UI=1`.

## Emergency legacy UI

```bash
export NETDUCTOR_LEGACY_ADMIN_UI=1
# reinstall unit / restart: netductor install … or systemctl restart netductor-api
ssh -L 8787:127.0.0.1:8787 primary
# open http://127.0.0.1:8787/admin/
```

Session: `netductor vpn session 72` or Telegram.

Prefer Mac Control: tunnel + `netductor-op session issue` + WebUI Control tab.
