# Установка (Netductor)

Debian-подобный VPS, root. Релиз: **v0.8.1**.

```bash
export NETDUCTOR_VERSION=0.8.1
curl -fsSL https://raw.githubusercontent.com/PavelNeyman/netductor/main/bootstrap.sh | bash
netductor install
# или
netductor tui --mode vps
```

Полный EN: [INSTALL.md](../INSTALL.md) · дальше [RUNBOOK.md](../RUNBOOK.md), [DEPLOY.md](../DEPLOY.md), [FLEET.md](../FLEET.md).

После `install` парольный SSH отключается — [SSH.md](../SSH.md).
