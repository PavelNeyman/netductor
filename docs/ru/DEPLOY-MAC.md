# Деплой с Mac

**RU** · [EN](../DEPLOY-MAC.md)

## Подготовка
1. Установить op: `brew install netductor` (только **реальные SHA** в Formula) или бинарь с Release.
2. Ключ: `~/.ssh/netductor_primary` (ed25519); passphrase опционально в мастере.
3. После bootstrap на VPS — SSH **:52222**, password выключен.

## Primary
TUI/Web/CLI: host, password (первый раз), SNI Reality, domain base, LE email, add-ons (tg/lampac/git…) отдельно.

```bash
netductor deploy primary --host IP --password … --key ~/.ssh/netductor_primary \
  --sni api.vk.me --domain-base … 
```

## Secondary
Деплой **с Mac**, не «через primary»:

```bash
export NETDUCTOR_SSH_PORT=52222
netductor deploy secondary --host IP --password … --key ~/.ssh/netductor_primary \
  --primary IP --sni …
```

Тот же pubkey на secondary.

## Edge / OpenWrt / MikroTik
Мастера в TUI/Web → тот же backend DeployEdge / DeploySite.

## Day-2
API ноды localhost :8787 через SSH tunnel; TG bot на primary; agent plane :8789 mTLS.
