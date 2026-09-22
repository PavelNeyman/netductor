# Open items

Baseline: **v0.8.50**.

## Gates
- restore-drill after primary wipe (`recover --from-secondary`)

## Operator / hardware
1. Hardware e2e (OpenWrt guest, Tapo, MikroTik)
2. Domain + HTTPS (P1 skipped for now)
3. SMTP when mailbox exists (P1 skipped)

## Optional later
- Path B user-bot
- More FormT coverage for long wizard strings

## Done (0.8.49–0.8.50)
- Agent backup_pull + recovery :8790
- Offline secondary alert on backup
- Recovery bind/UFW/CIDR controls
- Harden-last; no Mac key on primary
