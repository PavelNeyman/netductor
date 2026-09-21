# Развёртывание netductor

Полная схема двух нод: [FLEET.md](FLEET.md) · [AGENT_HANDOFF.md](AGENT_HANDOFF.md)

## Primary

```bash
wget -qO /usr/local/bin/netductor \
  https://github.com/PavelNeyman/netductor/releases/download/v0.8.22/netductor-linux-amd64
chmod 755 /usr/local/bin/netductor
# secrets telegram_* в /etc/netductor/secrets/
netductor install
netductor vpn set-sni api.vk.me
netductor doctor
```

## Secondary (с primary)

```bash
netductor fleet provision-secondary --host RU_IP --password '…' --sni api.vk.me
```

## Recover

```bash
netductor recover --key "$KEY" backup.ndenc
```

Смотри EN [DEPLOY.md](../DEPLOY.md) для admin, backup peer, update, uninstall.
