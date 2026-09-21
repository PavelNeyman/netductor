# mTLS (RU)

Полная актуальная версия: [EN](../MTLS.md)

# mTLS agent plane

Port **:8789** (TLS 1.3, client cert required). Plain **:8788** only with `NETDUCTOR_PLAIN_AGENT=1`.

## CLI

```bash
netductor mtls ensure
netductor mtls list
netductor mtls issue-client <node-id>
netductor mtls rotate <node-id>    # new cert + revoke old serial
netductor mtls revoke <serial|node-id>
netductor mtls revoked
```

After **rotate**, copy `secrets/mtls/clients/<id>/` to the device (re-run edge/secondary provision) — old serial is refused immediately.

## Doctor

Warns when CA/server/client or per-node certs expire within **30 days**; flags revoked clients still on disk.

## API (session)

- `GET /api/mtls/certs`
- `POST /api/mtls/revoke` `{"node_id"}` or `{"serial"}`
- `POST /api/mtls/rotate` `{"node_id"}`

## Rate limit

`:8789` — 180 req/min/IP; after repeated 429s the IP is **banned** (~15 min). Env: `NETDUCTOR_PLANE_BAN_AFTER`, `NETDUCTOR_PLANE_BAN_TTL_MIN`.
