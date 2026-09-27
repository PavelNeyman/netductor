**RU** · [EN](../DEPLOY-MAC.md)

# Деплой с Mac

## Бинарник
`netductor-op` — workstation. Не путать с `netductor` на VPS.

## UI
- **TUI** — вкладки, мастер деплоя, settings (remote host, key)
- **Web** — localhost, session Bearer / tunnel к :8787

## Сценарий флота
1. Primary: SSH password → install COMPONENTS → harden 52222 → key only
2. Secondary: provision agent + Reality + uplink
3. Domain/LE → redirect :8443
4. Addons по выбору

## Remote mode
TUI может управлять уже стоящей VPS по SSH (не только локальный apply).

## Credentials
Сбор секретов всех нод на Mac после успеха — [OPERATOR_CREDENTIALS](OPERATOR_CREDENTIALS.md).

## Brew
[BREW.md](BREW.md) — только реальные SHA256.
