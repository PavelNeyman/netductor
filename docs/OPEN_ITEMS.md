# Open items

Baseline: **v0.8.31**.

## Gates (green)
`go test` / version pins / core UI parity

## Operator (needs you)
1. **Hardware e2e** — OpenWrt (+ guest), Tapo/NVR, MikroTik+RPi  
2. **Domain + HTTPS** — redirect / Admin TLS  
3. **SMTP alerts** — when mailbox exists  
4. **Restore-drill** — full `.ndenc` on clean VPS  

## Done recently (do not re-open)
- Guest Wi‑Fi software path (hardware e2e still open)
- TG guest wait `cmd_result`; mTLS certs in doctor/TG
- Plane `:8789` rate-limit + ban; SSH **52222** + fail2ban
- Thin git: CLI + API + Admin + TG + pipelines (`docs/SELFHOST-GIT.md`)
- Path B user-bot — **dropped**

## Deferred
- Status-without-VPN  
- Messenger eval (SimpleX etc.) — not core  
- ~~Optional local OCI registry + crane~~ **done** (v0.8.31)  
- TUI dual-path cleanup (`bubbles/list`) — low priority  
- Residual “relay” string cosmetics in rare docs  

## Security (:8789)
mTLS required for useful traffic; 180 req/min/IP; ban after repeated 429. No IP allowlist for edge (NAT).
