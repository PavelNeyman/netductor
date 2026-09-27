**RU** · [EN](../RUNBOOK-INSTALL-RECOVER.md)

# Runbook: install / update / recover

## Чистый primary (Mac op)
1. TUI/Web wizard primary: IP, password, hostname, SNI, bot token, admin id, domain/LE optional
2. Harden SSH 52222 + operator key
3. COMPONENTS: sing-box, blocky, api, tg, backup…
4. Credentials → `~/.netductor/credentials/`

## Secondary
1. Wizard secondary: IP, password, primary URL/IP, agent join
2. Reality inbound + uplink к primary; agent → :8789 via SP когда есть
3. Тот же Mac pubkey

## Update
`netductor update` / agent pin с Release; op через brew с **реальными SHA**.

## Recover
См. [RECOVER-DRILL.md](RECOVER-DRILL.md). Ключ бэкапа offline. После recover — vpn apply + Reality checklist.

## Dual-node smoke (следующий шаг владельца)
Оба VPS с нуля или после recover: doctor, VPN client, agent heartbeat, backup pull, api-public toggle, Users→SP policy sync.
