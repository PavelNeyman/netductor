# Open items

Baseline: **v0.8.37**.

## Gates (green)
`go test` / version pins / git+registry UI parity / GHA-subset runner

## Operator (needs you)
1. **Hardware e2e** — OpenWrt (+ guest), Tapo/NVR, MikroTik+RPi  
2. **Domain + HTTPS** — redirect / Admin TLS  
3. **SMTP alerts** — when mailbox exists  
4. **Restore-drill** — full `.ndenc` on clean VPS  

## Done (do not re-open)
- Thin git + shell pipelines + GHA-subset workflows
- **Isolated CI** in docker/podman (host = data only; no host language SDKs) (`docs/SELFHOST-GIT.md`)
- Local OCI registry + crane + catalog tags + optional htpasswd
- Pipeline/workflow **artifacts**; doctor git/registry checks
- CLI ↔ Admin ↔ TG parity for git/registry
- mTLS revoke/rotate, plane rate-limit/ban, SSH 52222 + fail2ban
- Guest Wi‑Fi software path (hardware e2e still open)
- TUI single deploy path; allowlist removed for edge NAT
- Path B user-bot — **dropped**

## Ideas (not scheduled)
- **Guest Wi‑Fi seller TG bot** — staff grant/deny/TTL without PIN / without being on guest SSID; separate bot token optional

## Deferred (low)
- Status-without-VPN  
- Messenger eval  
- Residual “relay” string cosmetics in rare docs  
- GHA `matrix` / full Actions compatibility  

## Security notes
- Agent plane `:8789` mTLS; plain `:8788` only `NETDUCTOR_PLAIN_AGENT=1`
- Registry default `127.0.0.1:5000`; auth optional if exposed
- Git over SSH key-only (52222); API/Admin/TG require session/ACL
- Artifact reads path-contained under `git-artifacts/`
- API systemd restart allowlist: `netductor-*` only
