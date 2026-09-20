# Runbook — день 1

База **v0.8.1**. Полный текст: [RUNBOOK.md](../RUNBOOK.md).

Кратко:

1. `NETDUCTOR_VERSION=0.8.1` + `bootstrap.sh` → `netductor install` → `doctor`
2. TG secrets · VPN link (`vless` / `hy2`)
3. Secondary: `netductor fleet provision-secondary --host RU_IP …` · `secondary sync`
4. Admin только через VPN/туннель

CLI `relay` **нет** — используйте `secondary` / `fleet`.
