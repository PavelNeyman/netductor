# Open items

Baseline: **v0.8.48**.

## Gates
`go test` · primary/secondary e2e (this cycle)

## Operator
1. Hardware e2e (OpenWrt/guest, Tapo, MikroTik)
2. Domain + HTTPS
3. SMTP when mailbox exists
4. Restore-drill `.ndenc`

## Done this cycle
- Auto deploy primary: lampac + registry + git
- Secondary re-provision with SSH key after password off
- Post-harden SSH port switch in DeployPrimary
