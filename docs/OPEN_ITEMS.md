# Open items

Baseline: **v0.8.49**.

## Gates
- `go test ./...`
- primary/secondary e2e after binary upgrade to 0.8.49 (backup_pull + recovery :8790)

## Operator / hardware
1. Hardware e2e (OpenWrt guest Wi‑Fi, Tapo NVR, MikroTik site)
2. Domain + HTTPS (redirect / admin TLS)
3. SMTP alerts when mailbox available
4. Full restore-drill: kill primary → `recover --from-secondary` on new VPS

## Optional later
- `backup_pull` metrics/alerts in TG if secondary offline during backup
- Firewall secondary :8790 to operator IPs only
- Path B user-bot (deferred)
- UI FormT polish for long CLI strings

## Done (recent)
- Harden-last secondary provision; no Mac key on primary
- Agent backup_pull + recovery API
- SyncComponentsFromDisk (lampac/git/registry)
- Fleet SetSecondary from secondary store
- Lampac/registry/git on primary deploy path
