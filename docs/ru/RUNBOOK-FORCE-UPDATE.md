**RU** · [EN](../RUNBOOK-FORCE-UPDATE.md)

# Принудительное обновление бинарников (сломанный релиз)

Если нода на **0.9.111** (или update/бот не поднимаются) — подменить бинарники с GitHub, **без** wipe.

Цель: **v0.9.121**. SSH **52222**, ключ `~/.ssh/netductor`.

## Primary

```bash
VER=0.9.121
ssh -i ~/.ssh/netductor -p 52222 root@PRIMARY_IP bash -s <<EOF
set -euo pipefail
VER=$VER
cd /tmp
curl -fsSL -o netductor "https://github.com/PavelNeyman/netductor/releases/download/v\${VER}/netductor-linux-amd64"
curl -fsSL -o netductor-tg "https://github.com/PavelNeyman/netductor/releases/download/v\${VER}/netductor-tg-linux-amd64"
chmod 755 netductor netductor-tg
systemctl stop netductor-telegram-bot netductor-api 2>/dev/null || true
install -m 755 netductor /usr/local/bin/netductor
install -m 755 netductor-tg /usr/local/bin/netductor-tg
echo "\$VER" > /etc/netductor/VERSION
systemctl start netductor-api netductor-telegram-bot
netductor version
netductor stack watchdog-install 2>/dev/null || true
EOF
```

## Secondary

```bash
VER=0.9.121
ssh -i ~/.ssh/netductor -p 52222 root@SECONDARY_IP bash -s <<EOF
set -euo pipefail
VER=$VER
cd /tmp
curl -fsSL -o netductor "https://github.com/PavelNeyman/netductor/releases/download/v\${VER}/netductor-linux-amd64"
curl -fsSL -o netductor-agent "https://github.com/PavelNeyman/netductor/releases/download/v\${VER}/netductor-agent-linux-amd64"
chmod 755 netductor netductor-agent
install -m 755 netductor /usr/local/bin/netductor
install -m 755 netductor-agent /usr/local/bin/netductor-agent
systemctl restart netductor-secondary-agent sing-box 2>/dev/null || true
netductor version 2>/dev/null || true
EOF
```

## Mac

```bash
brew reinstall netductor
netductor-op version
```

Конфиги и VPN-пользователи не трогаются. Подробности и проверки — EN-версия этого файла.
