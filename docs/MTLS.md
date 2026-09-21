# mTLS agent plane

Port **:8789** (TLS 1.3, client cert required). Plain **:8788** only with `NETDUCTOR_PLAIN_AGENT=1`.

## Rotate + auto-push

```bash
netductor mtls rotate <node-id>
```

1. Primary issues **new** client cert; **old serial stays valid** for grace (`NETDUCTOR_MTLS_GRACE_HOURS`, default **24**).
2. Enqueues **`mtls_refresh`** on edge and/or secondary.
3. Agent downloads material API with **current** cert, writes PEMs, restarts.
4. Heartbeat `cert_serial` → **ConfirmRotate** → old serial **revoked**.
5. Grace expiry without confirm → force-revoke old.

## CA dual-trust rollover

```bash
netductor mtls rollover start
netductor mtls rollover issue <id>
netductor mtls rollover finish
netductor mtls rollover abort|status
```

## CLI language

`NETDUCTOR_LANG=ru|en` — mtls strings via `internal/cli18n`.
