# Open items

Baseline: **v0.8.29**.

## Gates (green)
go test / go vet / version pins / UI parity (core)

## Operator (needs you)
1. **Hardware e2e** — OpenWrt (incl. guest), Tapo/NVR, MikroTik+RPi  
2. **Domain + HTTPS** — redirect / Admin TLS when domain exists  
3. **SMTP alerts** — when mailbox is available  
4. **Restore-drill** — full `.ndenc` on clean VPS after major change  

## Soft / optional product
- TG guest wait for `cmd_result` — **done** (0.8.29)  
- Cert observability (doctor + TG Certs + API) — **done** / polish  
- Plane `:8789` rate-limit + temp ban after repeated 429 — **done** (`PlaneLimiter`)  
- Status-without-VPN — **deferred**  
- Path B end-user bot — **dropped** (not planned)  
- Messenger (SimpleX etc.) — **deferred**, not core  

## Self-host git
See [SELFHOST-GIT.md](SELFHOST-GIT.md). Prefer **bare git+SSH** for single operator; Forgejo only if web/CI UI needed.

## Self-host git (old) (discussion, not in tree)
Prefer **Forgejo** (or Gitea) on primary **behind VPN only**:
- Git + issues/PRs  
- Built-in Actions or **Woodpecker** for CI  
- Container registry (Forgejo/Gitea registry) or **Harbor**/minimal registry  
Avoid full GitLab unless you need its full DevOps suite (heavier RAM).

## Tech debt
- Residual “relay” strings in rare docs/one-liners → secondary  
- Admin residual EN-only JS strings  
- TUI dead-path cleanup (`bubbles/list` dual paths) — low priority  
- Long doctor/CLI lines → cli18n growth  

## Security stance (:8789)
mTLS required for useful traffic. Rate-limit (180/min/IP) + ban after N consecutive 429 (default 5 → 15 min ban). Prefer this over public fail2ban on SSH alone; do not IP-allowlist edge (NAT). Optional: tighter cloud SG + `NETDUCTOR_PLANE_BAN_*` env.


## Git (thin)
`netductor git` v0.8.29 — bare only; pipeline UI later. See SELFHOST-GIT.md.
